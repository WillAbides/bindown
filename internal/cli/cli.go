package cli

import (
	"fmt"
	"runtime"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v3/internal/configfile"
)

var kongVars = kong.Vars{
	"configfile_help":    `file with bindown config`,
	"configfile_default": `bindown.yml`,
	"cellar_dir_help":    `directory where installs will be cached`,
	"install_help":       `install a bin`,
	"system_default":     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	"system_help":        `target system in the format of <os>/<architecture>`,
}

type configOpts struct {
	Configfile string `kong:"type=path,help=${configfile_help},default=${configfile_default},env='BINDOWN_CONFIG_FILE'"`
	CellarDir  string `kong:"type=path,help=${cellar_dir_help},env='BINDOWN_CELLAR'"`
}

var cli struct {
	Version versionCmd `kong:"cmd"`
	Install installCmd `kong:"cmd,help=${install_help}"`
	Config  configCmd  `kong:"cmd"`
}

func configFile(kctx *kong.Context, filename string) *configfile.ConfigFile {
	config, err := configfile.LoadConfigFile(filename)
	kctx.FatalIfErrorf(err, "error loading config from %q", filename)
	return config
}

func newParser(kongOptions ...kong.Option) *kong.Kong {
	kongOptions = append(kongOptions,
		kong.Completers{
			"binpath": binPathCompleter,
			"bin":     binCompleter,
			"system":  systemCompleter,
		},
		kongVars,
		installKongVars,
		configKongVars,
		kong.UsageOnError(),
	)
	return kong.Must(&cli, kongOptions...)
}

//Run let's light this candle
func Run(args []string, kongOptions ...kong.Option) {
	parser := newParser(kongOptions...)

	kongCtx, err := parser.Parse(args)
	parser.FatalIfErrorf(err)
	err = kongCtx.Run()
	parser.FatalIfErrorf(err)
}
