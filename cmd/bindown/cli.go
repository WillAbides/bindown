package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v4/internal/bindown"
	"github.com/willabides/kongplete"
)

var kongVars = kong.Vars{
	"configfile_help":                 `file with bindown config. default is the first one of bindown.yml, bindown.yaml, bindown.json, .bindown.yml, .bindown.yaml or .bindown.json`,
	"cache_help":                      `directory downloads will be cached`,
	"install_help":                    `download, extract and install a dependency`,
	"system_default":                  string(bindown.CurrentSystem),
	"system_help":                     `target system in the format of <os>/<architecture>`,
	"systems_help":                    `target systems in the format of <os>/<architecture>`,
	"add_checksums_help":              `add checksums to the config file`,
	"prune_checksums_help":            `remove unnecessary checksums from the config file`,
	"sync_checksums_help":             `add checksums to the config file and remove unnecessary checksums`,
	"config_format_help":              `formats the config file`,
	"config_validate_help":            `validate that installs work`,
	"config_install_completions_help": `install shell completions`,
	"config_extract_path_help":        `output path to directory where the downloaded archive is extracted`,
	"install_force_help":              `force install even if it already exists`,
	"install_target_file_help":        `where to write the file`,
	"install_dependency_help":         `dependency to install`,
	"download_force_help":             `force download even if the file already exists`,
	"download_target_file_help":       `filename and path for the downloaded file. Default downloads to cache.`,
	"download_dependency_help":        `name of the dependency to download`,
	"allow_missing_checksum":          `allow missing checksums`,
	"download_help":                   `download a dependency but don't extract or install it`,
	"extract_dependency_help":         `name of the dependency to extract`,
	"extract_help":                    `download and extract a dependency but don't install it`,
	"extract_target_dir_help":         `path to extract to. Default extracts to cache.`,
	"checksums_dep_help":              `name of the dependency to update`,
	"trust_cache_help":                `trust the cache contents and do not recheck existing downloads and extracts in the cache`,
	"all_deps_help":                   `select all dependencies`,
}

type rootCmd struct {
	JSONConfig bool   `kong:"name=json,help='treat config file as json instead of yaml'"`
	Configfile string `kong:"type=path,help=${configfile_help},env='BINDOWN_CONFIG_FILE'"`
	CacheDir   string `kong:"name=cache,type=path,help=${cache_help},env='BINDOWN_CACHE'"`
	Quiet      bool   `kong:"short='q',help='suppress output to stdout'"`

	Download        downloadCmd        `kong:"cmd,help=${download_help}"`
	Extract         extractCmd         `kong:"cmd,help=${extract_help}"`
	Install         installCmd         `kong:"cmd,help=${install_help}"`
	Format          fmtCmd             `kong:"cmd,help=${config_format_help}"`
	Dependency      dependencyCmd      `kong:"cmd,help='manage dependencies'"`
	Template        templateCmd        `kong:"cmd,help='manage templates'"`
	TemplateSource  templateSourceCmd  `kong:"cmd,help='manage template sources'"`
	SupportedSystem supportedSystemCmd `kong:"cmd,help='manage supported systems'"`
	Checksums       checksumsCmd       `kong:"cmd,help='manage checksums'"`
	Init            initCmd            `kong:"cmd,help='create an empty config file'"`
	Cache           cacheCmd           `kong:"cmd,help='manage the cache'"`

	Version            versionCmd                   `kong:"cmd,help='show bindown version'"`
	InstallCompletions kongplete.InstallCompletions `kong:"cmd,help=${config_install_completions_help}"`
}

var defaultConfigFilenames = []string{
	"bindown.yml",
	"bindown.yaml",
	"bindown.json",
	".bindown.yml",
	".bindown.yaml",
	".bindown.json",
}

func loadConfigFile(ctx *runContext, noDefaultDirs bool) (*bindown.Config, error) {
	filename := ctx.rootCmd.Configfile
	if filename == "" {
		for _, configFilename := range defaultConfigFilenames {
			info, err := os.Stat(configFilename)
			if err == nil && !info.IsDir() {
				filename = configFilename
				break
			}
		}
	}
	configFile, err := bindown.NewConfig(ctx, filename, noDefaultDirs)
	if err != nil {
		return nil, err
	}
	if ctx.rootCmd.CacheDir != "" {
		configFile.Cache = ctx.rootCmd.CacheDir
	}
	return configFile, nil
}

