package main

import (
	"fmt"
	"os"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

func (a *App) setupHotkey() {
	go a.listenHotkey()
}

// listenHotkey detekuje display server za běhu.
// Preferuje X11 (nebo XWayland) — XGrabKey funguje globálně i na GNOME Wayland.
// Čistý Wayland bez DISPLAY použije xdg-desktop-portal GlobalShortcuts.
func (a *App) listenHotkey() {
	if os.Getenv("DISPLAY") != "" {
		a.listenHotkeyX11()
	} else if os.Getenv("WAYLAND_DISPLAY") != "" {
		a.listenHotkeyWayland()
	}
}

// ── X11 ───────────────────────────────────────────────────────────────────────
// Ctrl+Shift+S → Connect SSH
// Ctrl+Shift+D → Mount

func (a *App) listenHotkeyX11() {
	conn, err := xgb.NewConn()
	if err != nil {
		return
	}
	defer conn.Close()

	setup := xproto.Setup(conn)
	root := setup.Roots[0].Root

	keycodeS, errS := findKeycode(conn, setup, 0x0073) // 's'
	keycodeD, errD := findKeycode(conn, setup, 0x0064) // 'd'
	if errS != nil && errD != nil {
		return
	}

	const base uint16 = xproto.ModMaskControl | xproto.ModMaskShift
	extras := []uint16{0, xproto.ModMask2, xproto.ModMaskLock, xproto.ModMask2 | xproto.ModMaskLock}

	for _, extra := range extras {
		if errS == nil {
			xproto.GrabKey(conn, true, root, base|extra, keycodeS,
				xproto.GrabModeAsync, xproto.GrabModeAsync)
		}
		if errD == nil {
			xproto.GrabKey(conn, true, root, base|extra, keycodeD,
				xproto.GrabModeAsync, xproto.GrabModeAsync)
		}
	}

	for {
		ev, err := conn.WaitForEvent()
		if err != nil {
			return
		}
		if kp, ok := ev.(xproto.KeyPressEvent); ok {
			switch kp.Detail {
			case keycodeS:
				a.ShowConnectDialog()
			case keycodeD:
				a.ShowMountDialog()
			}
		}
	}
}

func findKeycode(conn *xgb.Conn, setup *xproto.SetupInfo, target xproto.Keysym) (xproto.Keycode, error) {
	min := setup.MinKeycode
	max := setup.MaxKeycode
	count := int(max-min) + 1

	mapping, err := xproto.GetKeyboardMapping(conn, min, byte(count)).Reply()
	if err != nil {
		return 0, err
	}

	kpk := int(mapping.KeysymsPerKeycode)
	for i := 0; i < count; i++ {
		for j := 0; j < kpk; j++ {
			if mapping.Keysyms[i*kpk+j] == target {
				return min + xproto.Keycode(i), nil
			}
		}
	}
	return 0, fmt.Errorf("keysym 0x%x nenalezen", target)
}

// ── Wayland (xdg-desktop-portal GlobalShortcuts) ─────────────────────────────
// Ctrl+Shift+S → Connect SSH
// Ctrl+Shift+D → Mount

func (a *App) listenHotkeyWayland() {
	conn, err := dbus.SessionBus()
	if err != nil {
		return
	}

	portal := conn.Object(
		"org.freedesktop.portal.Desktop",
		"/org/freedesktop/portal/desktop",
	)

	sigCh := make(chan *dbus.Signal, 16)
	conn.Signal(sigCh)
	defer conn.RemoveSignal(sigCh)

	if err := conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.portal.Request"),
		dbus.WithMatchMember("Response"),
	); err != nil {
		return
	}

	// 1. CreateSession
	var createReqPath dbus.ObjectPath
	err = portal.Call(
		"org.freedesktop.portal.GlobalShortcuts.CreateSession", 0,
		map[string]dbus.Variant{
			"session_handle_token": dbus.MakeVariant("mountlysession"),
		},
	).Store(&createReqPath)
	if err != nil {
		return
	}

	createResp, ok := waitPortalResponse(sigCh, createReqPath)
	if !ok {
		return
	}
	sh, ok := createResp["session_handle"].Value().(dbus.ObjectPath)
	if !ok {
		return
	}

	// 2. BindShortcuts — dvě zkratky
	shortcuts := []map[string]dbus.Variant{
		{
			"id":                dbus.MakeVariant("open-connect"),
			"description":       dbus.MakeVariant("Mountly: otevřít Connect SSH dialog"),
			"preferred_trigger": dbus.MakeVariant("<Control><Shift>s"),
		},
		{
			"id":                dbus.MakeVariant("open-mount"),
			"description":       dbus.MakeVariant("Mountly: otevřít Mount dialog"),
			"preferred_trigger": dbus.MakeVariant("<Control><Shift>d"),
		},
	}
	var bindReqPath dbus.ObjectPath
	err = portal.Call(
		"org.freedesktop.portal.GlobalShortcuts.BindShortcuts", 0,
		sh, shortcuts, "", map[string]dbus.Variant{},
	).Store(&bindReqPath)
	if err != nil {
		return
	}

	if _, ok := waitPortalResponse(sigCh, bindReqPath); !ok {
		return
	}

	// 3. Nasloucháme signálu Activated
	if err := conn.AddMatchSignal(
		dbus.WithMatchObjectPath(sh),
		dbus.WithMatchInterface("org.freedesktop.portal.GlobalShortcuts"),
		dbus.WithMatchMember("Activated"),
	); err != nil {
		return
	}

	for sig := range sigCh {
		if sig.Name != "org.freedesktop.portal.GlobalShortcuts.Activated" {
			continue
		}
		if len(sig.Body) >= 2 {
			if id, ok := sig.Body[1].(string); ok {
				switch id {
				case "open-connect":
					a.ShowConnectDialog()
				case "open-mount":
					a.ShowMountDialog()
				}
			}
		}
	}
}

func waitPortalResponse(sigCh <-chan *dbus.Signal, reqPath dbus.ObjectPath) (map[string]dbus.Variant, bool) {
	t := time.NewTimer(60 * time.Second)
	defer t.Stop()
	for {
		select {
		case sig, open := <-sigCh:
			if !open {
				return nil, false
			}
			if sig.Path != reqPath {
				continue
			}
			if len(sig.Body) < 2 {
				return nil, false
			}
			code, ok := sig.Body[0].(uint32)
			if !ok || code != 0 {
				return nil, false
			}
			results, ok := sig.Body[1].(map[string]dbus.Variant)
			if !ok {
				return nil, false
			}
			return results, true
		case <-t.C:
			return nil, false
		}
	}
}
