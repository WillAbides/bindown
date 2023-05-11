package main

import (
	"fmt"
	"strings"

	"github.com/willabides/bindown/v4/internal/bindown"
)

type templateCmd struct {
	List             templateListCmd             `kong:"cmd,help='list templates'"`
	Remove           templateRemoveCmd           `kong:"cmd,help='remove a template'"`
	UpdateFromSource templateUpdateFromSourceCmd `kong:"cmd,help='update a template from source'"`
	UpdateVars       templateUpdateVarCmd        `kong:"cmd,help='update template vars'"`
}

type templateUpdateVarCmd struct {
	Template string            `kong:"arg,predictor=localTemplate"`
	Set      map[string]string `kong:"help='add or update a var'"`
	Unset    []string          `kong:"help='remove a var'"`
}

func (c *templateUpdateVarCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}
	if len(c.Set) > 0 {
		err = config.SetTemplateVars(c.Template, c.Set)
		if err != nil {
			return err
		}
	}
	if len(c.Unset) > 0 {
		err = config.UnsetTemplateVars(c.Template, c.Unset)
		if err != nil {
			return err
		}
	}
	return config.WriteFile(ctx.rootCmd.JSONConfig)
}

type templateUpdateFromSourceCmd struct {
	Source   string `kong:"help='source of the update in the form of source_name#template_name',predictor=templateSource"`
	Template string `kong:"arg,help='template to update',predictor=localTemplateFromSource"`
}

func (c *templateUpdateFromSourceCmd) Run(ctx *runContext) error {
	if c.Source == "" {
		c.Source = c.Template
	}

	srcParts := strings.SplitN(c.Source, "#", 2)
	if len(srcParts) != 2 {
		return fmt.Errorf("source must be formatted as source#name (with the #)")
	}
	src := srcParts[0]
	srcTmpl := srcParts[1]

	cfg, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}

	if cfg.Templates == nil {
		cfg.Templates = map[string]*bindown.Dependency{}
	}
	err = cfg.CopyTemplateFromSource(ctx, src, srcTmpl, c.Template)
	if err != nil {
		return err
	}
	return cfg.WriteFile(ctx.rootCmd.JSONConfig)
}

type templateListCmd struct {
	Source string `kong:"help='source of templates to list',predictor=templateSource"`
}

func (c *templateListCmd) Run(ctx *runContext) error {
	cfg, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}

	templates, err := cfg.ListTemplates(ctx, c.Source)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.stdout, strings.Join(templates, "\n"))
	return nil
}

type templateRemoveCmd struct {
	Template string `kong:"arg,predictor=localTemplate"`
}

func (c *templateRemoveCmd) Run(ctx *runContext) error {
	cfg, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}

	if cfg.Templates == nil {
		return fmt.Errorf("no template named %q", c.Template)
	}
	if _, ok := cfg.Templates[c.Template]; !ok {
		return fmt.Errorf("no template named %q", c.Template)
	}
	delete(cfg.Templates, c.Template)
	return cfg.WriteFile(ctx.rootCmd.JSONConfig)
}
