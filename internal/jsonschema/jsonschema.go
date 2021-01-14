package jsonschema

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/qri-io/jsonschema"
	"github.com/willabides/bindown/v3/internal/jsonschema/schemafiles"
	"github.com/willabides/bindown/v3/internal/util"
)

//go:generate go-bindata -nometadata -pkg schemafiles -o schemafiles/schemafiles.go ../../bindown.schema.json

var jsonSchema *jsonschema.RootSchema

func init() {
	schemaData, err := schemafiles.Asset("../../bindown.schema.json")
	util.Must(err)
	jsonSchema = new(jsonschema.RootSchema)
	util.Must(json.Unmarshal(schemaData, jsonSchema))
}

// ValidateConfig checks whether cfg meets the json schema.
func ValidateConfig(cfg []byte) error {
	cfgJSON, err := yaml.YAMLToJSON(cfg)
	if err != nil {
		return fmt.Errorf("config is not valid yaml (or json)")
	}
	validationErrs, err := jsonSchema.ValidateBytes(cfgJSON)
	if err != nil {
		return fmt.Errorf("unexpected error running jsonSchema.ValidateBytes: %v", err)
	}
	if len(validationErrs) == 0 {
		return nil
	}
	sort.Slice(validationErrs, func(i, j int) bool {
		return validationErrs[i].Error() < validationErrs[j].Error()
	})
	msgs := make([]string, len(validationErrs))
	for i, validationErr := range validationErrs {
		msgs[i] = validationErr.Error()
	}
	return fmt.Errorf("invalid config:\n%s", strings.Join(msgs, "\n"))
}
