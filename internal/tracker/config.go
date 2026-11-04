package tracker

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	APIKey  string `json:"api_key"`
}

func DefaultConfig() Config {
	return Config{BaseURL: "https://api.openai.com/v1", Model: "gpt-4o-mini"}
}

// LoadConfig overlays stored fields on the defaults. An explicitly empty model
// or API key is valid for manual use of the app.
func LoadConfig(path string) (Config, error) {
	c := DefaultConfig()
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return Config{}, errors.New("无法读取 AI 配置文件")
	}
	defer f.Close()
	const maxConfigBytes = 1 << 20
	data, err := io.ReadAll(io.LimitReader(f, maxConfigBytes+1))
	if err != nil || len(data) > maxConfigBytes {
		return Config{}, errors.New("无法读取 AI 配置文件或文件过大")
	}
	if strings.TrimSpace(string(data)) == "null" {
		return Config{}, errors.New("AI 配置文件格式无效")
	}
	if err := json.Unmarshal(data, &c); err != nil {
		// Do not expose parser errors: a provider/key may be present in input.
		return Config{}, errors.New("AI 配置文件格式无效")
	}
	c = normalizedConfig(c)
	if _, err := parseBaseURL(c.BaseURL); err != nil {
		return Config{}, err
	}
	return c, nil
}

// SaveConfig atomically replaces the configuration without ever writing a
// world-readable file. Existing parent directories are not chmodded, since a
// caller may legitimately choose a shared parent such as their home directory.
func SaveConfig(path string, c Config) error {
	c = normalizedConfig(c)
	if _, err := parseBaseURL(c.BaseURL); err != nil {
		return err
	}
	if err := ensureParentDirectory(path); err != nil {
		return errors.New("无法创建配置目录")
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return errors.New("无法编码 AI 配置")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".tracker-config-*")
	if err != nil {
		return errors.New("无法创建配置临时文件")
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := f.Chmod(0600); err != nil {
		return errors.New("无法设置配置文件权限")
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return errors.New("无法写入 AI 配置")
	}
	if err := f.Sync(); err != nil {
		return errors.New("无法同步 AI 配置")
	}
	if err := f.Close(); err != nil {
		return errors.New("无法关闭 AI 配置文件")
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return errors.New("无法替换 AI 配置文件")
	}
	return nil
}

func normalizedConfig(c Config) Config {
	c.BaseURL = strings.TrimSpace(c.BaseURL)
	c.Model = strings.TrimSpace(c.Model)
	c.APIKey = strings.TrimSpace(c.APIKey)
	return c
}

func parseBaseURL(value string) (*url.URL, error) {
	u, err := url.Parse(value)
	if err != nil || !validHTTPURL(value) || u.Fragment != "" || u.Opaque != "" {
		return nil, errors.New("AI endpoint 必须是有效的 HTTP(S) 地址，且不能包含账号或片段")
	}
	return u, nil
}

func ensureParentDirectory(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0700)
}
