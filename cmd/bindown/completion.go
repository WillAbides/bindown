package main

import (
	"context"
	"os"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/posener/complete"
	"github.com/willabides/bindown/v3/internal/bindown"
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

func completionConfig(ctx context.Context, args []string) *bindown.Config {
	path := findConfigFileForCompletion(args)
	if path == "" {
		return nil
	}
	configFile, err := bindown.NewConfig(ctx, path, true)
	if err != nil {
		return nil
	}
	return configFile
}

func allDependencies(cfg *bindown.Config) []string {
	if cfg == nil {
		return []string{}
	}
	system := bindown.CurrentSystem
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
		srcCfg, err := bindown.NewConfig(ctx, srcURL, true)
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
			opts = append(opts, string(system))
		}
		return complete.PredictSet(opts...).Predict(a)
	}
}

var allSystemsCompleter = complete.PredictFunc(func(a complete.Args) []string {
	return append([]string{"current"}, strings.Split(goDists, "\n")...)
})
