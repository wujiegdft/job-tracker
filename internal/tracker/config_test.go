package tracker

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigDefaultsAndAtomicPrivateSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "ai.json")
	got, err := LoadConfig(path)
	if err != nil || got != DefaultConfig() {
		t.Fatalf("missing file: %+v, %v", got, err)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatal("loading a missing config should not create directories")
	}
	first := DefaultConfig()
	first.APIKey = "fake-test-key-one"
	if err := SaveConfig(path, first); err != nil {
		t.Fatal(err)
	}
	checkMode(t, filepath.Dir(path), 0700)
	checkMode(t, path, 0600)
	old, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer old.Close()
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	second := Config{BaseURL: " http://localhost:8080/v1/chat/completions ", Model: "", APIKey: ""}
	if err := SaveConfig(path, second); err != nil {
		t.Fatal(err)
	}
	second.BaseURL = strings.TrimSpace(second.BaseURL)
	got, err = LoadConfig(path)
	if err != nil || got != second {
		t.Fatalf("empty model/key should work for manual mode: %+v, %v", got, err)
	}
	// An already-open reader sees the complete old inode, proving replacement
	// rather than truncation/rewrite of the old file.
	stillOld, err := io.ReadAll(old)
	if err != nil || !bytes.Equal(stillOld, original) {
		t.Fatal("SaveConfig did not atomically replace the old file")
	}
	checkMode(t, path, 0600)
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 || entries[0].Name() != "ai.json" {
		t.Fatalf("temporary config files left behind: %v, %v", entries, err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"", "file:///tmp/key", "ftp://example.com", "https:///v1", "https://user:fake-secret@example.com/v1", "https://example.com/v1#fragment"} {
		invalid := first
		invalid.BaseURL = endpoint
		err := SaveConfig(path, invalid)
		if err == nil || strings.Contains(err.Error(), first.APIKey) || strings.Contains(err.Error(), "fake-secret") {
			t.Fatal("invalid endpoint accepted or secret leaked")
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed save changed the existing config")
	}
}

func TestLoadConfigErrorsDoNotExposeInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai.json")
	for _, body := range []string{
		`{"api_key":"fake-secret", BROKEN`,
		`{"base_url":"file:///fake-secret"}`,
		`[]`, `null`,
	} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadConfig(path); err == nil || strings.Contains(err.Error(), "fake-secret") {
			t.Fatal("corrupt config accepted or secret leaked")
		}
	}
	if err := os.WriteFile(path, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := LoadConfig(path); err != nil || got != DefaultConfig() {
		t.Fatalf("partial config did not retain defaults: %+v, %v", got, err)
	}
}
