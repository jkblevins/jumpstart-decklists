package scryfall

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9-]`)

// safeFileName converts a card name to a filesystem-safe string.
func safeFileName(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = nonAlphaNum.ReplaceAllString(s, "")
	return s
}

// cacheDir returns the default cache directory path.
func cacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "jumpforge")
}

// readCacheFile reads a cached card response if it exists and is fresh.
func readCacheFile(dir, safeName string, ttl time.Duration) ([]byte, bool) {
	path := filepath.Join(dir, safeName+".json")
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if time.Since(info.ModTime()) > ttl {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

// writeCacheFile writes card data to the cache directory.
func writeCacheFile(dir, safeName string, data []byte) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, safeName+".json")
	return os.WriteFile(path, data, 0644)
}
