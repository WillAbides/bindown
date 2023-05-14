package builddep

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v4/internal/bindown"
	"gopkg.in/yaml.v3"
)

func selectFirstCandidate(candidates []*archiveFileCandidate, candidate *archiveFileCandidate) error {
	*candidate = *candidates[0]
	return nil
}

func Test_sanity(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	ctx := context.Background()
	tkn := os.Getenv("GITHUB_TOKEN")
	if tkn == "" {
		t.Skip("GITHUB_TOKEN not set")
	}
	urls, version, homepage, description, err := QueryGitHubRelease(ctx, "willabides/bindown", "v3.16.1", tkn)
	require.NoError(t, err)
	require.Equal(t, "3.16.1", version)
	require.Equal(t, 18, len(urls))
	initialConfig := `
systems:
  - linux/386
  - darwin/amd64
  - windows/amd64
  - darwin/386
`
	var cfg bindown.Config
	err = yaml.Unmarshal([]byte(initialConfig), &cfg)
	require.NoError(t, err)
	err = addDependency(ctx, &cfg, "bindown", version, homepage, description, urls, selectFirstCandidate)
	require.NoError(t, err)
	got, err := yaml.Marshal(&cfg)
	require.NoError(t, err)
	fmt.Println(string(got))

	want := `
dependencies:
    bindown:
        template: bindown
        vars:
            version: 3.16.1
templates:
    bindown:
        homepage: https://github.com/WillAbides/bindown
        url: https://github.com/WillAbides/bindown/releases/download/v{{.version}}/bindown_{{.version}}_{{.os}}_{{.arch}}{{.urlSuffix}}
        archive_path: bindown{{.archivePathSuffix}}
        bin: bindown
        vars:
            archivePathSuffix: ""
            urlSuffix: .tar.gz
        overrides:
            - matcher:
                os:
                    - windows
              dependency:
                vars:
                    archivePathSuffix: .exe
        systems:
            - darwin/amd64
            - linux/386
            - windows/amd64
        required_vars:
            - version
systems:
    - linux/386
    - darwin/amd64
    - windows/amd64
    - darwin/386
url_checksums:
    https://github.com/WillAbides/bindown/releases/download/v3.16.1/bindown_3.16.1_darwin_amd64.tar.gz: 724502e502dd7929fa717c0aab0bc759d8bf221ccf58b535f16732a574fe560f
    https://github.com/WillAbides/bindown/releases/download/v3.16.1/bindown_3.16.1_linux_386.tar.gz: 35cca77fb8bad4d7a1644a2cd0b61a34ec4ef5d74943077e22faab4aba9fda66
    https://github.com/WillAbides/bindown/releases/download/v3.16.1/bindown_3.16.1_windows_amd64.tar.gz: d84601ef49a8f7339a96bd4a8e0bf89d3253a3403e84f4d25595cff786eafb88
`
	require.Equal(t, strings.TrimSpace(want), string(bytes.TrimSpace(got)))
}

//nolint:unused // this is for occasional manual testing
func adhocRelease(t *testing.T, repo, tag string) {
	ctx := context.Background()
	tkn := os.Getenv("GITHUB_TOKEN")
	if tkn == "" {
		t.Skip("GITHUB_TOKEN not set")
	}
	urls, version, homepage, description, err := QueryGitHubRelease(ctx, repo, tag, tkn)
	require.NoError(t, err)
	var cfg bindown.Config
	err = addDependency(ctx, &cfg, "x", version, homepage, description, urls, selectFirstCandidate)
	require.NoError(t, err)
	got, err := yaml.Marshal(&cfg)
	require.NoError(t, err)
	fmt.Println(string(got))
}
