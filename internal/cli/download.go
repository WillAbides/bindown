package cli

import (
	"path"
	"path/filepath"

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
	config := configFile(kctx, d.ConfigOpts.Configfile)
	binary := path.Base(d.TargetFile)
	binDir := path.Dir(d.TargetFile)
	system := bindown.SystemInfo{
		OS:   d.OSArchOpts.OS,
		Arch: d.OSArchOpts.Arch,
	}
	cellarDir := cli.Config.ConfigOpts.CellarDir
	if cellarDir == "" {
		cellarDir = filepath.Join(binDir, ".bindown")
	}
	return config.Install(binary, system, &bindown.ConfigInstallOpts{
		CellarDir: cellarDir,
		TargetDir: binDir,
		Force:     d.Force,
	})
}
