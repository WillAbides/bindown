package internal

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

func errOut(msg string) {
	_, err := fmt.Fprintln(os.Stderr, msg)
	if err != nil {
		panic(err)
	}
}

func exitErr(msg string) {
	errOut(msg)
	os.Exit(1)
}

//Main is a function that can be called from a main package
func Main() {
	configPath := flag.String("config", "buildtools.json", "json file with tool definitions")
	force := flag.Bool("force", false, "force download even if it already exists")
	opSys := flag.String("os", runtime.GOOS, "download for this operating system")
	arch := flag.String("arch", runtime.GOARCH, "download for this architecture")
	flag.Parse()
	downloaders, err := fromFile(*configPath)
	if err != nil {
		exitErr(fmt.Sprintf("error loading config: %v", err))
	}
	wantPath := flag.Arg(0)
	if wantPath == "" {
		exitErr("target file path is required")
	}
	targetDir := path.Dir(wantPath)
	targetFile := path.Base(wantPath)
	err = downloaders.installTool(targetFile, targetDir, *force, *opSys, *arch)
	if err != nil {
		errOut(fmt.Sprintf("error: %v", err))
	}
}

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

type downloaders map[string][]*downloader

func (d downloaders) installTool(toolName, binDir string, force bool, opSys, arch string) error {
	dl, ok := d.forInstall(toolName, opSys, arch)
	if !ok {
		return fmt.Errorf("no config for %s - %s - %s", toolName, opSys, arch)
	}
	return dl.install(binDir, force)
}

func (d downloaders) forInstall(toolName, os, arch string) (*downloader, bool) {
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

//fromFile builds a new downloaders from a json file
func fromFile(filepath string) (downloaders, error) {
	var dls downloaders
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
