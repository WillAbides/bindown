package bindown

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	bootstrapper "github.com/willabides/bindown/v4/internal/build-bootstrapper"
	"github.com/willabides/bindown/v4/internal/cache"
)

//go:embed wrapper.gotmpl
var wrapperTmplText string

func install(
	ctx context.Context,
	dep *Dependency,
	targetPath, cacheDir string,
	force, toCache, missingSums bool,
) (_ string, errOut error) {
	dep.mustBeBuilt()
	if toCache {
		instCache := &cache.Cache{Root: filepath.Join(cacheDir, "bin")}
		key := dep.cacheKey()
		validateFn := func(dir string) error {
			filename := filepath.Join(dir, dep.binName())
			if !FileExists(filename) {
				return fmt.Errorf("file %q does not exist", filename)
			}
			return nil
		}
		popFn := func(dir string) error {
			filename := filepath.Join(dir, dep.binName())
			_, err := install(ctx, dep, filename, cacheDir, force, false, missingSums)
			return err
		}
		dir, unlock, err := instCache.Dir(key, validateFn, popFn)
		if err != nil {
			return "", err
		}
		err = unlock()
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, dep.binName()), nil
	}

	dlCache := cache.Cache{Root: filepath.Join(cacheDir, "downloads")}
	dlFile, key, dlUnlock, err := downloadDependency(dep, &dlCache, missingSums, force)
	if err != nil {
		return "", err
	}
	defer deferErr(&errOut, dlUnlock)

	extractsCache := cache.Cache{Root: filepath.Join(cacheDir, "extracts")}
	extractDir, exUnlock, err := extractDependencyToCache(ctx, dlFile, cacheDir, key, &extractsCache, force)
	if err != nil {
		return "", err
	}
	defer deferErr(&errOut, exUnlock)

	var binName string
	if dep.BinName != nil {
		binName = *dep.BinName
	}
	if binName == "" {
		binName = dep.name
	}
	archivePath := filepath.FromSlash(binName)
	if dep.ArchivePath != nil {
		archivePath = filepath.FromSlash(*dep.ArchivePath)
	}
	extractBin := filepath.Join(extractDir, archivePath)
	if dep.Link != nil && *dep.Link {
		return targetPath, linkBin(targetPath, extractBin)
	}
	if FileExists(targetPath) {
		err = os.RemoveAll(targetPath)
		if err != nil {
			return "", err
		}
	}
	err = os.MkdirAll(filepath.Dir(targetPath), 0o755)
	if err != nil {
		return "", err
	}
	err = copyFile(extractBin, targetPath)
	if err != nil {
		return "", err
	}
	targetStat, err := os.Stat(targetPath)
	if err != nil {
		return "", err
	}
	err = os.Chmod(targetPath, addExec(targetStat.Mode()))
	if err != nil {
		return "", err
	}
	return targetPath, nil
}

type wrapperTmplVars struct {
	DependencyName string
	BindownExec    string
	ConfigFile     string
	FlagArgs       string
}

var wrapperTmpl = template.Must(template.New("wrapper").Parse(wrapperTmplText))

func createWrapper(name, target, bindownExec, cacheDir, configFile string, missingSums bool) (string, error) {
	wrapperDir := filepath.Dir(target)
	err := os.MkdirAll(wrapperDir, 0o750)
	if err != nil {
		return "", err
	}
	if bindownExec == "" {
		bindownExec = "bindown"
	} else {
		bindownExec, err = relPath(wrapperDir, bindownExec)
		if err != nil {
			return "", err
		}
		if !strings.HasPrefix(bindownExec, ".") && !filepath.IsAbs(bindownExec) {
			bindownExec = "./" + bindownExec
		}
	}

	configFile, err = relPath(wrapperDir, configFile)
	if err != nil {
		return "", err
	}

	flagArgs := `--to-cache`
	if missingSums {
		flagArgs += " \\\n    --allow-missing-checksum"
	}
	addFlagArg := func(name, value string) {
		flagArgs += fmt.Sprintf(" \\\n    %s %q", name, value)
	}
	addFlagArg("--configfile", configFile)

	err = os.MkdirAll(cacheDir, 0o750)
	if err != nil {
		return "", err
	}
	cacheDir, err = relPath(wrapperDir, cacheDir)
	if err != nil {
		return "", err
	}
	addFlagArg("--cache", cacheDir)

	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o750)
	if err != nil {
		return "", err
	}

	err = wrapperTmpl.Execute(file, wrapperTmplVars{
		DependencyName: name,
		BindownExec:    bindownExec,
		ConfigFile:     configFile,
		FlagArgs:       flagArgs,
	})
	if err != nil {
		return "", err
	}
	err = file.Close()
	if err != nil {
		return "", err
	}
	return target, nil
}

func createBindownWrapper(target, cacheDir, tag, baseURL string) (string, error) {
	wrapperDir := filepath.Dir(target)
	err := os.MkdirAll(wrapperDir, 0o750)
	if err != nil {
		return "", err
	}
	binDir := filepath.Join(cacheDir, "bootstrapped")
	err = os.MkdirAll(binDir, 0o750)
	if err != nil {
		return "", err
	}
	binDir, err = relPath(wrapperDir, binDir)
	if err != nil {
		return "", err
	}
	content, err := bootstrapper.Build(tag, &bootstrapper.BuildOpts{
		BinDir:  binDir,
		Wrap:    true,
		BaseURL: baseURL,
	})
	if err != nil {
		return "", err
	}
	err = os.WriteFile(target, []byte(content), 0o750)
	if err != nil {
		return "", err
	}
	return target, nil
}

// relPath returns target relative to base and converted to slash-separated path.
// Unlike filepath.Rel, it converts both paths to absolute paths before calculating the relative path.
func relPath(base, target string) (string, error) {
	// if it works without abs, use that
	rel, err := filepath.Rel(base, target)
	if err == nil {
		return filepath.ToSlash(rel), nil
	}

	// convert to abs and try again
	base, err = filepath.Abs(base)
	if err != nil {
		return "", err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return "", err
	}
	rel, err = filepath.Rel(base, target)
	if err == nil {
		return filepath.ToSlash(rel), nil
	}

	// resolve symlinks and try again
	base, err = filepath.EvalSymlinks(base)
	if err != nil {
		return "", err
	}
	target, err = filepath.EvalSymlinks(target)
	if err != nil {
		return "", err
	}
	rel, err = filepath.Rel(base, target)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

// addExec adds exec for each read bit in mode
func addExec(mode os.FileMode) os.FileMode {
	if mode&0o4 != 0 {
		mode |= 0o1
	}
	if mode&0o40 != 0 {
		mode |= 0o10
	}
	if mode&0o400 != 0 {
		mode |= 0o100
	}
	return mode
}
