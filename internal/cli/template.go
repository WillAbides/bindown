package cli

import (
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v3"
)

type templateCmd struct {
	List   templateListCmd   `kong:"cmd,help='list templates'"`
	Remove templateRemoveCmd `kong:"cmd,help='remove a template'"`
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
