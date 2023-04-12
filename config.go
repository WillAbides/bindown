package bindown

import (
	"context"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/willabides/bindown/v3/internal/cache"
	"gopkg.in/yaml.v2"
)

// Config is our main config
type Config struct {
	Cache           string                 `json:"cache,omitempty" yaml:"cache,omitempty"`
	TrustCache      bool                   `json:"trust_cache,omitempty" yaml:"trust_cache,omitempty"`
	InstallDir      string                 `json:"install_dir,omitempty" yaml:"install_dir,omitempty"`
	Systems         []SystemInfo           `json:"systems,omitempty" yaml:"systems,omitempty"`
	Dependencies    map[string]*Dependency `json:"dependencies,omitempty" yaml:",omitempty"`
	Templates       map[string]*Dependency `json:"templates,omitempty" yaml:",omitempty"`
	TemplateSources map[string]string      `json:"template_sources,omitempty" yaml:"template_sources,omitempty"`
	URLChecksums    map[string]string      `json:"url_checksums,omitempty" yaml:"url_checksums,omitempty"`
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
func (c *Config) BinName(depName string, system SystemInfo) (string, error) {
	dep, err := c.buildDependency(depName, system)
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
func (c *Config) BuildDependency(depName string, info SystemInfo) (*Dependency, error) {
	dep, err := c.buildDependency(depName, info)
	if err != nil {
		return nil, err
	}
	return &dep.Dependency, nil
}

func (c *Config) buildDependency(depName string, info SystemInfo) (*builtDependency, error) {
	dep := c.Dependencies[depName]
	if dep == nil {
		return nil, fmt.Errorf("no dependency configured with the name %q", depName)
	}
	dep = dep.clone()
	err := dep.applyTemplate(c.Templates, 0)
	if err != nil {
		return nil, err
	}
	dep.applyOverrides(info, 0)
	if dep.Vars == nil {
		dep.Vars = map[string]string{}
	}
	if _, ok := dep.Vars["os"]; !ok {
		dep.Vars["os"] = info.OS
	}
	if _, ok := dep.Vars["arch"]; !ok {
		dep.Vars["arch"] = info.Arch
	}
	dep.Vars = varsWithSubstitutions(dep.Vars, dep.Substitutions)
	err = dep.interpolateVars(info)
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
	return &builtDependency{
		Dependency: *dep,
		name:       depName,
		system:     info,
		checksum:   checksum,
		url:        *dep.URL,
	}, nil
}

func (c *Config) allDependencyNames() []string {
	if len(c.Dependencies) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(c.Dependencies))
	for dl := range c.Dependencies {
		result = append(result, dl)
	}
	return result
}

// ConfigAddChecksumsOptions contains options for Config.AddChecksums
type ConfigAddChecksumsOptions struct {
	// Only add checksums for these dependencies. When Dependencies is empty, AddChecksums adds checksums for all
	// configured dependencies.
	Dependencies []string

	// Only add checksums for these system targets. When Systems is empty, AddChecksums adds checksums for all known
	// builds configured for each dependency.
	Systems []SystemInfo
}

// DefaultSystems returns c.Systems if it isn't empty. Otherwise returns the runtime system.
func (c *Config) DefaultSystems() []SystemInfo {
	if len(c.Systems) > 0 {
		return c.Systems
	}
	return []SystemInfo{
		{
			OS:   runtime.GOOS,
			Arch: runtime.GOARCH,
		},
	}
}

// AddChecksums downloads, calculates checksums and adds them to the config's URLChecksums. AddChecksums skips urls that
// already exist in URLChecksums.
func (c *Config) AddChecksums(dependencies []string, systems []SystemInfo) error {
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
			var dep *builtDependency
			dep, err = c.buildDependency(depName, system)
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

func (c *Config) addChecksum(dependencyName string, sysInfo SystemInfo) error {
	dep, err := c.buildDependency(dependencyName, sysInfo)
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

// ConfigValidateOptions contains options for Config.Validate
type ConfigValidateOptions struct {
	// Only validates these dependencies. When Dependencies is empty, Validate validates all configured dependencies.
	Dependencies []string

	// Only validates system targets. When Systems is empty, AddChecksums validates all known builds configured for each
	// dependency.
	Systems []SystemInfo
}

// Validate installs the downloader to a temporary directory and returns an error if it was unsuccessful.
func (c *Config) Validate(dependencies []string, systems []SystemInfo) (errOut error) {
	runtime.Version()
	if len(dependencies) == 0 {
		dependencies = c.allDependencyNames()
	}
	tmpCacheDir, err := os.MkdirTemp("", "bindown-cache")
	if err != nil {
		return err
	}
	defer deferErr(&errOut, func() error {
		cleanErr := filepath.WalkDir(tmpCacheDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			err = os.Chmod(path, 0o777)
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			return os.Remove(path)
		})
		if cleanErr != nil {
			return cleanErr
		}
		return os.RemoveAll(tmpCacheDir)
	})
	tmpBinDir, err := os.MkdirTemp("", "bindown-bin")
	if err != nil {
		return err
	}
	defer deferErr(&errOut, func() error {
		return os.RemoveAll(tmpBinDir)
	})
	c.InstallDir = tmpBinDir
	c.Cache = tmpCacheDir
	for _, depName := range dependencies {
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
	TargetFile           string
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
	sysInfo SystemInfo,
	opts *ConfigDownloadDependencyOpts,
) (_ string, errOut error) {
	if opts == nil {
		opts = &ConfigDownloadDependencyOpts{}
	}
	dep, err := c.buildDependency(name, sysInfo)
	if err != nil {
		return "", err
	}
	dlFile, _, unlock, err := downloadDependency(dep, c.downloadsCache(), c.TrustCache, opts.AllowMissingChecksum, opts.Force)
	if err != nil {
		return "", err
	}
	defer deferErr(&errOut, unlock)
	cachedFile := dlFile
	if opts.TargetFile == "" {
		return cachedFile, nil
	}
	err = os.MkdirAll(filepath.Dir(opts.TargetFile), 0o755)
	if err != nil {
		return "", err
	}
	err = copyFile(dlFile, opts.TargetFile)
	if err != nil {
		return "", err
	}
	return opts.TargetFile, nil
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
	TargetDirectory      string
	Force                bool
	AllowMissingChecksum bool
}

// ExtractDependency downloads and extracts a dependency
func (c *Config) ExtractDependency(dependencyName string, sysInfo SystemInfo, opts *ConfigExtractDependencyOpts) (_ string, errOut error) {
	if opts == nil {
		opts = &ConfigExtractDependencyOpts{}
	}
	dep, err := c.buildDependency(dependencyName, sysInfo)
	if err != nil {
		return "", err
	}
	dlFile, key, dlUnlock, err := downloadDependency(
		dep,
		c.downloadsCache(),
		c.TrustCache,
		opts.AllowMissingChecksum,
		opts.Force,
	)
	if err != nil {
		return "", err
	}
	defer deferErr(&errOut, dlUnlock)

	if opts.TargetDirectory != "" {
		err = extract(dlFile, opts.TargetDirectory)
		if err != nil {
			return "", err
		}
		return opts.TargetDirectory, nil
	}
	outDir, unlock, err := extractDependencyToCache(
		dlFile,
		c.Cache,
		key,
		c.extractsCache(),
		c.TrustCache,
		opts.Force,
	)
	if err != nil {
		return "", err
	}
	defer deferErr(&errOut, unlock)
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
func (c *Config) InstallDependency(dependencyName string, sysInfo SystemInfo, opts *ConfigInstallDependencyOpts) (_ string, errOut error) {
	if opts == nil {
		opts = &ConfigInstallDependencyOpts{}
	}
	dep, err := c.buildDependency(dependencyName, sysInfo)
	if err != nil {
		return "", err
	}
	dlFile, key, dlUnlock, err := downloadDependency(
		dep,
		c.downloadsCache(),
		c.TrustCache,
		opts.AllowMissingChecksum,
		opts.Force,
	)
	if err != nil {
		return "", err
	}
	defer deferErr(&errOut, dlUnlock)

	extractDir, exUnlock, err := extractDependencyToCache(
		dlFile,
		c.Cache,
		key,
		c.extractsCache(),
		c.TrustCache,
		opts.Force,
	)
	if err != nil {
		return "", err
	}
	defer deferErr(&errOut, exUnlock)
	targetPath := opts.TargetPath
	if targetPath == "" {
		var binName string
		binName, err = c.BinName(dependencyName, sysInfo)
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

// AddDependencyFromTemplate adds a dependency to the config
func (c *Config) AddDependencyFromTemplate(ctx context.Context, templateName string, opts *AddDependencyFromTemplateOpts) error {
	if opts == nil {
		opts = new(AddDependencyFromTemplateOpts)
	}
	dependencyName := opts.DependencyName
	if dependencyName == "" {
		dependencyName = strings.Split(templateName, "#")[0]
	}
	if c.Dependencies == nil {
		c.Dependencies = map[string]*Dependency{}
	}
	if c.Dependencies[dependencyName] != nil {
		return fmt.Errorf("dependency named %q already exists", dependencyName)
	}
	templateName, err := c.addOrGetTemplate(ctx, templateName, opts.TemplateSource)
	if err != nil {
		return err
	}
	dep := new(Dependency)
	dep.Vars = opts.Vars
	dep.Template = &templateName
	c.Dependencies[dependencyName] = dep
	return nil
}

func (c *Config) addOrGetTemplate(ctx context.Context, name, src string) (string, error) {
	destName := name
	if src != "" {
		destName = fmt.Sprintf("%s#%s", src, name)
	}
	if _, ok := c.Templates[destName]; ok {
		return destName, nil
	}
	if src == "" {
		return "", fmt.Errorf("no template named %q", name)
	}
	tmplSrc := src
	tmplSrcs := c.TemplateSources
	if tmplSrcs == nil {
		tmplSrcs = map[string]string{}
	}
	if _, ok := tmplSrcs[tmplSrc]; ok {
		tmplSrc = tmplSrcs[tmplSrc]
	}
	err := c.addTemplateFromSource(ctx, tmplSrc, name, destName)
	if err != nil {
		return "", err
	}
	return destName, nil
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
	return c.addTemplateFromSource(ctx, tmplSrc, srcTemplate, destName)
}

// addTemplateFromSource copies a template from another config file
func (c *Config) addTemplateFromSource(ctx context.Context, src, srcTemplate, destName string) error {
	srcCfg, err := ConfigFromURL(ctx, src)
	if err != nil {
		return err
	}
	tmpl := srcCfg.Templates[srcTemplate]
	if tmpl == nil {
		return fmt.Errorf("source has no template named %q", srcTemplate)
	}
	if c.Templates == nil {
		c.Templates = map[string]*Dependency{}
	}
	c.Templates[destName] = tmpl
	return nil
}

func (c *Config) templatesList() []string {
	var templates []string
	for tmpl := range c.Templates {
		templates = append(templates, tmpl)
	}
	sort.Strings(templates)
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
	return ConfigFromURL(ctx, c.TemplateSources[name])
}

// DependencySystems returns the supported systems of either the config or the dependency if one is not empty
// if both are not empty, it returns the intersection of the lists
func (c *Config) DependencySystems(depName string) ([]SystemInfo, error) {
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
		return c.DefaultSystems(), nil
	}
	if len(c.Systems) == 0 {
		return dep.Systems, nil
	}
	mp := make(map[SystemInfo]bool, len(c.Systems))
	for _, system := range c.Systems {
		mp[system] = true
	}
	result := make([]SystemInfo, 0, len(dep.Systems))
	for _, system := range dep.Systems {
		if mp[system] {
			result = append(result, system)
		}
	}
	return result, nil
}

// ConfigFromURL loads a config from a URL
func ConfigFromURL(ctx context.Context, cfgSrc string) (*Config, error) {
	cfgURL, err := url.Parse(cfgSrc)
	if err != nil {
		return nil, err
	}
	switch cfgURL.Scheme {
	case "", "file":
		cfg, err := LoadConfigFile(ctx, cfgURL.Path, true)
		if err != nil {
			return nil, err
		}
		return &cfg.Config, nil
	case "http", "https":
		return configFromHTTP(ctx, cfgSrc)
	default:
		return nil, fmt.Errorf("invalid src: %s", cfgSrc)
	}
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
	return configFromYAML(ctx, data)
}

func configFromYAML(ctx context.Context, data []byte) (*Config, error) {
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
