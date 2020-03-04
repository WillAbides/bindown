package bindown

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func logCloseErr(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Println(err)
	}
}

// mustHexHash is like hexHash but panics on err
// this should only be used with hashers that are guaranteed to return a nil error from Write()
func mustHexHash(hasher hash.Hash, data ...[]byte) string {
	hsh, err := hexHash(hasher, data...)
	if err != nil {
		panic(err)
	}
	return hsh
}

// hexHash returns a hex representation of data's hash
// This will only return non-nil error if given a hasher that can return a non-nil error from Write()
func hexHash(hasher hash.Hash, data ...[]byte) (string, error) {
	hasher.Reset()
	for _, datum := range data {
		_, err := hasher.Write(datum)
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// fileChecksum returns the hex checksum of a file
func fileChecksum(filename string) (string, error) {
	fileBytes, err := ioutil.ReadFile(filename) //nolint:gosec
	if err != nil {
		return "", err
	}
	return hexHash(sha256.New(), fileBytes)
}

//fileExistsWithChecksum returns true if the file both exists and has a matching checksum
func fileExistsWithChecksum(filename, checksum string) (bool, error) {
	if !fileExists(filename) {
		return false, nil
	}
	got, err := fileChecksum(filename)
	if err != nil {
		return false, err
	}
	return checksum == got, nil
}

//fileExists asserts that a file exists
func fileExists(path string) bool {
	if _, err := os.Stat(filepath.FromSlash(path)); !os.IsNotExist(err) {
		return true
	}
	return false
}

func copyFile(src, dst string) error {
	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !srcStat.Mode().IsRegular() {
		return fmt.Errorf("not a regular file")
	}

	rdr, err := os.Open(src) //nolint:gosec
	if err != nil {
		return err
	}
	defer logCloseErr(rdr)

	writer, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcStat.Mode())
	if err != nil {
		return err
	}
	defer logCloseErr(writer)

	_, err = io.Copy(writer, rdr)
	return err
}
