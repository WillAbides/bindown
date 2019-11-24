package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindownloader"
)

var kongVars = kong.Vars{
	"arch_help":          `download for this architecture`,
	"arch_default":       runtime.GOARCH,
	"os_help":            `download for this operating system`,
	"os_default":         runtime.GOOS,
	"configfile_help":    `file with tool definitions`,
	"configfile_default": `buildtools.json`,
	"force_help":         `force download even if it already exists`,
	"cellar_dir_help":    `directory where downloads will be cached`,
	"config_format_help": `formats the config file`,
	"download_help":      `download a bin`,
}

var version = "unknown"

var cli struct {
	Version    versionCmd  `kong:"cmd"`
	Download   downloadCmd `kong:"cmd,help=${download_help}"`
	Config     configCmd   `kong:"cmd"`
	Configfile string      `kong:"type=path,help=${configfile_help},default=${configfile_default}"`
	CellarDir  string      `kong:"type=path,help=${cellar_dir_help}"`
}

type versionCmd struct{}

func (*versionCmd) Run(k *kong.Context) error {
	k.Printf("version %s", version)
	return nil
}

type downloadCmd struct {
	Arch       string `kong:"help=${arch_help},default=${arch_default}"`
	OS         string `kong:"help=${os_help},default=${os_default}"`
	Force      bool   `kong:"help=${force_help}"`
	TargetFile string `kong:"required=true,arg,help='file to download'"`
}

func (d *downloadCmd) Run(*kong.Context) error {
	config, err := bindownloader.LoadConfigFile(cli.Configfile)
	if err != nil {
		return fmt.Errorf("error loading config from %q", cli.Configfile)
	}
	binary := path.Base(d.TargetFile)
	binDir := path.Dir(d.TargetFile)

	downloader := config.Downloader(binary, d.OS, d.Arch)
	if downloader == nil {
		return fmt.Errorf(`no downloader configured for:
bin: %s
os: %s
arch: %s`, binary, d.OS, d.Arch)
	}

	installOpts := bindownloader.InstallOpts{
		DownloaderName: binary,
		TargetDir:      binDir,
		Force:          d.Force,
		CellarDir:      cli.CellarDir,
	}

	return downloader.Install(installOpts)
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

func main() {
	parser := kong.Must(&cli, kongVars, kong.UsageOnError())

	kongCtx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)
	kongCtx.FatalIfErrorf(kongCtx.Run(kongCtx))
}
