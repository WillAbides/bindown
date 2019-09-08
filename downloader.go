package bindownloader

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type downloader struct {
	URL        string `json:"url"`
	Checksum   string `json:"checksum"`
	LinkSource string `json:"symlink,omitempty"`
	BinName    string `json:"bin"`
	MoveFrom   string `json:"move-from"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
}

func (d *downloader) downloadableName() string {
	return filepath.Base(filepath.FromSlash(d.URL))
}

func (d *downloader) downloadablePath(targetDir string) string {
	return filepath.Join(targetDir, d.downloadableName())
}

func (d *downloader) binPath(targetDir string) string {
	return filepath.Join(targetDir, d.BinName)
}

func (d *downloader) chmod(targetDir string) error {
	return os.Chmod(d.binPath(targetDir), 0755) //nolint:gosec
}

func (d *downloader) move(targetDir string) error {
	if d.MoveFrom == "" {
		return nil
	}
	err := rm(d.binPath(targetDir))
	if err != nil {
		return err
	}
	from := filepath.Join(targetDir, filepath.FromSlash(d.MoveFrom))
	to := d.binPath(targetDir)
	return os.Rename(from, to)
}

func (d *downloader) link(targetDir string) error {
	if d.LinkSource == "" {
		return nil
	}
	if fileExists(d.binPath(targetDir)) {
		err := rm(d.binPath(targetDir))
		if err != nil {
			return err
		}
	}
	return os.Symlink(filepath.FromSlash(d.LinkSource), d.binPath(targetDir))
}

func (d *downloader) isTar() bool {
	return strings.HasSuffix(d.URL, ".tar.gz")
}

func (d *downloader) extract(targetDir string) error {
	if !d.isTar() {
		return nil
	}
	tarPath := filepath.Join(targetDir, d.downloadableName())
	cmd := exec.Command("tar", "-C", targetDir, "-xzf", tarPath) //nolint:gosec
	err := cmd.Run()
	if err != nil {
		return err
	}
	return rm(tarPath)
}

func (d *downloader) download(targetDir string) error {
	return downloadFile(d.downloadablePath(targetDir), d.URL)
}

func (d *downloader) validateChecksum(targetDir string) error {
	targetFile := d.downloadablePath(targetDir)
	file, err := os.Open(targetFile) //nolint:gosec
	if err != nil {
		return err
	}
	defer logCloseErr(file)
	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return err
	}
	result := hex.EncodeToString(hash.Sum(nil))
	if d.Checksum != result {
		defer func() {
			delErr := rm(targetFile)
			if delErr != nil {
				log.Printf("Error deleting suspicious file at %q. Please delete it manually", targetFile)
			}
		}()
		return fmt.Errorf(`checksum mismatch in downloaded file %q 
wanted: %s
got: %s`, targetFile, d.Checksum, result)
	}
	return nil
}

func (d *downloader) install(targetDir string, force bool) error {
	if fileExists(d.binPath(targetDir)) && !force {
		return nil
	}
	var err error
	for _, fn := range []func(string) error{d.download, d.validateChecksum, d.extract, d.link, d.move, d.chmod} {
		err = fn(targetDir)
		if err != nil {
			break
		}
	}
	return err
}
