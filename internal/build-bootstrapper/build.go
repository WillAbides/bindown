package main

import (
	"bytes"
	"embed"
	"errors"
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
	bindownPath := "bin/bootstrapped/bindown"
	makeCmd := exec.Command("make", bindownPath)
	makeCmd.Dir = repoRoot
	err := makeCmd.Run()
	if err != nil {
		var execErr *exec.ExitError
		if errors.As(err, &execErr) {
			err = fmt.Errorf("stderr: %s\nerr: %w", string(execErr.Stderr), err)
		}
		return fmt.Errorf("failed to build bindown: %w", err)
	}
	bindownCmd := exec.Command(bindownPath, arg...)
	bindownCmd.Dir = repoRoot
	err = bindownCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run bindown: %w", err)
	}
	return nil
}

func build(tag, repoRoot string) (_ string, errOut error) {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() {
		rmErr := os.RemoveAll(tmpDir)
		if errOut == nil && rmErr != nil {
			errOut = fmt.Errorf("failed to remove temp dir: %w", rmErr)
		}
	}()
	err = execBindown(repoRoot, "install", "shfmt")
	if err != nil {
		return "", fmt.Errorf("failed to install shfmt: %w", err)
	}
	err = execBindown(repoRoot, "install", "shellcheck")
	if err != nil {
		return "", fmt.Errorf("failed to install shellcheck: %w", err)
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
		return "", fmt.Errorf("failed to add dependency bindown-checksums: %w", err)
	}
	defer func() {
		removeErr := execBindown(repoRoot, "dependency", "remove", "bindown-checksums")
		if errOut != nil && removeErr != nil {
			errOut = fmt.Errorf("failed to remove dependency: %w", removeErr)
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
		return "", fmt.Errorf("failed to extract bindown-checksums: %w", err)
	}
	checksums, err := os.ReadFile(filepath.Join(checksumsDir, "checksums.txt"))
	if err != nil {
		return "", fmt.Errorf("failed to read checksums: %w", err)
	}
	shlibContent, err := assets.ReadFile("assets/shlib.sh")
	if err != nil {
		return "", fmt.Errorf("failed to read shlib: %w", err)
	}
	mainContent, err := assets.ReadFile("assets/main.sh")
	if err != nil {
		return "", fmt.Errorf("failed to read main: %w", err)
	}
	tmplContent, err := assets.ReadFile("assets/bootstrap-bindown.gotmpl")
	if err != nil {
		return "", fmt.Errorf("failed to read template: %w", err)
	}
	tmpl, err := template.New("").Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	var tmplOut bytes.Buffer
	err = tmpl.Execute(&tmplOut, map[string]string{
		"tag":       tag,
		"checksums": strings.TrimSpace(string(checksums)),
		"shlib":     string(shlibContent),
		"main":      string(mainContent),
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	shfmtCmd := exec.Command(filepath.Join(repoRoot, "bin", "shfmt"), "-i", "2", "-ci", "-sr", "-")
	shfmtCmd.Stdin = &tmplOut
	formatted, err := shfmtCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to format: %w", err)
	}
	shellcheckCmd := exec.Command(filepath.Join(repoRoot, "bin", "shellcheck"), "--shell", "sh", "-")
	shellcheckCmd.Stdin = bytes.NewReader(formatted)
	err = shellcheckCmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to shellcheck: %w", err)
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
