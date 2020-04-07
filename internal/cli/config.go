package cli

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/alecthomas/kong"
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
	TargetFile string     `kong:"arg,required=true,help=${config_extract_path_help},completer=binpath"`
	OSArchOpts osArchOpts `kong:"embed"`
}

func (d configExtractPathCmd) Run(kctx *kong.Context) error {
	config := configFile(kctx)
	binary := path.Base(d.TargetFile)
	binDir := path.Dir(d.TargetFile)
	downloader := config.Downloader(binary, d.OSArchOpts.OS, d.OSArchOpts.Arch)
	if downloader == nil {
		return fmt.Errorf(`no downloader configured for:
bin: %s
os: %s
arch: %s`, binary, d.OSArchOpts.OS, d.OSArchOpts.Arch)
	}

	cellarDir := cli.Config.ConfigOpts.CellarDir
	if cellarDir == "" {
		cellarDir = filepath.Join(binDir, ".bindown")
	}
	extractDir, err := config.DownloaderExtractDir(downloader, binary, cellarDir)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(kctx.Stdout, "%s:  %s\n", "extract-dir", extractDir)
	if err != nil {
		return err
	}
	return nil
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
