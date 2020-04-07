package cli

import (
	"fmt"
	"path"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v3"
)

var downloadKongVars = kong.Vars{
	"download_force_help":       `force download even if it already exists`,
	"download_target_file_help": `file to download`,
}

type downloadCmd struct {
	Force      bool       `kong:"help=${download_force_help}"`
	TargetFile string     `kong:"required=true,arg,help=${download_target_file_help},completer=binpath"`
	ConfigOpts configOpts `kong:"embed"`
	OSArchOpts osArchOpts `kong:"embed"`
}

func (d *downloadCmd) Run(kctx *kong.Context) error {
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

	installOpts := bindown.InstallOpts{
		DownloaderName: binary,
		TargetDir:      binDir,
		Force:          d.Force,
		CellarDir:      d.ConfigOpts.CellarDir,
		URLChecksums:   config.URLChecksums,
	}

	return downloader.Install(installOpts)
}
