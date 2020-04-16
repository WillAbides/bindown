package cli

import (
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v3"
)

type dependencyCmd struct {
	List dependencyListCmd `kong:"cmd,help='list configured dependencies'"`
	Add  dependencyAddCmd  `kong:"cmd,help='add a template-based dependency'"`
}

type dependencyListCmd struct{}

func (c *dependencyListCmd) Run(ctx *kong.Context) error {
	cfg, err := configLoader.Load(cli.Configfile, true)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout, strings.Join(allDependencies(cfg.(*bindown.ConfigFile)), "\n"))
	return nil
}

type dependencyAddCmd struct {
	Name             string            `kong:"arg"`
	Template         string            `kong:"arg"`
	TemplateSource   string            `kong:"name=source,help='template source'"`
	Vars             map[string]string `kong:"name=var"`
	SkipRequiredVars bool              `kong:"name=skipvars,help='do not prompt for required vars'"`
}

func (c *dependencyAddCmd) Run(ctx *kong.Context) error {
	config, err := configLoader.Load(cli.Configfile, true)
	if err != nil {
		return err
	}
	tmpl := c.Template
	tmplSrc := c.TemplateSource
	if tmplSrc == "" {
		tmplParts := strings.SplitN(tmpl, "#", 2)
		if len(tmplParts) == 2 {
			tmpl = tmplParts[1]
			tmplSrc = tmplParts[0]
		}
	}

	if c.Vars == nil {
		c.Vars = map[string]string{}
	}
	err = config.AddDependencyFromTemplate(tmpl, &bindown.AddDependencyFromTemplateOpts{
		DependencyName: c.Name,
		TemplateSource: tmplSrc,
		Vars:           c.Vars,
	})
	if err != nil {
		return err
	}
	if c.SkipRequiredVars {
		return config.Write(cli.JSONConfig)
	}
	missingVars, err := config.MissingDependencyVars(c.Name)
	if err != nil {
		return err
	}
	dep := config.(*bindown.ConfigFile).Dependencies[c.Name]
	for _, missingVar := range missingVars {
		dep.Vars, err = requestRequiredVar(ctx, missingVar, dep.Vars)
		if err != nil {
			return err
		}
	}
	return config.Write(cli.JSONConfig)
}
