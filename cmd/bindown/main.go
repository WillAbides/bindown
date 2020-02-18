package main

import (
	"os"

	"github.com/alecthomas/kong"
	"github.com/posener/complete"
	"github.com/willabides/kongplete"
)

var kongVars = kong.Vars{
	"configfile_help":    `file with bindown config`,
	"configfile_default": `bindown.yml|bindown.json`,
	"cellar_dir_help":    `directory where downloads will be cached`,
	"download_help":      `download a bin`,
}

var cli struct {
	Version    versionCmd  `kong:"cmd"`
	Download   downloadCmd `kong:"cmd,help=${download_help}"`
	Config     configCmd   `kong:"cmd"`
	Configfile string      `kong:"type=multipath,help=${configfile_help},default=${configfile_default},env='BINDOWN_CONFIG_FILE',predictor=file"`
	CellarDir  string      `kong:"type=path,help=${cellar_dir_help},env='BINDOWN_CELLAR',predictor=dir"`
}

func main() {
	parser := kong.Must(
		&cli,
		kongVars,
		downloadKongVars,
		configKongVars,
		kong.UsageOnError(),
		kong.NamedMapper("multipath", kong.MapperFunc(multipathMapper)),
	)

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

	kongCtx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)
	kongCtx.FatalIfErrorf(kongCtx.Run(kongCtx))
}
