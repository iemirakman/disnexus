package systemproxy

// Enable verilen port üzerinden sistem SOCKS proxy'sini aktifleştirir.
// Platforma özel implementasyonlar _darwin, _windows, _linux dosyalarındadır.
func Enable(port string) error {
	return enablePlatform(port)
}

// Disable sistem SOCKS proxy ayarını devre dışı bırakır.
func Disable() error {
	return disablePlatform()
}
