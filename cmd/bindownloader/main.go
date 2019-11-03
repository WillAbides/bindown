package main

import (
	"os"
	"path"
	"runtime"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindownloader"
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

	downloaders, err := bindownloader.LoadConfigFile(cli.Config)
	if err != nil {
		kctx.Errorf("error loading config from %q\n", cli.Config)
		os.Exit(1)
	}

	binary := path.Base(cli.TargetFile)
	binDir := path.Dir(cli.TargetFile)

	downloader := downloaders.Downloader(binary, cli.OS, cli.Arch)
	if downloader == nil {
		kctx.Errorf(`no downloader configured for:
bin: %s
os: %s
arch: %s
`, binary, cli.OS, cli.Arch)
		os.Exit(1)
	}

	err = downloader.Install(binDir, cli.Force)

	kctx.FatalIfErrorf(err)
}