// fileWriter covers terminal.FileWriter. Needed for survey
type fileWriter interface {
	io.Writer
	Fd() uintptr
}

type SimpleFileWriter struct {
	io.Writer
}

func (s SimpleFileWriter) Fd() uintptr {
	return 0
}

// fileReader covers terminal.FileReader. Needed for survey
type fileReader interface {
	io.Reader
	Fd() uintptr
}

type runContext struct {
	parent  context.Context
	stdin   fileReader
	stdout  fileWriter
	rootCmd *rootCmd
}

func newRunContext(ctx context.Context) *runContext {
	return &runContext{
		parent: ctx,
	}
}

func (r *runContext) Deadline() (deadline time.Time, ok bool) {
	return r.parent.Deadline()
}

func (r *runContext) Done() <-chan struct{} {
	return r.parent.Done()
}

func (r *runContext) Err() error {
	return r.parent.Err()
}

func (r *runContext) Value(key any) any {
	return r.parent.Value(key)
}

type runOpts struct {
	stdin       fileReader
	stdout      fileWriter
	stderr      io.Writer
	cmdName     string
	exitHandler func(int)
}

// Run let's light this candle
func Run(ctx context.Context, args []string, opts *runOpts) {
	if opts == nil {
		opts = &runOpts{}
	}
	var root rootCmd
	runCtx := newRunContext(ctx)
	runCtx.rootCmd = &root
	runCtx.stdin = opts.stdin
	if runCtx.stdin == nil {
		runCtx.stdin = os.Stdin
	}
	runCtx.stdout = opts.stdout
	if runCtx.stdout == nil {
		runCtx.stdout = os.Stdout
	}
	stderr := opts.stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	kongOptions := []kong.Option{
		kong.HelpOptions{Compact: true},
		kong.BindTo(runCtx, &runCtx),
		kongVars,
		kong.UsageOnError(),
		kong.Writers(runCtx.stdout, stderr),
	}
	if opts.exitHandler != nil {
		kongOptions = append(kongOptions, kong.Exit(opts.exitHandler))
	}
	if opts.cmdName != "" {
		kongOptions = append(kongOptions, kong.Name(opts.cmdName))
	}

	parser := kong.Must(&root, kongOptions...)
	runCompletion(ctx, parser)

	kongCtx, err := parser.Parse(args)
	parser.FatalIfErrorf(err)
	if root.Quiet {
		runCtx.stdout = SimpleFileWriter{io.Discard}
		kongCtx.Stdout = io.Discard
	}
	err = kongCtx.Run()
	kongCtx.FatalIfErrorf(err)
}

func runCompletion(ctx context.Context, parser *kong.Kong) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	kongplete.Complete(parser,
		kongplete.WithPredictor("bin", binCompleter(ctx)),
		kongplete.WithPredictor("allSystems", allSystemsCompleter),
		kongplete.WithPredictor("templateSource", templateSourceCompleter(ctx)),
		kongplete.WithPredictor("system", systemCompleter(ctx)),
		kongplete.WithPredictor("localTemplate", localTemplateCompleter(ctx)),
		kongplete.WithPredictor("localTemplateFromSource", localTemplateFromSourceCompleter(ctx)),
		kongplete.WithPredictor("template", templateCompleter(ctx)),
	)
}

type initCmd struct{}

func (c *initCmd) Run(ctx *runContext) error {
	for _, filename := range defaultConfigFilenames {
		info, err := os.Stat(filename)
		if err == nil && !info.IsDir() {
			return fmt.Errorf("%s already exists", filename)
		}
	}
	configfile := ctx.rootCmd.Configfile
	if configfile == "" {
		configfile = ".bindown.yaml"
	}
	file, err := os.Create(configfile)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}
	cfg := &bindown.Config{
		Filename: file.Name(),
	}
	return cfg.WriteFile(ctx.rootCmd.JSONConfig)
}

type fmtCmd struct{}

func (c fmtCmd) Run(ctx *runContext, cli *rootCmd) error {
	ctx.rootCmd.CacheDir = ""
	config, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}
	return config.WriteFile(ctx.rootCmd.JSONConfig)
}

