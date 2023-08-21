package main

import (
	"fmt"
	"slices"
	"text/tabwriter"

	"github.com/willabides/bindown/v4/internal/bindown"
)

type templateSourceCmd struct {
	List   templateSourceListCmd   `kong:"cmd,help='list configured template sources'"`
	Add    templateSourceAddCmd    `kong:"cmd,help='add a template source'"`
	Remove templateSourceRemoveCmd `kong:"cmd,help='remove a template source'"`
}

type templateSourceListCmd struct{}

func (c *templateSourceListCmd) Run(ctx *runContext) error {
	cfg, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(ctx.stdout, 0, 0, 1, ' ', 0)
	sourceNames := bindown.MapKeys(cfg.TemplateSources)
	slices.Sort(sourceNames)
	for _, name := range sourceNames {
		fmt.Fprintln(w, name+"\t"+cfg.TemplateSources[name])
	}
	return w.Flush()
}

type templateSourceAddCmd struct {
	Name   string `kong:"arg"`
	Source string `kong:"arg"`
}

func (c *templateSourceAddCmd) Run(ctx *runContext) error {
	cfg, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}

	if cfg.TemplateSources == nil {
		cfg.TemplateSources = map[string]string{}
	}
	if _, ok := cfg.TemplateSources[c.Name]; ok {
		return fmt.Errorf("template source already exists")
	}
	cfg.TemplateSources[c.Name] = c.Source
	return cfg.WriteFile(ctx.rootCmd.JSONConfig)
}

type templateSourceRemoveCmd struct {
	Name string `kong:"arg,predictor=templateSource"`
}

func (c *templateSourceRemoveCmd) Run(ctx *runContext) error {
	cfg, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}

	if cfg.TemplateSources == nil {
		return fmt.Errorf("no template source named %q", c.Name)
	}
	if _, ok := cfg.TemplateSources[c.Name]; !ok {
		return fmt.Errorf("no template source named %q", c.Name)
	}
	delete(cfg.TemplateSources, c.Name)
	return cfg.WriteFile(ctx.rootCmd.JSONConfig)
}
