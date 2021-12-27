package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v3"
)

type templateSourceCmd struct {
	List   templateSourceListCmd   `kong:"cmd,help='list configured template sources'"`
	Add    templateSourceAddCmd    `kong:"cmd,help='add a template source'"`
	Remove templateSourceRemoveCmd `kong:"cmd,help='remove a template source'"`
}

type templateSourceListCmd struct{}

func (c *templateSourceListCmd) Run(ctx *kong.Context) error {
	cfg, err := configLoader.Load(cli.Configfile, true)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(ctx.Stdout, 0, 0, 1, ' ', 0)
	for name, val := range cfg.(*bindown.ConfigFile).TemplateSources {
		fmt.Fprintln(w, name+"\t"+val)
	}
	return w.Flush()
}

type templateSourceAddCmd struct {
	Name   string `kong:"arg"`
	Source string `kong:"arg"`
}

func (c *templateSourceAddCmd) Run() error {
	cfgIface, err := configLoader.Load(cli.Configfile, true)
	if err != nil {
		return err
	}
	cfg := cfgIface.(*bindown.ConfigFile)
	if cfg.TemplateSources == nil {
		cfg.TemplateSources = map[string]string{}
	}
	if _, ok := cfg.TemplateSources[c.Name]; ok {
		return fmt.Errorf("template source already exists")
	}
	cfg.TemplateSources[c.Name] = c.Source
	return cfg.Write(cli.JSONConfig)
}

type templateSourceRemoveCmd struct {
	Name string `kong:"arg,predictor=templateSource"`
}

func (c *templateSourceRemoveCmd) Run() error {
	cfgIface, err := configLoader.Load(cli.Configfile, true)
	if err != nil {
		return err
	}
	cfg := cfgIface.(*bindown.ConfigFile)
	if cfg.TemplateSources == nil {
		return fmt.Errorf("no template source named %q", c.Name)
	}
	if _, ok := cfg.TemplateSources[c.Name]; !ok {
		return fmt.Errorf("no template source named %q", c.Name)
	}
	delete(cfg.TemplateSources, c.Name)
	return cfg.Write(cli.JSONConfig)
}
