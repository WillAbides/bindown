package builddep

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/mholt/archiver/v4"
	"github.com/willabides/bindown/v3/internal/bindown"
	"golang.org/x/exp/slices"
)

type dlFile struct {
	origUrl      string
	url          string
	osSub        *systemSub
	archSub      *systemSub
	suffix       string
	isArchive    bool
	priority     int
	archiveFiles []*archiveFile
	checksum     string
}

func (f *dlFile) clone() *dlFile {
	clone := *f
	clone.archiveFiles = slices.Clone(f.archiveFiles)
	for i, file := range f.archiveFiles {
		cf := *file
		clone.archiveFiles[i] = &cf
	}
	osSub := *f.osSub
	clone.osSub = &osSub
	archSub := *f.archSub
	clone.archSub = &archSub
	return &clone
}

func (f *dlFile) setArchiveFiles(ctx context.Context, binName, version string) error {
	if !f.isArchive {
		return nil
	}
	parsedUrl, err := url.Parse(f.origUrl)
	if err != nil {
		return err
	}
	filename := path.Base(parsedUrl.EscapedPath())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.origUrl, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		//nolint:errcheck // ignore error
		_ = resp.Body.Close()
	}()
	hasher := sha256.New()
	reader := io.TeeReader(resp.Body, hasher)
	format, reader, err := archiver.Identify(filename, reader)
	if err != nil {
		if errors.Is(err, archiver.ErrNoMatch) {
			err = fmt.Errorf("unable to identify archive format for %s", filename)
		}
		return err
	}
	// reader needs to be an io.ReaderAt and io.Seeker for zip
	_, isZip := format.(archiver.Zip)
	if isZip {
		var b []byte
		b, err = io.ReadAll(reader)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	extractor, ok := format.(archiver.Extractor)
	if !ok {
		return errors.New("format does not support extraction")
	}
	err = extractor.Extract(ctx, reader, nil, func(_ context.Context, af archiver.File) error {
		if af.IsDir() {
			return nil
		}
		executable := af.Mode().Perm()&0o100 != 0
		if !executable && f.osSub.normalized == "windows" {
			executable = strings.HasSuffix(af.Name(), ".exe")
		}
		f.archiveFiles = append(f.archiveFiles, parseArchiveFile(af.NameInArchive, binName, f.osSub.val, f.archSub.val, version, executable))
		return nil
	})
	if err != nil {
		return err
	}
	slices.SortFunc(f.archiveFiles, archiveFileLess)
	// read remaining bytes to calculate hash
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return err
	}
	f.checksum = hex.EncodeToString(hasher.Sum(nil))
	return err
}

func (f *dlFile) system() bindown.System {
	if f.osSub == nil || f.archSub == nil {
		panic("system called on dlFile without osSub or archSub")
	}
	return bindown.System(f.osSub.normalized + "/" + f.archSub.normalized)
}
