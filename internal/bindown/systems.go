package bindown

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
)

// SystemInfo contains os and architecture for a target system
type SystemInfo struct {
	OS   string
	Arch string
}

func (s *SystemInfo) String() string {
	return fmt.Sprintf("%s/%s", s.OS, s.Arch)
}

// UnmarshalText implements encoding.TextUnmarshaler
func (s *SystemInfo) UnmarshalText(text []byte) error {
	if string(text) == "current" {
		s.OS = runtime.GOOS
		s.Arch = runtime.GOARCH
		return nil
	}
	parts := bytes.Split(text, []byte{'/'})
	if len(parts) != 2 {
		return fmt.Errorf(`systemInfo must be in the form "os/architecture"`)
	}
	s.OS = string(parts[0])
	s.Arch = string(parts[1])
	return nil
}

// MarshalText implements encoding.TextMarshaler
func (s SystemInfo) MarshalText() (text []byte, err error) {
	return []byte(s.String()), nil
}

func (s SystemInfo) System() System {
	system := System(fmt.Sprintf("%s/%s", s.OS, s.Arch))
	system.validate()
	return system
}

// System is a string that represents a target system in the form of "os/architecture"
type System string

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

func (s System) String() string {
	return string(s)
}

func (s System) SystemInfo() SystemInfo {
	s.validate()
	return SystemInfo{
		OS:   s.OS(),
		Arch: s.Arch(),
	}
}

//nolint:unused // transitioning
func systemsToInfos(systems []System) []SystemInfo {
	if systems == nil {
		return nil
	}
	infos := make([]SystemInfo, len(systems))
	for i, system := range systems {
		infos[i] = system.SystemInfo()
	}
	return infos
}

func infosToSystems(infos []SystemInfo) []System {
	if infos == nil {
		return nil
	}
	systems := make([]System, len(infos))
	for i, info := range infos {
		systems[i] = info.System()
	}
	return systems
}
