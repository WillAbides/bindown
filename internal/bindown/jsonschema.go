package bindown

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
	validator "github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

var (
	_jsonSchema     *jsonschema.Schema
	_jsonSchemaText []byte
)

// validateConfig checks whether cfg meets the json schema.
func validateConfig(cfg []byte) error {
	var val any
	err := yaml.Unmarshal(cfg, &val)
	if err != nil {
		return fmt.Errorf("config is not valid yaml (or json)")
	}
	if _jsonSchemaText == nil {
		var s *jsonschema.Schema
		s, err = GetJSONSchema()
		if err != nil {
			return err
		}
		_jsonSchemaText, err = json.Marshal(s)
		if err != nil {
			return err
		}
	}
	vSchema, err := validator.CompileString("", string(_jsonSchemaText))
	if err != nil {
		return err
	}
	err = vSchema.Validate(val)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	return nil
}

func GetJSONSchema() (*jsonschema.Schema, error) {
	if _jsonSchema == nil {
		r := &jsonschema.Reflector{}
		r.ExpandedStruct = true
		err := r.AddGoComments("github.com/willabides/bindown/v4", "./")
		if err != nil {
			return nil, err
		}
		_jsonSchema = r.Reflect(&Config{})
		_jsonSchema.ID = "https://willabides.github.io/bindown/bindown.schema.json"
	}
	return _jsonSchema, nil
}
