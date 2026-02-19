package main

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Config struct {
	MountBase        string `json:"mount_base"`
	Terminal         string `json:"terminal"`
	RemoteMountPath  string `json:"remote_mount_path"`
}

type App struct {
	ctx            context.Context
	mountHistory   []string
	trayUpdate     func() // called after history changes; set by platform tray impl
	config         Config
	startupCommand string // "mount", "connect", nebo ""
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.loadConfig()
	a.loadMountHistory()
	a.setupTray()
	a.setupHotkey()
	go a.listenSocket()
	if a.startupCommand != "" {
		go func() {
			time.Sleep(300 * time.Millisecond)
			switch a.startupCommand {
			case "connect":
				a.ShowConnectDialog()
			default:
				a.ShowMountDialog()
			}
		}()
	}
}

// listenSocket naslouchá na Unix socketu — fallback pro GNOME Wayland.
// GNOME custom shortcuts: daktela-gui-tools --show-mount / daktela-gui-tools --show-connect
func (a *App) listenSocket() {
	path := socketPath()
	os.Remove(path)
	listener, err := net.Listen("unix", path)
	if err != nil {
		return
	}
	defer listener.Close()
	defer os.Remove(path)

	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go func() {
			buf := make([]byte, 16)
			n, _ := conn.Read(buf)
			conn.Close()
			if strings.TrimSpace(string(buf[:n])) == "connect" {
				a.ShowConnectDialog()
			} else {
				a.ShowMountDialog()
			}
		}()
	}
}

// ── Config ────────────────────────────────────────────────────────────────────

func (a *App) configDir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		d = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(d, "daktela-gui-tools")
}

func (a *App) configFilePath() string {
	return filepath.Join(a.configDir(), "config.json")
}

func (a *App) loadConfig() {
	home, _ := os.UserHomeDir()
	a.config = Config{
		MountBase:       filepath.Join(home, "Projects", "Daktela"),
		Terminal:        defaultTerminal,
		RemoteMountPath: "/var/lib/daktela/custom",
	}
	data, err := os.ReadFile(a.configFilePath())
	if err != nil {
		a.saveConfigFile()
		return
	}
	json.Unmarshal(data, &a.config)
}

func (a *App) saveConfigFile() {
	path := a.configFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	data, _ := json.MarshalIndent(a.config, "", "  ")
	os.WriteFile(path, data, 0644)
}

// ── History ───────────────────────────────────────────────────────────────────

func (a *App) historyFilePath() string {
	return filepath.Join(a.configDir(), "history.json")
}

func (a *App) loadMountHistory() {
	data, err := os.ReadFile(a.historyFilePath())
	if err != nil {
		a.mountHistory = []string{}
		return
	}
	var v struct {
		History []string `json:"history"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		a.mountHistory = []string{}
		return
	}
	a.mountHistory = v.History
}

func (a *App) saveMountHistoryFile() {
	path := a.historyFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	v := struct {
		History []string `json:"history"`
	}{History: a.mountHistory}
	data, _ := json.Marshal(v)
	os.WriteFile(path, data, 0644)
}

func (a *App) GetMountHistory() []string {
	return a.mountHistory
}

// SaveMount přesune name na začátek (nejnovější nahoře), uloží, aktualizuje tray.
func (a *App) SaveMount(name string) {
	filtered := a.mountHistory[:0]
	for _, item := range a.mountHistory {
		if item != name {
			filtered = append(filtered, item)
		}
	}
	a.mountHistory = append([]string{name}, filtered...)
	if len(a.mountHistory) > 50 {
		a.mountHistory = a.mountHistory[:50]
	}
	a.saveMountHistoryFile()
	if a.trayUpdate != nil {
		a.trayUpdate()
	}
}

// ── Mount ─────────────────────────────────────────────────────────────────────

func (a *App) RunMount(name string) string {
	base := filepath.Join(a.config.MountBase, name)
	var sb strings.Builder

	run := func(args ...string) {
		sb.WriteString("$ " + strings.Join(args, " ") + "\n")
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if len(out) > 0 {
			sb.WriteString(strings.TrimRight(string(out), "\n") + "\n")
		}
		if err != nil {
			sb.WriteString("error: " + err.Error() + "\n")
		}
		sb.WriteString("\n")
	}

	run(unmountArgs(base)...)
	run("mkdir", "-p", base)
	remotePath := strings.TrimRight(a.config.RemoteMountPath, "/") + "/"
	run("sshfs",
		"root@"+name+".daktela.com:"+remotePath,
		base,
		"-o", "ServerAliveInterval=10,ServerAliveCountMax=6")

	return strings.TrimSpace(sb.String())
}

// ExecuteMount voláno z trayi — přeřadí historii a zobrazí result okno.
func (a *App) ExecuteMount(name string) {
	a.SaveMount(name)
	runtime.WindowSetSize(a.ctx, 500, 300)
	runtime.WindowSetTitle(a.ctx, "Mount: "+name)
	runtime.WindowShow(a.ctx)
	runtime.EventsEmit(a.ctx, "show-mount-result", name)
}

func (a *App) ShowMountDialog() {
	runtime.WindowSetSize(a.ctx, 360, 200)
	runtime.WindowSetTitle(a.ctx, "Mount")
	runtime.WindowShow(a.ctx)
	runtime.EventsEmit(a.ctx, "show-mount-dialog")
}

// ── Connect SSH ───────────────────────────────────────────────────────────────

// ExecuteConnect uloží do historie, otevře terminál s SSH příkazem.
func (a *App) ExecuteConnect(name string) {
	a.SaveMount(name) // sdílená historie
	parts := strings.Fields(a.config.Terminal)
	if len(parts) == 0 {
		return
	}
	args := append(parts[1:], "ssh", "root@"+name+".daktela.com")
	exec.Command(parts[0], args...).Start()
}

func (a *App) ShowConnectDialog() {
	runtime.WindowSetSize(a.ctx, 360, 200)
	runtime.WindowSetTitle(a.ctx, "Connect SSH")
	runtime.WindowShow(a.ctx)
	runtime.EventsEmit(a.ctx, "show-connect-dialog")
}

// ── Execute command ───────────────────────────────────────────────────────────

func (a *App) ExecuteCommand(command string) string {
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "Error: " + err.Error() + "\n" + string(output)
	}
	return string(output)
}

func (a *App) HideWindow() {
	runtime.WindowHide(a.ctx)
}