type installCmd struct {
	Force                bool           `kong:"help=${install_force_help}"`
	Dependency           []string       `kong:"arg,name=dependency,help=${download_dependency_help},predictor=bin"`
	All                  bool           `kong:"help=${all_deps_help}"`
	TargetFile           string         `kong:"type=path,name=output,type=file,help=${install_target_file_help}"`
	System               bindown.System `kong:"name=system,default=${system_default},help=${system_help},predictor=allSystems"`
	AllowMissingChecksum bool           `kong:"name=allow-missing-checksum,help=${allow_missing_checksum}"`
}

func (d *installCmd) BeforeApply(k *kong.Context) error {
	optionalDependency(k)
	return nil
}

func (d *installCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, false)
	if err != nil {
		return err
	}
	if d.All {
		if len(d.Dependency) > 0 {
			return fmt.Errorf("cannot specify dependencies when using --all")
		}
		d.Dependency = allDependencies(config)
	}
	switch len(d.Dependency) {
	case 0:
		return fmt.Errorf("must specify at least one dependency")
	case 1:
	default:
		if d.TargetFile != "" {
			return fmt.Errorf("cannot specify --output when installing multiple dependencies")
		}
	}
	for _, dep := range d.Dependency {
		var pth string
		pth, err = config.InstallDependency(dep, d.System, &bindown.ConfigInstallDependencyOpts{
			TargetPath:           d.TargetFile,
			Force:                d.Force,
			AllowMissingChecksum: d.AllowMissingChecksum,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(ctx.stdout, "installed %s to %s\n", dep, pth)
	}
	return nil
}

type downloadCmd struct {
	Force                bool           `kong:"help=${download_force_help}"`
	System               bindown.System `kong:"name=system,default=${system_default},help=${system_help},predictor=allSystems"`
	Dependency           []string       `kong:"arg,help=${download_dependency_help},predictor=bin"`
	All                  bool           `kong:"help=${all_deps_help}"`
	AllowMissingChecksum bool           `kong:"name=allow-missing-checksum,help=${allow_missing_checksum}"`
}

func (d *downloadCmd) BeforeApply(k *kong.Context) error {
	optionalDependency(k)
	return nil
}

func (d *downloadCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, false)
	if err != nil {
		return err
	}
	if d.All {
		if len(d.Dependency) > 0 {
			return fmt.Errorf("cannot specify dependencies when using --all")
		}
		d.Dependency = allDependencies(config)
	}
	if len(d.Dependency) == 0 {
		return fmt.Errorf("must specify at least one dependency")
	}
	for _, dep := range d.Dependency {
		var pth string
		pth, err = config.DownloadDependency(dep, d.System, &bindown.ConfigDownloadDependencyOpts{
			Force:                d.Force,
			AllowMissingChecksum: d.AllowMissingChecksum,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(ctx.stdout, "downloaded %s to %s\n", dep, pth)
	}
	return nil
}

type extractCmd struct {
	System               bindown.System `kong:"name=system,default=${system_default},help=${system_help},predictor=allSystems"`
	Dependency           []string       `kong:"arg,help=${extract_dependency_help},predictor=bin"`
	All                  bool           `kong:"help=${all_deps_help}"`
	AllowMissingChecksum bool           `kong:"name=allow-missing-checksum,help=${allow_missing_checksum}"`
}

func (d *extractCmd) BeforeApply(k *kong.Context) error {
	optionalDependency(k)
	return nil
}

func (d *extractCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, false)
	if err != nil {
		return err
	}
	if d.All {
		if len(d.Dependency) > 0 {
			return fmt.Errorf("cannot specify dependencies when using --all")
		}
		d.Dependency = allDependencies(config)
	}
	if len(d.Dependency) == 0 {
		return fmt.Errorf("must specify at least one dependency")
	}
	for _, dep := range d.Dependency {
		var pth string
		pth, err = config.ExtractDependency(dep, d.System, &bindown.ConfigExtractDependencyOpts{
			AllowMissingChecksum: d.AllowMissingChecksum,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(ctx.stdout, "extracted %s to %s\n", dep, pth)
	}
	return nil
}

// optionalDependency sets dependency positional to optional. We do this because we want to allow --all to be
// equivalent to specifying all dependencies but want the help output to indicate that a dependency is required.
func optionalDependency(k *kong.Context) {
	for _, pos := range k.Selected().Positional {
		if pos.Name == "dependency" {
			pos.Required = false
		}
	}
}
