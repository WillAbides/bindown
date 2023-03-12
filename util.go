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

// copyStringMap returns a copy of mp
func copyStringMap(mp map[string]string) map[string]string {
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

// executeTemplate executes a template
func executeTemplate(tmplString, goos, arch string, vars map[string]string) (string, error) {
	vars = copyStringMap(vars)
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

// hexHash returns a hex representation of data's hash
func hexHash(hasher hash.Hash, data ...[]byte) string {
	hasher.Reset()
	for _, datum := range data {
		_, err := hasher.Write(datum)
		if err != nil {
			// hash.Hash.Write() never returns an error
			// https://github.com/golang/go/blob/go1.17/src/hash/hash.go#L27-L29
			panic(err)
		}
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

// fileChecksum returns the hex checksum of a file
func fileChecksum(filename string) (string, error) {
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return hexHash(sha256.New(), fileBytes), nil
}

// fileExists asserts that a file exists
func fileExists(path string) bool {
	if _, err := os.Stat(filepath.FromSlash(path)); !os.IsNotExist(err) {
		return true
	}
	return false
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
func copyFile(src, dst string, closeCloser func(io.Closer)) error {
	if closeCloser == nil {
		closeCloser = func(_ io.Closer) {}
	}
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
	defer closeCloser(rdr)

	writer, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcStat.Mode())
	if err != nil {
		return err
	}
	defer closeCloser(writer)

	_, err = io.Copy(writer, rdr)
	return err
}
