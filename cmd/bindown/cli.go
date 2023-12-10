package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v4/internal/bindown"
	"github.com/willabides/kongplete"
)

var kongVars = kong.Vars{
	"configfile_help":                 `file with bindown config. default is the first one of bindown.yml, bindown.yaml, bindown.json, .bindown.yml, .bindown.yaml or .bindown.json`,
	"cache_help":                      `directory downloads will be cached`,
	"install_help":                    `download, extract and install a dependency`,
	"wrap_help":                       `create a wrapper script for a dependency`,
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
	"output_help":                     `where to write the file. this is a directory unless a single dependency is selected and the path isn't an existing directory`,
	"download_force_help":             `force download even if the file already exists`,
	"allow_missing_checksum":          `allow missing checksums`,
	"download_help":                   `download a dependency but don't extract or install it`,
	"extract_help":                    `download and extract a dependency but don't install it`,
	"checksums_dep_help":              `name of the dependency to update`,
	"all_deps_help":                   `select all dependencies`,
	"dependency_help":                 `name of dependency`,
	"install_to_cache_help":           `install to cache instead of install dir`,
	"install_wrapper_help":            `install a wrapper script instead of the binary`,
	"install_bindown_help":            `path to bindown executable to use in wrapper`,
	"bootstrap_tag_default":           defaultBootstrapTag(),
}

type rootCmd struct {
	JSONConfig bool   `kong:"name=json,help='treat config file as json instead of yaml'"`
	Configfile string `kong:"type=path,help=${configfile_help},env='BINDOWN_CONFIG_FILE'"`
	CacheDir   string `kong:"name=cache,type=path,help=${cache_help},env='BINDOWN_CACHE'"`
	Quiet      bool   `kong:"short='q',help='suppress output to stdout'"`

	Download        downloadCmd        `kong:"cmd,help=${download_help}"`
	Extract         extractCmd         `kong:"cmd,help=${extract_help}"`
	Install         installCmd         `kong:"cmd,help=${install_help}"`
	Wrap            wrapCmd            `kong:"cmd,help=${wrap_help}"`
	Format          fmtCmd             `kong:"cmd,help=${config_format_help}"`
	Dependency      dependencyCmd      `kong:"cmd,help='manage dependencies'"`
	Template        templateCmd        `kong:"cmd,help='manage templates'"`
	TemplateSource  templateSourceCmd  `kong:"cmd,help='manage template sources'"`
	SupportedSystem supportedSystemCmd `kong:"cmd,help='manage supported systems'"`
	Checksums       checksumsCmd       `kong:"cmd,help='manage checksums'"`
	Init            initCmd            `kong:"cmd,help='create an empty config file'"`
	Cache           cacheCmd           `kong:"cmd,help='manage the cache'"`
	Bootstrap       bootstrapCmd       `kong:"cmd,help='create bootstrap script for bindown'"`

	Version            versionCmd                   `kong:"cmd,help='show bindown version'"`
	InstallCompletions kongplete.InstallCompletions `kong:"cmd,help=${config_install_completions_help}"`
}

