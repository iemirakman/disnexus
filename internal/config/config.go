package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppConfig uygulama ayarlarını tutan yapıdır.
type AppConfig struct {
	SOCKS5Port string `json:"socks5_port"`
	TargetCIDR string `json:"target_cidr"`
	AutoStart  bool   `json:"auto_start"`
	Language   string `json:"language"`
}

// GetConfigPath ayar dosyasının tam yolunu döner.
func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(configDir, "disnexus")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "config.json"), nil
}

// LoadConfig ayarları yükler, dosya yoksa varsayılan değerleri döner.
func LoadConfig() *AppConfig {
	path, _ := GetConfigPath()
	conf := &AppConfig{
		SOCKS5Port: "1080",
		TargetCIDR: "10.0.0.0/8",
		AutoStart:  false,
		Language:   "auto",
	}

	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, conf)
	}
	if conf.Language == "" {
		conf.Language = "auto"
	}
	return conf
}

// Save ayarları JSON dosyasına kaydeder.
func (c *AppConfig) Save() error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
