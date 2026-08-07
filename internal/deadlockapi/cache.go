package deadlockapi

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

const cacheVersion = 1

type cache struct{ directory string }

type cachedResponse struct {
	CacheVersion int             `json:"cache_version"`
	FetchedAt    time.Time       `json:"fetched_at"`
	Data         json.RawMessage `json:"data"`
}

func newCache(directory string) (*cache, error) {
	if directory == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return nil, err
		}
		directory = filepath.Join(base, "deadlore", "deadlock-api")
	} else {
		directory = filepath.Join(directory, "deadlock-api")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}
	return &cache{directory: directory}, nil
}

func (c *cache) load(key string, target any) (time.Time, bool, error) {
	contents, err := os.ReadFile(c.path(key))
	if errors.Is(err, os.ErrNotExist) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	var response cachedResponse
	if err := json.Unmarshal(contents, &response); err != nil {
		return time.Time{}, false, err
	}
	if response.CacheVersion != cacheVersion {
		return time.Time{}, false, nil
	}
	if err := json.Unmarshal(response.Data, target); err != nil {
		return time.Time{}, false, err
	}
	return response.FetchedAt, true, nil
}

func (c *cache) save(key string, value any, fetchedAt time.Time) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	contents, err := json.Marshal(cachedResponse{CacheVersion: cacheVersion, FetchedAt: fetchedAt, Data: data})
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(c.directory, ".response-*.json")
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

func (c *cache) clear() error {
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

func (c *cache) path(key string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(key))))
	return filepath.Join(c.directory, hex.EncodeToString(sum[:])+".json")
}
