package bindown

import (
	"bytes"
	"fmt"
	"sort"
)

//SystemInfo contains os and architecture for a target system
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

//UnmarshalText implements encoding.TextUnmarshaler
func (s *SystemInfo) UnmarshalText(text []byte) error {
	parts := bytes.Split(text, []byte{'/'})
	if len(parts) != 2 {
		return fmt.Errorf(`systemInfo must be in the form "os/architecture"`)
	}
	s.OS = string(parts[0])
	s.Arch = string(parts[1])
	return nil
}

//MarshalText implements encoding.TextMarshaler
func (s SystemInfo) MarshalText() (text []byte, err error) {
	return []byte(s.String()), nil
}

//Equal tests equality
func (s SystemInfo) Equal(other SystemInfo) bool {
	return s.OS == other.OS && s.Arch == other.Arch
}

func SystemInfosSort(systems []SystemInfo) []SystemInfo {
	sort.Slice(systems, func(i, j int) bool {
		return systems[i].String() < systems[j].String()
	})
	return systems
}

func SystemInfosIntersection(a, b []SystemInfo) []SystemInfo {
	mp := map[SystemInfo]bool{}
	for _, system := range a {
		mp[system] = true
	}
	result := make([]SystemInfo, 0, len(b))
	for _, system := range b {
		if mp[system] {
			result = append(result, system)
		}
	}
	return SystemInfosUnique(result)
}

func SystemInfosUnique(systems []SystemInfo) []SystemInfo {
	mp := map[SystemInfo]bool{}
	for _, system := range systems {
		mp[system] = true
	}
	result := make([]SystemInfo, 0, len(mp))
	for system := range mp {
		result = append(result, system)
	}
	return SystemInfosSort(result)
}
