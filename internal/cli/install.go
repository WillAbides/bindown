package cli

import (
	"path"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v3"
)

var installKongVars = kong.Vars{
	"install_force_help":       `force install even if it already exists`,
	"install_target_file_help": `file to install`,
}

type installCmd struct {
	Force      bool               `kong:"help=${install_force_help}"`
	TargetFile string             `kong:"required=true,arg,help=${install_target_file_help},completer=binpath"`
	ConfigOpts configOpts         `kong:"embed"`
	System     bindown.SystemInfo `kong:"name=system,default=${system_default},help=${system_help},completer=system"`
}

func (d *installCmd) Run(kctx *kong.Context) error {
	config := configFile(kctx, d.ConfigOpts.Configfile)
	binary := path.Base(d.TargetFile)
	binDir := path.Dir(d.TargetFile)
	cellarDir := cli.Config.ConfigOpts.CellarDir
	if cellarDir == "" {
		cellarDir = filepath.Join(binDir, ".bindown")
	}
	return config.Install(binary, d.System, &bindown.ConfigInstallOpts{
		CellarDir: cellarDir,
		TargetDir: binDir,
		Force:     d.Force,
	})
}
