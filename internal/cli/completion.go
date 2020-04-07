package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/killa-beez/gopkgs/sets/builtins"
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

func completionConfig(args []string) *bindown.ConfigFile {
	path := findConfigFileForCompletion(args)
	if path == "" {
		return nil
	}
	configFile, err := bindown.LoadConfigFile(path)
	if err != nil {
		return nil
	}
	return configFile
}

func allBins(cfg *bindown.ConfigFile) []string {
	if cfg == nil {
		return []string{}
	}
	bins := builtins.NewStringSet(len(cfg.Downloaders) * 10)
	for dlName, downloaders := range cfg.Downloaders {
		for _, dl := range downloaders {
			if dl.BinName == "" {
				bins.Add(dlName)
				continue
			}
			bins.Add(dl.BinName)
		}
	}
	return bins.Values()
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

var osCompleter = kong.CompleteSet(strings.Split(goosVals, "\n")...)

//from `go tool dist list | cut -f 1 -d '/' | sort | uniq`
const goosVals = `aix
android
darwin
dragonfly
freebsd
illumos
js
linux
nacl
netbsd
openbsd
plan9
solaris
windows`

var archCompleter = kong.CompleteSet(strings.Split(goarchVals, "\n")...)

//from `go tool dist list | cut -f 2 -d '/' | sort | uniq`
const goarchVals = `386
amd64
amd64p32
arm
arm64
mips
mips64
mips64le
mipsle
ppc64
ppc64le
s390x
wasm`
