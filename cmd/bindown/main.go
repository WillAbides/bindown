package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/alecthomas/kong"
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
	Configfile string      `kong:"type=multipath,help=${configfile_help},default=${configfile_default},env='BINDOWN_CONFIG_FILE'"`
	CellarDir  string      `kong:"type=path,help=${cellar_dir_help},env='BINDOWN_CELLAR'"`
}

func multipathMapper(ctx *kong.DecodeContext, target reflect.Value) error {
	if target.Kind() != reflect.String {
		return fmt.Errorf("\"multipath\" type must be applied to a string not %s", target.Type())
	}
	var path string
	err := ctx.Scan.PopValueInto("file", &path)
	if err != nil {
		return err
	}

	for _, configFile := range strings.Split(path, "|") {
		configFile = kong.ExpandPath(configFile)
		stat, err := os.Stat(configFile)
		if err != nil {
			continue
		}
		if stat.IsDir() {
			continue
		}
		target.SetString(configFile)
		return nil
	}
	return fmt.Errorf("not found")
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

	kongCtx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)
	kongCtx.FatalIfErrorf(kongCtx.Run(kongCtx))
}
