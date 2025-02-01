package main

const defaultVersion = "unknown"

// Version the version to display for `bindown version`
var Version = defaultVersion

func getVersion() string {
	if Version == defaultVersion {
		return ""
	}
	return Version
}

type versionCmd struct{}

func (*versionCmd) Run(ctx *runContext) error {
	mustPrintf(ctx.stdout, "bindown: version %s\n", Version)
	return nil
}
