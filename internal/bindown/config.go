package bindown

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/willabides/bindown/v4/internal/cache"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// The directory where bindown will cache downloads and extracted files. This is relative to the directory where
	// the configuration file resides. cache paths should always use / as a delimiter even on Windows or other
	// operating systems where the native delimiter isn't /.
	Cache string `json:"cache,omitempty" yaml:"cache,omitempty"`

	// The directory that bindown installs files to. This is relative to the directory where the configuration file
	// resides. install_directory paths should always use / as a delimiter even on Windows or other operating systems
	// where the native delimiter isn't /.
	InstallDir string `json:"install_dir,omitempty" yaml:"install_dir,omitempty"`

	// List of systems supported by this config. Systems are in the form of os/architecture.
	Systems []System `json:"systems,omitempty" yaml:"systems,omitempty"`

	// Dependencies available for bindown to install.
	Dependencies map[string]*Dependency `json:"dependencies,omitempty" yaml:",omitempty"`

	// Templates that can be used by dependencies in this file.
	Templates map[string]*Dependency `json:"templates,omitempty" yaml:",omitempty"`

	// Upstream sources for templates.
	TemplateSources map[string]string `json:"template_sources,omitempty" yaml:"template_sources,omitempty"`

	// Checksums of downloaded files.
	URLChecksums map[string]string `json:"url_checksums,omitempty" yaml:"url_checksums,omitempty"`

	Filename string `json:"-" yaml:"-"`
}

// UnsetDependencyVars removes a dependency var. Noop if the var doesn't exist.
func (c *Config) UnsetDependencyVars(depName string, vars []string) error {
	dep := c.Dependencies[depName]
	if dep == nil {
		return fmt.Errorf("dependency %q does not exist", depName)
	}
	if dep.Vars == nil {
		return nil
	}
	for _, v := range vars {
		delete(dep.Vars, v)
	}
	return nil
}

// SetDependencyVars sets the value of a dependency's var. Adds or Updates the var.
func (c *Config) SetDependencyVars(depName string, vars map[string]string) error {
	dep := c.Dependencies[depName]
	if dep == nil {
		return fmt.Errorf("dependency %q does not exist", depName)
	}
	if dep.Vars == nil {
		dep.Vars = map[string]string{}
	}
	for k, v := range vars {
		dep.Vars[k] = v
	}
	return nil
}

// UnsetTemplateVars removes a template var. Noop if the var doesn't exist.
func (c *Config) UnsetTemplateVars(tmplName string, vars []string) error {
	tmpl := c.Templates[tmplName]
	if tmpl == nil {
		return fmt.Errorf("template %q does not exist", tmplName)
	}
	if tmpl.Vars == nil {
		return nil
	}
	for _, v := range vars {
		delete(tmpl.Vars, v)
	}
	return nil
}

// SetTemplateVars sets the value of a template's var. Adds or Updates the var.
func (c *Config) SetTemplateVars(tmplName string, vars map[string]string) error {
	tmpl := c.Templates[tmplName]
	if tmpl == nil {
		return fmt.Errorf("template %q does not exist", tmplName)
	}
	if tmpl.Vars == nil {
		tmpl.Vars = map[string]string{}
	}
	for k, v := range vars {
		tmpl.Vars[k] = v
	}
	return nil
}

// BinName returns the bin name for a downloader on a given system
func (c *Config) BinName(depName string, system System) (string, error) {
	dep, err := c.BuildDependency(depName, system)
	if err != nil {
		return "", err
	}
	if dep.BinName != nil && *dep.BinName != "" {
		return *dep.BinName, nil
	}
	return depName, nil
}

// MissingDependencyVars returns a list of vars that are required but undefined
func (c *Config) MissingDependencyVars(depName string) ([]string, error) {
	dep := c.Dependencies[depName]
	if dep == nil {
		return nil, fmt.Errorf("no dependency configured with the name %q", depName)
	}
	var result []string
	dep = dep.clone()
	err := dep.applyTemplate(c.Templates, 0)
	if err != nil {
		return nil, err
	}
	if dep.Vars == nil {
		return dep.RequiredVars, nil
	}
	for _, requiredVar := range dep.RequiredVars {
		if _, ok := dep.Vars[requiredVar]; !ok {
			result = append(result, requiredVar)
		}
	}
	return result, nil
}

