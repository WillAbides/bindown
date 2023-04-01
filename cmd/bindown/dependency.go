package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/willabides/bindown/v3"
	"github.com/willabides/bindown/v3/cmd/bindown/ifaces"
	"gopkg.in/yaml.v3"
)

type dependencyCmd struct {
	List       dependencyListCmd       `kong:"cmd,help='list configured dependencies'"`
	Add        dependencyAddCmd        `kong:"cmd,help='add a template-based dependency'"`
	Remove     dependencyRemoveCmd     `kong:"cmd,help='remove a dependency'"`
	Info       dependencyInfoCmd       `kong:"cmd,help='info about a dependency'"`
	ShowConfig dependencyShowConfigCmd `kong:"cmd,help='show dependency config'"`
	UpdateVars dependencyUpdateVarCmd  `kong:"cmd,help='update dependency vars'"`
	Validate   dependencyValidateCmd   `kong:"cmd,help='validate that installs work'"`
}

type dependencyUpdateVarCmd struct {
	Dependency    string            `kong:"arg,predictor=bin"`
	Set           map[string]string `kong:"help='add or update a var'"`
	Unset         []string          `kong:"help='remove a var'"`
	SkipChecksums bool              `kong:"name=skipchecksums,help='do not update checksums for this dependency'"`
}

func (c *dependencyUpdateVarCmd) Run(ctx context.Context) error {
	config, err := configLoader.Load(ctx, cli.Configfile, true)
	if err != nil {
		return err
	}
	if len(c.Set) > 0 {
		err = config.SetDependencyVars(c.Dependency, c.Set)
		if err != nil {
			return err
		}
	}
	if len(c.Unset) > 0 {
		err = config.UnsetDependencyVars(c.Dependency, c.Unset)
		if err != nil {
			return err
		}
	}
	missingVars, err := config.MissingDependencyVars(c.Dependency)
	if err != nil {
		return err
	}
	if len(missingVars) == 0 && !c.SkipChecksums {
		err = config.AddChecksums([]string{c.Dependency}, nil)
		if err != nil {
			return err
		}
	}
	return config.Write(cli.JSONConfig)
}

type dependencyShowConfigCmd struct {
	Dependency string `kong:"arg,predictor=bin"`
}

func (c *dependencyShowConfigCmd) Run(ctx context.Context, kctx *kong.Context) error {
	cfgIface, err := configLoader.Load(ctx, cli.Configfile, true)
	if err != nil {
		return err
	}
	cfg := cfgIface.(*bindown.ConfigFile)
	if cfg.Dependencies == nil || cfg.Dependencies[c.Dependency] == nil {
		return fmt.Errorf("no dependency named %q", c.Dependency)
	}
	if cli.JSONConfig {
		encoder := json.NewEncoder(kctx.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(cfg.Dependencies[c.Dependency])
	}
	return yaml.NewEncoder(kctx.Stdout).Encode(cfg.Dependencies[c.Dependency])
}

type dependencyInfoCmd struct {
	Dependency string               `kong:"arg,predictor=bin"`
	Systems    []bindown.SystemInfo `kong:"name=system,help=${systems_help},predictor=allSystems"`
	Vars       bool                 `kong:"help='include vars'"`
}

func (c *dependencyInfoCmd) Run(ctx context.Context, kctx *kong.Context) error {
	cfgIface, err := configLoader.Load(ctx, cli.Configfile, true)
	if err != nil {
		return err
	}
	cfg := cfgIface.(*bindown.ConfigFile)
	systems := c.Systems
	if len(systems) == 0 {
		systems, err = cfg.DependencySystems(c.Dependency)
		if err != nil {
			return err
		}
	}
	result := map[string]*bindown.Dependency{}
	for _, system := range systems {
		dep, err := cfg.BuildDependency(c.Dependency, system)
		if err != nil {
			return err
		}
		if dep.BinName == nil {
			dep.BinName = &c.Dependency
		}
		dep.Systems = nil
		if !c.Vars {
			dep.Vars = nil
			dep.RequiredVars = nil
		}
		result[system.String()] = dep
	}

	if cli.JSONConfig {
		encoder := json.NewEncoder(kctx.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	return yaml.NewEncoder(kctx.Stdout).Encode(result)
}

type dependencyListCmd struct{}

func (c *dependencyListCmd) Run(ctx context.Context, kctx *kong.Context) error {
	cfg, err := configLoader.Load(ctx, cli.Configfile, true)
	if err != nil {
		return err
	}
	fmt.Fprintln(kctx.Stdout, strings.Join(allDependencies(cfg.(*bindown.ConfigFile)), "\n"))
	return nil
}

type dependencyRemoveCmd struct {
	Dependency string `kong:"arg,predictor=bin"`
}

func (c *dependencyRemoveCmd) Run(ctx context.Context) error {
	cfgIface, err := configLoader.Load(ctx, cli.Configfile, true)
	if err != nil {
		return err
	}
	cfg := cfgIface.(*bindown.ConfigFile)
	if cfg.Dependencies == nil {
		return fmt.Errorf("no dependency named %q", c.Dependency)
	}
	if _, ok := cfg.Dependencies[c.Dependency]; !ok {
		return fmt.Errorf("no dependency named %q", c.Dependency)
	}
	delete(cfg.Dependencies, c.Dependency)
	return cfg.Write(cli.JSONConfig)
}

type dependencyAddCmd struct {
	Name             string            `kong:"arg"`
	Template         string            `kong:"arg,predictor=template"`
	TemplateSource   string            `kong:"name=source,help='template source',predictor=templateSource"`
	Vars             map[string]string `kong:"name=var"`
	SkipRequiredVars bool              `kong:"name=skipvars,help='do not prompt for required vars'"`
	SkipChecksums    bool              `kong:"name=skipchecksums,help='do not add checksums for this dependency'"`
}

func (c *dependencyAddCmd) Run(ctx context.Context, kctx *kong.Context) error {
	config, err := configLoader.Load(ctx, cli.Configfile, true)
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
	err = config.AddDependencyFromTemplate(ctx, tmpl, &bindown.AddDependencyFromTemplateOpts{
		DependencyName: c.Name,
		TemplateSource: tmplSrc,
		Vars:           c.Vars,
	})
	if err != nil {
		return err
	}
	if !c.SkipRequiredVars {
		err = c.promptRequiredVars(kctx, config)
		if err != nil {
			return err
		}
	}
	missingVars, err := config.MissingDependencyVars(c.Name)
	if err != nil {
		return err
	}
	if len(missingVars) == 0 && !c.SkipChecksums {
		err = config.AddChecksums([]string{c.Name}, nil)
		if err != nil {
			return err
		}
	}
	return config.Write(cli.JSONConfig)
}

func (c *dependencyAddCmd) promptRequiredVars(kctx *kong.Context, config ifaces.ConfigFile) error {
	missingVars, err := config.MissingDependencyVars(c.Name)
	if err != nil {
		return err
	}
	dep := config.(*bindown.ConfigFile).Dependencies[c.Name]
	for _, missingVar := range missingVars {
		dep.Vars, err = requestRequiredVar(kctx, missingVar, dep.Vars)
		if err != nil {
			return err
		}
	}
	return nil
}

type dependencyValidateCmd struct {
	Dependency string               `kong:"arg,predictor=bin"`
	Systems    []bindown.SystemInfo `kong:"name=system,predictor=allSystems"`
}

func (d dependencyValidateCmd) Run(ctx context.Context) error {
	config, err := configLoader.Load(ctx, cli.Configfile, false)
	if err != nil {
		return err
	}
	return config.Validate([]string{d.Dependency}, d.Systems)
}
