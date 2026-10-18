//go:build linux

package systemproxy

import (
	"fmt"
	"os/exec"
)

func enablePlatform(port string) error {
	// GNOME gsettings ile SOCKS proxy ayarla
	cmds := [][]string{
		{"gsettings", "set", "org.gnome.system.proxy", "mode", "manual"},
		{"gsettings", "set", "org.gnome.system.proxy.socks", "host", "127.0.0.1"},
		{"gsettings", "set", "org.gnome.system.proxy.socks", "port", port},
	}

	for _, args := range cmds {
		if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
			return fmt.Errorf("gsettings komutu başarısız (%v): %w", args, err)
		}
	}
	return nil
}

func disablePlatform() error {
	if err := exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run(); err != nil {
		return fmt.Errorf("proxy devre dışı bırakılamadı: %w", err)
	}
	return nil
}
