package main

import (
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v3"
)

type templateCmd struct {
	List             templateListCmd             `kong:"cmd,help='list templates'"`
	Remove           templateRemoveCmd           `kong:"cmd,help='remove a template'"`
	UpdateFromSource templateUpdateFromSourceCmd `kong:"cmd,help='update a template from source'"`
	UpdateVars       templateUpdateVarCmd        `kong:"cmd,help='update template vars'"`
}

type templateUpdateVarCmd struct {
	Dependency string            `kong:"arg,completer=bin"`
	Set        map[string]string `kong:"help='add or update a var'"`
	Unset      []string          `kong:"help='remove a var'"`
}

func (c *templateUpdateVarCmd) Run() error {
	config, err := configLoader.Load(cli.Configfile, true)
	if err != nil {
		return err
	}
	if len(c.Set) > 0 {
		err = config.SetTemplateVars(c.Dependency, c.Set)
		if err != nil {
			return err
		}
	}
	if len(c.Unset) > 0 {
		err = config.UnsetTemplateVars(c.Dependency, c.Unset)
		if err != nil {
			return err
		}
	}
	return config.Write(cli.JSONConfig)
}

type templateUpdateFromSourceCmd struct {
	Source   string `kong:"help='source of the update'"`
	Template string `kong:"arg,help='template to update'"`
}

func (c *templateUpdateFromSourceCmd) Run() error {
	if c.Source == "" {
		c.Source = c.Template
	}

	srcParts := strings.SplitN(c.Source, "#", 2)
	if len(srcParts) != 2 {
		return fmt.Errorf("source must be formated as name#source (with the #)")
	}
	src := srcParts[0]
	srcTmpl := srcParts[1]

	cfgIface, err := configLoader.Load(cli.Configfile, true)
	if err != nil {
		return err
	}
	cfg := cfgIface.(*bindown.ConfigFile)
	if cfg.Templates == nil {
		cfg.Templates = map[string]*bindown.Dependency{}
	}
	err = cfg.CopyTemplateFromSource(src, srcTmpl, c.Template)
	if err != nil {
		return err
	}
	return cfg.Write(cli.JSONConfig)
}

type templateListCmd struct {
	Source string `kong:"help='source of templates to list'"`
}

func (c *templateListCmd) Run(ctx *kong.Context) error {
	cfgIface, err := configLoader.Load(cli.Configfile, true)
	if err != nil {
		return err
	}
	cfg := cfgIface.(*bindown.ConfigFile)
	templates, err := cfg.ListTemplates(c.Source)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, strings.Join(templates, "\n"))
	return nil
}

type templateRemoveCmd struct {
	Name string `kong:"arg"`
}

func (c *templateRemoveCmd) Run() error {
	cfgIface, err := configLoader.Load(cli.Configfile, true)
	if err != nil {
		return err
	}
	cfg := cfgIface.(*bindown.ConfigFile)
	if cfg.Templates == nil {
		return fmt.Errorf("no template named %q", c.Name)
	}
	if _, ok := cfg.Templates[c.Name]; !ok {
		return fmt.Errorf("no template named %q", c.Name)
	}
	delete(cfg.Templates, c.Name)
	return cfg.Write(cli.JSONConfig)
}
