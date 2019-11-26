package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v2"
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
	config, err := bindown.LoadConfigFile(cli.Configfile)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(&config, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(cli.Configfile, b, 0600)
}

type configUpdateChecksumsCmd struct {
	TargetFile string `kong:"required=true,arg,help=${config_checksums_bin_help}"`
}

func (d *configUpdateChecksumsCmd) Run(kctx *kong.Context) error {
	config, err := bindown.LoadConfigFile(cli.Configfile)
	if err != nil {
		return fmt.Errorf("error loading config from %q", cli.Configfile)
	}
	tmpDir, err := ioutil.TempDir("", "bindown")
	if err != nil {
		return err
	}
	defer func() {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			kctx.Errorf("error deleting temp directory, %q", tmpDir)
		}
	}()

	binary := path.Base(d.TargetFile)
	binDir := path.Dir(d.TargetFile)

	cellarDir := cli.CellarDir
	if cellarDir == "" {
		cellarDir = filepath.Join(tmpDir, "cellar")
	}

	downloaders, ok := config.Downloaders[binary]
	if !ok {
		return fmt.Errorf("nothing configured for %q", binary)
	}

	for _, downloader := range downloaders {
		err = downloader.UpdateChecksum(bindown.UpdateChecksumOpts{
			DownloaderName: binary,
			CellarDir:      cellarDir,
			TargetDir:      binDir,
		})
		if err != nil {
			return err
		}
	}

	b, err := json.MarshalIndent(&config, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(cli.Configfile, b, 0600)
}

type configValidateCmd struct {
	Bin string `kong:"required=true,arg,help=${config_validate_bin_help}"`
}

func (d configValidateCmd) Run(kctx *kong.Context) error {
	config, err := bindown.LoadConfigFile(cli.Configfile)
	if err != nil {
		return fmt.Errorf("error loading config from %q", cli.Configfile)
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
