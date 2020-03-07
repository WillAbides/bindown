package cli

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/alecthomas/kong"
)

func multipathMapper(d *kong.DecodeContext, target reflect.Value) error {
	if target.Kind() != reflect.String {
		return fmt.Errorf("\"multipath\" type must be applied to a string not %s", target.Type())
	}
	var path string
	err := d.Scan.PopValueInto("file", &path)
	if err != nil {
		return err
	}

	existing := multifileFindExisting(path)
	if existing == "" {
		return fmt.Errorf("not found")
	}
	target.SetString(existing)
	return nil
}

func multifileFindExisting(multiFile string) string {
	for _, configFile := range strings.Split(multiFile, "|") {
		configFile = kong.ExpandPath(configFile)
		stat, err := os.Stat(configFile)
		if err != nil {
			continue
		}
		if stat.IsDir() {
			continue
		}
		return configFile
	}
	return ""
}
