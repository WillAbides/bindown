package bindown

import (
	"bytes"
	"fmt"
	"runtime"
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
