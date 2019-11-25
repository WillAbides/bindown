package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindownloader"
)

var configKongVars = kong.Vars{
	"config_format_help":        `formats the config file`,
	"config_checksums_help":     `update checksums in the config file`,
	"config_checksums_bin_help": `name of the binary to update`,
}

type configCmd struct {
	Format          configFmtCmd             `kong:"cmd,help=${config_format_help}"`
	UpdateChecksums configUpdateChecksumsCmd `kong:"cmd,help=${config_checksums_bin_help}"`
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

type configUpdateChecksumsCmd struct {
	TargetFile string `kong:"required=true,arg,help=${config_checksums_bin_help}"`
}

func (d *configUpdateChecksumsCmd) Run(*kong.Context) error {
	config, err := bindownloader.LoadConfigFile(cli.Configfile)
	if err != nil {
		return fmt.Errorf("error loading config from %q", cli.Configfile)
	}
	binary := path.Base(d.TargetFile)
	binDir := path.Dir(d.TargetFile)

	downloaders, ok := config[binary]
	if !ok {
		return fmt.Errorf("nothing configured for %q", binary)
	}

	for _, downloader := range downloaders {
		err = downloader.UpdateChecksum(bindownloader.UpdateChecksumOpts{
			DownloaderName: binary,
			CellarDir:      cli.CellarDir,
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
