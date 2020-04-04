package bindown

import (
	"fmt"
	"hash/fnv"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/mholt/archiver/v3"
	"github.com/willabides/bindown/v3/internal/util"
	"gopkg.in/yaml.v2"
)

// Downloader downloads a binary
type Downloader struct {
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	URL         string `json:"url"`
	Checksum    string `json:"checksum,omitempty" yaml:",omitempty"`
	ArchivePath string `json:"archive_path,omitempty" yaml:"archive_path,omitempty"`
	Link        bool   `json:"link,omitempty" yaml:",omitempty"`
	BinName     string `json:"bin,omitempty" yaml:"bin,omitempty"`

	// Deprecated: use ArchivePath
	MoveFrom string `json:"move-from,omitempty" yaml:"move-from,omitempty"`

	// Deprecated: use ArchivePath and Link
	LinkSource string `json:"symlink,omitempty" yaml:"symlink,omitempty"`
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

func (d *Downloader) moveOrLinkBin(targetDir, extractDir string) error {
	//noinspection GoDeprecation
	if d.LinkSource != "" {
		d.ArchivePath = d.LinkSource
		d.Link = true
	}
	//noinspection GoDeprecation
	if d.MoveFrom != "" {
		d.ArchivePath = d.MoveFrom
	}
	archivePath := filepath.FromSlash(d.ArchivePath)
	if archivePath == "" {
		archivePath = filepath.FromSlash(d.BinName)
	}
	var err error
	target := d.binPath(targetDir)
	if fileExists(target) {
		err = os.RemoveAll(target)
		if err != nil {
			return err
		}
	}

	extractDir, err = filepath.Abs(extractDir)
	if err != nil {
		return err
	}
	extractedBin := filepath.Join(extractDir, archivePath)

	if d.Link {
		targetDir, err = filepath.Abs(filepath.Dir(target))
		if err != nil {
			return err
		}

		targetDir, err = filepath.EvalSymlinks(targetDir)
		if err != nil {
			return err
		}

		extractedBin, err = filepath.EvalSymlinks(extractedBin)
		if err != nil {
			return err
		}

		var dst string
		dst, err = filepath.Rel(targetDir, extractedBin)
		if err != nil {
			return err
		}
		return os.Symlink(dst, target)
	}
	err = os.MkdirAll(filepath.Dir(target), 0750)
	if err != nil {
		return err
	}
	return os.Rename(extractedBin, target)
}

func (d *Downloader) extract(downloadDir, extractDir string) error {
	dlName, err := d.downloadableName()
	if err != nil {
		return err
	}
	err = os.RemoveAll(extractDir)
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
		return util.CopyFile(tarPath, filepath.Join(extractDir, dlName), logCloseErr)
	}
	return archiver.Unarchive(tarPath, extractDir)
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

func (d *Downloader) setDefaultBinName(defaultName string) {
	if d.BinName == "" {
		d.BinName = defaultName
	}
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
			delErr := os.RemoveAll(targetFile)
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

//UpdateChecksumOpts options for UpdateChecksum
type UpdateChecksumOpts struct {
	// CellarDir is the directory where downloads and extractions will be placed.  Default is a temp directory.
	CellarDir string
}

//UpdateChecksum updates the checksum based on a fresh download
func (d *Downloader) UpdateChecksum(opts UpdateChecksumOpts) error {
	cellarDir := opts.CellarDir
	var err error
	if cellarDir == "" {
		cellarDir, err = ioutil.TempDir("", "bindown")
		if err != nil {
			return err
		}
		defer func() {
			_ = os.RemoveAll(cellarDir) //nolint:errcheck
		}()
	}

	downloadDir := filepath.Join(cellarDir, "downloads", d.downloadsSubName())

	err = d.download(downloadDir)
	if err != nil {
		log.Printf("error downloading: %v", err)
		return err
	}

	dlPath, err := d.downloadablePath(downloadDir)
	if err != nil {
		return err
	}

	checkSum, err := fileChecksum(dlPath)
	if err != nil {
		return err
	}

	d.Checksum = checkSum
	return nil
}

//InstallOpts options for Install
type InstallOpts struct {
	// DownloaderName is the downloader's key from the config file
	DownloaderName string
	// CellarDir is the directory where downloads and extractions will be placed.  Default is a <TargetDir>/.bindown
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
	d.setDefaultBinName(opts.DownloaderName)
	cellarDir := opts.CellarDir
	if cellarDir == "" {
		cellarDir = filepath.Join(opts.TargetDir, ".bindown")
	}

	downloadDir := filepath.Join(cellarDir, "downloads", d.downloadsSubName())
	extractDir := filepath.Join(cellarDir, "extracts", d.extractsSubName())

	if opts.Force {
		err := os.RemoveAll(downloadDir)
		if err != nil {
			return err
		}
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

	err = d.moveOrLinkBin(opts.TargetDir, extractDir)
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

//ValidateOpts is options for Validate
type ValidateOpts struct {
	// DownloaderName is the downloader's key from the config file
	DownloaderName string
	// CellarDir is the directory where downloads and extractions will be placed.  Default is a temp directory.
	CellarDir string
}

//Validate installs the downloader to a temporary directory and returns an error if it was unsuccessful.
// If cellarDir is "", it will use a temp directory
func (d *Downloader) Validate(opts ValidateOpts) error {
	tmpDir, err := ioutil.TempDir("", "bindown")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(tmpDir) //nolint:errcheck
	}()
	binDir := filepath.Join(tmpDir, "bin")
	err = os.MkdirAll(binDir, 0700)
	if err != nil {
		return err
	}
	if opts.CellarDir == "" {
		opts.CellarDir = filepath.Join(tmpDir, "cellar")
	}

	dlYAML, err := yaml.Marshal(d)
	if err != nil {
		return err
	}

	installOpts := InstallOpts{
		DownloaderName: opts.DownloaderName,
		TargetDir:      binDir,
		Force:          true,
		CellarDir:      opts.CellarDir,
	}

	err = d.Install(installOpts)
	if err != nil {
		return fmt.Errorf("could not validate downloader:\n%s", string(dlYAML))
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
