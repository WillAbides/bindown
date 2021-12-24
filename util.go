package bindown

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// CopyStringMap returns a copy of mp
func CopyStringMap(mp map[string]string) map[string]string {
	result := make(map[string]string, len(mp))
	for k, v := range mp {
		result[k] = v
	}
	return result
}

// setStringMapDefault sets map[key] to val unless it is already set
func setStringMapDefault(mp map[string]string, key, val string) {
	_, ok := mp[key]
	if ok {
		return
	}
	mp[key] = val
}

// ExecuteTemplate executes a template
func ExecuteTemplate(tmplString, goos, arch string, vars map[string]string) (string, error) {
	vars = CopyStringMap(vars)
	setStringMapDefault(vars, "os", goos)
	setStringMapDefault(vars, "arch", arch)
	tmpl, err := template.New("").Option("missingkey=error").Parse(tmplString)
	if err != nil {
		fmt.Println(err.Error())
		return "", fmt.Errorf("%q is not a valid template", tmplString)
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, vars)
	if err != nil {
		return "", fmt.Errorf("error applying template: %v", err)
	}
	return buf.String(), nil
}

// DirectoryChecksum returns a hash of directory contents.
func DirectoryChecksum(inputDir string) (string, error) {
	hasher := fnv.New64a()
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		linfo, err := os.Lstat(path)
		if err != nil {
			return err
		}

		hPath := strings.TrimPrefix(strings.TrimPrefix(path, inputDir), string(filepath.Separator))
		_, err = hasher.Write([]byte(hPath))
		if err != nil {
			return err
		}

		// if it's a symlink, just add the target path to the hash
		if linfo.Mode()&os.ModeSymlink != 0 {
			var linkPath string
			linkPath, err = os.Readlink(path)
			if err != nil {
				return err
			}
			_, err = hasher.Write([]byte(linkPath))
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		fi, err := os.Open(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(hasher, fi)
		if err != nil {
			return err
		}

		return fi.Close()
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// HexHash returns a hex representation of data's hash
// This will only return non-nil error if given a hasher that can return a non-nil error from Write()
func HexHash(hasher hash.Hash, data ...[]byte) (string, error) {
	hasher.Reset()
	for _, datum := range data {
		_, err := hasher.Write(datum)
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// FileChecksum returns the hex checksum of a file
func FileChecksum(filename string) (string, error) {
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return HexHash(sha256.New(), fileBytes)
}

// FileExists asserts that a file exists
func FileExists(path string) bool {
	if _, err := os.Stat(filepath.FromSlash(path)); !os.IsNotExist(err) {
		return true
	}
	return false
}

// FileExistsWithChecksum returns true if the file both exists and has a matching checksum
func FileExistsWithChecksum(filename, checksum string) (bool, error) {
	if !FileExists(filename) {
		return false, nil
	}
	got, err := FileChecksum(filename)
	if err != nil {
		return false, err
	}
	return checksum == got, nil
}
