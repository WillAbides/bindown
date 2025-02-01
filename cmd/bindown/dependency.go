package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/willabides/bindown/v4/internal/bindown"
	"github.com/willabides/bindown/v4/internal/builddep"
)

type dependencyCmd struct {
	List               dependencyListCmd               `kong:"cmd,help='list configured dependencies'"`
	Add                dependencyAddCmd                `kong:"cmd,help='add a template-based dependency'"`
	AddByUrls          dependencyAddByUrlsCmd          `kong:"cmd,help='add a dependency by urls'"`
	AddByGithubRelease dependencyAddByGithubReleaseCmd `kong:"cmd,help='add a dependency by github release'"`
	Remove             dependencyRemoveCmd             `kong:"cmd,help='remove a dependency'"`
	Info               dependencyInfoCmd               `kong:"cmd,help='info about a dependency'"`
	ShowConfig         dependencyShowConfigCmd         `kong:"cmd,help='show dependency config'"`
	UpdateVars         dependencyUpdateVarsCmd         `kong:"cmd,help='update dependency vars'"`
	Validate           dependencyValidateCmd           `kong:"cmd,help='validate that installs work'"`
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
	return bindown.EncodeYaml(ctx.stdout, cfg.Dependencies[c.Dependency])
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
	return bindown.EncodeYaml(ctx.stdout, result)
}

type dependencyListCmd struct{}

func (c *dependencyListCmd) Run(ctx *runContext) error {
	cfg, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}
	mustPrintln(ctx.stdout, strings.Join(cfg.DependencyNames(), "\n"))
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
	Template         string            `kong:"arg,optional,predictor=template"`
	TemplateSource   string            `kong:"name=source,help='template source',predictor=templateSource"`
	Vars             map[string]string `kong:"name=var"`
	AcceptDefaults   bool              `kong:"short=y,help='accept default values for vars'"`
	SkipRequiredVars bool              `kong:"name=skipvars,help='do not prompt for required vars. implies --skipchecksums'"`
	SkipChecksums    bool              `kong:"name=skipchecksums,help='do not add checksums for this dependency'"`
}

func (c *dependencyAddCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}
	tmpl := c.Template
	if tmpl == "" {
		tmpl = c.Name
	}
	tmplSrc := c.TemplateSource
	if tmplSrc == "" {
		ts, t, ok := strings.Cut(tmpl, "#")
		if ok {
			tmplSrc, tmpl = ts, t
		}
	}

	if c.Vars == nil {
		c.Vars = map[string]string{}
	}
	dep, varVals, err := config.AddDependencyFromTemplate(ctx, tmpl, &bindown.AddDependencyFromTemplateOpts{
		DependencyName: c.Name,
		TemplateSource: tmplSrc,
		Vars:           c.Vars,
	})
	if err != nil {
		return err
	}
	// This shouldn't be possible, but just in case
	if dep.Template == nil || config.Templates == nil || config.Templates[*dep.Template] == nil {
		return fmt.Errorf("template not found: %q", tmpl)
	}
	if varVals == nil {
		varVals = map[string][]string{}
	}
	tmplCfg := config.Templates[*dep.Template]
	// Don't need to output the list of systems
	systems := tmplCfg.Systems
	tmplCfg.Systems = nil
	mustPrintf(ctx.stdout, "Adding dependency %q from template ", c.Name)
	err = bindown.EncodeYaml(ctx.stdout, map[string]bindown.Dependency{
		*dep.Template: *tmplCfg,
	})
	if err != nil {
		return err
	}
	tmplCfg.Systems = systems

	err = c.promptForVars(ctx, config, dep, varVals)
	if err != nil {
		return err
	}

	skipChecksums := c.SkipChecksums || c.SkipRequiredVars
	if !skipChecksums {
		err = config.AddChecksums([]string{c.Name}, nil)
		if err != nil {
			return err
		}
	}
	return config.WriteFile(ctx.rootCmd.JSONConfig)
}

