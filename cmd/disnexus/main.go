package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"disnexus/internal/config"
	"disnexus/internal/gui"
	"disnexus/internal/monitor"
	"disnexus/internal/proxy"
	"disnexus/internal/systemproxy"
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.Println("[main] DisNexus (Network Bonding & Load Balancing) başlatılıyor...")

	// ─── 0. Root Context (Graceful Shutdown için) ───
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ─── 1. Konfigürasyon Yükleme ───
	cfg := config.LoadConfig()
	log.Printf("[main] Ayarlar yüklendi — SOCKS5 Port: %s | Target CIDR: %s | Dil: %s",
		cfg.SOCKS5Port, cfg.TargetCIDR, cfg.Language)

	// ─── 2. Network Monitor Başlatma ───
	mon, err := monitor.NewMonitor(cfg.TargetCIDR)
	if err != nil {
		log.Fatalf("[main] Kritik Hata: Monitor başlatılamadı (%v). Geçerli bir CIDR girin.", err)
	}
	mon.Start(ctx)
	log.Println("[main] Ağ monitörü aktif (3 saniyede bir adaptör taraması yapılıyor)")

	// ─── 3. SOCKS5 Proxy Sunucusu Başlatma ───
	proxyServer := &proxy.Server{
		Port:    cfg.SOCKS5Port,
		Monitor: mon,
	}
	if err := proxyServer.Start(); err != nil {
		log.Fatalf("[main] Kritik Hata: SOCKS5 proxy başlatılamadı (%v). Port %s başka bir uygulama tarafından kullanılıyor olabilir.", err, cfg.SOCKS5Port)
	}

	// ─── 4. Sistem Proxy Ayarını Etkinleştirme ───
	if err := systemproxy.Enable(cfg.SOCKS5Port); err != nil {
		log.Printf("[main] Uyarı: Sistem proxy aktifleştirilemedi: %v", err)
	} else {
		log.Println("[main] Sistem SOCKS proxy başarıyla etkinleştirildi")
	}

	// ─── 5. Sinyal ve Kapanma Yönetimi (Graceful Shutdown & Rollback) ───
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("[main] Kapanma sinyali alındı (%v). Temizlik ve sistem proxy geri alma işlemi başlatılıyor...", sig)
		shutdownSequence(proxyServer, mon, cancel)
	}()

	// ─── 6. GUI (Fyne System Tray & Settings Window) Başlatma ───
	guiApp := gui.NewApp(mon, cfg, gui.Callbacks{
		OnSaveAndRestart: func(newCfg *config.AppConfig) {
			log.Println("[main] Ayarlar güncellendi, proxy ve monitor yeniden başlatılıyor...")
			proxyServer.Stop()
			mon.Stop()

			// Yeni ayarları uygula
			newMon, err := monitor.NewMonitor(newCfg.TargetCIDR)
			if err != nil {
				log.Printf("[main] Yeni CIDR geçersiz: %v", err)
				return
			}
			newMon.Start(ctx)

			newProxy := &proxy.Server{
				Port:    newCfg.SOCKS5Port,
				Monitor: newMon,
			}
			if err := newProxy.Start(); err != nil {
				log.Printf("[main] Yeni port (%s) başlatılamadı: %v", newCfg.SOCKS5Port, err)
				return
			}
			proxyServer = newProxy
			mon = newMon

			// Sistem proxy portunu güncelle
			_ = systemproxy.Enable(newCfg.SOCKS5Port)
			log.Println("[main] DisNexus başarıyla yeniden başlatıldı.")
		},
		OnToggle: func(start bool) {
			if start {
				log.Println("[main] Ağ birleştirme devam ediyor...")
			} else {
				log.Println("[main] Ağ birleştirme durduruldu.")
			}
		},
		OnQuit: func() {
			log.Println("[main] Kullanıcı çıkış talebinde bulundu.")
			shutdownSequence(proxyServer, mon, cancel)
		},
	})

	guiApp.Run()
}

// shutdownSequence tüm bileşenleri güvenle kapatır ve sistem proxy'sini orijinal haline getirir (Rollback).
func shutdownSequence(proxyServer *proxy.Server, mon *monitor.Monitor, cancel context.CancelFunc) {
	log.Println("[main] Sistem proxy ayarları devre dışı bırakılıyor (Disable)...")
	systemproxy.Disable()

	if proxyServer != nil {
		proxyServer.Stop()
	}

	if mon != nil {
		mon.Stop()
	}

	cancel()
	log.Println("[main] DisNexus güvenli bir şekilde kapatıldı.")
	os.Exit(0)
}
