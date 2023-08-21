package bindown

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/Masterminds/semver/v3"
	ignore "github.com/sabhiram/go-gitignore"
	"gopkg.in/yaml.v3"
)

//go:generate sh -c "go tool dist list > go_dist_list.txt"

//go:embed go_dist_list.txt
var GoDists string

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
	err := filepath.WalkDir(inputDir, func(path string, dirEntry os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		linfo, err := dirEntry.Info()
		if err != nil {
			return err
		}

		hPath := strings.TrimPrefix(path, inputDir)
		hPath = strings.TrimPrefix(filepath.ToSlash(hPath), "/")
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
			linkPath = filepath.ToSlash(linkPath)
			_, err = hasher.Write([]byte(linkPath))
			return err
		}

		if !linfo.Mode().IsRegular() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
		mustWriteToHash(hasher, content)
		return nil
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
	defer deferErr(&errOut, rdr.Close)

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

func SortBySemverOrString(vals []string) {
	semvers := make(map[string]*semver.Version)
	for _, val := range vals {
		ver, err := semver.NewVersion(val)
		if err == nil {
			semvers[val] = ver
		}
	}

	// descending order only if all values are semvers
	if len(vals) == len(semvers) {
		slices.SortFunc(vals, func(a, b string) int {
			return semvers[b].Compare(semvers[a])
		})
		return
	}

	slices.SortFunc(vals, func(a, b string) int {
		aVer, bVer := semvers[a], semvers[b]
		if aVer == nil || bVer == nil {
			if a == b {
				return 0
			}
			if a < b {
				return -1
			}
			return 1
		}
		return aVer.Compare(bVer)
	})
}

// Unique appends unique values from vals to buf and returns buf
func Unique[V comparable](vals, buf []V) []V {
	seen := make(map[V]bool)
	for _, val := range vals {
		if !seen[val] {
			seen[val] = true
			buf = append(buf, val)
		}
	}
	return buf
}

func dirIsGitIgnored(dir string) (bool, error) {
	ig, err := fileIsGitignored(dir)
	if err != nil {
		return false, err
	}
	if ig {
		return true, nil
	}
	return fileIsGitignored(filepath.Join(dir, "x"))
}

// fileIsGitignored returns true if the file is ignored by a .gitignore file in the same directory or any parent
// directory from the same git repo.
func fileIsGitignored(filename string) (bool, error) {
	dir := filepath.Dir(filename)
	repoBase, err := gitRepo(dir)
	if err != nil {
		return false, err
	}
	if repoBase == "" {
		return false, nil
	}
	for {
		ignoreFile := filepath.Join(dir, ".gitignore")
		var info os.FileInfo
		info, err = os.Stat(ignoreFile)
		if err != nil {
			if !os.IsNotExist(err) {
				return false, err
			}
		} else if info.Mode().Type().IsRegular() {
			var ig *ignore.GitIgnore
			ig, err = ignore.CompileIgnoreFile(ignoreFile)
			if err != nil {
				return false, err
			}
			var relFile string
			relFile, err = filepath.Rel(dir, filename)
			if err != nil {
				return false, err
			}
			if ig.MatchesPath(relFile) {
				return true, nil
			}
		}
		if dir == repoBase {
			break
		}
		dir = filepath.Dir(dir)
	}
	return false, nil
}

// gitRepo returns the path of the base git repo for a dir. Returns ""
// if dir is not in a git repo.
// Does not use git commands. Just checks for .git directory.
func gitRepo(dir string) (string, error) {
	dir = filepath.Clean(dir)
	info, err := os.Stat(filepath.Join(dir, ".git"))
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
	} else if info.IsDir() {
		return dir, nil
	}
	parent := filepath.Dir(dir)
	if parent == dir {
		return "", nil
	}
	return gitRepo(parent)
}

func MapKeys[M ~map[K]V, K comparable, V any](m M) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}
