package cli

import (
	"github.com/alecthomas/kong"
	"github.com/posener/complete"
	"github.com/willabides/kongplete"
)

var kongVars = kong.Vars{
	"configfile_help":    `file with bindown config`,
	"configfile_default": `bindown.yml`,
	"cellar_dir_help":    `directory where downloads will be cached`,
	"download_help":      `download a bin`,
}

type configOpts struct {
	Configfile string `kong:"type=path,help=${configfile_help},default=${configfile_default},env='BINDOWN_CONFIG_FILE',predictor=file"`
	CellarDir  string `kong:"type=path,help=${cellar_dir_help},env='BINDOWN_CELLAR',predictor=dir"`
}

var cli struct {
	Version  versionCmd  `kong:"cmd"`
	Download downloadCmd `kong:"cmd,help=${download_help}"`
	Config   configCmd   `kong:"cmd"`
}

func newParser(kongOptions ...kong.Option) *kong.Kong {
	kongOptions = append(kongOptions,
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

	kongplete.Complete(parser,
		kongplete.WithPredictors(map[string]complete.Predictor{
			"file":    complete.PredictFiles("*"),
			"dir":     complete.PredictDirs("*"),
			"binpath": binPathPredictor,
			"arch":    archPredictor,
			"os":      osPredictor,
			"bin":     binPredictor,
		}),
	)

	kongCtx, err := parser.Parse(args)
	parser.FatalIfErrorf(err)
	err = kongCtx.Run()
	parser.FatalIfErrorf(err)
}
