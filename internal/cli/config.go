package cli

import (
	"github.com/alecthomas/kong"
)

var configKongVars = kong.Vars{
	"config_format_help":              `formats the config file`,
	"config_checksums_help":           `update checksums in the config file`,
	"config_checksums_bin_help":       `name of the binary to update`,
	"config_validate_bin_help":        `name of the binary to validate`,
	"config_validate_help":            `validate that downloads work`,
	"config_install_completions_help": `install shell completions`,
}

type configCmd struct {
	Format             configFmtCmd               `kong:"cmd,help=${config_format_help}"`
	UpdateChecksums    configUpdateChecksumsCmd   `kong:"cmd,help=${config_checksums_bin_help}"`
	Validate           configValidateCmd          `kong:"cmd,help=${config_validate_help}"`
	InstallCompletions kong.InstallCompletionFlag `kong:"help=${config_install_completions_help}"`
	ConfigOpts         configOpts                 `kong:"embed"`
}

type configFmtCmd struct{}

func (c configFmtCmd) Run(kctx *kong.Context) error {
	config := configFile(kctx)
	if config != nil {
		return config.Write()
	}
	return nil
}

type configUpdateChecksumsCmd struct {
	TargetFile string `kong:"required=true,arg,help=${config_checksums_bin_help},completer=bin"`
}

func (d *configUpdateChecksumsCmd) Run(kctx *kong.Context) error {
	config := configFile(kctx)
	err := config.AddDownloaderChecksums(d.TargetFile, cli.Config.ConfigOpts.CellarDir)
	if err != nil {
		return err
	}
	return config.Write()
}

type configValidateCmd struct {
	Bin string `kong:"required=true,arg,help=${config_validate_bin_help},completer=bin"`
}

func (d configValidateCmd) Run(kctx *kong.Context) error {
	config := configFile(kctx)
	return config.Validate(d.Bin, cli.Config.ConfigOpts.CellarDir)
}
