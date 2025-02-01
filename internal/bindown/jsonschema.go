package bindown

import (
	_ "embed"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

//go:embed bindown.schema.json
var jsonSchemaText string

// validateConfig checks whether cfg meets the json schema.
func validateConfig(cfg []byte) error {
	var val any
	err := yaml.Unmarshal(cfg, &val)
	if err != nil {
		return fmt.Errorf("config is not valid yaml (or json)")
	}
	schema, err := jsonschema.CompileString("", jsonSchemaText)
	if err != nil {
		return err
	}
	err = schema.Validate(val)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	return nil
}
