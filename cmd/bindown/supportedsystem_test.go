package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/willabides/bindown/v3"
)

func Test_supportedSystemListCmd(t *testing.T) {
	for _, td := range []struct {
		name      string
		config    bindown.Config
		wantState resultState
	}{
		{
			name: "no systems",
			config: bindown.Config{
				Dependencies: map[string]*bindown.Dependency{
					// this demonstrates that a dependency's systems are not used
					"dep1": {
						URL:     ptr("foo"),
						Systems: []bindown.SystemInfo{{OS: "linux", Arch: "amd64"}},
					},
				},
			},
		},
		{
			name: "yes system",
			config: bindown.Config{
				Systems: []bindown.SystemInfo{
					{OS: "linux", Arch: "amd64"},
					{OS: "darwin", Arch: "amd64"},
					{OS: "windows", Arch: "amd64"},
				},
			},
			wantState: resultState{
				stdout: strings.Join([]string{
					"darwin/amd64",
					"linux/amd64",
					"windows/amd64",
				}, "\n"),
			},
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			runner := newCmdRunner(t)
			runner.writeConfig(&td.config)
			result := runner.run("supported-system", "list")
			result.assertState(td.wantState)
		})
	}
}

func Test_supportedSystemsRemoveCmd(t *testing.T) {
	for _, td := range []struct {
		name        string
		config      bindown.Config
		args        []string
		state       resultState
		wantSystems []string
	}{
		{
			name: "removes system",
			config: bindown.Config{
				Systems: []bindown.SystemInfo{
					{OS: "darwin", Arch: "amd64"},
					{OS: "linux", Arch: "amd64"},
					{OS: "windows", Arch: "amd64"},
				},
			},
			args:        []string{"linux/amd64"},
			wantSystems: []string{"darwin/amd64", "windows/amd64"},
		},
		{
			name: "no-op if system not found",
			config: bindown.Config{
				Systems: []bindown.SystemInfo{
					{OS: "darwin", Arch: "amd64"},
					{OS: "linux", Arch: "amd64"},
					{OS: "windows", Arch: "amd64"},
				},
			},
			args:        []string{"linux/386"},
			wantSystems: []string{"darwin/amd64", "linux/amd64", "windows/amd64"},
		},
		{
			name: "no systems",
			args: []string{"linux/386"},
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			runner := newCmdRunner(t)
			runner.writeConfig(&td.config)
			result := runner.run(append([]string{"supported-system", "remove"}, td.args...)...)
			result.assertState(td.state)
			cfg := runner.getConfigFile()
			gotSystems := make([]string, len(cfg.Systems))
			for i, system := range cfg.Systems {
				gotSystems[i] = system.String()
			}
			wantSystems := td.wantSystems
			if wantSystems == nil {
				wantSystems = []string{}
			}
			require.Equal(t, wantSystems, gotSystems)
		})
	}
}

func Test_supportedSystemAddCmd(t *testing.T) {
	for _, td := range []struct {
		name        string
		config      bindown.Config
		args        []string
		state       resultState
		wantSystems []string
	}{
		{
			name: "adds system",
			config: bindown.Config{
				Systems: []bindown.SystemInfo{
					{OS: "darwin", Arch: "amd64"},
				},
			},
			args:        []string{"linux/amd64"},
			wantSystems: []string{"darwin/amd64", "linux/amd64"},
		},
		{
			name: "no-op if system already exists",
			config: bindown.Config{
				Systems: []bindown.SystemInfo{
					{OS: "darwin", Arch: "amd64"},
					{OS: "linux", Arch: "amd64"},
				},
			},
			args:        []string{"linux/amd64"},
			wantSystems: []string{"darwin/amd64", "linux/amd64"},
		},
		{
			// we know this honor --skipchecksums because the test dependency url is not valid
			name: "honors --skipchecksums",
			config: bindown.Config{
				Dependencies: map[string]*bindown.Dependency{
					"dep1": {
						URL: ptr("foo"),
					},
				},
			},
			args:        []string{"linux/amd64", "--skipchecksums"},
			wantSystems: []string{"linux/amd64"},
		},
		{
			name: "works with existing checksums",
			config: bindown.Config{
				Dependencies: map[string]*bindown.Dependency{
					"dep1": {
						URL: ptr("foo"),
					},
				},
				URLChecksums: map[string]string{
					"foo": "deadbeef",
				},
			},
			args:        []string{"linux/amd64"},
			wantSystems: []string{"linux/amd64"},
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			runner := newCmdRunner(t)
			runner.writeConfig(&td.config)
			result := runner.run(append([]string{"supported-system", "add"}, td.args...)...)
			result.assertState(td.state)
			cfg := runner.getConfigFile()
			gotSystems := make([]string, len(cfg.Systems))
			for i, system := range cfg.Systems {
				gotSystems[i] = system.String()
			}
			wantSystems := td.wantSystems
			if wantSystems == nil {
				wantSystems = []string{}
			}
			require.Equal(t, wantSystems, gotSystems)
		})
	}
}