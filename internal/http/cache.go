package http

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CacheFilename is the name of the cache file, under the os temp directory.
const CacheFilename = "serverpilot-tools.cache"

// CacheLifetime is the amount of time that a cache file is valid for. It is based on the file mtime.
const CacheLifetime = 24 * time.Hour // 1 day

// tmpFileCacher implements the cacher interface. It caches serverpilot API
// responses in a temporary file.
type tmpFileCacher struct{}

func (c *tmpFileCacher) Has(key string) bool {
	b, err := os.ReadFile(c.filename())
	if err != nil {
		return false
	}

	return bytes.Contains(b, []byte(key))
}

func (c *tmpFileCacher) Get(key string) (string, error) {
	// If the cache file is older than the cache lifetime, delete it.
	if c.isExpired() {
		err := c.delete()
		if err != nil {
			return "", fmt.Errorf("could not delete expired cache file: %s", err)
		}
		return "", fmt.Errorf("cache file expired")
	}

	// Read the contents of the file.
	b, err := os.ReadFile(c.filename())
	if err != nil {
		return "", fmt.Errorf("could not read cache file: %s", err)
	}

	// Find the line that starts with the key.
	i := bytes.Index(b, []byte(key))
	// Find the end of the line.
	n := bytes.Index(b[i:], []byte("\n"))
	// Extract the value.
	value := string(b[i+len(key)+2 : i+n])

	return value, nil
}

func (c *tmpFileCacher) Set(key string, value string) error {
	// Create the file if it doesn't exist.
	f, err := os.OpenFile(c.filename(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("could not open cache file: %s", err)
	}
	defer f.Close()

	// Append the key/value pair to the file.
	_, err = f.WriteString(fmt.Sprintf("%s: %s\n", key, value))
	if err != nil {
		return fmt.Errorf("could not write to cache file: %s", err)
	}

	return nil
}

func (c *tmpFileCacher) filename() string {
	return filepath.Join(os.TempDir(), CacheFilename)
}

func (c *tmpFileCacher) isExpired() bool {
	// If the file doesn't exist, it's not expired.
	if _, err := os.Stat(c.filename()); os.IsNotExist(err) {
		return false
	}

	// If the file is older than the cache lifetime, it's expired.
	fi, err := os.Stat(c.filename())
	if err != nil {
		return true
	}
	if time.Since(fi.ModTime()) > CacheLifetime {
		return true
	}

	// Otherwise, it's not expired.
	return false
}

func (c *tmpFileCacher) delete() error {
	return os.Remove(c.filename())
}
