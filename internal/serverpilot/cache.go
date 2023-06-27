package serverpilot

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CacheFilename is the name of the cache file, under the os temp directory.
const CacheFilename = "serverpilot-tools.cache"

// CacheLifetime is the amount of time that a cache file is valid for. It is based on the file mtime.
const CacheLifetime = 10 * time.Minute

// tmpFileCacher implements the cacher interface. It caches serverpilot API
// responses in a temporary file.
type tmpFileCacher struct{}

func (c *tmpFileCacher) Get(key string) (string, error) {
	// If the cache file is older than the cache lifetime, delete it.
	if c.isExpired() {
		err := c.delete()
		if err != nil {
			return "", fmt.Errorf("could not delete expired cache file: %s", err)
		}
		return "", fmt.Errorf("cache file expired")
	}

	// Open the cache file.
	f, err := os.Open(c.filename())
	if err != nil {
		return "", fmt.Errorf("could not open cache file: %s", err)
	}
	defer f.Close()

	// Read the contents of the file.
	buf := new(strings.Builder)
	_, err = io.Copy(buf, f)
	if err != nil {
		return "", fmt.Errorf("could not read cache file: %s", err)
	}

	// Convert the contents to a map
	m, err := c.decode(buf.String())
	if err != nil {
		return "", fmt.Errorf("could not decode cache file: %s", err)
	}

	// Return the value for the given key
	val, ok := m[key]
	if !ok {
		return "", fmt.Errorf("could not find key %s in cache", key)
	}
	return val, nil
}

func (c *tmpFileCacher) Set(key string, value string) error {
	// Create the file if it doesn't exist.
	f, err := os.OpenFile(c.filename(), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("could not open cache file: %s", err)
	}
	defer f.Close()

	// Convert the contents to a map
	s := new(strings.Builder)
	_, err = io.Copy(s, f)
	m, err := c.decode(s.String())
	if err != nil {
		return fmt.Errorf("could not decode cache file: %s", err)
	}

	// Add the new key/value pair to the map
	m[key] = value

	// Encode the map back to a string
	contents, err := c.encode(m)
	if err != nil {
		return fmt.Errorf("could not encode cache file: %s", err)
	}

	// Write the contents to the file
	_, err = f.WriteString(contents)
	if err != nil {
		return fmt.Errorf("could not write to cache file: %s", err)
	}

	return nil
}

func (c *tmpFileCacher) filename() string {
	return filepath.Join(os.TempDir(), CacheFilename)
}

func (c *tmpFileCacher) decode(contents string) (map[string]string, error) {
	if contents == "" {
		return make(map[string]string), nil
	}

	var m map[string]string
	d := gob.NewDecoder(strings.NewReader(contents))
	err := d.Decode(&m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (c *tmpFileCacher) encode(m map[string]string) (string, error) {
	buf := new(strings.Builder)
	e := gob.NewEncoder(buf)
	err := e.Encode(m)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
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