func (r *rootCmd) BeforeApply(k *kong.Context) error {
	// set dependency positional to optional for install, wrap, download and extract.
	// We do this because we want to allow --all to be equivalent to specifying all
	// dependencies but want the help output to indicate that a dependency is required.
	if slices.Contains([]string{"install", "wrap", "download", "extract"}, k.Selected().Name) {
		for _, pos := range k.Selected().Positional {
			if pos.Name == "dependency" {
				pos.Required = false
			}
		}
	}
	return nil
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
	stderr  fileWriter
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
	stderr      fileWriter
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
	runCtx.stderr = opts.stderr
	if runCtx.stderr == nil {
		runCtx.stderr = os.Stderr
	}

	kongOptions := []kong.Option{
		kong.HelpOptions{Compact: true},
		kong.BindTo(runCtx, &runCtx),
		kongVars,
		kong.UsageOnError(),
		kong.Writers(runCtx.stdout, runCtx.stderr),
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
		kongplete.WithPredictor("wrap_bin", wrapBinCompleter(ctx)),
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
	Dependency           []string       `kong:"arg,name=dependency,help=${dependency_help},predictor=bin"`
	All                  bool           `kong:"help=${all_deps_help}"`
	Force                bool           `kong:"help=${install_force_help}"`
	Output               string         `kong:"type=path,name=output,type=file,help=${output_help}"`
	System               bindown.System `kong:"name=system,default=${system_default},help=${system_help},predictor=allSystems"`
	AllowMissingChecksum bool           `kong:"name=allow-missing-checksum,help=${allow_missing_checksum}"`
	ToCache              bool           `kong:"name=to-cache,help=${install_to_cache_help}"`

	// hidden options to be removed
	Wrapper     bool   `kong:"hidden,name=wrapper"`
	BindownExec string `kong:"hidden,name=bindown"`
}

func (d *installCmd) Run(ctx *runContext) error {
	if d.Wrapper {
		fmt.Fprintln(ctx.stderr, `--wrapper is deprecated and will be removed in a future version. Use "bindown wrap" instead.`)
		if d.ToCache {
			return fmt.Errorf("cannot use --to-cache and --wrapper together")
		}
		if d.Force {
			return fmt.Errorf("cannot use --force and --wrapper together")
		}
		cmd := &wrapCmd{
			Dependency:           d.Dependency,
			All:                  d.All,
			Output:               d.Output,
			AllowMissingChecksum: d.AllowMissingChecksum,
			BindownExec:          d.BindownExec,
		}
		return cmd.Run(ctx)
	}
	config, err := loadConfigFile(ctx, false)
	if err != nil {
		return err
	}

	return config.InstallDependencies(d.Dependency, d.System, &bindown.ConfigInstallDependenciesOpts{
		Output:               d.Output,
		Force:                d.Force,
		AllowMissingChecksum: d.AllowMissingChecksum,
		ToCache:              d.ToCache,
		Stdout:               ctx.stdout,
		AllDeps:              d.All,
	})
}

type wrapCmd struct {
	Dependency           []string `kong:"arg,name=dependency,help=${dependency_help},predictor=wrap_bin"`
	All                  bool     `kong:"help=${all_deps_help}"`
	Output               string   `kong:"type=path,name=output,type=file,help=${output_help}"`
	AllowMissingChecksum bool     `kong:"name=allow-missing-checksum,help=${allow_missing_checksum}"`
	BindownExec          string   `kong:"name=bindown,help=${install_bindown_help}"`
	BindownTag           string   `kong:"hidden,default=${bootstrap_tag_default}"`
	BaseURL              string   `kong:"hidden,name='base-url',default='https://github.com'"`
}

func (d *wrapCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, false)
	if err != nil {
		return err
	}
	return config.WrapDependencies(d.Dependency, &bindown.ConfigWrapDependenciesOpts{
		Output:               d.Output,
		AllowMissingChecksum: d.AllowMissingChecksum,
		BindownExec:          d.BindownExec,
		Stdout:               ctx.stdout,
		AllDeps:              d.All,
		BindownTag:           d.BindownTag,
		BindownWrapped:       os.Getenv("BINDOWN_WRAPPED"),
		BaseURL:              d.BaseURL,
	})
}

type downloadCmd struct {
	Dependency           []string       `kong:"arg,name=dependency,help=${dependency_help},predictor=bin"`
	All                  bool           `kong:"help=${all_deps_help}"`
	Force                bool           `kong:"help=${download_force_help}"`
	System               bindown.System `kong:"name=system,default=${system_default},help=${system_help},predictor=allSystems"`
	AllowMissingChecksum bool           `kong:"name=allow-missing-checksum,help=${allow_missing_checksum}"`
}

func (d *downloadCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, false)
	if err != nil {
		return err
	}
	return config.DownloadDependencies(d.Dependency, d.System, &bindown.ConfigDownloadDependenciesOpts{
		Force:                d.Force,
		AllowMissingChecksum: d.AllowMissingChecksum,
		AllDeps:              d.All,
		Stdout:               ctx.stdout,
	})
}

type extractCmd struct {
	Dependency           []string       `kong:"arg,name=dependency,help=${dependency_help},predictor=bin"`
	All                  bool           `kong:"help=${all_deps_help}"`
	System               bindown.System `kong:"name=system,default=${system_default},help=${system_help},predictor=allSystems"`
	AllowMissingChecksum bool           `kong:"name=allow-missing-checksum,help=${allow_missing_checksum}"`
}

func (d *extractCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, false)
	if err != nil {
		return err
	}
	return config.ExtractDependencies(d.Dependency, d.System, &bindown.ConfigExtractDependenciesOpts{
		AllowMissingChecksum: d.AllowMissingChecksum,
		AllDeps:              d.All,
		Stdout:               ctx.stdout,
	})
}
