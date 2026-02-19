package main

import (
	"embed"
	"fmt"
	"net"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func socketPath() string {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = fmt.Sprintf("/tmp/daktela-gui-tools-%d", os.Getuid())
	}
	return dir + "/daktela-gui-tools.sock"
}

func main() {
	// Zpracuj --show-mount / --show-connect: signalizuj běžící instanci a skonči,
	// nebo pokud žádná neběží, spusť app a dialog zobraz automaticky.
	startupCmd := ""
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--show-mount":
			startupCmd = "mount"
		case "--show-connect":
			startupCmd = "connect"
		}
	}

	if startupCmd != "" {
		conn, err := net.Dial("unix", socketPath())
		if err == nil {
			conn.Write([]byte(startupCmd))
			conn.Close()
			return
		}
		// Žádná instance neběží — spustí se níže
	}

	app := NewApp()
	app.startupCommand = startupCmd

	err := wails.Run(&options.App{
		Title:  "Execute Command",
		Width:  500,
		Height: 260,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:  &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:         app.startup,
		StartHidden:       true,
		HideWindowOnClose: true,
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			Icon: icon,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
