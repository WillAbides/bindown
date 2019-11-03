package main

import (
	"os"
	"path"
	"runtime"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindownloader/internal"
)

var kongVars = kong.Vars{
	"arch_help":      `download for this architecture`,
	"arch_default":   runtime.GOARCH,
	"os_help":        `download for this operating system`,
	"os_default":     runtime.GOOS,
	"config_help":    `file with tool definitions`,
	"config_default": `buildtools.json`,
	"force_help":     `force download even if it already exists`,
}

var cli struct {
	Arch       string `kong:"help=${arch_help},default=${arch_default}"`
	OS         string `kong:"help=${os_help},default=${os_default}"`
	Config     string `kong:"type=path,help=${config_help},default=${config_default}"`
	Force      bool   `kong:"help=${force_help}"`
	TargetFile string `kong:"arg,help='file to download'"`
}

func main() {
	parser := kong.Must(&cli, kongVars, kong.UsageOnError())
	kctx, err := parser.Parse(os.Args[1:])

	parser.FatalIfErrorf(err)

	targetDir := path.Dir(cli.TargetFile)
	targetFile := path.Base(cli.TargetFile)

	downloaders, err := internal.FromFile(cli.Config)
	if err != nil {
		kctx.Errorf("error loading config from %q\n", cli.Config)
		os.Exit(1)
	}

	config := internal.InstallToolConfig{
		ToolName: targetFile,
		BinDir:   targetDir,
		OpSys:    cli.OS,
		Arch:     cli.Arch,
		Force:    false,
	}

	kctx.FatalIfErrorf(downloaders.InstallTool(config))
}
