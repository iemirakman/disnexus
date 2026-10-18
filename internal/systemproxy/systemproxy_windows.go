//go:build windows

package systemproxy

import (
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const regPath = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

func enablePlatform(port string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("registry açılamadı: %w", err)
	}
	defer k.Close()

	proxyValue := fmt.Sprintf("socks=127.0.0.1:%s", port)

	if err := k.SetStringValue("ProxyServer", proxyValue); err != nil {
		return fmt.Errorf("ProxyServer yazılamadı: %w", err)
	}
	if err := k.SetDWordValue("ProxyEnable", 1); err != nil {
		return fmt.Errorf("ProxyEnable yazılamadı: %w", err)
	}

	// Bypass listesi — yerel adresleri atla
	bypass := strings.Join([]string{"localhost", "127.*", "10.*", "172.16.*", "192.168.*", "<local>"}, ";")
	if err := k.SetStringValue("ProxyOverride", bypass); err != nil {
		return fmt.Errorf("ProxyOverride yazılamadı: %w", err)
	}

	// WinINet'e değişikliği bildir
	refreshWinINet()

	return nil
}

func disablePlatform() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("registry açılamadı: %w", err)
	}
	defer k.Close()

	if err := k.SetDWordValue("ProxyEnable", 0); err != nil {
		return fmt.Errorf("ProxyEnable sıfırlanamadı: %w", err)
	}

	refreshWinINet()
	return nil
}

// refreshWinINet Internet Explorer / WinHTTP ayarlarının yeniden yüklenmesini tetikler.
func refreshWinINet() {
	// rundll32 ile WinINet cache'ini yenile
	_ = exec.Command("rundll32.exe", "wininet.dll,InternetSetOptionW").Run()
}
