package ifaces

import (
	"github.com/willabides/bindown/v3"
)

var _ ConfigFile = new(bindown.ConfigFile)

//ConfigFile a config file
type ConfigFile interface {
	Write(outputJSON bool) error
	AddChecksums(dependencies []string, systems []bindown.SystemInfo) error
	Validate(dependencies []string, systems []bindown.SystemInfo) error
	InstallDependency(dependencyName string, sysInfo bindown.SystemInfo, opts *bindown.ConfigInstallDependencyOpts) (string, error)
	DownloadDependency(dependencyName string, sysInfo bindown.SystemInfo, opts *bindown.ConfigDownloadDependencyOpts) (string, error)
	ExtractDependency(dependencyName string, sysInfo bindown.SystemInfo, opts *bindown.ConfigExtractDependencyOpts) (string, error)
	AddDependencyFromTemplate(templateName string, opts *bindown.AddDependencyFromTemplateOpts) error
	MissingDependencyVars(depName string) ([]string, error)
}

//ConfigLoader loads config files
type ConfigLoader interface {
	Load(filename string, noDefaultDirs bool) (ConfigFile, error)
}
