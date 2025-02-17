package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/willabides/bindown/v4/internal/testutil"
)

func Test_templateUpdateVarCmd(t *testing.T) {
	for _, td := range []struct {
		name      string
		config    string
		args      []string
		wantVars  map[string]string
		wantState resultState
	}{
		{
			name:   "no changes",
			config: `templates: {tmpl1: {url: foo}}`,
			args:   []string{"template", "update-vars", "tmpl1"},
		},
		{
			name:   "create var",
			config: `templates: {tmpl1: {url: foo}}`,
			args:   []string{"template", "update-vars", "tmpl1", "--set", "foo=bar"},
			wantVars: map[string]string{
				"foo": "bar",
			},
		},
		{
			name:   "update var",
			config: `templates: {tmpl1: {url: foo, vars: {foo: bar}}}`,
			args:   []string{"template", "update-vars", "tmpl1", "--set", "foo=baz"},
			wantVars: map[string]string{
				"foo": "baz",
			},
		},
		{
			name:   "unset var",
			config: `templates: {tmpl1: {url: foo, vars: {foo: bar}}}`,
			args:   []string{"template", "update-vars", "tmpl1", "--unset", "foo"},
		},
		{
			name:   "unset all vars",
			config: `templates: {tmpl1: {url: foo, vars: {foo: bar, baz: qux}}}`,
			args:   []string{"template", "update-vars", "tmpl1", "--unset", "foo", "--unset", "baz"},
		},
		{
			name:   "unset on empty vars",
			config: `templates: {tmpl1: {url: foo}}`,
			args:   []string{"template", "update-vars", "tmpl1", "--unset", "foo"},
		},
		{
			name:   "set var on non-existent template",
			args:   []string{"template", "update-vars", "fake", "--set", "foo=bar"},
			config: `{}`,
			wantState: resultState{
				stderr: `cmd: error: template "fake" does not exist`,
				exit:   1,
			},
		},
		{
			name:   "unset var on non-existent template",
			args:   []string{"template", "update-vars", "fake", "--unset", "foo"},
			config: `{}`,
			wantState: resultState{
				stderr: `cmd: error: template "fake" does not exist`,
				exit:   1,
			},
		},
	} {
		t.Run(td.name, func(t *testing.T) {
			runner := newCmdRunner(t)
			runner.writeConfigYaml(td.config)
			result := runner.run(td.args...)
			result.assertState(td.wantState)
			configFile := runner.getConfigFile()
			if td.wantVars != nil {
				require.Equal(t, td.wantVars, configFile.Templates["tmpl1"].Vars)
			} else {
				require.True(t,
					configFile.Templates == nil ||
						configFile.Templates["tmpl1"] == nil ||
						len(configFile.Templates["tmpl1"].Vars) == 0,
				)
			}
		})
	}
}

func Test_templateRemoveCmd(t *testing.T) {
	t.Run("remove template", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`
templates:
  tmpl1:
    url: foo
`)
		result := runner.run("template", "remove", "tmpl1")
		result.assertState(resultState{})
		configFile := runner.getConfigFile()
		require.True(t, configFile.Templates == nil || len(configFile.Templates) == 0)
	})

	t.Run("remove non-existent template", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`
templates:
  tmpl1:
    url: foo
`)
		result := runner.run("template", "remove", "tmpl2")
		result.assertState(resultState{
			stderr: `cmd: error: no template named "tmpl2"`,
			exit:   1,
		})
		configFile := runner.getConfigFile()
		require.Equal(t, 1, len(configFile.Templates))
	})
}

func Test_templateListCmd(t *testing.T) {
	remoteConfig := `
systems: ["linux/amd64", "darwin/amd64"]
templates:
  tmpl1:
    url: foo
  tmpl2:
    url: bar
`
	srcFile := filepath.Join(t.TempDir(), "template-source.yaml")
	err := os.WriteFile(srcFile, []byte(remoteConfig), 0o600)
	require.NoError(t, err)

	t.Run("local", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`templates: {tmpl1: {url: foo}, tmpl2: {url: bar}}`)
		result := runner.run("template", "list")
		result.assertState(resultState{
			stdout: "tmpl1\ntmpl2\n",
		})
	})

	t.Run("remote with path", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`template_sources: {source1: ` + srcFile + `}`)
		result := runner.run("template", "list", "--source", "source1")
		result.assertState(resultState{
			stdout: "tmpl1\ntmpl2\n",
		})
	})

	t.Run("remote with http url", func(t *testing.T) {
		server := testutil.ServeFile(t, srcFile, "/template-source.yaml", "")
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`template_sources: {source1: ` + server.URL + `/template-source.yaml}`)
		result := runner.run("template", "list", "--source", "source1")
		result.assertState(resultState{
			stdout: "tmpl1\ntmpl2\n",
		})
	})
}

func Test_templateUpdateFromSourceCmd(t *testing.T) {
	remoteConfig := `
systems: ["linux/amd64", "darwin/amd64"]
templates:
  tmpl1:
    url: foo
  tmpl2:
    url: bar
`
	srcFile := filepath.Join(t.TempDir(), "template-source.yaml")
	err := os.WriteFile(srcFile, []byte(remoteConfig), 0o600)
	require.NoError(t, err)
	server := testutil.ServeFile(t, srcFile, "/template-source.yaml", "")
	remoteURL := server.URL + "/template-source.yaml"

	t.Run("new template", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`template_sources: {source1: ` + remoteURL + `}`)
		result := runner.run("template", "update-from-source", "source1#tmpl1")
		result.assertState(resultState{})
		configFile := runner.getConfigFile()
		require.Equal(t, "foo", *configFile.Templates["source1#tmpl1"].URL)
	})

	t.Run("new template with --source", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`template_sources: {source1: ` + remoteURL + `}`)
		result := runner.run("template", "update-from-source", "my_tmpl", "--source", "source1#tmpl1")
		result.assertState(resultState{})
		configFile := runner.getConfigFile()
		require.Equal(t, "foo", *configFile.Templates["my_tmpl"].URL)
	})

	t.Run("invalid source name", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`template_sources: {source1: ` + remoteURL + `}`)
		result := runner.run("template", "update-from-source", "invalid")
		result.assertState(resultState{
			stderr: `cmd: error: source must be formatted as source#name (with the #)`,
			exit:   1,
		})
		configFile := runner.getConfigFile()
		require.Equal(t, 0, len(configFile.Templates))
	})

	t.Run("source not found", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`template_sources: {source1: ` + remoteURL + `}`)
		result := runner.run("template", "update-from-source", "source2#tmpl1")
		result.assertState(resultState{
			stderr: `cmd: error: no template source named "source2"`,
			exit:   1,
		})
		configFile := runner.getConfigFile()
		require.Equal(t, 0, len(configFile.Templates))
	})

	t.Run("source template not found", func(t *testing.T) {
		runner := newCmdRunner(t)
		runner.writeConfigYaml(`template_sources: {source1: ` + remoteURL + `}`)
		result := runner.run("template", "update-from-source", "source1#tmpl3")
		result.assertState(resultState{
			stderr: `cmd: error: source has no template named "tmpl3"`,
			exit:   1,
		})
		configFile := runner.getConfigFile()
		require.Equal(t, 0, len(configFile.Templates))
	})
}
