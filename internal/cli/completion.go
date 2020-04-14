package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v3"
	"github.com/willabides/bindown/v3/internal/configfile"
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
	return prepCompletionConfigFile(kongVars["configfile_default"])
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

func completionConfig(args []string) *configfile.ConfigFile {
	path := findConfigFileForCompletion(args)
	if path == "" {
		return nil
	}
	configFile, err := configfile.LoadConfigFile(path, true)
	if err != nil {
		return nil
	}
	return configFile
}

func allBins(cfg *configfile.ConfigFile) []string {
	if cfg == nil {
		return []string{}
	}
	system := bindown.SystemInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
	bins := make([]string, 0, len(cfg.Dependencies))
	for dlName := range cfg.Dependencies {
		bn, err := cfg.BinName(dlName, system)
		if err != nil {
			return []string{}
		}
		bins = append(bins, bn)
	}
	return bins
}

var binCompleter = kong.CompleterFunc(func(a kong.CompleterArgs) []string {
	cfg := completionConfig(a.Completed())
	return kong.CompleteSet(allBins(cfg)...).Options(a)
})

var binPathCompleter = kong.CompleterFunc(func(a kong.CompleterArgs) []string {
	cfg := completionConfig(a.Completed())
	bins := allBins(cfg)
	dir, _ := filepath.Split(a.Last())
	for i, bin := range bins {
		bins[i] = filepath.Join(dir, bin)
	}
	return kong.CompleteOr(
		kong.CompleteDirs(),
		kong.CompleteSet(bins...),
	).Options(a)
})

var systemCompleter = kong.CompleteSet(strings.Split(goDists, "\n")...)

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
