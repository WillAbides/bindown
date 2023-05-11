package main

import (
	"encoding/json"
	"os"

	"github.com/invopop/jsonschema"
	"github.com/willabides/bindown/v4/internal/bindown"
)

func main() {
	r := &jsonschema.Reflector{}
	r.ExpandedStruct = true
	err := r.AddGoComments("github.com/willabides/bindown/v4", "./internal/bindown/")
	if err != nil {
		panic(err)
	}
	schema := r.Reflect(&bindown.Config{})
	schema.ID = "https://willabides.github.io/bindown/bindown.schema.json"

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(schema)
	if err != nil {
		panic(err)
	}
}
