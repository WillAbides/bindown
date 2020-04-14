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
)

// Downloader downloads a binary
type Downloader struct {
	OS          string
	Arch        string
	URL         string
	ArchivePath string
	BinName     string
	Link        bool
	Vars        map[string]string

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

func (d *Downloader) moveOrLinkBin(target, extractDir string) error {
	d.requireApplyTemplates()
	archivePath := filepath.FromSlash(d.ArchivePath)
	if archivePath == "" {
		archivePath = filepath.FromSlash(d.BinName)
	}
	var err error
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
		var linkTargetDir string
		linkTargetDir, err = filepath.Abs(filepath.Dir(target))
		if err != nil {
			return err
		}

		linkTargetDir, err = filepath.EvalSymlinks(linkTargetDir)
		if err != nil {
			return err
		}

		extractedBin, err = filepath.EvalSymlinks(extractedBin)
		if err != nil {
			return err
		}

		var dst string
		dst, err = filepath.Rel(linkTargetDir, extractedBin)
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

//Extract extracts a downloaded file to extractDir
func (d *Downloader) Extract(downloadDir, extractDir string) error {
	dl := d.clone()
	err := dl.applyTemplates()
	if err != nil {
		return err
	}
	dlName, err := dl.downloadableName()
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
	// Cache is the directory where downloads and extractions will be placed.  Default is a temp directory.
	Cache        string
	URLChecksums map[string]string
}

//GetBinName returns the downloader's bin name after applying templates
func (d *Downloader) GetBinName() (string, error) {
	dl := d.clone()
	err := dl.applyTemplates()
	if err != nil {
		return "", err
	}
	return dl.BinName, nil
}

//GetURL returns the downloader/s url after applying templates
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
	cache := opts.Cache
	if cache == "" {
		cache, err = ioutil.TempDir("", "bindown")
		if err != nil {
			return "", err
		}
		defer func() {
			_ = os.RemoveAll(cache) //nolint:errcheck
		}()
	}

	downloadDir := filepath.Join(cache, "downloads", dl.downloadsSubName(opts.URLChecksums))

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

//DownloadsCacheDir returns a cache directory for this downloading downloader
func (d *Downloader) DownloadsCacheDir(cache string, knownChecksums map[string]string) string {
	var checksum string
	u, err := d.GetURL()
	if err == nil && knownChecksums != nil {
		if knownChecksums[u] != "" {
			checksum = knownChecksums[u]
		}
	}
	return downloadCacheDir(cache, checksum)
}

//ExtractsCacheDir returns a cache directory for this extracting downloader
func (d *Downloader) ExtractsCacheDir(cache string, knownChecksums map[string]string) string {
	var checksum string
	u, err := d.GetURL()
	if err == nil && knownChecksums != nil {
		if knownChecksums[u] != "" {
			checksum = knownChecksums[u]
		}
	}
	return extractCacheDir(cache, checksum, d.BinName)
}

//Download download the file to outputPath
func (d *Downloader) Download(outputPath, checksum string, force bool) error {
	dl := d.clone()
	err := dl.applyTemplates()
	if err != nil {
		return err
	}
	if force {
		err = os.RemoveAll(outputPath)
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
	err = os.MkdirAll(filepath.Dir(outputPath), 0750)
	if err != nil {
		return err
	}
	err = downloadFile(outputPath, dl.URL)
	if err != nil {
		return err
	}
	return validateFileChecksum(outputPath, checksum)
}

func downloadCacheDir(cache, checksum string) string {
	return filepath.Join(cache, "downloads", util.MustHexHash(fnv.New64a(), []byte(checksum)))
}

func extractCacheDir(cache, checksum, binName string) string {
	return filepath.Join(cache, "extracts", util.MustHexHash(fnv.New64a(), []byte(checksum), []byte(binName)))
}

//Install installs a bin
func (d *Downloader) Install(targetPath, extractDir string) error {
	dl := d.clone()
	err := dl.applyTemplates()
	if err != nil {
		return err
	}
	err = dl.moveOrLinkBin(targetPath, extractDir)
	if err != nil {
		return err
	}
	return os.Chmod(targetPath, 0750) //nolint:gosec
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
