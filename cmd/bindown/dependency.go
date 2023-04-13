package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/willabides/bindown/v3/internal/bindown"
	"gopkg.in/yaml.v2"
)

type dependencyCmd struct {
	List       dependencyListCmd       `kong:"cmd,help='list configured dependencies'"`
	Add        dependencyAddCmd        `kong:"cmd,help='add a template-based dependency'"`
	Remove     dependencyRemoveCmd     `kong:"cmd,help='remove a dependency'"`
	Info       dependencyInfoCmd       `kong:"cmd,help='info about a dependency'"`
	ShowConfig dependencyShowConfigCmd `kong:"cmd,help='show dependency config'"`
	UpdateVars dependencyUpdateVarsCmd `kong:"cmd,help='update dependency vars'"`
	Validate   dependencyValidateCmd   `kong:"cmd,help='validate that installs work'"`
}

type dependencyUpdateVarsCmd struct {
	Dependency    string            `kong:"arg,predictor=bin"`
	Set           map[string]string `kong:"help='add or update a var'"`
	Unset         []string          `kong:"help='remove a var'"`
	SkipChecksums bool              `kong:"name=skipchecksums,help='do not update checksums for this dependency'"`
}

func (c *dependencyUpdateVarsCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, true)
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
	return config.WriteFile(ctx.rootCmd.JSONConfig)
}

type dependencyShowConfigCmd struct {
	Dependency string `kong:"arg,predictor=bin"`
}

func (c *dependencyShowConfigCmd) Run(ctx *runContext) error {
	cfg, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}
	if cfg.Dependencies == nil || cfg.Dependencies[c.Dependency] == nil {
		return fmt.Errorf("no dependency named %q", c.Dependency)
	}
	if ctx.rootCmd.JSONConfig {
		encoder := json.NewEncoder(ctx.stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(cfg.Dependencies[c.Dependency])
	}
	return yaml.NewEncoder(ctx.stdout).Encode(cfg.Dependencies[c.Dependency])
}

type dependencyInfoCmd struct {
	Dependency string           `kong:"arg,predictor=bin"`
	Systems    []bindown.System `kong:"name=system,help=${systems_help},predictor=allSystems"`
	Vars       bool             `kong:"help='include vars'"`
}

func (c *dependencyInfoCmd) Run(ctx *runContext) error {
	cfg, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}
	var systems []bindown.System
	systems = append(systems, c.Systems...)
	if len(systems) == 0 {
		systems, err = cfg.DependencySystems(c.Dependency)
		if err != nil {
			return err
		}
	}
	result := map[bindown.System]*bindown.Dependency{}
	for _, system := range systems {
		var dep *bindown.Dependency
		dep, err = cfg.BuildDependency(c.Dependency, system)
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
		result[system] = dep
	}

	if ctx.rootCmd.JSONConfig {
		encoder := json.NewEncoder(ctx.stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	return yaml.NewEncoder(ctx.stdout).Encode(result)
}

type dependencyListCmd struct{}

func (c *dependencyListCmd) Run(ctx *runContext) error {
	cfg, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.stdout, strings.Join(allDependencies(cfg), "\n"))
	return nil
}

type dependencyRemoveCmd struct {
	Dependency string `kong:"arg,predictor=bin"`
}

func (c *dependencyRemoveCmd) Run(ctx *runContext) error {
	cfg, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}
	if cfg.Dependencies == nil {
		return fmt.Errorf("no dependency named %q", c.Dependency)
	}
	if _, ok := cfg.Dependencies[c.Dependency]; !ok {
		return fmt.Errorf("no dependency named %q", c.Dependency)
	}
	delete(cfg.Dependencies, c.Dependency)
	return cfg.WriteFile(ctx.rootCmd.JSONConfig)
}

type dependencyAddCmd struct {
	Name             string            `kong:"arg"`
	Template         string            `kong:"arg,predictor=template"`
	TemplateSource   string            `kong:"name=source,help='template source',predictor=templateSource"`
	Vars             map[string]string `kong:"name=var"`
	SkipRequiredVars bool              `kong:"name=skipvars,help='do not prompt for required vars'"`
	SkipChecksums    bool              `kong:"name=skipchecksums,help='do not add checksums for this dependency'"`
}

func (c *dependencyAddCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, true)
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
	missingVars, err := config.MissingDependencyVars(c.Name)
	if err != nil {
		return err
	}
	hasMissingVars := len(missingVars) > 0
	if hasMissingVars && !c.SkipRequiredVars {
		hasMissingVars = false
		scanner := bufio.NewScanner(ctx.stdin)
		for _, missingVar := range missingVars {
			fmt.Fprintf(ctx.stdout, "Please enter a value for required variable %q:\t", missingVar)
			scanner.Scan()
			err = scanner.Err()
			if err != nil {
				return err
			}
			val := scanner.Text()
			config.Dependencies[c.Name].Vars[missingVar] = val
		}
	}
	if !hasMissingVars && !c.SkipChecksums {
		err = config.AddChecksums([]string{c.Name}, nil)
		if err != nil {
			return err
		}
	}
	return config.WriteFile(ctx.rootCmd.JSONConfig)
}

type dependencyValidateCmd struct {
	Dependency string           `kong:"arg,predictor=bin"`
	Systems    []bindown.System `kong:"name=system,predictor=allSystems"`
}

func (d dependencyValidateCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, false)
	if err != nil {
		return err
	}
	return config.Validate(d.Dependency, d.Systems)
}
