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
	System bindown.SystemInfo `kong:"arg,completer=system,help='system to remove'"`
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
	System bindown.SystemInfo `kong:"arg,completer=allSystems,help='system to add'"`
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
	return cfg.Write(cli.JSONConfig)
}
