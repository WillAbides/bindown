package bindownloader

import (
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/mholt/archiver"
)

// Downloader downloads a binary
type Downloader struct {
	URL        string `json:"url"`
	Checksum   string `json:"checksum"`
	LinkSource string `json:"symlink,omitempty"`
	BinName    string `json:"bin"`
	MoveFrom   string `json:"move-from"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
}

func (d *Downloader) downloadableName() (string, error) {
	u, err := url.Parse(d.URL)
	if err != nil {
		return "", err
	}
	return path.Base(u.Path), nil
}

func (d *Downloader) downloadablePath(targetDir string) (string, error) {
	name, err := d.downloadableName()
	if err != nil {
		return "", err
	}
	return filepath.Join(targetDir, name), nil
}

func (d *Downloader) binPath(targetDir string) string {
	return filepath.Join(targetDir, d.BinName)
}

func (d *Downloader) chmod(targetDir string) error {
	return os.Chmod(d.binPath(targetDir), 0755) //nolint:gosec
}

func (d *Downloader) move(targetDir, extractDir string) error {
	if d.MoveFrom == "" {
		return nil
	}
	err := rm(d.binPath(targetDir))
	if err != nil {
		return err
	}
	from := filepath.Join(extractDir, filepath.FromSlash(d.MoveFrom))
	to := d.binPath(targetDir)
	return os.Rename(from, to)
}

func (d *Downloader) link(targetDir, extractDir string) error {
	if d.LinkSource == "" {
		return nil
	}
	var err error
	if fileExists(d.binPath(targetDir)) {
		err = rm(d.binPath(targetDir))
		if err != nil {
			return err
		}
	}
	extractDir, err = filepath.Abs(extractDir)
	if err != nil {
		return err
	}
	target := d.binPath(targetDir)
	targetDir, err = filepath.Abs(filepath.Dir(target))
	if err != nil {
		return err
	}

	dst, err := filepath.Rel(targetDir, filepath.Join(extractDir, filepath.FromSlash(d.LinkSource)))
	if err != nil {
		return err
	}
	return os.Symlink(dst, d.binPath(targetDir))
}

func (d *Downloader) extract(downloadDir, extractDir string) error {
	dlName, err := d.downloadableName()
	if err != nil {
		return err
	}
	err = os.MkdirAll(extractDir, 0750)
	if err != nil {
		return err
	}
	tarPath := filepath.Join(downloadDir, dlName)
	_, err = archiver.ByExtension(dlName)
	if err != nil {
		return copyFile(tarPath, filepath.Join(extractDir, dlName))
	}
	err = archiver.Unarchive(tarPath, extractDir)
	if err != nil {
		return err
	}
	return rm(tarPath)
}

func (d *Downloader) download(downloadDir string) error {
	dlPath, err := d.downloadablePath(downloadDir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(downloadDir, 0750)
	if err != nil {
		return err
	}
	ok, err := fileExistsWithChecksum(dlPath, d.Checksum)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return downloadFile(dlPath, d.URL)
}

func (d *Downloader) validateChecksum(targetDir string) error {
	targetFile, err := d.downloadablePath(targetDir)
	if err != nil {
		return err
	}
	result, err := fileChecksum(targetFile)
	if err != nil {
		return err
	}
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

//InstallOpts options for Install
type InstallOpts struct {
	// CellarDir is the directory where downloads and extractions will be placed.  Default is a <TargetDir>/.bindownloader
	CellarDir string
	// TargetDir is the directory where the executable should end up
	TargetDir string
	// Force - whether to force the install even if it already exists
	Force bool
}

func (d *Downloader) downloadsSubName() string {
	return mustHexHash(fnv.New64a(), []byte(d.Checksum))
}

func (d *Downloader) extractsSubName() string {
	return mustHexHash(fnv.New64a(), []byte(d.Checksum), []byte(d.BinName))
}

//Install downloads and installs a bin
func (d *Downloader) Install(opts InstallOpts) error {
	cellarDir := opts.CellarDir
	if cellarDir == "" {
		cellarDir = filepath.Join(opts.TargetDir, ".bindownloader")
	}

	downloadDir := filepath.Join(cellarDir, "downloads", d.downloadsSubName())
	extractDir := filepath.Join(cellarDir, "extracts", d.extractsSubName())

	if fileExists(d.binPath(opts.TargetDir)) && !opts.Force {
		return nil
	}

	err := d.download(downloadDir)
	if err != nil {
		log.Printf("error downloading: %v", err)
		return err
	}

	err = d.validateChecksum(downloadDir)
	if err != nil {
		log.Printf("error validating: %v", err)
		return err
	}

	err = d.extract(downloadDir, extractDir)
	if err != nil {
		log.Printf("error extracting: %v", err)
		return err
	}

	err = d.link(opts.TargetDir, extractDir)
	if err != nil {
		log.Printf("error linking: %v", err)
		return err
	}

	err = d.move(opts.TargetDir, extractDir)
	if err != nil {
		log.Printf("error moving: %v", err)
		return err
	}

	err = d.chmod(opts.TargetDir)
	if err != nil {
		log.Printf("error chmodding: %v", err)
		return err
	}

	return nil
}

func downloadFile(targetPath, url string) error {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return err
	}
	defer logCloseErr(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed downloading %s", url)
	}
	out, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer logCloseErr(out)
	_, err = io.Copy(out, resp.Body)
	return err
}