// BuildDependency returns a dependency with templates and overrides applied and variables interpolated for the given system.
func (c *Config) BuildDependency(depName string, system System) (*Dependency, error) {
	dep := c.Dependencies[depName]
	if dep == nil {
		return nil, fmt.Errorf("no dependency configured with the name %q", depName)
	}
	dep = dep.clone()
	err := dep.applyTemplate(c.Templates, 0)
	if err != nil {
		return nil, err
	}
	err = dep.applyOverrides(system, 0)
	if err != nil {
		return nil, err
	}
	if dep.Vars == nil {
		dep.Vars = map[string]string{}
	}
	if _, ok := dep.Vars["os"]; !ok {
		dep.Vars["os"] = system.OS()
	}
	if _, ok := dep.Vars["arch"]; !ok {
		dep.Vars["arch"] = system.Arch()
	}
	dep.Vars = varsWithSubstitutions(dep.Vars, dep.Substitutions)
	err = dep.interpolateVars(system)
	if err != nil {
		return nil, err
	}
	if dep.URL == nil {
		return nil, fmt.Errorf("dependency %q has no URL", depName)
	}
	checksum := ""
	if c.URLChecksums != nil && dep.URL != nil {
		checksum = c.URLChecksums[*dep.URL]
	}
	dep.built = true
	dep.name = depName
	dep.system = system
	dep.checksum = checksum
	dep.url = *dep.URL
	return dep, nil
}

// defaultSystems returns c.Systems if it isn't empty. Otherwise returns the runtime system.
func (c *Config) defaultSystems() []System {
	if len(c.Systems) > 0 {
		return c.Systems
	}
	return []System{CurrentSystem}
}

