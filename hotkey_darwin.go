package main

import "golang.design/x/hotkey"

func (a *App) setupHotkey() {
	go a.listenHotkeyDarwin()
}

func (a *App) listenHotkeyDarwin() {
	hkS := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, hotkey.KeyS)
	hkD := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModShift}, hotkey.KeyD)

	if err := hkS.Register(); err != nil {
		return
	}
	defer hkS.Unregister()

	if err := hkD.Register(); err != nil {
		return
	}
	defer hkD.Unregister()

	for {
		select {
		case <-hkS.Keydown():
			a.ShowConnectDialog()
		case <-hkD.Keydown():
			a.ShowMountDialog()
		}
	}
}
