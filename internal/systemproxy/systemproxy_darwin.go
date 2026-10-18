//go:build darwin

package systemproxy

import (
	"fmt"
	"os/exec"
	"strings"
)

// getActiveNetworkServices macOS'ta aktif ağ servislerini döner.
func getActiveNetworkServices() ([]string, error) {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return nil, fmt.Errorf("ağ servisleri listelenemedi: %w", err)
	}
	var services []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		// İlk satır başlık, yıldızlı olanlar devre dışı
		if line == "" || strings.HasPrefix(line, "An asterisk") || strings.HasPrefix(line, "*") {
			continue
		}
		services = append(services, line)
	}
	return services, nil
}

func enablePlatform(port string) error {
	services, err := getActiveNetworkServices()
	if err != nil {
		return err
	}
	for _, svc := range services {
		_ = exec.Command("networksetup", "-setsocksfirewallproxy", svc, "127.0.0.1", port).Run()
		_ = exec.Command("networksetup", "-setsocksfirewallproxystate", svc, "on").Run()
	}
	return nil
}

func disablePlatform() error {
	services, err := getActiveNetworkServices()
	if err != nil {
		return err
	}
	for _, svc := range services {
		_ = exec.Command("networksetup", "-setsocksfirewallproxystate", svc, "off").Run()
	}
	return nil
}
