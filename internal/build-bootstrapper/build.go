package bootstrapper

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"text/template"
)

//go:embed assets/*
var assets embed.FS

type BuildOpts struct {
	BaseURL string // defaults to https://github.com
	BinDir  string
	Wrap    bool
}

// Build builds a bootstrapper for the given tag
func Build(tag string, opts *BuildOpts) (_ string, errOut error) {
	if opts == nil {
		opts = &BuildOpts{}
	}
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = "https://github.com"
	}
	repoURL := fmt.Sprintf("%s/WillAbides/bindown", baseURL)
	checksumsURL := fmt.Sprintf(
		`%s/releases/download/%s/checksums.txt`,
		repoURL, tag,
	)
	resp, err := http.Get(checksumsURL)
	if err != nil {
		return "", err
	}
	defer func() { errOut = errors.Join(errOut, resp.Body.Close()) }()
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
	mainSrc := "assets/bootstrap-main.sh"
	if opts.Wrap {
		mainSrc = "assets/wrap-main.sh"
	}
	mainContent, err := assets.ReadFile(mainSrc)
	if err != nil {
		return "", err
	}
	libContent, err := assets.ReadFile("assets/lib.sh")
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
	binDir := "./bin"
	if opts.BinDir != "" {
		binDir = opts.BinDir
	}
	var tmplOut bytes.Buffer
	err = tmpl.Execute(&tmplOut, map[string]string{
		"tag":       tag,
		"checksums": strings.TrimSpace(string(checksums)),
		"shlib":     string(shlibContent),
		"lib":       string(libContent),
		"main":      string(mainContent),
		"bindir":    binDir,
		"repo_url":  repoURL,
	})
	if err != nil {
		return "", err
	}
	out := strings.TrimSpace(tmplOut.String()) + "\n"
	if runtime.GOOS == "windows" {
		out = windowsLineEndings(out)
	}
	return out, nil
}

func windowsLineEndings(in string) string {
	buf := bytes.NewBuffer(make([]byte, 0, len(in)))
	for i := 0; i < len(in); i++ {
		if in[i] != '\n' {
			buf.WriteByte(in[i])
			continue
		}
		if i == 0 || in[i-1] != '\r' {
			buf.WriteByte('\r')
		}
		buf.WriteByte('\n')
	}
	return buf.String()
}
