package cli

import (
	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v3"
)

var kongVars = kong.Vars{
	"configfile_help":    `file with bindown config`,
	"configfile_default": `bindown.yml`,
	"cellar_dir_help":    `directory where downloads will be cached`,
	"download_help":      `download a bin`,
}

type configOpts struct {
	Configfile string `kong:"type=path,help=${configfile_help},default=${configfile_default},env='BINDOWN_CONFIG_FILE'"`
	CellarDir  string `kong:"type=path,help=${cellar_dir_help},env='BINDOWN_CELLAR'"`
}

var cli struct {
	Version  versionCmd  `kong:"cmd"`
	Download downloadCmd `kong:"cmd,help=${download_help}"`
	Config   configCmd   `kong:"cmd"`
}

func configFile(kctx *kong.Context) *bindown.ConfigFile {
	config, err := bindown.LoadConfigFile(cli.Config.ConfigOpts.Configfile)
	kctx.FatalIfErrorf(err, "error loading config from %q", cli.Config.ConfigOpts.Configfile)
	return config
}

func newParser(kongOptions ...kong.Option) *kong.Kong {
	kongOptions = append(kongOptions,
		kong.Completers{
			"binpath": binPathCompleter,
			"arch":    archCompleter,
			"os":      osCompleter,
			"bin":     binCompleter,
		},
		kongVars,
		downloadKongVars,
		configKongVars,
		kong.UsageOnError(),
		kong.NamedMapper("multipath", kong.MapperFunc(multipathMapper)),
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
