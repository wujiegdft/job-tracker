package tracker

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigDefaultsAndAtomicPrivateSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "ai.json")
	got, err := LoadConfig(path)
	if err != nil || got != (Config{}) {
		t.Fatalf("missing file: %+v, %v", got, err)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Fatal("loading a missing config should not create directories")
	}
	first := Config{APIKey: " \t fake-test-key-one \n"}
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
	var saved map[string]any
	if err := json.Unmarshal(original, &saved); err != nil || len(saved) != 1 || saved["api_key"] != "fake-test-key-one" {
		t.Fatalf("saved config must contain only the trimmed API key: %v", err)
	}
	got, err = LoadConfig(path)
	if err != nil || got.APIKey != "fake-test-key-one" {
		t.Fatalf("API key did not round-trip: %v", err)
	}
	second := Config{APIKey: " \t\n "}
	if err := SaveConfig(path, second); err != nil {
		t.Fatal(err)
	}
	got, err = LoadConfig(path)
	if err != nil || got != (Config{}) {
		t.Fatalf("empty key should work for manual mode: %+v, %v", got, err)
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
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	saved = nil
	if err := json.Unmarshal(data, &saved); err != nil || len(saved) != 1 || saved["api_key"] != "" {
		t.Fatalf("manual config must persist only an empty API key: %v", err)
	}
}

func TestLoadConfigIgnoresLegacyProviderSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai.json")
	for _, body := range []string{
		`{"base_url":"https://attacker.invalid/steal","model":"untrusted","api_key":" fake-key "}`,
		`{"base_url":"file:///fake-secret","model":"","api_key":" fake-key "}`,
		`{"base_url":{"malicious":true},"model":[1,false],"api_key":" fake-key "}`,
		`{"base_url":123,"model":false,"api_key":" fake-key "}`,
	} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if got, err := LoadConfig(path); err != nil || got != (Config{APIKey: "fake-key"}) {
			t.Fatalf("legacy provider settings affected API key loading: %v", err)
		}
	}
}

func TestLoadConfigErrorsDoNotExposeInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ai.json")
	for _, body := range []string{
		`{"api_key":"fake-secret", BROKEN`,
		`{"api_key":["fake-secret"]}`,
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
	if got, err := LoadConfig(path); err != nil || got != (Config{}) {
		t.Fatalf("missing key should work for manual mode: %+v, %v", got, err)
	}
}
