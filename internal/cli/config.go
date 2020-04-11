package cli

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v3"
)

var configKongVars = kong.Vars{
	"config_format_help":              `formats the config file`,
	"config_checksums_help":           `update checksums in the config file`,
	"config_checksums_bin_help":       `name of the binary to update`,
	"config_validate_bin_help":        `name of the binary to validate`,
	"config_validate_help":            `validate that downloads work`,
	"config_install_completions_help": `install shell completions`,
	"config_extract_path_help":        `output path to directory where the downloaded archive is extracted`,
}

type configCmd struct {
	Format             configFmtCmd               `kong:"cmd,help=${config_format_help}"`
	UpdateChecksums    configUpdateChecksumsCmd   `kong:"cmd,help=${config_checksums_bin_help}"`
	Validate           configValidateCmd          `kong:"cmd,help=${config_validate_help}"`
	InstallCompletions kong.InstallCompletionFlag `kong:"help=${config_install_completions_help}"`
	ExtractPath        configExtractPathCmd       `kong:"cmd,help=${config_extract_path_help}"`
	ConfigOpts         configOpts                 `kong:"embed"`
}

type configExtractPathCmd struct {
	TargetFile string             `kong:"arg,required=true,help=${config_extract_path_help},completer=binpath"`
	System     bindown.SystemInfo `kong:"name=system,default=${system_default},help=${system_help},completer=system"`
}

func (d configExtractPathCmd) Run(kctx *kong.Context) error {
	config := configFile(kctx, cli.Config.ConfigOpts.Configfile)
	binary := path.Base(d.TargetFile)
	binDir := path.Dir(d.TargetFile)

	cellarDir := cli.Config.ConfigOpts.CellarDir
	if cellarDir == "" {
		cellarDir = filepath.Join(binDir, ".bindown")
	}
	extractDir, err := config.ExtractPath(binary, d.System, cellarDir)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(kctx.Stdout, extractDir)
	return err
}

type configFmtCmd struct{}

func (c configFmtCmd) Run(kctx *kong.Context) error {
	config := configFile(kctx, cli.Config.ConfigOpts.Configfile)
	if config != nil {
		return config.Write()
	}
	return nil
}

type configUpdateChecksumsCmd struct {
	TargetFile string               `kong:"required=true,arg,help=${config_checksums_bin_help},completer=bin"`
	Systems    []bindown.SystemInfo `kong:"name=system,default=${system_default},completer=system"`
}

func (d *configUpdateChecksumsCmd) Run(kctx *kong.Context) error {
	config := configFile(kctx, cli.Config.ConfigOpts.Configfile)
	err := config.AddChecksums(&bindown.ConfigAddChecksumsOptions{
		Dependencies: []string{filepath.Base(d.TargetFile)},
		Systems:      d.Systems,
	})
	if err != nil {
		return err
	}
	return config.Write()
}

type configValidateCmd struct {
	Bin     string               `kong:"required=true,arg,help=${config_validate_bin_help},completer=bin"`
	Systems []bindown.SystemInfo `kong:"name=system,default=${system_default},completer=system"`
}

func (d configValidateCmd) Run(kctx *kong.Context) error {
	config := configFile(kctx, cli.Config.ConfigOpts.Configfile)
	return config.Validate([]string{d.Bin}, d.Systems)
}
