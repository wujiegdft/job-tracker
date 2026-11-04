package tracker

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	DeepSeekBaseURL = "https://api.deepseek.com"
	DeepSeekModel   = "deepseek-flash"
)

type Config struct {
	APIKey string `json:"api_key"`
}

// LoadConfig reads only the API key, ignoring legacy provider settings.
// An empty API key is valid for manual use of the app.
func LoadConfig(path string) (Config, error) {
	var c Config
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
	c.APIKey = strings.TrimSpace(c.APIKey)
	return c, nil
}

// SaveConfig atomically replaces the configuration without ever writing a
// world-readable file. Existing parent directories are not chmodded, since a
// caller may legitimately choose a shared parent such as their home directory.
func SaveConfig(path string, c Config) error {
	c.APIKey = strings.TrimSpace(c.APIKey)
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

func ensureParentDirectory(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0700)
}
