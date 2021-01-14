package bindown

import (
	"bytes"
	"fmt"
)

// SystemInfo contains os and architecture for a target system
type SystemInfo struct {
	OS   string
	Arch string
}

func newSystemInfo(os, arch string) SystemInfo {
	return SystemInfo{
		OS:   os,
		Arch: arch,
	}
}

func (s *SystemInfo) String() string {
	return fmt.Sprintf("%s/%s", s.OS, s.Arch)
}

// UnmarshalText implements encoding.TextUnmarshaler
func (s *SystemInfo) UnmarshalText(text []byte) error {
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

// Equal tests equality
func (s SystemInfo) Equal(other SystemInfo) bool {
	return s.OS == other.OS && s.Arch == other.Arch
}
