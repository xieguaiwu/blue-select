// internal/config/config.go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 保存设备名到 MAC 的映射与默认设备。
type Config struct {
	DefaultDevice string            `json:"default_device"`
	Devices       map[string]string `json:"devices"` // name -> MAC
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "blue-select", "config.json"), nil
}

// Load 读取配置。文件不存在时返回空配置，不报错。
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{Devices: map[string]string{}}, nil
	}
	if err != nil {
		return nil, err
	}
	c := &Config{}
	if err := json.Unmarshal(data, c); err != nil {
		return nil, err
	}
	if c.Devices == nil {
		c.Devices = map[string]string{}
	}
	return c, nil
}

// Save 写入配置，目录不存在则创建，权限 0644。
func Save(c *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
