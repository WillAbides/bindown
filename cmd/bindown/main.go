package main

import (
	"context"
	_ "embed"
	"os"
)

var version string

//go:generate sh -c "go tool dist list > go_dist_list.txt"

//go:embed go_dist_list.txt
var goDists string

func main() {
	ctx := context.Background()
	if version != "" {
		Version = version
	}
	Run(ctx, os.Args[1:], nil)
}
