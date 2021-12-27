package main

import (
	"github.com/alecthomas/kong"
)

// Version version to display for `bindown version`
var Version = "unknown"

type versionCmd struct{}

func (*versionCmd) Run(k *kong.Context) error {
	k.Printf("version %s", Version)
	return nil
}
