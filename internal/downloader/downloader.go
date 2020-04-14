package downloader

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
	"strings"

	"github.com/mholt/archiver/v3"
	"github.com/willabides/bindown/v3/internal/util"
	"gopkg.in/yaml.v2"
)

// Downloader downloads a binary
type Downloader struct {
	OS          string
	Arch        string
	URL         string
	ArchivePath string            `yaml:"archive_path,omitempty"`
	BinName     string            `yaml:"bin,omitempty"`
	Link        bool              `yaml:",omitempty"`
	Vars        map[string]string `yaml:"vars,omitempty"`

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
	if util.FileExists(target) {
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
	return util.CopyFile(extractedBin, target, nil)
}

func (d *Downloader) extract(downloadDir, extractDir string) error {
	d.requireApplyTemplates()
	dlName, err := d.downloadableName()
	if err != nil {
		return err
	}
	extractSumFile := filepath.Join(downloadDir, ".extractsum")
	if wantSum, sumErr := ioutil.ReadFile(extractSumFile); sumErr == nil { //nolint:gosec
		var exs string
		exs, sumErr = util.DirectoryChecksum(extractDir)
		if sumErr == nil && exs == strings.TrimSpace(string(wantSum)) {
			return nil
		}
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
	err = archiver.Unarchive(tarPath, extractDir)
	if err != nil {
		return err
	}
	extractSum, err := util.DirectoryChecksum(extractDir)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(extractSumFile, []byte(extractSum), 0640)
}

func (d *Downloader) download(downloadDir, wantChecksum string) error {
	d.requireApplyTemplates()
	dlPath, err := d.downloadablePath(downloadDir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(downloadDir, 0750)
	if err != nil {
		return err
	}
	ok, err := util.FileExistsWithChecksum(dlPath, wantChecksum)
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

func validateFileChecksum(filename, checksum string) error {
	result, err := util.FileChecksum(filename)
	if err != nil {
		return err
	}
	if checksum != result {
		defer func() {
			delErr := os.RemoveAll(filename)
			if delErr != nil {
				log.Printf("Error deleting suspicious file at %q. Please delete it manually", filename)
			}
		}()
		return fmt.Errorf(`checksum mismatch in downloaded file %q 
wanted: %s
got: %s`, filename, checksum, result)
	}
	return nil
}

func (d *Downloader) validateChecksum(targetDir, checksum string) error {
	d.requireApplyTemplates()
	dl := d.clone()
	targetFile, err := dl.downloadablePath(targetDir)
	if err != nil {
		return err
	}
	return validateFileChecksum(targetFile, checksum)
}

//UpdateChecksumOpts options for UpdateChecksum
type UpdateChecksumOpts struct {
	// CellarDir is the directory where downloads and extractions will be placed.  Default is a temp directory.
	CellarDir    string
	URLChecksums map[string]string
}

//GetURL returns the downloaders url after applying templates
func (d *Downloader) GetURL() (string, error) {
	dl := d.clone()
	err := dl.applyTemplates()
	if err != nil {
		return "", err
	}
	return dl.URL, nil
}

//GetUpdatedChecksum downloads the archive and returns its actual checksum.
func (d *Downloader) GetUpdatedChecksum(opts UpdateChecksumOpts) (string, error) {
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

	downloadDir := filepath.Join(cellarDir, "downloads", dl.downloadsSubName(opts.URLChecksums))

	err = dl.download(downloadDir, "")
	if err != nil {
		log.Printf("error downloading: %v", err)
		return "", err
	}

	dlPath, err := dl.downloadablePath(downloadDir)
	if err != nil {
		return "", err
	}

	return util.FileChecksum(dlPath)
}

func (d *Downloader) downloadsSubName(knownChecksums map[string]string) string {
	var checksum string
	u, err := d.GetURL()
	if err == nil && knownChecksums != nil {
		if knownChecksums[u] != "" {
			checksum = knownChecksums[u]
		}
	}
	return util.MustHexHash(fnv.New64a(), []byte(checksum))
}

//ExtractsSubName returns the subdirectory where this will be extracted
func (d *Downloader) ExtractsSubName(knownChecksums map[string]string) string {
	var checksum string
	u, err := d.GetURL()
	if err == nil && knownChecksums != nil {
		if knownChecksums[u] != "" {
			checksum = knownChecksums[u]
		}
	}
	return util.MustHexHash(fnv.New64a(), []byte(checksum), []byte(d.BinName))
}

//Download download the file to outputPath
func (d *Downloader) Download(outputPath, checksum string, force bool) error {
	dl := d.clone()
	err := dl.applyTemplates()
	if err != nil {
		return err
	}
	if force {
		err = os.Remove(outputPath)
		if err != nil {
			return err
		}
	}
	ok, err := util.FileExistsWithChecksum(outputPath, checksum)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	err = downloadFile(outputPath, dl.URL)
	if err != nil {
		return err
	}
	return validateFileChecksum(outputPath, checksum)
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
	// Checksum is the checksum we want this download to have
	Checksum string
}

//Install downloads and installs a bin
func (d *Downloader) Install(opts InstallOpts) error {
	dl := d.clone()
	err := dl.applyTemplates()
	if err != nil {
		return err
	}

	dl.setDefaultBinName(opts.DownloaderName)
	cellarDir := opts.CellarDir
	if cellarDir == "" {
		cellarDir = filepath.Join(opts.TargetDir, ".bindown")
	}

	downloadDir := filepath.Join(cellarDir, "downloads", util.MustHexHash(fnv.New64a(), []byte(opts.Checksum)))
	extractDir := filepath.Join(cellarDir, "extracts", util.MustHexHash(fnv.New64a(), []byte(opts.Checksum), []byte(dl.BinName)))

	if opts.Force {
		err = os.RemoveAll(downloadDir)
		if err != nil {
			return err
		}
	}

	dlPath, err := dl.downloadablePath(downloadDir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(downloadDir, 0750)
	if err != nil {
		return err
	}
	err = dl.Download(dlPath, opts.Checksum, false)
	if err != nil {
		return err
	}

	err = dl.extract(downloadDir, extractDir)
	if err != nil {
		log.Printf("error extracting: %v", err)
		return err
	}

	err = dl.moveOrLinkBin(opts.TargetDir, extractDir)
	if err != nil {
		log.Printf("error moving: %v", err)
		return err
	}

	err = dl.chmod(opts.TargetDir)
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
	// Checksum is the checksum we want this download to have
	Checksum string
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
		Checksum:       opts.Checksum,
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

func logCloseErr(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Println(err)
	}
}
