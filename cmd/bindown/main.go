package main

import (
	"os"

	"github.com/willabides/bindown/v3/internal/cli"
)

func main() {
	cli.Run(os.Args[1:])
}
