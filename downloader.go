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
	"github.com/willabides/bindown/v2/internal/util"
	"gopkg.in/yaml.v2"
)

// Downloader downloads a binary
type Downloader struct {
	OS          string            `json:"os"`
	Arch        string            `json:"arch"`
	URL         string            `json:"url"`
	ArchivePath string            `json:"archive_path,omitempty" yaml:"archive_path,omitempty"`
	BinName     string            `json:"bin,omitempty" yaml:"bin,omitempty"`
	Link        bool              `json:"link,omitempty" yaml:",omitempty"`
	Checksum    string            `json:"checksum,omitempty" yaml:",omitempty"`
	Vars        map[string]string `json:"vars,omitempty" yaml:"vars,omitempty"`

	//set to true by applyTemplates
	tmplApplied bool
}

func (d *Downloader) clone() *Downloader {
	c := new(Downloader)
	*c = *d
	if c.Vars != nil {
		c.Vars = util.CopyStringMap(c.Vars)
	}
	return c
}

func (d *Downloader) requireApplyTemplates() {
	if !d.tmplApplied {
		panic("templates not applied")
	}
}

func (d *Downloader) applyTemplates() error {
	executeTemplate := func(tmpl string) (string, error) {
		return util.ExecuteTemplate(tmpl, d.OS, d.Arch, d.Vars)
	}
	var err error
	d.URL, err = executeTemplate(d.URL)
	if err != nil {
		return err
	}
	d.ArchivePath, err = executeTemplate(d.ArchivePath)
	if err != nil {
		return err
	}
	d.BinName, err = executeTemplate(d.BinName)
	if err != nil {
		return err
	}
	d.tmplApplied = true
	return nil
}

func (d *Downloader) downloadableName() (string, error) {
	d.requireApplyTemplates()
	u, err := url.Parse(d.URL)
	if err != nil {
		return "", err
	}
	return path.Base(u.Path), nil
}

func (d *Downloader) downloadablePath(targetDir string) (string, error) {
	d.requireApplyTemplates()
	name, err := d.downloadableName()
	if err != nil {
		return "", err
	}
	return filepath.Join(targetDir, name), nil
}

func (d *Downloader) binPath(targetDir string) string {
	d.requireApplyTemplates()
	return filepath.Join(targetDir, d.BinName)
}

func (d *Downloader) chmod(targetDir string) error {
	return os.Chmod(d.binPath(targetDir), 0755) //nolint:gosec
}

func (d *Downloader) moveOrLinkBin(targetDir, extractDir string) error {
	d.requireApplyTemplates()
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
	d.requireApplyTemplates()
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
	d.requireApplyTemplates()
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

func (d *Downloader) validateChecksum(targetDir string, knownChecksums map[string]string) error {
	d.requireApplyTemplates()
	dl := d.clone()
	u, err := dl.url()
	if err != nil {
		return err
	}
	if knownChecksums != nil && dl.Checksum == "" {
		dl.Checksum = knownChecksums[u]
	}
	targetFile, err := dl.downloadablePath(targetDir)
	if err != nil {
		return err
	}
	result, err := fileChecksum(targetFile)
	if err != nil {
		return err
	}
	if dl.Checksum != result {
		defer func() {
			delErr := os.RemoveAll(targetFile)
			if delErr != nil {
				log.Printf("Error deleting suspicious file at %q. Please delete it manually", targetFile)
			}
		}()
		return fmt.Errorf(`checksum mismatch in downloaded file %q 
wanted: %s
got: %s`, targetFile, dl.Checksum, result)
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
	sum, err := d.getUpdatedChecksum(opts)
	if err != nil {
		return err
	}
	d.Checksum = sum
	return nil
}

func (d *Downloader) url() (string, error) {
	dl := d.clone()
	err := dl.applyTemplates()
	if err != nil {
		return "", err
	}
	return dl.URL, nil
}

//getUpdatedChecksum downloads the archive and returns its actual checksum.
func (d *Downloader) getUpdatedChecksum(opts UpdateChecksumOpts) (string, error) {
	dl := d.clone()
	err := dl.applyTemplates()
	if err != nil {
		return "", err
	}
	cellarDir := opts.CellarDir
	if cellarDir == "" {
		cellarDir, err = ioutil.TempDir("", "bindown")
		if err != nil {
			return "", err
		}
		defer func() {
			_ = os.RemoveAll(cellarDir) //nolint:errcheck
		}()
	}

	downloadDir := filepath.Join(cellarDir, "downloads", dl.downloadsSubName())

	err = dl.download(downloadDir)
	if err != nil {
		log.Printf("error downloading: %v", err)
		return "", err
	}

	dlPath, err := dl.downloadablePath(downloadDir)
	if err != nil {
		return "", err
	}

	return fileChecksum(dlPath)
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
	// Map of known checksums to validate against
	URLChecksums map[string]string
}

func (d *Downloader) downloadsSubName() string {
	return mustHexHash(fnv.New64a(), []byte(d.Checksum))
}

func (d *Downloader) extractsSubName() string {
	return mustHexHash(fnv.New64a(), []byte(d.Checksum), []byte(d.BinName))
}

//Install downloads and installs a bin
func (d *Downloader) Install(opts InstallOpts) error {
	err := d.applyTemplates()
	if err != nil {
		return err
	}
	d.setDefaultBinName(opts.DownloaderName)
	cellarDir := opts.CellarDir
	if cellarDir == "" {
		cellarDir = filepath.Join(opts.TargetDir, ".bindown")
	}

	downloadDir := filepath.Join(cellarDir, "downloads", d.downloadsSubName())
	extractDir := filepath.Join(cellarDir, "extracts", d.extractsSubName())

	if opts.Force {
		err = os.RemoveAll(downloadDir)
		if err != nil {
			return err
		}
	}

	err = d.download(downloadDir)
	if err != nil {
		log.Printf("error downloading: %v", err)
		return err
	}

	err = d.validateChecksum(downloadDir, opts.URLChecksums)
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
	err := d.applyTemplates()
	if err != nil {
		return err
	}
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
