package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/killa-beez/gopkgs/sets/builtins"
	"github.com/posener/complete"
	"github.com/willabides/bindown/v2"
)

func findConfigFileForPredictor(args []string) string {
	for i, arg := range args {
		if len(args) == i+1 {
			continue
		}
		if arg != "--configfile" {
			continue
		}
		return multifileFindExisting(args[i+1])
	}
	cf, ok := os.LookupEnv("BINDOWN_CONFIG_FILE")
	if ok {
		return multifileFindExisting(cf)
	}
	return multifileFindExisting(kongVars["configfile_default"])
}

func predictorConfig(args []string) *bindown.ConfigFile {
	path := findConfigFileForPredictor(args)
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

var binPredictor = complete.PredictFunc(func(a complete.Args) []string {
	cfg := predictorConfig(a.Completed)
	return complete.PredictSet(allBins(cfg)...).Predict(a)
})

var binPathPredictor = complete.PredictFunc(func(a complete.Args) []string {
	cfg := predictorConfig(a.Completed)
	bins := allBins(cfg)
	dir, _ := filepath.Split(a.Last)
	for i, bin := range bins {
		bins[i] = filepath.Join(dir, bin)
	}
	return complete.PredictOr(
		complete.PredictDirs("*"),
		complete.PredictSet(bins...),
	).Predict(a)
})

var osPredictor = complete.PredictSet(strings.Split(goosVals, "\n")...)

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

var archPredictor = complete.PredictSet(strings.Split(goarchVals, "\n")...)

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
