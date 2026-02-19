package main

import (
	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const maxHistorySlots = 10

type historySlot struct {
	parent  *systray.MenuItem
	mount   *systray.MenuItem
	connect *systray.MenuItem
}

func (a *App) setupTray() {
	go systray.Run(a.onTrayReady, nil)
}

func (a *App) onTrayReady() {
	systray.SetIcon(icon)
	systray.SetTooltip("Mountly")

	mConnect := systray.AddMenuItem("Connect SSH  (Ctrl+Shift+S)", "Open Connect SSH dialog")
	mMount := systray.AddMenuItem("Mount  (Ctrl+Shift+D)", "Open Mount dialog")
	systray.AddSeparator()

	slots := make([]historySlot, maxHistorySlots)
	for i := range slots {
		slots[i].parent = systray.AddMenuItem("", "")
		slots[i].parent.Hide()
		slots[i].mount = slots[i].parent.AddSubMenuItem("Mount", "")
		slots[i].connect = slots[i].parent.AddSubMenuItem("Connect SSH", "")
	}

	systray.AddSeparator()
	mExit := systray.AddMenuItem("Exit", "Exit Mountly")

	// Callback used by SaveMount to refresh visible history items.
	a.trayUpdate = func() {
		limit := len(a.mountHistory)
		if limit > maxHistorySlots {
			limit = maxHistorySlots
		}
		for i := range slots {
			if i < limit {
				slots[i].parent.SetTitle(a.mountHistory[i])
				slots[i].parent.Show()
			} else {
				slots[i].parent.Hide()
			}
		}
	}
	a.trayUpdate()

	// Top-level buttons.
	mConnect.Click(func() { go a.ShowConnectDialog() })
	mMount.Click(func() { go a.ShowMountDialog() })
	mExit.Click(func() { runtime.Quit(a.ctx) })

	// History slot handlers.
	for i := range slots {
		i := i
		slot := &slots[i]
		slot.mount.Click(func() {
			if i < len(a.mountHistory) {
				go a.ExecuteMount(a.mountHistory[i])
			}
		})
		slot.connect.Click(func() {
			if i < len(a.mountHistory) {
				go a.ExecuteConnect(a.mountHistory[i])
			}
		})
	}
}
