package ifaces

import (
	"github.com/willabides/bindown/v3"
	"github.com/willabides/bindown/v3/internal/configfile"
)

var _ ConfigFile = new(configfile.ConfigFile)

//ConfigFile a config file
type ConfigFile interface {
	Write(outputJSON bool) error
	AddChecksums(dependencies []string, systems []bindown.SystemInfo) error
	Validate(dependencies []string, systems []bindown.SystemInfo) error
	InstallDependency(dependencyName string, sysInfo bindown.SystemInfo, opts *bindown.ConfigInstallDependencyOpts) (string, error)
	DownloadDependency(dependencyName string, sysInfo bindown.SystemInfo, opts *bindown.ConfigDownloadDependencyOpts) (string, error)
	ExtractDependency(dependencyName string, sysInfo bindown.SystemInfo, opts *bindown.ConfigExtractDependencyOpts) (string, error)
}

//ConfigLoader loads config files
type ConfigLoader interface {
	Load(filename string, noDefaultDirs bool) (ConfigFile, error)
}
