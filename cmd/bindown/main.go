package main

import (
	"os"

	"github.com/willabides/bindown/v2/internal/cli"
)

var version string

func main() {
	if version != "" {
		cli.Version = version
	}
	cli.Run(os.Args[1:])
}
