package bindown

import (
	"fmt"
	"runtime"
	"strings"
)

// System is a string that represents a target system in the form of "os/architecture"
type System string

// CurrentSystem is the system that bindown is running on
var CurrentSystem = System(runtime.GOOS + "/" + runtime.GOARCH)

func (s System) validate() {
	if len(strings.Split(string(s), "/")) != 2 {
		panic(fmt.Sprintf(`invalid system %q`, s))
	}
}

func (s System) OS() string {
	s.validate()
	return strings.Split(string(s), "/")[0]
}

func (s System) Arch() string {
	s.validate()
	return strings.Split(string(s), "/")[1]
}
