package gui

import (
	_ "embed"
	"fmt"
	_ "image/png"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"disnexus/internal/config"
	"disnexus/internal/i18n"
	"disnexus/internal/monitor"
)

//go:embed assets/disnexus.png
var disnexusLogoPNG []byte

// Callbacks ana modülden GUI'ye aktarılan işlevlerdir.
type Callbacks struct {
	OnSaveAndRestart func(cfg *config.AppConfig) // Ayar kaydedilince çağrılır
	OnToggle         func(start bool)            // Başlat/Durdur
	OnQuit           func()                      // Çıkış
}

// App GUI uygulamasını temsil eder.
type App struct {
	fyneApp     fyne.App
	settingsWin fyne.Window
	mon         *monitor.Monitor
	cfg         *config.AppConfig
	cb          Callbacks
	deskApp     desktop.App

	// Durum
	bonding bool

	// Tray menü öğeleri ve menü referansı
	statusItem *fyne.MenuItem
	toggleItem *fyne.MenuItem
	quitItem   *fyne.MenuItem
	trayMenu   *fyne.Menu
}

// NewApp yeni bir GUI uygulaması oluşturur.
func NewApp(mon *monitor.Monitor, cfg *config.AppConfig, cb Callbacks) *App {
	return &App{
		mon:     mon,
		cfg:     cfg,
		cb:      cb,
		bonding: true,
	}
}

// getLogoResource gömülü logoyu Fyne Resource olarak döner.
func (a *App) getLogoResource() fyne.Resource {
	return fyne.NewStaticResource("disnexus.png", disnexusLogoPNG)
}

// Run GUI'yi başlatır (blocking — ana goroutine'de çağrılmalı).
func (a *App) Run() {
	a.fyneApp = app.NewWithID("com.disnexus.app")
	logoRes := a.getLogoResource()
	a.fyneApp.SetIcon(logoRes)

	// Ayarlar penceresini oluştur
	a.settingsWin = a.fyneApp.NewWindow(i18n.T(a.cfg.Language, "settings_title"))
	a.settingsWin.Resize(fyne.NewSize(420, 420))
	a.settingsWin.SetIcon(logoRes)

	// KRİTİK GEREKSİNİM: Ayarlar penceresinin "X" butonuna basıldığında uygulama kapanmamalı, gizlenmeli.
	a.settingsWin.SetCloseIntercept(func() {
		a.settingsWin.Hide()
	})

	a.settingsWin.SetContent(a.buildSettingsUI())

	// System Tray (macOS MenuBar / Windows Tray) entegrasyonu
	if deskApp, ok := a.fyneApp.(desktop.App); ok {
		a.deskApp = deskApp
		a.setupSystemTray(logoRes)
	}

	// Periyodik olarak aktif IP sayısını tray menüde güncelle
	go func() {
		for {
			ips := a.mon.GetActiveIPs()
			a.UpdateStatus(len(ips))
			select {
			case <-a.mon.OnChange():
			}
		}
	}()

	a.fyneApp.Run()
}

// setupSystemTray tepsi menüsünü yapılandırır.
func (a *App) setupSystemTray(logoRes fyne.Resource) {
	a.statusItem = fyne.NewMenuItem(fmt.Sprintf(i18n.T(a.cfg.Language, "active_links"), 0), func() {})
	a.statusItem.Disabled = true

	stopLabel := i18n.T(a.cfg.Language, "stop_bonding")
	a.toggleItem = fyne.NewMenuItem(stopLabel, func() {
		a.bonding = !a.bonding
		if a.bonding {
			a.toggleItem.Label = i18n.T(a.cfg.Language, "stop_bonding")
			log.Println("[gui] Ağ birleştirme devam ediyor...")
		} else {
			a.toggleItem.Label = i18n.T(a.cfg.Language, "start_bonding")
			log.Println("[gui] Ağ birleştirme durduruldu.")
		}
		if a.trayMenu != nil {
			a.trayMenu.Refresh()
		}
		if a.cb.OnToggle != nil {
			a.cb.OnToggle(a.bonding)
		}
	})

	settingsItem := fyne.NewMenuItem(i18n.T(a.cfg.Language, "open_settings"), func() {
		a.settingsWin.Show()
		a.settingsWin.RequestFocus()
	})

	// REQUIREMENT 1: exitItem.IsQuit = true
	a.quitItem = fyne.NewMenuItem(i18n.T(a.cfg.Language, "quit"), func() {
		if a.cb.OnQuit != nil {
			a.cb.OnQuit()
		}
		a.fyneApp.Quit()
	})
	a.quitItem.IsQuit = true

	a.trayMenu = fyne.NewMenu("DisNexus",
		a.statusItem,
		fyne.NewMenuItemSeparator(),
		settingsItem,
		a.toggleItem,
		fyne.NewMenuItemSeparator(),
		a.quitItem,
	)

	a.deskApp.SetSystemTrayMenu(a.trayMenu)
	a.deskApp.SetSystemTrayIcon(logoRes)
}

