package bindownloader

import (
	"strings"
)

// Downloaders map Downloader name to Downloader
type Downloaders map[string][]*Downloader

// Downloader returns a Downloader for the given binary, os and arch.
func (d Downloaders) Downloader(binary, os, arch string) *Downloader {
	l, ok := d[binary]
	if !ok {
		return nil
	}
	for _, d := range l {
		if eqOS(os, d.OS) && eqArch(arch, d.Arch) {
			return d
		}
	}
	return nil
}

func eqOS(a, b string) bool {
	return strings.EqualFold(normalizeOS(a), normalizeOS(b))
}

func eqArch(a, b string) bool {
	return strings.EqualFold(normalizeArch(a), normalizeArch(b))
}

func normalizeArch(arch string) string {
	return strings.ToLower(arch)
}

func normalizeOS(os string) string {
	for _, v := range []string{
		"osx", "darwin", "macos",
	} {
		if strings.EqualFold(v, os) {
			return "darwin"
		}
	}
	return strings.ToLower(os)
}
