package deadlockio

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dorkitude/deadlore/internal/wiki"
)

const cacheVersion = 1

type cache struct{ directory string }

func newCache(directory string) (*cache, error) {
	if directory == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return nil, err
		}
		directory = filepath.Join(base, "deadlore", "deadlockio")
	} else {
		directory = filepath.Join(directory, "deadlockio")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}
	return &cache{directory: directory}, nil
}

func (c *cache) load(key string) (*wiki.Page, error) {
	contents, err := os.ReadFile(c.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var page wiki.Page
	if err := json.Unmarshal(contents, &page); err != nil {
		return nil, err
	}
	if page.CacheVersion != cacheVersion {
		return nil, nil
	}
	return &page, nil
}

func (c *cache) save(key string, page *wiki.Page) error {
	page.CacheVersion = cacheVersion
	contents, err := json.Marshal(page)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(c.directory, ".page-*.json")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, c.path(key))
}

func (c *cache) path(key string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(key))))
	return filepath.Join(c.directory, hex.EncodeToString(sum[:])+".json")
}

func (c *cache) status() (count int, newest time.Time, err error) {
	entries, err := os.ReadDir(c.directory)
	if err != nil {
		return 0, time.Time{}, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return 0, time.Time{}, err
		}
		count++
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
	}
	return count, newest, nil
}

func (c *cache) clear(key string) error {
	if key == "" {
		entries, err := os.ReadDir(c.directory)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				if err := os.Remove(filepath.Join(c.directory, entry.Name())); err != nil {
					return err
				}
			}
		}
		return nil
	}
	err := os.Remove(c.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
