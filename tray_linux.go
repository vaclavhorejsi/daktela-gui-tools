package main

import (
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/prop"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	sniPath      = dbus.ObjectPath("/StatusNotifierItem")
	menuPath     = dbus.ObjectPath("/MenuBar")
	sniIface     = "org.kde.StatusNotifierItem"
	menuIface    = "com.canonical.dbusmenu"
	watcherSvc   = "org.kde.StatusNotifierWatcher"
	watcherPath  = dbus.ObjectPath("/StatusNotifierWatcher")
	watcherIface = "org.kde.StatusNotifierWatcher"
)

type MenuNode struct {
	ID       int32
	Props    map[string]dbus.Variant
	Children []dbus.Variant
}

type MenuItem struct {
	ID    int32
	Props map[string]dbus.Variant
}

type MenuEvent struct {
	ID        int32
	EventID   string
	Data      dbus.Variant
	Timestamp uint32
}

type trayItem struct{ app *App }
type trayMenu struct {
	app      *App
	conn     *dbus.Conn
	revision uint32
}

func (m *trayMenu) notifyLayoutUpdated() {
	m.revision++
	if m.conn != nil {
		m.conn.Emit(menuPath, menuIface+".LayoutUpdated", m.revision, int32(0))
	}
}

func (t *trayItem) Activate(x, y int32) *dbus.Error {
	runtime.WindowShow(t.app.ctx)
	return nil
}

func (t *trayItem) ContextMenu(x, y int32) *dbus.Error        { return nil }
func (t *trayItem) SecondaryActivate(x, y int32) *dbus.Error  { return nil }
func (t *trayItem) Scroll(delta int32, dir string) *dbus.Error { return nil }

func (m *trayMenu) GetLayout(parentId, depth int32, props []string) (uint32, MenuNode, *dbus.Error) {
	children := []dbus.Variant{
		// Execute command — skrytý, kód zachován pro budoucí použití
		dbus.MakeVariant(MenuNode{
			ID: 1,
			Props: map[string]dbus.Variant{
				"label":   dbus.MakeVariant("Execute command"),
				"enabled": dbus.MakeVariant(true),
				"visible": dbus.MakeVariant(false),
			},
			Children: []dbus.Variant{},
		}),
		dbus.MakeVariant(MenuNode{
			ID:       2,
			Props:    map[string]dbus.Variant{"type": dbus.MakeVariant("separator")},
			Children: []dbus.Variant{},
		}),
		dbus.MakeVariant(MenuNode{
			ID: 6,
			Props: map[string]dbus.Variant{
				"label":   dbus.MakeVariant("Connect SSH  (Ctrl+Shift+S)"),
				"enabled": dbus.MakeVariant(true),
				"visible": dbus.MakeVariant(true),
			},
			Children: []dbus.Variant{},
		}),
		dbus.MakeVariant(MenuNode{
			ID: 4,
			Props: map[string]dbus.Variant{
				"label":   dbus.MakeVariant("Mount  (Ctrl+Shift+D)"),
				"enabled": dbus.MakeVariant(true),
				"visible": dbus.MakeVariant(true),
			},
			Children: []dbus.Variant{},
		}),
		// Oddělovač před historií
		dbus.MakeVariant(MenuNode{
			ID:       7,
			Props:    map[string]dbus.Variant{"type": dbus.MakeVariant("separator")},
			Children: []dbus.Variant{},
		}),
	}

	// Historie — každá položka má submenu s Mount a Connect SSH
	// IDs: parent 10-19 | Mount child 100-109 | Connect child 200-209
	history := m.app.mountHistory
	limit := len(history)
	if limit > 10 {
		limit = 10
	}
	for i := 0; i < limit; i++ {
		children = append(children, dbus.MakeVariant(MenuNode{
			ID: int32(10 + i),
			Props: map[string]dbus.Variant{
				"label":            dbus.MakeVariant(history[i]),
				"enabled":          dbus.MakeVariant(true),
				"visible":          dbus.MakeVariant(true),
				"children-display": dbus.MakeVariant("submenu"),
			},
			Children: []dbus.Variant{
				dbus.MakeVariant(MenuNode{
					ID: int32(100 + i),
					Props: map[string]dbus.Variant{
						"label":   dbus.MakeVariant("Mount"),
						"enabled": dbus.MakeVariant(true),
						"visible": dbus.MakeVariant(true),
					},
					Children: []dbus.Variant{},
				}),
				dbus.MakeVariant(MenuNode{
					ID: int32(200 + i),
					Props: map[string]dbus.Variant{
						"label":   dbus.MakeVariant("Connect SSH"),
						"enabled": dbus.MakeVariant(true),
						"visible": dbus.MakeVariant(true),
					},
					Children: []dbus.Variant{},
				}),
			},
		}))
	}

	children = append(children,
		dbus.MakeVariant(MenuNode{
			ID:       5,
			Props:    map[string]dbus.Variant{"type": dbus.MakeVariant("separator")},
			Children: []dbus.Variant{},
		}),
		dbus.MakeVariant(MenuNode{
			ID: 3,
			Props: map[string]dbus.Variant{
				"label":   dbus.MakeVariant("Exit"),
				"enabled": dbus.MakeVariant(true),
				"visible": dbus.MakeVariant(true),
			},
			Children: []dbus.Variant{},
		}),
	)

	return m.revision, MenuNode{
		ID:       0,
		Props:    map[string]dbus.Variant{},
		Children: children,
	}, nil
}

