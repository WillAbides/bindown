package cache

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/rogpeppe/go-internal/lockedfile"
)

type (
	populateFunc func(string) error
	validateFunc func(fs.FS) error
)

type Cache struct {
	Root string
}

// Dir returns a fs.FS for the given key, populating the cache if necessary.
// The returned fs.FS is valid until unlock is called. After that the contents may change unexpectedly.
func (c *Cache) Dir(key string, validate validateFunc, populate populateFunc) (_ fs.FS, unlock func() error, _ error) {
	var err error
	key, err = parseKey(key)
	if err != nil {
		return nil, nil, err
	}
	lock, err := c.rLock(key)
	if err != nil {
		return nil, nil, err
	}
	dir := filepath.Join(c.Root, key)
	fsDir := os.DirFS(dir)
	validateErr := validateDir(dir, validate)
	if validateErr == nil {
		return fsDir, lock.Close, nil
	}
	if populate == nil {
		return nil, nil, errors.Join(validateErr, lock.Close())
	}
	err = lock.Close()
	if err != nil {
		return nil, nil, err
	}
	err = c.populate(key, validate, populate)
	if err != nil {
		return nil, nil, err
	}
	lock, err = c.rLock(key)
	if err != nil {
		return nil, nil, err
	}
	err = validateDir(dir, validate)
	if err != nil {
		return nil, nil, errors.Join(err, lock.Close())
	}
	return fsDir, lock.Close, nil
}

// Evict removes acquires a write lock and removes the cache entry for the given key.
func (c *Cache) Evict(key string) (errOut error) {
	var err error
	key, err = parseKey(key)
	if err != nil {
		return err
	}
	lock, err := c.lock(key)
	if err != nil {
		return err
	}
	defer func() {
		errOut = errors.Join(errOut, lock.Close())
	}()
	dir := filepath.Join(c.Root, key)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return errors.New("not a directory")
	}
	err = os.RemoveAll(dir)
	if err != nil {
		return err
	}
	return os.Remove(c.lockfile(key))
}

func (c *Cache) lockfile(key string) string {
	return filepath.Join(c.locksDir(), key)
}

func (c *Cache) locksDir() string {
	return filepath.Join(c.Root, ".locks")
}

func (c *Cache) rLockRoot() (io.Closer, error) {
	return acquireRLock(c.lockfile(".root"))
}

func (c *Cache) lockRoot() (io.Closer, error) {
	return acquireLock(c.lockfile(".root"))
}

func (c *Cache) lock(key string) (io.Closer, error) {
	rootLock, err := c.rLockRoot()
	if err != nil {
		return nil, err
	}
	file, err := lockedfile.Create(c.lockfile(key))
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(c.Root, key)
	err = unsealDir(dir)
	if err != nil {
		return nil, err
	}
	return &writeLock{
		rootLock: rootLock,
		lock:     file,
		dir:      dir,
	}, nil
}

func (c *Cache) rLock(key string) (io.Closer, error) {
	rootLock, err := c.rLockRoot()
	if err != nil {
		return nil, err
	}
	rLock, err := acquireRLock(c.lockfile(key))
	if err != nil {
		return nil, err
	}
	return &readLock{
		rootLock: rootLock,
		lock:     rLock,
	}, nil
}

func acquireRLock(lockfile string) (io.Closer, error) {
	var rLock io.Closer
	for i := 0; i < 8; i++ {
		err := os.MkdirAll(filepath.Dir(lockfile), 0o777)
		if err != nil {
			return nil, err
		}
		rLock, err = lockedfile.Open(lockfile)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		rLock = nil
		var wl io.Closer
		wl, err = lockedfile.Create(lockfile)
		if err != nil {
			return nil, err
		}
		err = wl.Close()
		if err != nil {
			return nil, err
		}
	}
	if rLock == nil {
		return nil, errors.New("failed to acquire lock")
	}
	return rLock, nil
}

func acquireLock(lockfile string) (io.Closer, error) {
	err := os.MkdirAll(filepath.Dir(lockfile), 0o777)
	if err != nil {
		return nil, err
	}
	return lockedfile.Create(lockfile)
}

func (c *Cache) populate(key string, validate validateFunc, populate populateFunc) (errOut error) {
	lock, err := c.lock(key)
	if err != nil {
		return err
	}
	defer func() {
		errOut = errors.Join(errOut, lock.Close())
	}()
	dir := filepath.Join(c.Root, key)
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o777)
		if err != nil {
			return err
		}
		return populate(dir)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("not a directory")
	}
	if validateDir(dir, validate) == nil {
		return nil
	}
	err = os.RemoveAll(dir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(dir, 0o777)
	if err != nil {
		return err
	}
	return populate(dir)
}

// RemoveRoot removes a cache root and all of its contents. This is the nuclear option.
func RemoveRoot(root string) (errOut error) {
	c := &Cache{Root: root}
	rootLock, err := c.lockRoot()
	if err != nil {
		return err
	}
	defer func() {
		closeErr := rootLock.Close()
		if errOut == nil {
			errOut = closeErr
		}
	}()
	err = unsealDir(root)
	if err != nil {
		return err
	}
	return os.RemoveAll(root)
}

type writeLock struct {
	rootLock io.Closer
	lock     io.Closer
	dir      string
}

func (l *writeLock) Close() (errOut error) {
	sealDir(l.dir)
	return errors.Join(l.lock.Close(), l.rootLock.Close())
}

type readLock struct {
	rootLock io.Closer
	lock     io.Closer
}

func (l *readLock) Close() (errOut error) {
	return errors.Join(l.lock.Close(), l.rootLock.Close())
}

func validateDir(dir string, validate validateFunc) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("entry does not exist")
		}
		return err
	}
	if !info.IsDir() {
		return errors.New("not a directory")
	}
	if validate == nil {
		return nil
	}
	return validate(os.DirFS(dir))
}

func parseKey(key string) (string, error) {
	key = filepath.FromSlash(key)
	// key must be a valid file name without path separators
	if key != filepath.Base(key) {
		return "", errors.New("invalid key")
	}
	// reserve dot files for internal use
	if strings.HasPrefix(key, ".") {
		return "", errors.New("invalid key")
	}
	return key, nil
}

// sealDir removes the write permission from a directory and all its contents.
// This is best-effort, and will not fail if the permissions cannot be changed.
//
//nolint:errcheck // this is best-effort
func sealDir(dir string) {
	var files []string
	_ = filepath.WalkDir(dir, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		files = append(files, path)
		return nil
	})
	// go backwards to remove write permissions from subdirectories first
	for i := len(files) - 1; i >= 0; i-- {
		f := files[i]
		stat, err := os.Lstat(f)
		if err != nil {
			continue
		}
		if stat.Mode()&0o222 == 0 {
			continue
		}
		_ = os.Chmod(f, stat.Mode()&^0o222)
	}
}

func unsealDir(dir string) error {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.Chmod(path, info.Mode()|0o222)
	})
}
