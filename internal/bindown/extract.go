package bindown

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archives"
	"github.com/willabides/bindown/v4/internal/cache"
)

func extractDependencyToCache(
	ctx context.Context,
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
		exErr := extract(ctx, archivePath, dir)
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
func extract(ctx context.Context, archivePath, extractDir string) (errOut error) {
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
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() { errOut = errors.Join(errOut, file.Close()) }()
	format, reader, err := archives.Identify(ctx, archivePath, file)
	if err != nil {
		return copyFile(tarPath, filepath.Join(extractDir, dlName))
	}
	switch x := format.(type) {
	case archives.Extractor:
		return extractExtractor(ctx, reader, x, extractDir)
	case archives.Decompressor:
		return extractDecompressor(reader, x, tarPath)
	default:
		return copyFile(tarPath, filepath.Join(extractDir, dlName))
	}
}

func extractDecompressor(reader io.Reader, decompressor archives.Decompressor, dest string) (errOut error) {
	srcRdr, err := decompressor.OpenReader(reader)
	if err != nil {
		return err
	}
	defer func() { errOut = errors.Join(errOut, srcRdr.Close()) }()
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { errOut = errors.Join(errOut, destFile.Close()) }()
	_, err = io.Copy(destFile, srcRdr)
	return err
}

func extractExtractor(ctx context.Context, reader io.Reader, extractor archives.Extractor, base string) (errOut error) {
	cleanBase := filepath.Clean(base)

	// securePath ensures the path is safely relative to the target directory.
	securePath := func(relativePath string) (string, error) {
		relativePath = filepath.Clean("/" + relativePath)
		relativePath = strings.TrimPrefix(relativePath, string(os.PathSeparator))

		dstPath := filepath.Join(base, relativePath)

		cleanDst := filepath.Clean(dstPath)
		if !strings.HasPrefix(cleanDst+string(os.PathSeparator), cleanBase+string(os.PathSeparator)) {
			return "", fmt.Errorf("illegal file path: %s", dstPath)
		}
		return dstPath, nil
	}

	return extractor.Extract(ctx, reader, func(_ context.Context, info archives.FileInfo) (errOut error) {
		dstPath, err := securePath(info.NameInArchive)
		if err != nil {
			return err
		}

		parentDir := filepath.Dir(dstPath)

		err = os.MkdirAll(parentDir, 0o700)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		if info.LinkTarget != "" {
			var targetPath string
			targetPath, err = securePath(info.LinkTarget)
			if err != nil {
				return fmt.Errorf("invalid symlink target: %w", err)
			}
			return os.Symlink(targetPath, dstPath)
		}

		// Check and handle parent directory permissions
		originalMode, err := os.Stat(parentDir)
		if err != nil {
			return err
		}

		// If parent directory is read-only, temporarily make it writable
		if originalMode.Mode().Perm()&0o200 == 0 {
			err = os.Chmod(parentDir, originalMode.Mode()|0o200)
			if err != nil {
				return fmt.Errorf("chmod parent directory: %w", err)
			}
			defer func() {
				err = os.Chmod(parentDir, originalMode.Mode())
				if err != nil {
					errOut = fmt.Errorf("restoring original permissions: %w", err)
				}
			}()
		}

		// Handle regular files
		file, err := info.Open()
		if err != nil {
			return fmt.Errorf("opening file: %w", err)
		}
		defer func() { errOut = errors.Join(errOut, file.Close()) }()

		dstFile, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY, info.Mode())
		if err != nil {
			return fmt.Errorf("create file: %w", err)
		}
		defer func() { errOut = errors.Join(errOut, dstFile.Close()) }()

		_, err = io.Copy(dstFile, file)
		return err
	})
}
