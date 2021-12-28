package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v3"
	"github.com/willabides/bindown/v3/cmd/bindown/ifaces"
	"github.com/willabides/kongplete"
)

//go:generate mockgen -source ifaces/ifaces.go -destination mocks/$GOFILE -package mocks

var kongVars = kong.Vars{
	"configfile_help":                 `file with bindown config. default is the first one of bindown.yml, bindown.yaml, bindown.json, .bindown.yml, .bindown.yaml or .bindown.json`,
	"cache_help":                      `directory downloads will be cached`,
	"install_help":                    `download, extract and install a dependency`,
	"system_default":                  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	"system_help":                     `target system in the format of <os>/<architecture>`,
	"systems_help":                    `target systems in the format of <os>/<architecture>`,
	"add_checksums_help":              `add checksums to the config file`,
	"prune_checksums_help":            `remove unnecessary checksums from the config file`,
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
	"download_help":                   `download a dependency but don't extract or install it`,
	"extract_dependency_help":         `name of the dependency to extract`,
	"extract_help":                    `download and extract a dependency but don't install it`,
	"extract_target_dir_help":         `path to extract to. Default extracts to cache.`,
	"checksums_dep_help":              `name of the dependency to update`,
}

var cli struct {
	JSONConfig bool   `kong:"name=json,help='treat config file as json instead of yaml'"`
	Configfile string `kong:"type=path,help=${configfile_help},env='BINDOWN_CONFIG_FILE'"`
	Cache      string `kong:"type=path,help=${cache_help},env='BINDOWN_CACHE'"`

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

	Version            versionCmd                   `kong:"cmd,help='show bindown version'"`
	InstallCompletions kongplete.InstallCompletions `kong:"cmd,help=${config_install_completions_help}"`

	AddChecksums addChecksumsCmd `kong:"cmd,hidden"`
	Validate     validateCmd     `kong:"cmd,hidden"`
}

type defaultConfigLoader struct{}

var defaultConfigFilenames = []string{
	"bindown.yml",
	"bindown.yaml",
	"bindown.json",
	".bindown.yml",
	".bindown.yaml",
	".bindown.json",
}

func (d defaultConfigLoader) Load(filename string, noDefaultDirs bool) (ifaces.ConfigFile, error) {
	if filename != "" {
		return bindown.LoadConfigFile(filename, noDefaultDirs)
	}
	for _, configFilename := range defaultConfigFilenames {
		info, err := os.Stat(configFilename)
		if err == nil && !info.IsDir() {
			filename = configFilename
			break
		}
	}
	return bindown.LoadConfigFile(filename, noDefaultDirs)
}

var configLoader ifaces.ConfigLoader = defaultConfigLoader{}

func newParser(kongOptions ...kong.Option) *kong.Kong {
	kongOptions = append(kongOptions,
		kongVars,
		kong.UsageOnError(),
	)
	return kong.Must(&cli, kongOptions...)
}

// Run let's light this candle
func Run(args []string, kongOptions ...kong.Option) {
	kongOptions = append(kongOptions,
		kong.HelpOptions{
			Compact: true,
		},
	)
	parser := newParser(kongOptions...)
	kongplete.Complete(parser,
		kongplete.WithPredictor("bin", binCompleter),
		kongplete.WithPredictor("allSystems", allSystemsCompleter),
		kongplete.WithPredictor("templateSource", templateSourceCompleter),
		kongplete.WithPredictor("system", systemCompleter),
	)

	kongCtx, err := parser.Parse(args)
	parser.FatalIfErrorf(err)
	err = kongCtx.Run()
	parser.FatalIfErrorf(err)
}

type initCmd struct{}

func (c *initCmd) Run() error {
	for _, filename := range defaultConfigFilenames {
		info, err := os.Stat(filename)
		if err == nil && !info.IsDir() {
			return fmt.Errorf("%s already exists", filename)
		}
	}
	file, err := os.Create(".bindown.yaml")
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}
	cfg := &bindown.ConfigFile{
		Filename: file.Name(),
	}
	return cfg.Write(cli.JSONConfig)
}

type fmtCmd struct{}

func (c fmtCmd) Run(_ *kong.Context) error {
	cli.Cache = ""
	config, err := configLoader.Load(cli.Configfile, true)
	if err != nil {
		return err
	}
	return config.Write(cli.JSONConfig)
}

type validateCmd struct {
	Dependency string               `kong:"required=true,arg,predictor=bin"`
	Systems    []bindown.SystemInfo `kong:"name=system,predictor=allSystems"`
}

func (d validateCmd) Run(ctx *kong.Context) error {
	config, err := configLoader.Load(cli.Configfile, false)
	if err != nil {
		return err
	}
	return config.Validate([]string{d.Dependency}, d.Systems)
}

type installCmd struct {
	Force      bool               `kong:"help=${install_force_help}"`
	Dependency string             `kong:"required=true,arg,help=${download_dependency_help},predictor=bin"`
	TargetFile string             `kong:"type=path,name=output,type=file,help=${install_target_file_help}"`
	System     bindown.SystemInfo `kong:"name=system,default=${system_default},help=${system_help},predictor=allSystems"`
}

func (d *installCmd) Run(ctx *kong.Context) error {
	config, err := configLoader.Load(cli.Configfile, false)
	if err != nil {
		return err
	}
	pth, err := config.InstallDependency(d.Dependency, d.System, &bindown.ConfigInstallDependencyOpts{
		TargetPath: d.TargetFile,
		Force:      d.Force,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "installed %s to %s\n", d.Dependency, pth)
	return nil
}

type downloadCmd struct {
	Force      bool               `kong:"help=${download_force_help}"`
	System     bindown.SystemInfo `kong:"name=system,default=${system_default},help=${system_help},predictor=allSystems"`
	Dependency string             `kong:"required=true,arg,help=${download_dependency_help},predictor=bin"`
	TargetFile string             `kong:"name=output,help=${download_target_file_help}"`
}

func (d *downloadCmd) Run(ctx *kong.Context) error {
	config, err := configLoader.Load(cli.Configfile, false)
	if err != nil {
		return err
	}
	pth, err := config.DownloadDependency(d.Dependency, d.System, &bindown.ConfigDownloadDependencyOpts{
		TargetFile: d.TargetFile,
		Force:      d.Force,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "downloaded %s to %s\n", d.Dependency, pth)
	return nil
}

type extractCmd struct {
	System     bindown.SystemInfo `kong:"name=system,default=${system_default},help=${system_help},predictor=allSystems"`
	Dependency string             `kong:"required=true,arg,help=${extract_dependency_help},predictor=bin"`
	TargetDir  string             `kong:"name=output,help=${extract_target_dir_help}"`
}

func (d *extractCmd) Run(ctx *kong.Context) error {
	config, err := configLoader.Load(cli.Configfile, false)
	if err != nil {
		return err
	}
	pth, err := config.ExtractDependency(d.Dependency, d.System, &bindown.ConfigExtractDependencyOpts{
		TargetDirectory: d.TargetDir,
		Force:           false,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "extracted %s to %s\n", d.Dependency, pth)
	return nil
}

func requestRequiredVar(ctx *kong.Context, name string, vars map[string]string) (map[string]string, error) {
	fmt.Fprintf(ctx.Stdout, "Please enter a value for required variable %q:\t", name)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	err := scanner.Err()
	if err != nil {
		return nil, err
	}
	val := scanner.Text()

	vars[name] = val
	return vars, nil
}
