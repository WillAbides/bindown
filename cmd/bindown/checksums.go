package main

import (
	"github.com/willabides/bindown/v4/internal/bindown"
)

type checksumsCmd struct {
	Add   addChecksumsCmd   `kong:"cmd,help=${add_checksums_help}"`
	Prune pruneChecksumsCmd `kong:"cmd,help=${prune_checksums_help}"`
}

type addChecksumsCmd struct {
	Dependency []string         `kong:"help=${checksums_dep_help},predictor=bin"`
	Systems    []bindown.System `kong:"name=system,help=${systems_help},predictor=allSystems"`
}

func (d *addChecksumsCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}
	err = config.AddChecksums(d.Dependency, d.Systems)
	if err != nil {
		return err
	}
	return config.WriteFile(ctx.rootCmd.JSONConfig)
}

type pruneChecksumsCmd struct{}

func (d *pruneChecksumsCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}
	err = config.PruneChecksums()
	if err != nil {
		return err
	}
	return config.WriteFile(ctx.rootCmd.JSONConfig)
}
