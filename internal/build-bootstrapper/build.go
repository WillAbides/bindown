package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed assets/*
var assets embed.FS

func build(tag, repoRoot string) (_ string, errOut error) {
	checksumsURL := fmt.Sprintf(
		`https://github.com/WillAbides/bindown/releases/download/%s/checksums.txt`,
		tag,
	)
	resp, err := http.Get(checksumsURL)
	if err != nil {
		return "", err
	}
	defer func() {
		err = resp.Body.Close()
		if errOut == nil {
			errOut = err
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("got status %d from %s", resp.StatusCode, checksumsURL)
	}
	checksums, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	shlibContent, err := assets.ReadFile("assets/shlib.sh")
	if err != nil {
		return "", err
	}
	mainContent, err := assets.ReadFile("assets/main.sh")
	if err != nil {
		return "", err
	}
	tmplContent, err := assets.ReadFile("assets/bootstrap-bindown.gotmpl")
	if err != nil {
		return "", err
	}
	tmpl, err := template.New("").Parse(string(tmplContent))
	if err != nil {
		return "", err
	}
	var tmplOut bytes.Buffer
	err = tmpl.Execute(&tmplOut, map[string]string{
		"tag":       tag,
		"checksums": strings.TrimSpace(string(checksums)),
		"shlib":     string(shlibContent),
		"main":      string(mainContent),
	})
	if err != nil {
		return "", err
	}
	shfmtCmd := exec.Command(filepath.Join(repoRoot, "bin", "shfmt"), "-i", "2", "-ci", "-sr", "-")
	shfmtCmd.Stdin = &tmplOut
	formatted, err := shfmtCmd.Output()
	if err != nil {
		return "", err
	}
	shellcheckCmd := exec.Command(filepath.Join(repoRoot, "bin", "shellcheck"), "--shell", "sh", "-")
	shellcheckCmd.Stdin = bytes.NewReader(formatted)
	err = shellcheckCmd.Run()
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}

func main() {
	var tag, repoRoot string
	flag.StringVar(&tag, "tag", "", "tag to build")
	flag.StringVar(&repoRoot, "repo-root", ".", "path to bindown repo root")
	flag.Parse()
	if tag == "" {
		panic("tag is required")
	}
	got, err := build(tag, repoRoot)
	if err != nil {
		panic(err)
	}
	fmt.Println(got)
}
