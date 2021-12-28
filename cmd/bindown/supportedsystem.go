package main

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v3"
)

type supportedSystemCmd struct {
	List   supportedSystemListCmd    `kong:"cmd,help='list supported systems'"`
	Add    supportedSystemAddCmd     `kong:"cmd,help='add a supported system'"`
	Remove supportedSystemsRemoveCmd `kong:"cmd,help='remove a supported system'"`
}

type supportedSystemListCmd struct{}

func (c *supportedSystemListCmd) Run(ctx *kong.Context) error {
	cfgIface, err := configLoader.Load(cli.Configfile, true)
	if err != nil {
		return err
	}
	cfg := cfgIface.(*bindown.ConfigFile)
	for _, system := range cfg.Systems {
		fmt.Fprintln(ctx.Stdout, system.String())
	}
	return nil
}

type supportedSystemsRemoveCmd struct {
	System bindown.SystemInfo `kong:"arg,predictor=system,help='system to remove'"`
}

func (c *supportedSystemsRemoveCmd) Run() error {
	cfgIface, err := configLoader.Load(cli.Configfile, true)
	if err != nil {
		return err
	}
	cfg := cfgIface.(*bindown.ConfigFile)
	systems := cfg.Systems
	newSystems := make([]bindown.SystemInfo, 0, len(systems))
	for _, system := range systems {
		if system.String() != c.System.String() {
			newSystems = append(newSystems, system)
		}
	}
	cfg.Systems = newSystems
	return cfg.Write(cli.JSONConfig)
}

type supportedSystemAddCmd struct {
	System        bindown.SystemInfo `kong:"arg,predictor=allSystems,help='system to add'"`
	SkipChecksums bool               `kong:"name=skipchecksums,help='do not add checksums for this system'"`
}

func (c *supportedSystemAddCmd) Run() error {
	cfgIface, err := configLoader.Load(cli.Configfile, true)
	if err != nil {
		return err
	}
	cfg := cfgIface.(*bindown.ConfigFile)
	for _, system := range cfg.Systems {
		if system.String() == c.System.String() {
			return nil
		}
	}
	cfg.Systems = append(cfg.Systems, c.System)
	if !c.SkipChecksums {
		var updateDeps []string
		updateDeps, err = dependenciesWithSystem(cfg, c.System)
		if err != nil {
			return err
		}
		err = cfg.AddChecksums(updateDeps, []bindown.SystemInfo{c.System})
		if err != nil {
			return err
		}
	}
	return cfg.Write(cli.JSONConfig)
}

func dependenciesWithSystem(cfg *bindown.ConfigFile, system bindown.SystemInfo) ([]string, error) {
	deps := make([]string, 0, len(cfg.Dependencies))
	for depName := range cfg.Dependencies {
		depSystems, err := cfg.DependencySystems(depName)
		if err != nil {
			return nil, err
		}
		for _, s := range depSystems {
			if s.OS == system.OS && s.Arch == system.Arch {
				deps = append(deps, depName)
				break
			}
		}
	}
	return deps, nil
}
