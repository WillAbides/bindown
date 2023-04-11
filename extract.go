package bindown

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mholt/archiver/v3"
	"github.com/willabides/bindown/v3/internal/cache"
)

func extractDependencyToCache(
	archivePath, cacheDir, key string,
	exCache *cache.Cache,
	trustCache, force bool,
) (outDir string, fsDir fs.FS, unlock func() error, errOut error) {
	extractSumsDir := filepath.Join(cacheDir, ".extract_sums")
	err := os.MkdirAll(extractSumsDir, 0o755)
	if err != nil {
		return "", nil, nil, err
	}
	extractSumFile := filepath.Join(extractSumsDir, key+".sum")

	validator := func(dir fs.FS) error {
		if trustCache {
			return nil
		}
		wantSum, vErr := os.ReadFile(extractSumFile)
		if vErr != nil {
			return vErr
		}
		gotSum, vErr := fsDirectoryChecksum(dir)
		if vErr != nil {
			return vErr
		}
		if gotSum != string(wantSum) {
			return fmt.Errorf("expected checksum %s, got %s", wantSum, gotSum)
		}
		return nil
	}

	extractor := func(dir string) error {
		exErr := extract(archivePath, dir)
		if exErr != nil {
			return exErr
		}
		gotSum, exErr := fsDirectoryChecksum(os.DirFS(dir))
		if exErr != nil {
			return exErr
		}
		return os.WriteFile(extractSumFile, []byte(gotSum), 0o644)
	}

	if force {
		err = exCache.Evict(key)
		if err != nil {
			return "", nil, nil, err
		}
	}
	fsDir, unlock, err = exCache.Dir(key, validator, extractor)
	if err != nil {
		return "", nil, nil, err
	}
	outDir = filepath.Join(exCache.Root, key)
	return outDir, fsDir, unlock, nil
}

// extract extracts an archive
func extract(archivePath, extractDir string) error {
	dlName := filepath.Base(archivePath)
	downloadDir := filepath.Dir(archivePath)

	err := os.RemoveAll(extractDir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(extractDir, 0o750)
	if err != nil {
		return err
	}
	tarPath := filepath.Join(downloadDir, dlName)
	_, err = archiver.ByExtension(dlName)
	if err != nil {
		return copyFile(tarPath, filepath.Join(extractDir, dlName), nil)
	}
	return archiver.Unarchive(tarPath, extractDir)
}