// buildSettingsUI ayarlar penceresinin içeriğini oluşturur.
func (a *App) buildSettingsUI() fyne.CanvasObject {
	logoImg := canvas.NewImageFromResource(a.getLogoResource())
	logoImg.FillMode = canvas.ImageFillContain
	logoImg.SetMinSize(fyne.NewSize(80, 80))

	titleLabel := widget.NewLabelWithStyle(
		i18n.T(a.cfg.Language, "header_title"),
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	cidrLabel := widget.NewLabel(i18n.T(a.cfg.Language, "target_cidr_label"))
	cidrEntry := widget.NewEntry()
	cidrEntry.SetPlaceHolder("Örn: 10.0.0.0/8")
	cidrEntry.SetText(a.cfg.TargetCIDR)

	portLabel := widget.NewLabel(i18n.T(a.cfg.Language, "socks5_port_label"))
	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("Örn: 1080")
	portEntry.SetText(a.cfg.SOCKS5Port)

	langLabel := widget.NewLabel(i18n.T(a.cfg.Language, "language_label"))
	langOptions := []string{"Sistem Dili (Otomatik)", "Türkçe", "English"}

	selectedLangOption := "Sistem Dili (Otomatik)"
	if a.cfg.Language == "tr" {
		selectedLangOption = "Türkçe"
	} else if a.cfg.Language == "en" {
		selectedLangOption = "English"
	}

	langSelect := widget.NewSelect(langOptions, func(s string) {})
	langSelect.SetSelected(selectedLangOption)

	statusLabel := widget.NewLabel("")

	saveBtn := widget.NewButton(i18n.T(a.cfg.Language, "save_btn"), func() {
		newCIDR := cidrEntry.Text
		newPort := portEntry.Text

		if newCIDR == "" || newPort == "" {
			statusLabel.SetText(i18n.T(a.cfg.Language, "fill_all"))
			return
		}

		chosenLang := "auto"
		switch langSelect.Selected {
		case "Türkçe":
			chosenLang = "tr"
		case "English":
			chosenLang = "en"
		default:
			chosenLang = "auto"
		}

		a.cfg.TargetCIDR = newCIDR
		a.cfg.SOCKS5Port = newPort
		a.cfg.Language = chosenLang

		if err := a.cfg.Save(); err != nil {
			statusLabel.SetText(fmt.Sprintf(i18n.T(a.cfg.Language, "save_error"), err))
			return
		}

		statusLabel.SetText(i18n.T(a.cfg.Language, "saved_success"))

		a.settingsWin.SetTitle(i18n.T(a.cfg.Language, "settings_title"))
		a.settingsWin.SetContent(a.buildSettingsUI())
		a.settingsItemLabelUpdate()

		if a.cb.OnSaveAndRestart != nil {
			a.cb.OnSaveAndRestart(a.cfg)
		}
	})

	form := container.NewVBox(
		container.NewCenter(logoImg),
		titleLabel,
		cidrLabel,
		cidrEntry,
		portLabel,
		portEntry,
		langLabel,
		langSelect,
		widget.NewSeparator(),
		saveBtn,
		statusLabel,
	)

	return container.NewPadded(form)
}

func (a *App) settingsItemLabelUpdate() {
	if a.quitItem != nil {
		a.quitItem.Label = i18n.T(a.cfg.Language, "quit")
	}
	if a.toggleItem != nil {
		if a.bonding {
			a.toggleItem.Label = i18n.T(a.cfg.Language, "stop_bonding")
		} else {
			a.toggleItem.Label = i18n.T(a.cfg.Language, "start_bonding")
		}
	}
	if a.trayMenu != nil {
		a.trayMenu.Refresh()
	}
}

// UpdateStatus tray durum metnini günceller.
func (a *App) UpdateStatus(count int) {
	if a.statusItem != nil {
		a.statusItem.Label = fmt.Sprintf(i18n.T(a.cfg.Language, "active_links"), count)
		if a.trayMenu != nil {
			a.trayMenu.Refresh()
		}
	}
}

