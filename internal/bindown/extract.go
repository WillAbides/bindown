package bindown

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
	"github.com/willabides/bindown/v4/internal/cache"
)

func extractDependencyToCache(
	archivePath, cacheDir, key string,
	exCache *cache.Cache,
	force bool,
) (extractDir string, unlock func() error, _ error) {
	extractSumsDir := filepath.Join(cacheDir, ".extract_sums")
	err := os.MkdirAll(extractSumsDir, 0o755)
	if err != nil {
		return "", nil, err
	}
	extractSumFile := filepath.Join(extractSumsDir, key+".sum")

	extractor := func(dir string) error {
		exErr := extract(archivePath, dir)
		if exErr != nil {
			return exErr
		}
		gotSum, exErr := directoryChecksum(dir)
		if exErr != nil {
			return exErr
		}
		return os.WriteFile(extractSumFile, []byte(gotSum), 0o644)
	}

	if force {
		err = exCache.Evict(key)
		if err != nil {
			return "", nil, err
		}
	}
	return exCache.Dir(key, nil, extractor)
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
	byExt, err := archiver.ByExtension(dlName)
	if err != nil {
		return copyFile(tarPath, filepath.Join(extractDir, dlName))
	}
	switch x := byExt.(type) {
	case archiver.Unarchiver:
		return x.Unarchive(tarPath, extractDir)
	case archiver.Decompressor:
		dest := filepath.Join(
			extractDir,
			strings.TrimSuffix(dlName, filepath.Ext(dlName)),
		)
		return archiver.FileCompressor{Decompressor: x}.DecompressFile(tarPath, dest)
	default:
		return copyFile(tarPath, filepath.Join(extractDir, dlName))
	}
}
