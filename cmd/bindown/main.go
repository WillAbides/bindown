package main

import (
	"os"
)

var version string

func main() {
	Run(os.Args[1:])
}
