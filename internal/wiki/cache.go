package wiki

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const cacheDirectoryName = "deadlore"

type Cache struct {
	directory string
}

func NewCache(directory string) (*Cache, error) {
	if directory == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return nil, err
		}
		directory = filepath.Join(base, cacheDirectoryName)
	}

	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}

	return &Cache{directory: directory}, nil
}

func (c *Cache) Load(title string) (*Page, error) {
	contents, err := os.ReadFile(c.path(title))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var page Page
	if err := json.Unmarshal(contents, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

func (c *Cache) Save(page *Page) error {
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
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, c.path(page.Title))
}

func (c *Cache) Clear(title string) error {
	if title == "" {
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

	err := os.Remove(c.path(title))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (c *Cache) Status() (count int, newest time.Time, err error) {
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

func (c *Cache) path(title string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(title))))
	return filepath.Join(c.directory, hex.EncodeToString(sum[:])+".json")
}
