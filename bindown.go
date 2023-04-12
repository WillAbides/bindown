package bindown

import (
	"context"

	"github.com/willabides/bindown/v3/internal/bindown"
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
	ic := internalizeConfig(c)
	err := ic.UnsetDependencyVars(depName, vars)
	*c = *externalizeConfig(ic)
	return err
}

// SetDependencyVars sets the value of a dependency's var. Adds or Updates the var.
func (c *Config) SetDependencyVars(depName string, vars map[string]string) error {
	ic := internalizeConfig(c)
	err := ic.SetDependencyVars(depName, vars)
	*c = *externalizeConfig(ic)
	return err
}

// UnsetTemplateVars removes a template var. Noop if the var doesn't exist.
func (c *Config) UnsetTemplateVars(tmplName string, vars []string) error {
	ic := internalizeConfig(c)
	err := ic.UnsetTemplateVars(tmplName, vars)
	*c = *externalizeConfig(ic)
	return err
}

// SetTemplateVars sets the value of a template's var. Adds or Updates the var.
func (c *Config) SetTemplateVars(tmplName string, vars map[string]string) error {
	ic := internalizeConfig(c)
	err := ic.SetTemplateVars(tmplName, vars)
	*c = *externalizeConfig(ic)
	return err
}

// BinName returns the bin name for a downloader on a given system
func (c *Config) BinName(depName string, system SystemInfo) (string, error) {
	return internalizeConfig(c).BinName(depName, bindown.SystemInfo(system))
}

// MissingDependencyVars returns a list of vars that are required but undefined
func (c *Config) MissingDependencyVars(depName string) ([]string, error) {
	return internalizeConfig(c).MissingDependencyVars(depName)
}

// BuildDependency returns a dependency with templates and overrides applied and variables interpolated for the given system.
func (c *Config) BuildDependency(depName string, info SystemInfo) (*Dependency, error) {
	iDep, err := internalizeConfig(c).BuildDependency(depName, bindown.SystemInfo(info))
	if err != nil {
		return nil, err
	}
	return externalizeDependency(iDep), nil
}

// ConfigAddChecksumsOptions was never used but included by mistake. It will be removed in a future release.
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
	return externalizeSystems(internalizeConfig(c).DefaultSystems())
}

// AddChecksums downloads, calculates checksums and adds them to the config's URLChecksums. AddChecksums skips urls that
// already exist in URLChecksums.
func (c *Config) AddChecksums(dependencies []string, systems []SystemInfo) error {
	ic := internalizeConfig(c)
	err := ic.AddChecksums(dependencies, internalizeSystems(systems))
	*c = *externalizeConfig(ic)
	return err
}

// PruneChecksums removes checksums for dependencies that are not used by any configured system.
func (c *Config) PruneChecksums() error {
	ic := internalizeConfig(c)
	err := ic.PruneChecksums()
	*c = *externalizeConfig(ic)
	return err
}

// ConfigValidateOptions was never used but included by mistake. It will be removed in a future release.
type ConfigValidateOptions struct {
	// Only validates these dependencies. When Dependencies is empty, Validate validates all configured dependencies.
	Dependencies []string

	// Only validates system targets. When Systems is empty, AddChecksums validates all known builds configured for each
	// dependency.
	Systems []SystemInfo
}

// Validate installs the downloader to a temporary directory and returns an error if it was unsuccessful.
func (c *Config) Validate(dependencies []string, systems []SystemInfo) (errOut error) {
	return internalizeConfig(c).Validate(dependencies, internalizeSystems(systems))
}

func (c *Config) ClearCache() error {
	return internalizeConfig(c).ClearCache()
}

// ConfigDownloadDependencyOpts options for Config.DownloadDependency
type ConfigDownloadDependencyOpts struct {
	TargetFile           string
	Force                bool
	AllowMissingChecksum bool
}

// DownloadDependency downloads a dependency
func (c *Config) DownloadDependency(
	name string,
	sysInfo SystemInfo,
	opts *ConfigDownloadDependencyOpts,
) (_ string, errOut error) {
	internalOpts := bindown.ConfigDownloadDependencyOpts(*opts)
	return internalizeConfig(c).DownloadDependency(name, bindown.SystemInfo(sysInfo), &internalOpts)
}

// ConfigExtractDependencyOpts options for Config.ExtractDependency
type ConfigExtractDependencyOpts struct {
	TargetDirectory      string
	Force                bool
	AllowMissingChecksum bool
}

// ExtractDependency downloads and extracts a dependency
func (c *Config) ExtractDependency(dependencyName string, sysInfo SystemInfo, opts *ConfigExtractDependencyOpts) (_ string, errOut error) {
	internalOpts := bindown.ConfigExtractDependencyOpts(*opts)
	return internalizeConfig(c).ExtractDependency(dependencyName, bindown.SystemInfo(sysInfo), &internalOpts)
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
	internalOpts := bindown.ConfigInstallDependencyOpts(*opts)
	return internalizeConfig(c).InstallDependency(dependencyName, bindown.SystemInfo(sysInfo), &internalOpts)
}

