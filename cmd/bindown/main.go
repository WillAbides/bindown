package main

import (
	"os"

	"github.com/alecthomas/kong"
)

var kongVars = kong.Vars{
	"configfile_help":    `file with bindown config`,
	"configfile_default": `bindown.json`,
	"cellar_dir_help":    `directory where downloads will be cached`,
	"download_help":      `download a bin`,
}

var cli struct {
	Version    versionCmd  `kong:"cmd"`
	Download   downloadCmd `kong:"cmd,help=${download_help}"`
	Config     configCmd   `kong:"cmd"`
	Configfile string      `kong:"type=path,help=${configfile_help},default=${configfile_default}"`
	CellarDir  string      `kong:"type=path,help=${cellar_dir_help}"`
}

func main() {
	parser := kong.Must(
		&cli,
		kongVars,
		downloadKongVars,
		configKongVars,
		kong.UsageOnError(),
	)

	kongCtx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)
	kongCtx.FatalIfErrorf(kongCtx.Run(kongCtx))
}
