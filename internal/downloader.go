package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/mholt/archiver"
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

func (d *downloader) extract(targetDir string) error {
	tarPath := filepath.Join(targetDir, d.downloadableName())
	_, err := archiver.ByExtension(d.downloadableName())
	if err != nil {
		return nil
	}
	err = archiver.Unarchive(tarPath, targetDir)
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
	err := d.download(targetDir)
	if err != nil {
		log.Printf("error downloading: %v", err)
		return err
	}

	err = d.validateChecksum(targetDir)
	if err != nil {
		log.Printf("error validating: %v", err)
		return err
	}

	err = d.extract(targetDir)
	if err != nil {
		log.Printf("error extracting: %v", err)
		return err
	}

	err = d.link(targetDir)
	if err != nil {
		log.Printf("error linking: %v", err)
		return err
	}

	err = d.move(targetDir)
	if err != nil {
		log.Printf("error moving: %v", err)
		return err
	}

	err = d.chmod(targetDir)
	if err != nil {
		log.Printf("error chmodding: %v", err)
		return err
	}

	return nil
}
