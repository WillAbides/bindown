package bindown

import (
	"github.com/willabides/bindown/v3/internal/bindown"
	"golang.org/x/exp/maps"
)

func internalizeDependency(d *Dependency) *bindown.Dependency {
	if d == nil {
		return nil
	}
	dep := &bindown.Dependency{
		Template:      d.Template,
		URL:           d.URL,
		ArchivePath:   d.ArchivePath,
		BinName:       d.BinName,
		Link:          d.Link,
		Vars:          d.Vars,
		RequiredVars:  d.RequiredVars,
		Overrides:     internalizeOverrides(d.Overrides),
		Substitutions: d.Substitutions,
		Systems:       internalizeSystems(d.Systems),
	}
	return dep.Clone()
}

func externalizeDependency(iDep *bindown.Dependency) *Dependency {
	if iDep == nil {
		return nil
	}
	iDep = iDep.Clone()
	return &Dependency{
		Template:      iDep.Template,
		URL:           iDep.URL,
		ArchivePath:   iDep.ArchivePath,
		BinName:       iDep.BinName,
		Link:          iDep.Link,
		Vars:          iDep.Vars,
		RequiredVars:  iDep.RequiredVars,
		Overrides:     externalizeOverrides(iDep.Overrides),
		Substitutions: iDep.Substitutions,
		Systems:       externalizeSystems(iDep.Systems),
	}
}

func internalizeDependencyMap(external map[string]*Dependency) map[string]*bindown.Dependency {
	return bindown.TransformMap(external, internalizeDependency)
}

func externalizeDependencyMap(internal map[string]*bindown.Dependency) map[string]*Dependency {
	return bindown.TransformMap(internal, externalizeDependency)
}

func internalizeConfig(cfg *Config) *bindown.Config {
	return &bindown.Config{
		Cache:           cfg.Cache,
		TrustCache:      cfg.TrustCache,
		InstallDir:      cfg.InstallDir,
		Systems:         internalizeSystems(cfg.Systems),
		Dependencies:    internalizeDependencyMap(cfg.Dependencies),
		Templates:       internalizeDependencyMap(cfg.Templates),
		TemplateSources: maps.Clone(cfg.TemplateSources),
		URLChecksums:    maps.Clone(cfg.URLChecksums),
	}
}

func externalizeConfig(cfg *bindown.Config) *Config {
	return &Config{
		Cache:           cfg.Cache,
		TrustCache:      cfg.TrustCache,
		InstallDir:      cfg.InstallDir,
		Systems:         externalizeSystems(cfg.Systems),
		Dependencies:    externalizeDependencyMap(cfg.Dependencies),
		Templates:       externalizeDependencyMap(cfg.Templates),
		TemplateSources: maps.Clone(cfg.TemplateSources),
		URLChecksums:    maps.Clone(cfg.URLChecksums),
	}
}

func externalizeSystems(internal []bindown.SystemInfo) []SystemInfo {
	return bindown.TransformSlice(internal, func(iSys bindown.SystemInfo) SystemInfo {
		return SystemInfo(iSys)
	})
}

func internalizeSystems(external []SystemInfo) []bindown.SystemInfo {
	return bindown.TransformSlice(external, func(oSys SystemInfo) bindown.SystemInfo {
		return bindown.SystemInfo(oSys)
	})
}

func internalizeOverrides(overrides []DependencyOverride) []bindown.DependencyOverride {
	return bindown.TransformSlice(overrides, func(override DependencyOverride) bindown.DependencyOverride {
		return bindown.DependencyOverride{
			OverrideMatcher: bindown.OverrideMatcher(override.OverrideMatcher),
			Dependency:      *internalizeDependency(&override.Dependency),
		}
	})
}

func externalizeOverrides(iOverrides []bindown.DependencyOverride) []DependencyOverride {
	return bindown.TransformSlice(iOverrides, func(iOverride bindown.DependencyOverride) DependencyOverride {
		return DependencyOverride{
			OverrideMatcher: OverrideMatcher(iOverride.OverrideMatcher),
			Dependency:      *externalizeDependency(&iOverride.Dependency),
		}
	})
}
