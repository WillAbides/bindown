package main

import (
	"os"
	"path"
	"runtime"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindownloader"
)

var kongVars = kong.Vars{
	"arch_help":             `download for this architecture`,
	"arch_default":          runtime.GOARCH,
	"os_help":               `download for this operating system`,
	"os_default":            runtime.GOOS,
	"config_help":           `file with tool definitions`,
	"config_default":        `buildtools.json`,
	"force_help":            `force download even if it already exists`,
	"cellar_dir_help":       `directory where downloads will be cached`,
	"update_checksums_help": `update the config's checksums for the given binary`,
}

var cli struct {
	Arch            string `kong:"help=${arch_help},default=${arch_default}"`
	OS              string `kong:"help=${os_help},default=${os_default}"`
	Config          string `kong:"type=path,help=${config_help},default=${config_default}"`
	Force           bool   `kong:"help=${force_help}"`
	TargetFile      string `kong:"arg,help='file to download'"`
	CellarDir       string `kong:"type=path,help=${cellar_dir_help}"`
	UpdateChecksums bool   `kong:"help=${update_checksums_help}"`
}

func main() {
	parser := kong.Must(&cli, kongVars, kong.UsageOnError())
	kctx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)

	config, err := bindownloader.LoadConfigFile(cli.Config)
	if err != nil {
		kctx.Errorf("error loading config from %q\n", cli.Config)
		os.Exit(1)
	}

	binary := path.Base(cli.TargetFile)
	binDir := path.Dir(cli.TargetFile)

	if cli.UpdateChecksums {
		err := updateChecksums(config, binary, binDir)
		kctx.FatalIfErrorf(err)
		kctx.Exit(0)
	}

	downloader := config.Downloader(binary, cli.OS, cli.Arch)
	if downloader == nil {
		kctx.Errorf(`no downloader configured for:
bin: %s
os: %s
arch: %s
`, binary, cli.OS, cli.Arch)
		os.Exit(1)
	}

	installOpts := bindownloader.InstallOpts{
		DownloaderName: binary,
		TargetDir:      binDir,
		Force:          cli.Force,
		CellarDir:      cli.CellarDir,
	}

	err = downloader.Install(installOpts)

	kctx.FatalIfErrorf(err)
}

func updateChecksums(config bindownloader.Config, binary, binDir string) error {
	//downloaders := config.Downloaders(binary)
	//for _, downloader := range downloaders {
	//	err := downloader.UpdateChecksum(bindownloader.UpdateChecksumOpts{
	//		DownloaderName: binary,
	//		CellarDir:      binDir,
	//		TargetDir:      cli.CellarDir,
	//	})
	//	if err != nil {
	//		return err
	//	}
	//}
	return nil
}
