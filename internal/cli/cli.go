package cli

import (
	"fmt"

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
	Configfile string `kong:"type=existingfile,help=${configfile_help},default=${configfile_default},env='BINDOWN_CONFIG_FILE',predictor=file"`
	CellarDir  string `kong:"type=path,help=${cellar_dir_help},env='BINDOWN_CELLAR',predictor=dir"`
}

func (c *configOpts) BeforeApply(k *kong.Context) error {
	fmt.Println("hi")
	return nil
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

func run(parser *kong.Kong, args []string) error {
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
	if err != nil {
		return err
	}
	return kongCtx.Run()
}

//Run let's light this candle
func Run(args []string, kongOptions ...kong.Option) {
	parser := newParser(kongOptions...)
	parser.FatalIfErrorf(run(parser, args))
}
