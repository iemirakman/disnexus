package i18n

import (
	"fmt"
	"os"
	"strings"
)

var translations = map[string]map[string]string{
	"tr": {
		"active_links":      "Aktif Hat: %d Adet",
		"open_settings":     "Ayarları Aç",
		"stop_bonding":      "Birleştirmeyi Durdur",
		"start_bonding":     "Birleştirmeyi Başlat",
		"quit":              "Çıkış",
		"settings_title":    "DisNexus — Ayarlar",
		"header_title":      "DisNexus Ağ Birleştirici",
		"target_cidr_label": "Target CIDR (Hedef Ağ Bloğu):",
		"socks5_port_label": "SOCKS5 Proxy Portu:",
		"language_label":    "Dil / Language:",
		"save_btn":          "Kaydet ve Yeniden Başlat",
		"saved_success":     "✔ Ayarlar kaydedildi!",
		"fill_all":          "⚠ Tüm alanları doldurun.",
		"save_error":        "⚠ Kaydedilemedi: %v",
	},
	"en": {
		"active_links":      "Active Links: %d",
		"open_settings":     "Open Settings",
		"stop_bonding":      "Stop Bonding",
		"start_bonding":     "Start Bonding",
		"quit":              "Quit",
		"settings_title":    "DisNexus — Settings",
		"header_title":      "DisNexus Network Bonder",
		"target_cidr_label": "Target CIDR:",
		"socks5_port_label": "SOCKS5 Proxy Port:",
		"language_label":    "Language:",
		"save_btn":          "Save & Restart",
		"saved_success":     "✔ Settings saved!",
		"fill_all":          "⚠ Please fill all fields.",
		"save_error":        "⚠ Failed to save: %v",
	},
}

func DetectSystemLang() string {
	langEnv := os.Getenv("LANG")
	if langEnv == "" {
		langEnv = os.Getenv("LC_ALL")
	}
	if langEnv == "" {
		langEnv = os.Getenv("LC_MESSAGES")
	}
	langEnv = strings.ToLower(langEnv)
	if strings.Contains(langEnv, "tr") {
		return "tr"
	}
	return "en"
}

func GetLang(configLang string) string {
	if configLang == "auto" || configLang == "" {
		return DetectSystemLang()
	}
	if configLang == "tr" || configLang == "en" {
		return configLang
	}
	return "en"
}

func T(configLang, key string, args ...any) string {
	lang := GetLang(configLang)
	dict, ok := translations[lang]
	if !ok {
		dict = translations["en"]
	}
	format, ok := dict[key]
	if !ok {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(format, args...)
	}
	return format
}
