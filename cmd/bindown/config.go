package main

import (
	"fmt"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v2/pkg/config"
	"go.uber.org/multierr"
)

var configKongVars = kong.Vars{
	"config_format_help":        `formats the config file`,
	"config_checksums_help":     `update checksums in the config file`,
	"config_checksums_bin_help": `name of the binary to update`,
	"config_validate_bin_help":  `name of the binary to validate`,
	"config_validate_help":      `validate that downloads work`,
}

type configCmd struct {
	Format          configFmtCmd             `kong:"cmd,help=${config_format_help}"`
	UpdateChecksums configUpdateChecksumsCmd `kong:"cmd,help=${config_checksums_bin_help}"`
	Validate        configValidateCmd        `kong:"cmd,help=${config_validate_help}"`
}

type configFmtCmd struct{}

func (c configFmtCmd) Run() error {
	configFile, err := config.NewConfigFile(cli.Configfile)
	if err != nil {
		return err
	}
	return configFile.WriteFile()
}

type configUpdateChecksumsCmd struct {
	TargetFile string `kong:"required=true,arg,help=${config_checksums_bin_help}"`
}

func (d *configUpdateChecksumsCmd) Run(*kong.Context) error {
	configFile, err := config.NewConfigFile(cli.Configfile)
	if err != nil {
		return err
	}
	binary := filepath.Base(d.TargetFile)
	err = configFile.UpdateChecksums(binary, cli.CellarDir)
	if err != nil {
		return err
	}
	return configFile.WriteFile()
}

type configValidateCmd struct {
	Bin string `kong:"required=true,arg,help=${config_validate_bin_help}"`
}

func (d configValidateCmd) Run(kctx *kong.Context) error {
	config, err := config.NewConfigFile(cli.Configfile)
	if err != nil {
		return err
	}
	err = config.Validate(d.Bin, cli.CellarDir)
	if err == nil {
		return nil
	}
	errs := multierr.Errors(err)
	for _, gotErr := range errs {
		kctx.Printf("%s\n", gotErr.Error())
	}
	return fmt.Errorf("could not validate")
}
