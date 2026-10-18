package gui

import (
	_ "embed"
	"fmt"
	_ "image/png"

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

type Callbacks struct {
	OnSaveAndRestart func(cfg *config.AppConfig)
	OnToggle         func(start bool)
	OnQuit           func()
}

type App struct {
	fyneApp      fyne.App
	settingsWin  fyne.Window
	mon          *monitor.Monitor
	cfg          *config.AppConfig
	cb           Callbacks
	deskApp      desktop.App
	bonding      bool
	statusItem   *fyne.MenuItem
	settingsItem *fyne.MenuItem
	toggleItem   *fyne.MenuItem
	quitItem     *fyne.MenuItem
	trayMenu     *fyne.Menu
}

func NewApp(mon *monitor.Monitor, cfg *config.AppConfig, cb Callbacks) *App {
	return &App{mon: mon, cfg: cfg, cb: cb, bonding: true}
}

func (a *App) getLogoResource() fyne.Resource {
	return fyne.NewStaticResource("disnexus.png", disnexusLogoPNG)
}

func (a *App) Run() {
	HideDockIcon()

	a.fyneApp = app.NewWithID("com.disnexus.app")
	logoRes := a.getLogoResource()
	a.fyneApp.SetIcon(logoRes)

	a.settingsWin = a.fyneApp.NewWindow(i18n.T(a.cfg.Language, "settings_title"))
	a.settingsWin.Resize(fyne.NewSize(420, 420))
	a.settingsWin.SetIcon(logoRes)
	a.settingsWin.SetCloseIntercept(func() {
		a.settingsWin.Hide()
		HideDockIcon()
	})
	a.settingsWin.SetContent(a.buildSettingsUI())

	if deskApp, ok := a.fyneApp.(desktop.App); ok {
		a.deskApp = deskApp
		a.setupSystemTray(logoRes)
	}

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

func (a *App) setupSystemTray(logoRes fyne.Resource) {
	a.statusItem = fyne.NewMenuItem(i18n.T(a.cfg.Language, "active_links", 0), func() {})
	a.statusItem.Disabled = true

	a.settingsItem = fyne.NewMenuItem(i18n.T(a.cfg.Language, "open_settings"), func() {
		ShowDockIcon(disnexusLogoPNG)
		a.settingsWin.Show()
		a.settingsWin.RequestFocus()
	})

	a.toggleItem = fyne.NewMenuItem(i18n.T(a.cfg.Language, "stop_bonding"), func() {
		a.bonding = !a.bonding
		if a.bonding {
			a.toggleItem.Label = i18n.T(a.cfg.Language, "stop_bonding")
		} else {
			a.toggleItem.Label = i18n.T(a.cfg.Language, "start_bonding")
		}
		if a.trayMenu != nil {
			a.trayMenu.Refresh()
		}
		if a.cb.OnToggle != nil {
			a.cb.OnToggle(a.bonding)
		}
	})

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
		a.settingsItem,
		a.toggleItem,
		fyne.NewMenuItemSeparator(),
		a.quitItem,
	)

	a.deskApp.SetSystemTrayMenu(a.trayMenu)
	a.deskApp.SetSystemTrayIcon(logoRes)
}

func (a *App) buildSettingsUI() fyne.CanvasObject {
	logoImg := canvas.NewImageFromResource(a.getLogoResource())
	logoImg.FillMode = canvas.ImageFillContain
	logoImg.SetMinSize(fyne.NewSize(80, 80))

	titleLabel := widget.NewLabelWithStyle(i18n.T(a.cfg.Language, "header_title"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	cidrLabel := widget.NewLabel(i18n.T(a.cfg.Language, "target_cidr_label"))
	cidrEntry := widget.NewEntry()
	cidrEntry.SetPlaceHolder("10.0.0.0/8")
	cidrEntry.SetText(a.cfg.TargetCIDR)

	portLabel := widget.NewLabel(i18n.T(a.cfg.Language, "socks5_port_label"))
	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("1080")
	portEntry.SetText(a.cfg.SOCKS5Port)

	langLabel := widget.NewLabel(i18n.T(a.cfg.Language, "language_label"))
	langAutoStr := i18n.T(a.cfg.Language, "lang_auto")
	langTrStr := i18n.T(a.cfg.Language, "lang_tr")
	langEnStr := i18n.T(a.cfg.Language, "lang_en")

	langOpts := []string{langAutoStr, langTrStr, langEnStr}

	selLang := langAutoStr
	if a.cfg.Language == "tr" {
		selLang = langTrStr
	} else if a.cfg.Language == "en" {
		selLang = langEnStr
	}

	langSelect := widget.NewSelect(langOpts, func(s string) {})
	langSelect.SetSelected(selLang)
	statusLabel := widget.NewLabel("")

	saveBtn := widget.NewButton(i18n.T(a.cfg.Language, "save_btn"), func() {
		if cidrEntry.Text == "" || portEntry.Text == "" {
			statusLabel.SetText(i18n.T(a.cfg.Language, "fill_all"))
			return
		}

		chosenLang := "auto"
		if langSelect.Selected == langTrStr {
			chosenLang = "tr"
		} else if langSelect.Selected == langEnStr {
			chosenLang = "en"
		}

		a.cfg.TargetCIDR = cidrEntry.Text
		a.cfg.SOCKS5Port = portEntry.Text
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

	return container.NewPadded(container.NewVBox(
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
	))
}

func (a *App) settingsItemLabelUpdate() {
	if a.statusItem != nil {
		ips := a.mon.GetActiveIPs()
		a.statusItem.Label = i18n.T(a.cfg.Language, "active_links", len(ips))
	}
	if a.settingsItem != nil {
		a.settingsItem.Label = i18n.T(a.cfg.Language, "open_settings")
	}
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

func (a *App) UpdateStatus(count int) {
	if a.statusItem != nil {
		a.statusItem.Label = i18n.T(a.cfg.Language, "active_links", count)
		if a.trayMenu != nil {
			a.trayMenu.Refresh()
		}
	}
}
