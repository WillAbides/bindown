package main

import (
	"fmt"

	"github.com/willabides/bindown/v3/internal/bindown"
	"golang.org/x/exp/slices"
)

type supportedSystemCmd struct {
	List   supportedSystemListCmd    `kong:"cmd,help='list supported systems'"`
	Add    supportedSystemAddCmd     `kong:"cmd,help='add a supported system'"`
	Remove supportedSystemsRemoveCmd `kong:"cmd,help='remove a supported system'"`
}

type supportedSystemListCmd struct{}

func (c *supportedSystemListCmd) Run(ctx *runContext) error {
	cfg, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}

	for _, system := range cfg.Systems {
		fmt.Fprintln(ctx.stdout, system.String())
	}
	return nil
}

type supportedSystemsRemoveCmd struct {
	System bindown.SystemInfo `kong:"arg,predictor=system,help='system to remove'"`
}

func (c *supportedSystemsRemoveCmd) Run(ctx *runContext) error {
	cfg, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}

	systems := cfg.Systems
	newSystems := make([]bindown.SystemInfo, 0, len(systems))
	for _, system := range systems {
		if system.String() != c.System.String() {
			newSystems = append(newSystems, system)
		}
	}
	cfg.Systems = newSystems
	return cfg.WriteFile(ctx.rootCmd.JSONConfig)
}

type supportedSystemAddCmd struct {
	System        bindown.SystemInfo `kong:"arg,predictor=allSystems,help='system to add'"`
	SkipChecksums bool               `kong:"name=skipchecksums,help='do not add checksums for this system'"`
}

func (c *supportedSystemAddCmd) Run(ctx *runContext) error {
	cfg, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}

	for _, system := range cfg.Systems {
		if system.String() == c.System.String() {
			return nil
		}
	}
	cfg.Systems = append(cfg.Systems, c.System)
	var depsForSystem []string
	if !c.SkipChecksums {
		for depName := range cfg.Dependencies {
			depSystems, depErr := cfg.DependencySystems(depName)
			if depErr != nil {
				return depErr
			}
			if slices.Contains(depSystems, c.System) {
				depsForSystem = append(depsForSystem, depName)
			}
		}
		if len(depsForSystem) > 0 {
			err = cfg.AddChecksums(depsForSystem, []bindown.SystemInfo{c.System})
			if err != nil {
				return err
			}
		}
	}
	return cfg.WriteFile(ctx.rootCmd.JSONConfig)
}
