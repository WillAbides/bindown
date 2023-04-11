package cache

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache_Dir(t *testing.T) {
	t.Run("reads existing file", func(t *testing.T) {
		cache := testCache(t)
		testFile := filepath.Join(cache.Root, "foo", "foo.txt")
		mustWriteFile(t, testFile, "bar")
		dir, unlock, err := cache.Dir("foo", fooValidator, nil)
		require.NoError(t, err)
		assertFsFile(t, dir, "foo.txt", "bar")
		mustUnlock(t, unlock)
	})

	t.Run("reads existing file with no validator", func(t *testing.T) {
		cache := testCache(t)
		testFile := filepath.Join(cache.Root, "foo", "foo.txt")
		mustWriteFile(t, testFile, "bar")
		dir, unlock, err := cache.Dir("foo", nil, nil)
		require.NoError(t, err)
		assertFsFile(t, dir, "foo.txt", "bar")
		mustUnlock(t, unlock)
	})

	t.Run("populates cache", func(t *testing.T) {
		cache := testCache(t)
		dir, unlock, err := cache.Dir("foo", fooValidator, fooPopulator)
		require.NoError(t, err)
		assertFsFile(t, dir, "foo.txt", "bar")
		mustUnlock(t, unlock)
	})

	t.Run("re-populates cache when invalid", func(t *testing.T) {
		cache := testCache(t)
		testFile := filepath.Join(cache.Root, "foo", "foo.txt")
		mustWriteFile(t, testFile, "invalid")
		extraFile := filepath.Join(cache.Root, "foo", "extra.txt")
		mustWriteFile(t, extraFile, "extra")
		dir, unlock, err := cache.Dir("foo", fooValidator, fooPopulator)
		require.NoError(t, err)
		assertFsFile(t, dir, "foo.txt", "bar")
		assertFsFileNotExist(t, dir, "extra.txt")
		mustUnlock(t, unlock)
	})

	t.Run("errors when populator is nil on new cache", func(t *testing.T) {
		cache := testCache(t)
		_, _, err := cache.Dir("foo", fooValidator, nil)
		require.EqualError(t, err, "entry does not exist")
	})

	t.Run("errors when populator is nil on invalid cache", func(t *testing.T) {
		cache := testCache(t)
		testFile := filepath.Join(cache.Root, "foo", "foo.txt")
		mustWriteFile(t, testFile, "invalid")
		_, _, err := cache.Dir("foo", fooValidator, nil)
		require.EqualError(t, err, "invalid entry")
	})

	t.Run("errors when populated content is invalid", func(t *testing.T) {
		cache := testCache(t)
		_, _, err := cache.Dir("foo", fooValidator, func(dir string) error {
			return nil
		})
		require.EqualError(t, err, "open foo.txt: no such file or directory")
	})

	t.Run("errors when populator returns error", func(t *testing.T) {
		cache := testCache(t)
		_, _, err := cache.Dir("foo", fooValidator, func(dir string) error {
			return assert.AnError
		})
		require.EqualError(t, err, assert.AnError.Error())
	})

	t.Run("errors when dir is a file", func(t *testing.T) {
		cache := testCache(t)
		testFile := filepath.Join(cache.Root, "foo.txt")
		mustWriteFile(t, testFile, "bar")
		_, _, err := cache.Dir("foo.txt", nil, nil)
		require.EqualError(t, err, "not a directory")
	})

	t.Run("multiple read locks", func(t *testing.T) {
		cache := testCache(t)
		dir1, unlock1, err := cache.Dir("foo", fooValidator, fooPopulator)
		require.NoError(t, err)
		dir2, unlock2, err := cache.Dir("foo", fooValidator, fooPopulator)
		require.NoError(t, err)
		assertFsFile(t, dir1, "foo.txt", "bar")
		assertFsFile(t, dir2, "foo.txt", "bar")
		mustUnlock(t, unlock1)
		mustUnlock(t, unlock2)
	})

	t.Run("release then re-acquire lock", func(t *testing.T) {
		cache := testCache(t)
		dir1, unlock1, err := cache.Dir("foo", fooValidator, fooPopulator)
		require.NoError(t, err)
		assertFsFile(t, dir1, "foo.txt", "bar")
		mustUnlock(t, unlock1)
		dir2, unlock2, err := cache.Dir("foo", fooValidator, fooPopulator)
		require.NoError(t, err)
		assertFsFile(t, dir2, "foo.txt", "bar")
		mustUnlock(t, unlock2)
	})

	t.Run("invalid keys", func(t *testing.T) {
		cache := testCache(t)
		keys := []string{
			"/foo",
			"../foo",
			"foo/../bar",
			"foo/",
			"",
			".foo",
		}
		for _, key := range keys {
			t.Run(key, func(t *testing.T) {
				_, _, err := cache.Dir(key, fooValidator, fooPopulator)
				require.EqualError(t, err, "invalid key")
			})
		}
	})

	t.Run("directory is replaced by a file after read lock released", func(t *testing.T) {
		cache := testCache(t)

		testDir := filepath.Join(cache.Root, "foo")
		testFile := filepath.Join(testDir, "foo.txt")
		mustWriteFile(t, testFile, "bar")
		validate := func(dir fs.FS) error {
			err := os.RemoveAll(testDir)
			assert.NoError(t, err)
			mustWriteFile(t, testDir, "bar")
			return assert.AnError
		}
		_, _, err := cache.Dir("foo", validate, fooPopulator)
		require.EqualError(t, err, "not a directory")
	})

	t.Run("entry becomes valid after read lock released", func(t *testing.T) {
		cache := testCache(t)
		testFile := filepath.Join(cache.Root, "foo", "foo.txt")
		mustWriteFile(t, testFile, "bar")
		validateCallCount := 0
		validate := func(dir fs.FS) error {
			validateCallCount++
			if validateCallCount == 1 {
				return assert.AnError
			}
			return fooValidator(dir)
		}
		dir, unlock, err := cache.Dir("foo", validate, fooPopulator)
		require.NoError(t, err)
		assertFsFile(t, dir, "foo.txt", "bar")
		mustUnlock(t, unlock)
	})
}

