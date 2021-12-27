package main

import (
	"os"
)

var version string

func main() {
	if version != "" {
		Version = version
	}
	Run(os.Args[1:])
}