// AddChecksums downloads, calculates checksums and adds them to the config's URLChecksums. AddChecksums skips urls that
// already exist in URLChecksums.
func (c *Config) AddChecksums(dependencies []string, systems []System) error {
	if len(dependencies) == 0 && c.Dependencies != nil {
		dependencies = make([]string, 0, len(c.Dependencies))
		for dlName := range c.Dependencies {
			dependencies = append(dependencies, dlName)
		}
	}
	var err error
	for _, depName := range dependencies {
		depSystems := systems
		if len(depSystems) == 0 {
			depSystems, err = c.DependencySystems(depName)
			if err != nil {
				return err
			}
		}
		dp := c.Dependencies[depName]
		if dp == nil {
			return fmt.Errorf("no dependency configured with the name %q", depName)
		}
		for _, system := range depSystems {
			err = c.addChecksum(depName, system)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// PruneChecksums removes checksums for dependencies that are not used by any configured system.
func (c *Config) PruneChecksums() error {
	allURLS := make(map[string]bool, len(c.Dependencies)*8)
	for depName := range c.Dependencies {
		systems, err := c.DependencySystems(depName)
		if err != nil {
			return err
		}
		for _, system := range systems {
			var dep *Dependency
			dep, err = c.BuildDependency(depName, system)
			if err != nil {
				return err
			}
			allURLS[dep.url] = true
		}
	}
	for u := range c.URLChecksums {
		if !allURLS[u] {
			delete(c.URLChecksums, u)
		}
	}
	return nil
}

func (c *Config) addChecksum(dependencyName string, system System) error {
	dep, err := c.BuildDependency(dependencyName, system)
	if err != nil {
		return err
	}
	existingSum := c.URLChecksums[dep.url]
	if existingSum != "" {
		return nil
	}
	sum, err := getURLChecksum(dep.url, "")
	if err != nil {
		return err
	}
	if c.URLChecksums == nil {
		c.URLChecksums = make(map[string]string, 1)
	}
	c.URLChecksums[dep.url] = sum
	return nil
}

// Validate installs the downloader to a temporary directory and returns an error if it was unsuccessful.
func (c *Config) Validate(depName string, systems []System) (errOut error) {
	tmpDir, err := os.MkdirTemp("", "bindown-validate")
	if err != nil {
		return err
	}
	defer deferErr(&errOut, func() error {
		return os.RemoveAll(tmpDir)
	})
	installDir, cacheDir := c.InstallDir, c.Cache
	c.InstallDir = filepath.Join(tmpDir, "bin")
	c.Cache = filepath.Join(tmpDir, "cache")
	defer func() {
		c.InstallDir, c.Cache = installDir, cacheDir
	}()
	depSystems := systems
	if len(depSystems) == 0 {
		depSystems, err = c.DependencySystems(depName)
		if err != nil {
			return err
		}
	}
	for _, system := range depSystems {
		_, err = c.InstallDependency(depName, system, &ConfigInstallDependencyOpts{
			Force: true,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) ClearCache() error {
	err := cache.RemoveRoot(c.downloadsCache().Root)
	if err != nil {
		return err
	}
	err = cache.RemoveRoot(c.extractsCache().Root)
	if err != nil {
		return err
	}
	return os.RemoveAll(c.Cache)
}

// ConfigDownloadDependencyOpts options for Config.DownloadDependency
type ConfigDownloadDependencyOpts struct {
	Force                bool
	AllowMissingChecksum bool
}

func (c *Config) downloadsCache() *cache.Cache {
	return &cache.Cache{
		Root: filepath.Join(c.Cache, "downloads"),
	}
}

func (c *Config) extractsCache() *cache.Cache {
	return &cache.Cache{
		Root: filepath.Join(c.Cache, "extracts"),
	}
}

func cacheKey(hashMaterial string) string {
	hasher := fnv.New64a()
	mustWriteToHash(hasher, []byte(hashMaterial))
	return hex.EncodeToString(hasher.Sum(nil))
}

// DownloadDependency downloads a dependency
func (c *Config) DownloadDependency(
	name string,
	system System,
	opts *ConfigDownloadDependencyOpts,
) (_ string, errOut error) {
	if opts == nil {
		opts = &ConfigDownloadDependencyOpts{}
	}
	dep, err := c.BuildDependency(name, system)
	if err != nil {
		return "", err
	}
	dlFile, _, unlock, err := downloadDependency(dep, c.downloadsCache(), opts.AllowMissingChecksum, opts.Force)
	if err != nil {
		return "", err
	}
	err = unlock()
	if err != nil {
		return "", err
	}
	return dlFile, nil
}

func urlFilename(dlURL string) (string, error) {
	u, err := url.Parse(dlURL)
	if err != nil {
		return "", err
	}
	return path.Base(u.EscapedPath()), nil
}

// ConfigExtractDependencyOpts options for Config.ExtractDependency
type ConfigExtractDependencyOpts struct {
	Force                bool
	AllowMissingChecksum bool
}

// ExtractDependency downloads and extracts a dependency
func (c *Config) ExtractDependency(dependencyName string, system System, opts *ConfigExtractDependencyOpts) (_ string, errOut error) {
	if opts == nil {
		opts = &ConfigExtractDependencyOpts{}
	}
	dep, err := c.BuildDependency(dependencyName, system)
	if err != nil {
		return "", err
	}
	dlFile, key, dlUnlock, err := downloadDependency(dep, c.downloadsCache(), opts.AllowMissingChecksum, opts.Force)
	if err != nil {
		return "", err
	}
	defer deferErr(&errOut, dlUnlock)

	outDir, unlock, err := extractDependencyToCache(dlFile, c.Cache, key, c.extractsCache(), opts.Force)
	if err != nil {
		return "", err
	}
	err = unlock()
	if err != nil {
		return "", err
	}
	return outDir, nil
}

// ConfigInstallDependencyOpts provides options for Config.InstallDependency
type ConfigInstallDependencyOpts struct {
	// TargetPath is the path where the executable should end up
	TargetPath string
	// Force - install even if it already exists
	Force bool
	// AllowMissingChecksum - whether to allow missing checksum
	AllowMissingChecksum bool
}

// InstallDependency downloads, extracts and installs a dependency
func (c *Config) InstallDependency(dependencyName string, system System, opts *ConfigInstallDependencyOpts) (_ string, errOut error) {
	if opts == nil {
		opts = &ConfigInstallDependencyOpts{}
	}
	dep, err := c.BuildDependency(dependencyName, system)
	if err != nil {
		return "", err
	}
	dlFile, key, dlUnlock, err := downloadDependency(dep, c.downloadsCache(), opts.AllowMissingChecksum, opts.Force)
	if err != nil {
		return "", err
	}
	defer deferErr(&errOut, dlUnlock)

	extractDir, exUnlock, err := extractDependencyToCache(dlFile, c.Cache, key, c.extractsCache(), opts.Force)
	if err != nil {
		return "", err
	}
	defer deferErr(&errOut, exUnlock)
	targetPath := opts.TargetPath
	if targetPath == "" {
		var binName string
		binName, err = c.BinName(dependencyName, system)
		if err != nil {
			return "", err
		}
		targetPath = filepath.Join(c.InstallDir, binName)
	}
	return install(dep, targetPath, extractDir)
}

// AddDependencyFromTemplateOpts options for AddDependencyFromTemplate
type AddDependencyFromTemplateOpts struct {
	TemplateSource string
	DependencyName string
	Vars           map[string]string
}

// AddDependencyFromTemplate adds a dependency to the config. Returns a map of known values for template vars
func (c *Config) AddDependencyFromTemplate(ctx context.Context, templateName string, opts *AddDependencyFromTemplateOpts) (*Dependency, map[string][]string, error) {
	if opts == nil {
		opts = &AddDependencyFromTemplateOpts{}
	}
	dependencyName := opts.DependencyName
	if dependencyName == "" {
		dependencyName = strings.Split(templateName, "#")[0]
	}
	if c.Dependencies == nil {
		c.Dependencies = map[string]*Dependency{}
	}
	if c.Dependencies[dependencyName] != nil {
		return nil, nil, fmt.Errorf("dependency named %q already exists", dependencyName)
	}
	templateName, varVals, err := c.addOrGetTemplate(ctx, templateName, opts.TemplateSource)
	if err != nil {
		return nil, nil, err
	}
	dep := &Dependency{
		Overrideable: Overrideable{
			Vars: opts.Vars,
		},
		Template: &templateName,
	}
	c.Dependencies[dependencyName] = dep
	return dep, varVals, nil
}

func (c *Config) addOrGetTemplate(ctx context.Context, name, src string) (destName string, varVals map[string][]string, _ error) {
	destName = name
	if src != "" {
		destName = fmt.Sprintf("%s#%s", src, name)
	}
	if _, ok := c.Templates[destName]; ok {
		return destName, nil, nil
	}
	if src == "" {
		return "", nil, fmt.Errorf("no template named %q", name)
	}
	tmplSrc := src
	tmplSrcs := c.TemplateSources
	if tmplSrcs == nil {
		tmplSrcs = map[string]string{}
	}
	if _, ok := tmplSrcs[tmplSrc]; ok {
		tmplSrc = tmplSrcs[tmplSrc]
	}
	var err error
	varVals, err = c.addTemplateFromSource(ctx, tmplSrc, name, destName)
	if err != nil {
		return "", nil, err
	}
	return destName, varVals, nil
}

// CopyTemplateFromSource copies a template from source
func (c *Config) CopyTemplateFromSource(ctx context.Context, src, srcTemplate, destName string) error {
	if c.TemplateSources == nil {
		return fmt.Errorf("no template source named %q", src)
	}
	tmplSrc := c.TemplateSources[src]
	if tmplSrc == "" {
		return fmt.Errorf("no template source named %q", src)
	}
	_, err := c.addTemplateFromSource(ctx, tmplSrc, srcTemplate, destName)
	return err
}

// addTemplateFromSource copies a template from another config file
func (c *Config) addTemplateFromSource(ctx context.Context, src, srcTemplate, destName string) (map[string][]string, error) {
	srcCfg, err := NewConfig(ctx, src, true)
	if err != nil {
		return nil, err
	}
	tmpl := srcCfg.Templates[srcTemplate]
	if tmpl == nil {
		return nil, fmt.Errorf("source has no template named %q", srcTemplate)
	}
	varVals := map[string][]string{}
	for _, dep := range srcCfg.Dependencies {
		if dep.Template == nil || *dep.Template != srcTemplate {
			continue
		}
		for k, v := range dep.Vars {
			varVals[k] = append(varVals[k], v)
		}
	}
	if c.Templates == nil {
		c.Templates = map[string]*Dependency{}
	}
	c.Templates[destName] = tmpl
	return varVals, nil
}

func (c *Config) templatesList() []string {
	var templates []string
	for tmpl := range c.Templates {
		templates = append(templates, tmpl)
	}
	slices.Sort(templates)
	return templates
}

// ListTemplates lists templates available in this config or one of its template sources.
func (c *Config) ListTemplates(ctx context.Context, templateSource string) ([]string, error) {
	if templateSource == "" {
		return c.templatesList(), nil
	}
	srcCfg, err := c.templateSourceConfig(ctx, templateSource)
	if err != nil {
		return nil, err
	}
	return srcCfg.templatesList(), nil
}

func (c *Config) templateSourceConfig(ctx context.Context, name string) (*Config, error) {
	if c.TemplateSources == nil || c.TemplateSources[name] == "" {
		return nil, fmt.Errorf("no template source named %q", name)
	}
	return NewConfig(ctx, c.TemplateSources[name], true)
}

// DependencySystems returns the supported systems of either the config or the dependency if one is not empty
// if both are not empty, it returns the intersection of the lists
func (c *Config) DependencySystems(depName string) ([]System, error) {
	if c.Dependencies == nil || c.Dependencies[depName] == nil {
		return c.Systems, nil
	}
	dep := c.Dependencies[depName]

	dep = dep.clone()
	err := dep.applyTemplate(c.Templates, 0)
	if err != nil {
		return nil, err
	}

	if len(dep.Systems) == 0 {
		return c.defaultSystems(), nil
	}
	if len(c.Systems) == 0 {
		return dep.Systems, nil
	}
	mp := make(map[System]bool, len(c.Systems))
	for _, system := range c.Systems {
		mp[system] = true
	}
	result := make([]System, 0, len(dep.Systems))
	for _, system := range dep.Systems {
		if mp[system] {
			result = append(result, system)
		}
	}
	return result, nil
}

func (c *Config) WriteFile(outputJSON bool) (errOut error) {
	if c.Filename == "" {
		return fmt.Errorf("no filename specified")
	}
	if filepath.Ext(c.Filename) == ".json" {
		outputJSON = true
	}
	file, err := os.Create(c.Filename)
	if err != nil {
		return err
	}
	defer deferErr(&errOut, file.Close)
	slices.Sort(c.Systems)
	if outputJSON {
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		return encoder.Encode(c)
	}
	return EncodeYaml(file, &c)
}

// NewConfig loads a config from a URL
func NewConfig(ctx context.Context, cfgSrc string, noDefaultDirs bool) (*Config, error) {
	cfgURL, err := url.Parse(cfgSrc)
	if err == nil {
		if cfgURL.Scheme == "http" || cfgURL.Scheme == "https" {
			return configFromHTTP(ctx, cfgSrc)
		}
	}
	data, err := os.ReadFile(cfgSrc)
	if err != nil {
		return nil, err
	}
	cfg, err := ConfigFromYAML(ctx, data)
	if err != nil {
		return nil, err
	}
	cfg.Filename = cfgSrc
	if noDefaultDirs {
		return cfg, nil
	}
	if cfg.Cache == "" {
		cfg.Cache, err = findCacheDir(filepath.Dir(cfgSrc))
		if err != nil {
			return nil, err
		}
	}
	if cfg.InstallDir == "" {
		cfg.InstallDir = filepath.Join(filepath.Dir(cfgSrc), "bin")
	}
	return cfg, nil
}

// findCacheDir decides between .bindown and .cache for the cache directory to use when
// none is specified. This is necessary because v4 mistakenly made .cache the default.
// We want to use .bindown, but will revert to .cache if it is in .gitignore and .bindown
// does not exist.
func findCacheDir(cfgDir string) (string, error) {
	// if .bindown exists, use it
	bindownDir := filepath.Join(cfgDir, ".bindown")
	info, err := os.Stat(bindownDir)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if err == nil && info.IsDir() {
		return bindownDir, nil
	}
	// if .bindown is in .gitignore, use it
	ig, err := dirIsGitIgnored(bindownDir)
	if err != nil {
		return "", err
	}
	if ig {
		return bindownDir, nil
	}
	// if .cache is in .gitignore, use it
	cacheDir := filepath.Join(cfgDir, ".cache")
	ig, err = dirIsGitIgnored(cacheDir)
	if err != nil {
		return "", err
	}
	if ig {
		return cacheDir, nil
	}
	// default to .bindown
	return bindownDir, nil
}

func configFromHTTP(ctx context.Context, src string) (*Config, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", src, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("error downloading %q", src)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return ConfigFromYAML(ctx, data)
}

func ConfigFromYAML(ctx context.Context, data []byte) (*Config, error) {
	err := validateConfig(ctx, data)
	if err != nil {
		return nil, err
	}
	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	cfg.Cache = filepath.FromSlash(cfg.Cache)
	cfg.InstallDir = filepath.FromSlash(cfg.InstallDir)
	return &cfg, nil
}
