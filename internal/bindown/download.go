package bindown

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/willabides/bindown/v3/internal/cache"
)

func downloadDependency(
	dep *Dependency,
	dlCache *cache.Cache,
	trustCache, allowMissingChecksum, force bool,
) (cachedFile, key string, unlock func() error, errOut error) {
	if !dep.built {
		panic("downloadDependency called on non-built dependency")
	}
	dlFile, err := urlFilename(dep.url)
	if err != nil {
		return "", "", nil, err
	}

	var downloader func(dir string) error
	checksum := dep.checksum
	if checksum == "" {
		if !allowMissingChecksum {
			err = fmt.Errorf("no checksum configured for %s", dep.name)
			return "", "", nil, err
		}
		var tempDir string
		tempDir, err = os.MkdirTemp("", "bindown")
		if err != nil {
			return "", "", nil, err
		}
		defer deferErr(&errOut, func() error {
			return os.RemoveAll(tempDir)
		})
		tempFile := filepath.Join(tempDir, dlFile)
		checksum, err = getURLChecksum(dep.url, tempFile)
		if err != nil {
			return "", "", nil, err
		}
		downloader = func(dir string) (dlErrOut error) {
			return copyFile(tempFile, filepath.Join(dir, dlFile))
		}
	} else {
		downloader = func(dir string) error {
			ok, dlErr := fileExistsWithChecksum(filepath.Join(dir, dlFile), checksum)
			if dlErr != nil || ok {
				return dlErr
			}
			gotSum, dlErr := downloadFile(filepath.Join(dir, dlFile), dep.url)
			if dlErr != nil {
				return dlErr
			}
			if checksum != gotSum {
				return fmt.Errorf(`checksum mismatch in downloaded file %q 
wanted: %s
got: %s`, cachedFile, checksum, gotSum)
			}
			return nil
		}
	}
	key = cacheKey(checksum)
	if force {
		err = dlCache.Evict(key)
		if err != nil {
			return "", "", nil, err
		}
	}

	validator := func(dir string) error {
		got, sumErr := fileChecksum(filepath.Join(dir, dlFile))
		if sumErr != nil {
			return sumErr
		}
		if got != checksum {
			return fmt.Errorf("expected checksum %s, got %s", checksum, got)
		}
		return nil
	}
	if trustCache && !force {
		validator = nil
	}

	dir, unlock, err := dlCache.Dir(key, validator, downloader)
	if err != nil {
		return "", "", nil, err
	}
	return filepath.Join(dir, dlFile), key, unlock, nil
}

// downloadFile downloads the file at url to targetPath. It returns the checksum of the file.
func downloadFile(targetPath, url string) (_ string, errOut error) {
	hasher := sha256.New()
	err := os.MkdirAll(filepath.Dir(targetPath), 0o750)
	if err != nil {
		return "", err
	}
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer deferErr(&errOut, resp.Body.Close)
	bodyReader := io.TeeReader(resp.Body, hasher)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("failed downloading %s", url)
	}
	out, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer deferErr(&errOut, out.Close)
	_, err = io.Copy(out, bodyReader)
	if err != nil {
		return "", err
	}
	sum := hex.EncodeToString(hasher.Sum(nil))
	return sum, nil
}

// getURLChecksum returns the checksum of the file at dlURL. If tempFile is specified
// it will be used as the temporary file to download the file to and it will be the caller's
// responsibility to clean it up. Otherwise, a temporary file will be created and cleaned up
// automatically.
func getURLChecksum(dlURL, tempFile string) (_ string, errOut error) {
	if tempFile == "" {
		downloadDir, err := os.MkdirTemp("", "bindown")
		if err != nil {
			return "", err
		}
		tempFile = filepath.Join(downloadDir, "download")
		defer deferErr(&errOut, func() error {
			return os.RemoveAll(downloadDir)
		})
	}
	return downloadFile(tempFile, dlURL)
}
