package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed assets/*
var assets embed.FS

func execBindown(repoRoot string, arg ...string) error {
	bindownPath := filepath.FromSlash("bin/bootstrapped/bindown")
	//nolint:gosec // subprocess launch with variable
	makeCmd := exec.Command("make", bindownPath)
	makeCmd.Dir = repoRoot
	err := makeCmd.Run()
	if err != nil {
		return err
	}
	//nolint:gosec // subprocess launch with variable
	bindownCmd := exec.Command(bindownPath, arg...)
	bindownCmd.Dir = repoRoot
	return bindownCmd.Run()
}

func build(tag, repoRoot string) (_ string, errOut error) {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", err
	}
	defer func() {
		rmErr := os.RemoveAll(tmpDir)
		if errOut == nil {
			errOut = rmErr
		}
	}()
	err = execBindown(repoRoot, "install", "shfmt")
	if err != nil {
		return "", err
	}
	err = execBindown(repoRoot, "install", "shellcheck")
	if err != nil {
		return "", err
	}
	err = execBindown(
		repoRoot,
		"dependency",
		"add",
		"bindown-checksums",
		"bindown-checksums",
		"--var",
		fmt.Sprintf("tag=%s", tag),
		"--skipchecksums",
	)
	if err != nil {
		return "", err
	}
	defer func() {
		removeErr := execBindown(repoRoot, "dependency", "remove", "bindown-checksums")
		if errOut != nil {
			errOut = removeErr
		}
	}()
	checksumsDir := filepath.Join(tmpDir, "checksums")
	err = execBindown(
		repoRoot,
		"extract",
		"bindown-checksums",
		"--allow-missing-checksum",
		"--output",
		checksumsDir,
	)
	if err != nil {
		return "", err
	}
	checksums, err := os.ReadFile(filepath.Join(checksumsDir, "checksums.txt"))
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
	//nolint:gosec // subprocess launch with variable
	shfmtCmd := exec.Command(filepath.Join(repoRoot, "bin", "shfmt"), "-i", "2", "-ci", "-sr", "-")
	shfmtCmd.Stdin = &tmplOut
	formatted, err := shfmtCmd.Output()
	if err != nil {
		return "", err
	}
	//nolint:gosec // subprocess launch with variable
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
