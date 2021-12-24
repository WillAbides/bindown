package util

import (
	"fmt"
	"io"
	"os"
)

// CopyFile copies file from src to dst
func CopyFile(src, dst string, closeCloser func(io.Closer)) error {
	if closeCloser == nil {
		closeCloser = func(_ io.Closer) {}
	}
	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcStat.Mode().IsRegular() {
		return fmt.Errorf("not a regular file")
	}

	rdr, err := os.Open(src)
	if err != nil {
		return err
	}
	defer closeCloser(rdr)

	writer, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcStat.Mode())
	if err != nil {
		return err
	}
	defer closeCloser(writer)

	_, err = io.Copy(writer, rdr)
	return err
}
