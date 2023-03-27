package main

import (
	"context"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/posener/complete"
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

func getCompletionSource(args []string) string {
	for i, arg := range args {
		if len(args) == i+1 {
			continue
		}
		if arg != "--source" {
			continue
		}
		return args[i+1]
	}
	return ""
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

func completionConfig(ctx context.Context, args []string) *bindown.ConfigFile {
	path := findConfigFileForCompletion(args)
	if path == "" {
		return nil
	}
	configFile, err := bindown.LoadConfigFile(ctx, path, true)
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

func templateSourceCompleter(ctx context.Context) complete.PredictFunc {
	return func(a complete.Args) []string {
		cfg := completionConfig(ctx, a.Completed)
		if cfg == nil {
			return []string{}
		}

		opts := make([]string, 0, len(cfg.TemplateSources))
		for src := range cfg.TemplateSources {
			opts = append(opts, src)
		}
		return complete.PredictSet(opts...).Predict(a)
	}
}

func templateCompleter(ctx context.Context) complete.PredictFunc {
	return func(a complete.Args) []string {
		cfg := completionConfig(ctx, a.Completed)
		if cfg == nil {
			return []string{}
		}
		srcName := getCompletionSource(a.Completed)
		if srcName == "" {
			return localTemplateCompleter(ctx)(a)
		}
		srcURL, ok := cfg.TemplateSources[srcName]
		if !ok {
			return []string{}
		}
		srcCfg, err := bindown.ConfigFromURL(ctx, srcURL)
		if err != nil {
			return []string{}
		}
		opts := make([]string, 0, len(srcCfg.TemplateSources))
		for src := range srcCfg.Templates {
			opts = append(opts, src)
		}
		return complete.PredictSet(opts...).Predict(a)
	}
}

func localTemplateCompleter(ctx context.Context) complete.PredictFunc {
	return func(a complete.Args) []string {
		cfg := completionConfig(ctx, a.Completed)
		if cfg == nil {
			return []string{}
		}

		opts := make([]string, 0, len(cfg.Templates))
		for tmpl := range cfg.Templates {
			opts = append(opts, tmpl)
		}
		return complete.PredictSet(opts...).Predict(a)
	}
}

func localTemplateFromSourceCompleter(ctx context.Context) complete.PredictFunc {
	return func(a complete.Args) []string {
		cfg := completionConfig(ctx, a.Completed)
		if cfg == nil {
			return []string{}
		}

		opts := make([]string, 0, len(cfg.Templates))
		for tmpl := range cfg.Templates {
			if strings.Contains(tmpl, "#") {
				opts = append(opts, tmpl)
			}
		}
		return complete.PredictSet(opts...).Predict(a)
	}
}

func binCompleter(ctx context.Context) complete.PredictFunc {
	return func(a complete.Args) []string {
		cfg := completionConfig(ctx, a.Completed)
		return complete.PredictSet(allDependencies(cfg)...).Predict(a)
	}
}

func systemCompleter(ctx context.Context) complete.PredictFunc {
	return func(a complete.Args) []string {
		cfg := completionConfig(ctx, a.Completed)
		opts := make([]string, 0, len(cfg.Systems))
		for _, system := range cfg.Systems {
			opts = append(opts, system.String())
		}
		return complete.PredictSet(opts...).Predict(a)
	}
}

var allSystemsCompleter = complete.PredictFunc(func(a complete.Args) []string {
	return append([]string{"current"}, strings.Split(goDists, "\n")...)
})

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
