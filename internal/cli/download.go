package cli

import (
	"fmt"
	"path"
	"runtime"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v2"
)

var downloadKongVars = kong.Vars{
	"download_arch_help":        `download for this architecture`,
	"download_arch_default":     runtime.GOARCH,
	"download_os_help":          `download for this operating system`,
	"download_os_default":       runtime.GOOS,
	"download_force_help":       `force download even if it already exists`,
	"download_target_file_help": `file to download`,
}

type downloadCmd struct {
	Arch       string     `kong:"help=${download_arch_help},default=${download_arch_default},completer=arch"`
	OS         string     `kong:"help=${download_os_help},default=${download_os_default},completer=os"`
	Force      bool       `kong:"help=${download_force_help}"`
	TargetFile string     `kong:"required=true,arg,help=${download_target_file_help},completer=binpath"`
	ConfigOpts configOpts `kong:"embed"`
}

func (d *downloadCmd) Run(kctx *kong.Context) error {
	config := configFile(kctx)
	binary := path.Base(d.TargetFile)
	binDir := path.Dir(d.TargetFile)

	downloader := config.Downloader(binary, d.OS, d.Arch)
	if downloader == nil {
		return fmt.Errorf(`no downloader configured for:
bin: %s
os: %s
arch: %s`, binary, d.OS, d.Arch)
	}

	installOpts := bindown.InstallOpts{
		DownloaderName: binary,
		TargetDir:      binDir,
		Force:          d.Force,
		CellarDir:      d.ConfigOpts.CellarDir,
	}

	return downloader.Install(installOpts)
}
