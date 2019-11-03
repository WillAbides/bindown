package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func downloadFile(targetPath, url string) error {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return err
	}
	defer logCloseErr(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed downloading %s", url)
	}
	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer logCloseErr(out)
	_, err = io.Copy(out, resp.Body)
	return err
}

//Downloaders map downloader name to downloader
type Downloaders map[string][]*downloader

//InstallToolConfig config for InstallTool
type InstallToolConfig struct {
	ToolName string
	BinDir   string
	OpSys    string
	Arch     string
	Force    bool
}

//InstallTool installs a tool
func (d Downloaders) InstallTool(config InstallToolConfig) error {
	dl, ok := d.forInstall(config.ToolName, config.OpSys, config.Arch)
	if !ok {
		return fmt.Errorf("no config for %s - %s - %s", config.ToolName, config.OpSys, config.Arch)
	}
	return dl.install(config.BinDir, config.Force)
}

func (d Downloaders) forInstall(toolName, os, arch string) (*downloader, bool) {
	l, ok := d[toolName]
	if !ok {
		return nil, false
	}
	for _, d := range l {
		if eqOS(os, d.OS) && eqArch(arch, d.Arch) {
			return d, true
		}
	}
	return nil, false
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

//FromFile builds a new downloaders from a json file
func FromFile(filepath string) (Downloaders, error) {
	var dls Downloaders
	jsonFile, err := os.Open(filepath) //nolint:gosec
	if err != nil {
		return dls, err
	}
	defer func() {
		cerr := jsonFile.Close()
		if cerr != nil {
			log.Println("error closing a file: ", err)
		}
	}()
	err = json.NewDecoder(jsonFile).Decode(&dls)
	return dls, err
}

func logCloseErr(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Println(err)
	}
}

//fileExists asserts that a file exists
func fileExists(path string) bool {
	if _, err := os.Stat(filepath.FromSlash(path)); !os.IsNotExist(err) {
		return true
	}
	return false
}

func rm(path string) error {
	err := os.RemoveAll(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return fmt.Errorf(`failed to remove %s: %v`, path, err)
}
