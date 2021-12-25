package main

import (
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v3"
)

func findConfigFileForCompletion(args []string) string {
	for i, arg := range args {
		if len(args) == i+1 {
			continue
		}
		if arg != "--configfile" {
			continue
		}
		return prepCompletionConfigFile(args[i+1])
	}
	cf, ok := os.LookupEnv("BINDOWN_CONFIG_FILE")
	if ok {
		return prepCompletionConfigFile(cf)
	}
	for _, cf := range defaultConfigFilenames {
		if _, err := os.Stat(cf); err == nil {
			return prepCompletionConfigFile(cf)
		}
	}
	return prepCompletionConfigFile("")
}

// prepCompletionConfigFile expands the path and returns "" if it isn't an existing file
func prepCompletionConfigFile(path string) string {
	path = kong.ExpandPath(path)
	stat, err := os.Stat(path)
	if err != nil {
		return ""
	}
	if stat.IsDir() {
		return ""
	}
	return path
}

func completionConfig(args []string) *bindown.ConfigFile {
	path := findConfigFileForCompletion(args)
	if path == "" {
		return nil
	}
	configFile, err := bindown.LoadConfigFile(path, true)
	if err != nil {
		return nil
	}
	return configFile
}

func allDependencies(cfg *bindown.ConfigFile) []string {
	if cfg == nil {
		return []string{}
	}
	system := bindown.SystemInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
	dependencies := make([]string, 0, len(cfg.Dependencies))
	for depName := range cfg.Dependencies {
		bn, err := cfg.BinName(depName, system)
		if err != nil {
			return []string{}
		}
		dependencies = append(dependencies, bn)
	}
	sort.Strings(dependencies)
	return dependencies
}

var templateSourceCompleter = kong.CompleterFunc(func(a kong.CompleterArgs) []string {
	cfg := completionConfig(a.Completed())
	if cfg == nil {
		return []string{}
	}

	opts := make([]string, 0, len(cfg.TemplateSources))
	for src := range cfg.TemplateSources {
		opts = append(opts, src)
	}
	return kong.CompleteSet(opts...).Options(a)
})

var binCompleter = kong.CompleterFunc(func(a kong.CompleterArgs) []string {
	cfg := completionConfig(a.Completed())
	return kong.CompleteSet(allDependencies(cfg)...).Options(a)
})

var systemCompleter = kong.CompleterFunc(func(a kong.CompleterArgs) []string {
	cfg := completionConfig(a.Completed())
	opts := make([]string, 0, len(cfg.Systems))
	for _, system := range cfg.Systems {
		opts = append(opts, system.String())
	}
	return kong.CompleteSet(opts...).Options(a)
})

var allSystemsCompleter = kong.CompleteSet(strings.Split(goDists, "\n")...)

// from `go tool dist list`
const goDists = `aix/ppc64
android/386
android/amd64
android/arm
android/arm64
darwin/386
darwin/amd64
darwin/arm
darwin/arm64
dragonfly/amd64
freebsd/386
freebsd/amd64
freebsd/arm
freebsd/arm64
illumos/amd64
js/wasm
linux/386
linux/amd64
linux/arm
linux/arm64
linux/mips
linux/mips64
linux/mips64le
linux/mipsle
linux/ppc64
linux/ppc64le
linux/riscv64
linux/s390x
netbsd/386
netbsd/amd64
netbsd/arm
netbsd/arm64
openbsd/386
openbsd/amd64
openbsd/arm
openbsd/arm64
plan9/386
plan9/amd64
plan9/arm
solaris/amd64
windows/386
windows/amd64
windows/arm`