func (c *dependencyAddCmd) promptForVars(
	ctx *runContext,
	config *bindown.Config,
	dep *bindown.Dependency,
	varVals map[string][]string,
) error {
	if c.SkipRequiredVars {
		return nil
	}
	missingVars, err := config.MissingDependencyVars(c.Name)
	if err != nil {
		return err
	}
	if len(missingVars) == 0 {
		return nil
	}
	for _, d := range config.Dependencies {
		if d.Template == nil || dep.Template == nil || *d.Template != *dep.Template {
			continue
		}
		for k, v := range d.Vars {
			varVals[k] = append([]string{v}, varVals[k]...)
		}
	}

	questions := make([]*survey.Question, 0, len(missingVars))
	for _, missingVar := range missingVars {
		prompt := survey.Input{
			Message: missingVar,
		}
		knownVals := varVals[missingVar]
		bindown.SortBySemverOrString(knownVals)
		knownVals = bindown.Unique(knownVals, knownVals[:0])
		if len(knownVals) > 0 {
			defaultVal := knownVals[0]
			if c.AcceptDefaults {
				config.Dependencies[c.Name].Vars[missingVar] = defaultVal
				continue
			}
			prompt.Default = defaultVal
			prompt.Suggest = func(toComplete string) []string {
				suggestions := make([]string, 0, len(knownVals))
				for _, v := range knownVals {
					if strings.HasPrefix(v, toComplete) {
						suggestions = append(suggestions, v)
					}
				}
				return suggestions
			}
		}
		questions = append(questions, &survey.Question{
			Name:   missingVar,
			Prompt: &prompt,
		})
	}
	answers := map[string]any{}
	if len(questions) > 0 {
		err = survey.Ask(questions, &answers, survey.WithStdio(ctx.stdin, ctx.stdout, nil), survey.WithShowCursor(true))
		if err != nil {
			return err
		}
	}
	for k, v := range answers {
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("expected string for %q, got %T", k, v)
		}
		config.Dependencies[c.Name].Vars[k] = s
	}

	mustPrintf(ctx.stdout, "Adding dependency %q from template\n", c.Name)

	return nil
}

type dependencyAddByUrlsCmd struct {
	Name         string   `kong:"arg,help='dependency name'"`
	Version      string   `kong:"arg,help='dependency version'"`
	Homepage     string   `kong:"name=homepage,help='dependency homepage'"`
	Description  string   `kong:"name=description,help='dependency description'"`
	URL          []string `kong:"arg,help='dependency URL'"`
	Force        bool     `kong:"name=force,help='overwrite existing dependency'"`
	Experimental bool     `kong:"required,name=experimental,help='enable experimental features',env='BINDOWN_EXPERIMENTAL'"`
}

func (c *dependencyAddByUrlsCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}
	if config.Dependencies != nil && config.Dependencies[c.Name] != nil && !c.Force {
		return fmt.Errorf("dependency %q already exists", c.Name)
	}
	err = builddep.AddDependency(ctx, config, c.Name, c.Version, c.Homepage, c.Description, c.URL)
	if err != nil {
		return err
	}
	return config.WriteFile(ctx.rootCmd.JSONConfig)
}

type dependencyAddByGithubReleaseCmd struct {
	Release      string `kong:"arg,help='github release URL or \"owner/repo(@tag)\"'"`
	Name         string `kong:"name to use instead of repo name"`
	Version      string `kong:"version to use instead of release tag"`
	Homepage     string `kong:"name=homepage,help='dependency homepage'"`
	Description  string `kong:"name=description,help='dependency description'"`
	Force        bool   `kong:"name=force,help='overwrite existing dependency'"`
	Experimental bool   `kong:"required,name=experimental,help='enable experimental features',env='BINDOWN_EXPERIMENTAL'"`
	GithubToken  string `kong:"hidden,env='GITHUB_TOKEN'"`
}

var (
	releaseShortExp = regexp.MustCompile(`^([^/]+)/([^/^@]+)@?(.+)?$`)
	releaseURLExp   = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/releases/tag/([^/]+)`)
)

func (c *dependencyAddByGithubReleaseCmd) Run(ctx *runContext) error {
	config, err := loadConfigFile(ctx, true)
	if err != nil {
		return err
	}
	var owner, repo, tag string
	switch {
	case releaseURLExp.MatchString(c.Release):
		m := releaseURLExp.FindStringSubmatch(c.Release)
		owner, repo, tag = m[1], m[2], m[3]
	case releaseShortExp.MatchString(c.Release):
		m := releaseShortExp.FindStringSubmatch(c.Release)
		owner, repo, tag = m[1], m[2], m[3]
	default:
		return fmt.Errorf(`invalid release URL or "owner/repo(@tag)"`)
	}
	urls, releaseVer, repoPage, repoDesc, err := builddep.QueryGitHubRelease(ctx, fmt.Sprintf("%s/%s", owner, repo), tag, c.GithubToken)
	if err != nil {
		return err
	}
	ver := c.Version
	if ver == "" {
		ver = releaseVer
	}
	name := c.Name
	if name == "" {
		name = repo
	}
	homepage := c.Homepage
	if homepage == "" {
		homepage = repoPage
	}
	description := c.Description
	if description == "" {
		description = repoDesc
	}
	if config.Dependencies != nil && config.Dependencies[name] != nil && !c.Force {
		return fmt.Errorf("dependency %q already exists", name)
	}
	err = builddep.AddDependency(ctx, config, name, ver, homepage, description, urls)
	if err != nil {
		return err
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
	return config.Validate(ctx, d.Dependency, d.Systems)
}
