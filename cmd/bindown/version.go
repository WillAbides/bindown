package main

import (
	"fmt"
)

// Version the version to display for `bindown version`
var Version = "unknown"

type versionCmd struct{}

func (*versionCmd) Run(ctx *runContext) error {
	fmt.Fprintf(ctx.stdout, "bindown: version %s\n", Version)
	return nil
}
