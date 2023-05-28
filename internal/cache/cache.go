package cache

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rogpeppe/go-internal/lockedfile"
)

type (
	populateFunc func(string) error
	validateFunc func(string) error
)

type Cache struct {
	Root string
	// Set to true to make the content read-only on disk.
	// TODO: find a way to use this without the annoying side effect of requiring sudo to rm -rf the cache.
	ReadOnly bool
}

// Dir returns a fs.FS for the given key, populating the cache if necessary.
// The returned fs.FS is valid until unlock is called. After that the contents may change unexpectedly.
func (c *Cache) Dir(key string, validate validateFunc, populate populateFunc) (_ string, unlock func() error, _ error) {
	var err error
	key, err = parseKey(key)
	if err != nil {
		return "", nil, err
	}
	lock, err := c.rLock(key)
	if err != nil {
		return "", nil, err
	}
	dir := filepath.Join(c.Root, key)
	validateErr := validateDir(dir, validate)
	if validateErr == nil {
		return dir, lock.Close, nil
	}
	if populate == nil {
		return "", nil, errors.Join(validateErr, lock.Close())
	}
	err = lock.Close()
	if err != nil {
		return "", nil, err
	}
	err = c.populate(key, validate, populate)
	if err != nil {
		return "", nil, err
	}
	lock, err = c.rLock(key)
	if err != nil {
		return "", nil, err
	}
	err = validateDir(dir, validate)
	if err != nil {
		return "", nil, errors.Join(err, lock.Close())
	}
	return dir, lock.Close, nil
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
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer func() {
		closeErr := lock.Close()
		if closeErr != nil {
			errOut = errors.Join(errOut, fmt.Errorf("failed to close lock: %w", closeErr))
		}
	}()
	dir := filepath.Join(c.Root, key)
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to stat dir: %w", err)
	}
	if !info.IsDir() {
		return errors.New("not a directory")
	}
	err = os.RemoveAll(dir)
	if err != nil {
		return fmt.Errorf("failed to remove dir: %w", err)
	}
	err = os.Remove(c.lockfile(key))
	if err != nil {
		return fmt.Errorf("failed to remove lock: %w", err)
	}
	return nil
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
	if c.ReadOnly {
		err = unsealDir(dir)
		if err != nil {
			return nil, err
		}
	}
	return &writeLock{
		rootLock: rootLock,
		lock:     file,
		dir:      dir,
		readOnly: c.ReadOnly,
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
		return fmt.Errorf("failed to lock root: %w", err)
	}
	if c.ReadOnly {
		err = unsealDir(root)
		if err != nil {
			closeErr := rootLock.Close()
			if closeErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to unlock root: %w", closeErr))
			}
			return fmt.Errorf("failed to unseal root: %w", err)
		}
	}
	// Unlock root early to get around a Windows issue where you can't delete a directory that's locked.
	err = rootLock.Close()
	if err != nil {
		return fmt.Errorf("failed to unlock root: %w", err)
	}
	err = os.RemoveAll(root)
	if err != nil {
		return fmt.Errorf("failed to remove root: %w", err)
	}
	return nil
}

type writeLock struct {
	rootLock io.Closer
	lock     io.Closer
	dir      string
	readOnly bool
}

func (l *writeLock) Close() (errOut error) {
	if l.readOnly {
		sealDir(l.dir)
	}
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
	return validate(dir)
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
	_ = filepath.WalkDir(dir, func(path string, _ os.DirEntry, err error) error {
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
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
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
