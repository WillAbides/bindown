package main

import "testing"

func Test_versionCmd(t *testing.T) {
	runner := newCmdRunner(t)
	result := runner.run("version")
	result.assertState(resultState{
		stdout: "bindown: version unknown",
	})
}
