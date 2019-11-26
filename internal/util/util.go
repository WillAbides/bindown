package util

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

//Rm removes a file and filters out IsNotExist errors
func Rm(path string) error {
	err := os.RemoveAll(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return fmt.Errorf(`failed to remove %s: %v`, path, err)
}

//LogCloseErr logs errors closing closers.  Useful in defer statements.
func LogCloseErr(closer io.Closer) {
	err := closer.Close()
	if err != nil {
		log.Println(err)
	}
}

// MustHexHash is like hexHash but panics on err
// this should only be used with hashers that are guaranteed to return a nil error from Write()
func MustHexHash(hasher hash.Hash, data ...[]byte) string {
	hsh, err := HexHash(hasher, data...)
	if err != nil {
		panic(err)
	}
	return hsh
}

// HexHash returns a hex representation of data's hash
// This will only return non-nil error if given a hasher that can return a non-nil error from Write()
func HexHash(hasher hash.Hash, data ...[]byte) (string, error) {
	hasher.Reset()
	for _, datum := range data {
		_, err := hasher.Write(datum)
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// FileChecksum returns the hex checksum of a file
func FileChecksum(filename string) (string, error) {
	fileBytes, err := ioutil.ReadFile(filename) //nolint:gosec
	if err != nil {
		return "", err
	}
	return MustHexHash(sha256.New(), fileBytes), nil
}

//FileExistsWithChecksum returns true if the file both exists and has a matching checksum
func FileExistsWithChecksum(filename, checksum string) (bool, error) {
	if !FileExists(filename) {
		return false, nil
	}
	got, err := FileChecksum(filename)
	if err != nil {
		return false, err
	}
	return checksum == got, nil
}

//FileExists asserts that a file exists
func FileExists(path string) bool {
	if _, err := os.Stat(filepath.FromSlash(path)); !os.IsNotExist(err) {
		return true
	}
	return false
}

//CopyFile copys a file from src to dst
func CopyFile(src, dst string) error {
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
	defer LogCloseErr(rdr)

	writer, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, srcStat.Mode())
	if err != nil {
		return err
	}
	defer LogCloseErr(writer)

	_, err = io.Copy(writer, rdr)
	return err
}
