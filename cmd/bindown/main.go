package main

import (
	"context"
	_ "embed"
	"os"
)

var version string

func main() {
	ctx := context.Background()
	if version != "" {
		Version = version
	}
	Run(ctx, os.Args[1:], nil)
}
