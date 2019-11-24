package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindownloader"
)

var configKongVars = kong.Vars{
	"config_format_help":    `formats the config file`,
	"config_checksums_help": `update checksums in the config file`,
}

type configCmd struct {
	Format configFmtCmd `kong:"cmd,help=${config_format_help}"`
}

type configFmtCmd struct{}

func (c configFmtCmd) Run() error {
	config, err := bindownloader.LoadConfigFile(cli.Configfile)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(&config, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(cli.Configfile, b, 0600)
}
