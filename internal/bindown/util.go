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

	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
)

// executeTemplate executes a template
func executeTemplate(tmplString, goos, arch string, vars map[string]string) (string, error) {
	tmplData := map[string]string{
		"os":   goos,
		"arch": arch,
	}
	maps.Copy(tmplData, vars)
	tmpl, err := template.New("").Option("missingkey=error").Parse(tmplString)
	if err != nil {
		return "", fmt.Errorf("%q is not a valid template", tmplString)
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, tmplData)
	if err != nil {
		return "", fmt.Errorf("error applying template: %v", err)
	}
	return buf.String(), nil
}

// directoryChecksum returns a hash of directory contents.
func directoryChecksum(inputDir string) (string, error) {
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

func mustWriteToHash(hasher hash.Hash, data []byte) {
	_, err := hasher.Write(data)
	if err != nil {
		// hash.Hash.Write() never returns an error
		// https://github.com/golang/go/blob/go1.17/src/hash/hash.go#L27-L29
		panic(err)
	}
}

// fileChecksum returns the hex checksum of a file
func fileChecksum(filename string) (string, error) {
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	hasher := sha256.New()
	mustWriteToHash(hasher, fileBytes)
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// fileExists asserts that a file exist or symlink exists.
// Returns false for symlinks pointing to non-existent files.
func fileExists(path string) bool {
	_, statErr := os.Stat(filepath.FromSlash(path))
	return !os.IsNotExist(statErr)
}

// fileExistsWithChecksum returns true if the file both exists and has a matching checksum
func fileExistsWithChecksum(filename, checksum string) (bool, error) {
	if !fileExists(filename) {
		return false, nil
	}
	got, err := fileChecksum(filename)
	if err != nil {
		return false, err
	}
	return checksum == got, nil
}

// copyFile copies file from src to dst
// modeTrans is a function for setting the destination FileMode. It accepts the source FileMode. If nil, the unmodified
// source FileMode is used.
func copyFile(src, dst string) (errOut error) {
	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcStat.Mode().IsRegular() {
		return fmt.Errorf("not a regular file")
	}
	rdr, err := os.Open(src)
	if err != nil {
		return err
	}

	dstMode := srcStat.Mode()
	writer, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, dstMode)
	if err != nil {
		return err
	}
	defer deferErr(&errOut, writer.Close)

	_, err = io.Copy(writer, rdr)
	return err
}

func deferErr(errOut *error, fn func() error) {
	deferredErr := fn()
	if *errOut == nil {
		*errOut = deferredErr
	}
}

func clonePointer[T comparable](p *T) *T {
	if p == nil {
		return nil
	}
	val := *p
	return &val
}

// overrideValue sets p to override if override is not nil
func overrideValue[T comparable](p, override *T) *T {
	if override == nil {
		return p
	}
	return clonePointer(override)
}

func EncodeYaml(w io.Writer, v any) (errOut error) {
	encoder := yaml.NewEncoder(w)
	defer deferErr(&errOut, encoder.Close)
	encoder.SetIndent(2)
	return encoder.Encode(v)
}
