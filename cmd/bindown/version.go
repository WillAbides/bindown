package main

import (
	"github.com/alecthomas/kong"
)

var version = "unknown"

type versionCmd struct{}

func (*versionCmd) Run(k *kong.Context) error {
	k.Printf("version %s", version)
	return nil
}