func TestCache_Evict(t *testing.T) {
	t.Run("no-op for non-existent key", func(t *testing.T) {
		cache := testCache(t)
		err := cache.Evict("foo")
		require.NoError(t, err)
	})

	t.Run("evicts existing key", func(t *testing.T) {
		cache := testCache(t)
		dir, unlock, err := cache.Dir("foo", fooValidator, fooPopulator)
		require.NoError(t, err)
		assertFsFile(t, dir, "foo.txt", "bar")
		mustUnlock(t, unlock)
		err = cache.Evict("foo")
		require.NoError(t, err)
		// validate it's gone by trying to open it with no populator
		_, _, err = cache.Dir("foo", nil, nil)
		require.EqualError(t, err, "entry does not exist")
	})

	t.Run("errors when key is a file", func(t *testing.T) {
		cache := testCache(t)
		testFile := filepath.Join(cache.Root, "foo.txt")
		mustWriteFile(t, testFile, "bar")
		err := cache.Evict("foo.txt")
		require.EqualError(t, err, "not a directory")
	})

	t.Run("invalid keys", func(t *testing.T) {
		cache := testCache(t)
		keys := []string{
			"/foo",
			"../foo",
			"foo/../bar",
			"foo/",
			"",
			".foo",
		}
		for _, key := range keys {
			t.Run(key, func(t *testing.T) {
				err := cache.Evict(key)
				require.EqualError(t, err, "invalid key")
			})
		}
	})
}

var (
	fooValidator = fileValidator("foo.txt", "bar")
	fooPopulator = filePopulator("foo.txt", "bar")
)

func fileValidator(filename, want string) validateFunc {
	return func(dir fs.FS) error {
		b, err := fs.ReadFile(dir, filename)
		if err != nil {
			return err
		}
		if string(b) != want {
			return fmt.Errorf("invalid entry")
		}
		return nil
	}
}

func filePopulator(filename, content string) populateFunc {
	return func(dir string) error {
		n := filepath.Join(dir, filename)
		return os.WriteFile(n, []byte(content), 0o666)
	}
}

func assertFsFile(t testing.TB, dir fs.FS, name, content string) {
	t.Helper()
	b, err := fs.ReadFile(dir, name)
	assert.NoError(t, err)
	assert.Equal(t, content, string(b))
}

func assertFsFileNotExist(t testing.TB, dir fs.FS, name string) {
	t.Helper()
	_, err := dir.Open(name)
	assert.True(t, os.IsNotExist(err))
}

func mustWriteFile(t testing.TB, file, content string) {
	t.Helper()
	err := os.MkdirAll(filepath.Dir(file), 0o777)
	require.NoError(t, err)
	err = os.WriteFile(file, []byte(content), 0o666)
	require.NoError(t, err)
}

func mustUnlock(t testing.TB, unlock func() error) {
	t.Helper()
	require.NoError(t, unlock())
}

func testCache(t *testing.T) *Cache {
	t.Helper()
	dir := t.TempDir()
	t.Cleanup(func() {
		assert.NoError(t, RemoveRoot(dir))
	})
	return &Cache{
		Root: dir,
	}
}
