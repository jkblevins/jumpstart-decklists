package scryfall

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSafeFileName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Lightning Bolt", "lightning-bolt"},
		{"Goblin Guide", "goblin-guide"},
		{"Nissa, Who Shakes the World", "nissa-who-shakes-the-world"},
		{"Fire // Ice", "fire--ice"},
	}
	for _, tc := range tests {
		got := safeFileName(tc.input)
		if got != tc.want {
			t.Errorf("safeFileName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestCacheWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`{"name":"Lightning Bolt"}`)

	writeCacheFile(dir, "lightning-bolt", data)

	got, ok := readCacheFile(dir, "lightning-bolt", 7*24*time.Hour)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got) != string(data) {
		t.Errorf("cache data mismatch: got %q", string(got))
	}
}

func TestCacheExpired(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`{"name":"Old Card"}`)

	path := filepath.Join(dir, "old-card.json")
	os.WriteFile(path, data, 0644)
	// Set mod time to 8 days ago
	old := time.Now().Add(-8 * 24 * time.Hour)
	os.Chtimes(path, old, old)

	_, ok := readCacheFile(dir, "old-card", 7*24*time.Hour)
	if ok {
		t.Error("expected cache miss for expired entry")
	}
}

func TestCacheMiss(t *testing.T) {
	dir := t.TempDir()
	_, ok := readCacheFile(dir, "nonexistent", 7*24*time.Hour)
	if ok {
		t.Error("expected cache miss")
	}
}