func (m *trayMenu) GetGroupProperties(ids []int32, props []string) ([]MenuItem, *dbus.Error) {
	return []MenuItem{}, nil
}

func (m *trayMenu) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	return dbus.MakeVariant(""), nil
}

func (m *trayMenu) Event(id int32, eventId string, data dbus.Variant, timestamp uint32) *dbus.Error {
	if eventId != "clicked" {
		return nil
	}
	switch {
	case id == 1:
		runtime.WindowShow(m.app.ctx)
	case id == 3:
		runtime.Quit(m.app.ctx)
	case id == 4:
		go m.app.ShowMountDialog()
	case id == 6:
		go m.app.ShowConnectDialog()
	case id >= 100 && id <= 109: // Mount z historie
		idx := int(id) - 100
		if idx < len(m.app.mountHistory) {
			go m.app.ExecuteMount(m.app.mountHistory[idx])
		}
	case id >= 200 && id <= 209: // Connect SSH z historie
		idx := int(id) - 200
		if idx < len(m.app.mountHistory) {
			go m.app.ExecuteConnect(m.app.mountHistory[idx])
		}
	}
	return nil
}

func (m *trayMenu) EventGroup(events []MenuEvent) ([]int32, *dbus.Error) {
	for _, e := range events {
		m.Event(e.ID, e.EventID, e.Data, e.Timestamp)
	}
	return []int32{}, nil
}

func (m *trayMenu) AboutToShow(id int32) (bool, *dbus.Error) {
	return false, nil
}

func (m *trayMenu) AboutToShowGroup(ids []int32) ([]int32, []int32, *dbus.Error) {
	return []int32{}, []int32{}, nil
}

func (a *App) setupTray() {
	go a.runTray()
}

func (a *App) runTray() {
	conn, err := dbus.SessionBus()
	if err != nil {
		return
	}

	svcName := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	reply, err := conn.RequestName(svcName, dbus.NameFlagDoNotQueue)
	if err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		return
	}

	menu := &trayMenu{app: a, conn: conn, revision: 1}
	a.trayUpdate = func() { menu.notifyLayoutUpdated() }

	conn.Export(&trayItem{app: a}, sniPath, sniIface)
	conn.Export(menu, menuPath, menuIface)

	prop.Export(conn, sniPath, prop.Map{
		sniIface: {
			"Category":   {Value: "ApplicationStatus", Writable: false, Emit: prop.EmitTrue},
			"Id":         {Value: svcName, Writable: false, Emit: prop.EmitTrue},
			"Title":      {Value: "Mountly", Writable: false, Emit: prop.EmitTrue},
			"Status":     {Value: "Active", Writable: false, Emit: prop.EmitTrue},
			"IconName":   {Value: "utilities-terminal", Writable: false, Emit: prop.EmitTrue},
			"Menu":       {Value: menuPath, Writable: false, Emit: prop.EmitTrue},
			"ItemIsMenu": {Value: false, Writable: false, Emit: prop.EmitTrue},
			"WindowId":   {Value: int32(0), Writable: false, Emit: prop.EmitTrue},
		},
	})

	conn.Object(watcherSvc, watcherPath).
		Call(watcherIface+".RegisterStatusNotifierItem", 0, svcName)

	select {}
}