// AddDependencyFromTemplateOpts options for AddDependencyFromTemplate
type AddDependencyFromTemplateOpts struct {
	TemplateSource string
	DependencyName string
	Vars           map[string]string
}

// AddDependencyFromTemplate adds a dependency to the config
func (c *Config) AddDependencyFromTemplate(ctx context.Context, templateName string, opts *AddDependencyFromTemplateOpts) error {
	internalOpts := bindown.AddDependencyFromTemplateOpts(*opts)
	ic := internalizeConfig(c)
	err := ic.AddDependencyFromTemplate(ctx, templateName, &internalOpts)
	*c = *externalizeConfig(ic)
	return err
}

// CopyTemplateFromSource copies a template from source
func (c *Config) CopyTemplateFromSource(ctx context.Context, src, srcTemplate, destName string) error {
	ic := internalizeConfig(c)
	err := ic.CopyTemplateFromSource(ctx, src, srcTemplate, destName)
	*c = *externalizeConfig(ic)
	return err
}

// ListTemplates lists templates available in this config or one of its template sources.
func (c *Config) ListTemplates(ctx context.Context, templateSource string) ([]string, error) {
	return internalizeConfig(c).ListTemplates(ctx, templateSource)
}

// DependencySystems returns the supported systems of either the config or the dependency if one is not empty
// if both are not empty, it returns the intersection of the lists
func (c *Config) DependencySystems(depName string) ([]SystemInfo, error) {
	iSystems, err := internalizeConfig(c).DependencySystems(depName)
	if err != nil {
		return nil, err
	}
	return externalizeSystems(iSystems), nil
}

// ConfigFromURL loads a config from a URL
func ConfigFromURL(ctx context.Context, cfgSrc string) (*Config, error) {
	iCfg, err := bindown.ConfigFromURL(ctx, cfgSrc)
	if err != nil {
		return nil, err
	}
	return externalizeConfig(iCfg), nil
}

// ConfigFile is a file containing config
type ConfigFile struct {
	Filename string `json:"-"`
	Config
}

// LoadConfigFile loads a config file
func LoadConfigFile(ctx context.Context, filename string, noDefaultDirs bool) (*ConfigFile, error) {
	iCfgFile, err := bindown.LoadConfigFile(ctx, filename, noDefaultDirs)
	if err != nil {
		return nil, err
	}
	return &ConfigFile{
		Filename: iCfgFile.Filename,
		Config:   *externalizeConfig(&iCfgFile.Config),
	}, nil
}

// Write writes a file to disk
func (c *ConfigFile) Write(outputJSON bool) error {
	cfg := &bindown.ConfigFile{
		Filename: c.Filename,
		Config:   *internalizeConfig(&c.Config),
	}
	return cfg.Write(outputJSON)
}

// DependencyOverride overrides a dependency's configuration
type DependencyOverride struct {
	OverrideMatcher OverrideMatcher `json:"matcher" yaml:"matcher,omitempty"`
	Dependency      Dependency      `json:"dependency" yaml:",omitempty"`
}

// OverrideMatcher contains a list or oses and arches to match an override. If either os or arch is empty, all oses and arches match.
type OverrideMatcher map[string][]string

// Dependency is something to download, extract and install
type Dependency struct {
	Template      *string                      `json:"template,omitempty" yaml:",omitempty"`
	URL           *string                      `json:"url,omitempty" yaml:",omitempty"`
	ArchivePath   *string                      `json:"archive_path,omitempty" yaml:"archive_path,omitempty"`
	BinName       *string                      `json:"bin,omitempty" yaml:"bin,omitempty"`
	Link          *bool                        `json:"link,omitempty" yaml:",omitempty"`
	Vars          map[string]string            `json:"vars,omitempty" yaml:",omitempty"`
	RequiredVars  []string                     `json:"required_vars,omitempty" yaml:"required_vars,omitempty"`
	Overrides     []DependencyOverride         `json:"overrides,omitempty" yaml:",omitempty"`
	Substitutions map[string]map[string]string `json:"substitutions,omitempty" yaml:",omitempty"`
	Systems       []SystemInfo                 `json:"systems,omitempty" yaml:"systems,omitempty"`
}

// SystemInfo contains os and architecture for a target system
type SystemInfo struct {
	OS   string
	Arch string
}

func (s *SystemInfo) String() string {
	i := bindown.SystemInfo(*s)
	return i.String()
}

// UnmarshalText implements encoding.TextUnmarshaler
func (s *SystemInfo) UnmarshalText(text []byte) error {
	i := bindown.SystemInfo(*s)
	err := i.UnmarshalText(text)
	*s = SystemInfo(i)
	return err
}

// MarshalText implements encoding.TextMarshaler
func (s SystemInfo) MarshalText() (text []byte, err error) {
	return bindown.SystemInfo(s).MarshalText()
}
