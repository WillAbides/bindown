package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	bootstrapper "github.com/willabides/bindown/v4/internal/build-bootstrapper"
)

type bootstrapCmd struct {
	Tag     string `kong:"hidden"`
	BaseURL string `kong:"hidden,name='base-url',default='https://github.com'"`
	Output  string `kong:"help='output file, writes to stdout if not set',type='path'"`
}

func (c *bootstrapCmd) Run(ctx *runContext) error {
	tag := c.Tag
	if tag == "" {
		tag = getVersion()
	}
	if tag == "" {
		return fmt.Errorf("version is required")
	}
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	opts := bootstrapper.BuildOpts{BaseURL: c.BaseURL}
	content, err := bootstrapper.Build(tag, &opts)
	if err != nil {
		return err
	}
	if c.Output == "" {
		mustPrint(ctx.stdout, content)
		return nil
	}
	err = os.MkdirAll(filepath.Dir(c.Output), 0o755)
	if err != nil {
		return err
	}
	err = os.WriteFile(c.Output, []byte(content), 0o755)
	if err != nil {
		return err
	}
	return nil
}
