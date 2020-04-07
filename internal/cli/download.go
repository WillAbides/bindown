package cli

import (
	"fmt"
	"path"
	"path/filepath"
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
	"download_extract_dir_help": `output path to directory where the downloaded archive is extracted`,
}

type downloadCmd struct {
	Arch           string     `kong:"help=${download_arch_help},default=${download_arch_default},completer=arch"`
	OS             string     `kong:"help=${download_os_help},default=${download_os_default},completer=os"`
	Force          bool       `kong:"help=${download_force_help}"`
	TargetFile     string     `kong:"required=true,arg,help=${download_target_file_help},completer=binpath"`
	ShowExtractDir bool       `kong:"name='extract-dir',help=${download_extract_dir_help}"`
	ConfigOpts     configOpts `kong:"embed"`
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
		URLChecksums:   config.URLChecksums,
	}
	err := downloader.Install(installOpts)
	if err != nil {
		return err
	}

	if d.ShowExtractDir {
		var extractDir string
		cellarDir := d.ConfigOpts.CellarDir
		if cellarDir == "" {
			cellarDir = filepath.Join(binDir, ".bindown")
		}
		extractDir, err = config.DownloaderExtractDir(downloader, binary, cellarDir)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(kctx.Stdout, "%s:  %s\n", "extract-dir", extractDir)
		if err != nil {
			return err
		}
	}

	return nil
}
