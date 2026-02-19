# daktela-gui-tools

A GUI tool for managing Daktela servers — quickly mount remote directories via SSHFS or open SSH terminal sessions. Runs as a system tray application.

## Features

- **Mount** — mounts a server's remote directory (`/var/lib/daktela/custom/`) via SSHFS into a local folder
- **Connect SSH** — opens a terminal with an SSH connection to the server
- **History** — last 10 used servers accessible directly from the tray menu
- **Global hotkeys** — `Ctrl+Shift+D` (Mount), `Ctrl+Shift+S` (Connect SSH)
- **System tray** — runs in the background, window appears only when needed

---

## Requirements

### Linux
- `sshfs`
- `fusermount` (part of the `fuse` or `fuse3` package)
- GTK 3 + WebKitGTK (see build instructions below)

### macOS
- `sshfs` — easiest via [Homebrew](https://brew.sh): `brew install sshfs`
- Xcode Command Line Tools: `xcode-select --install`

---

## Installation

### From source

**Build dependencies:**

```bash
# Go 1.23+
# Node.js 20+
go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt install libwebkit2gtk-4.0-dev libgtk-3-dev pkg-config
```

**Linux (Ubuntu 24.04+ / Wayland with WebKitGTK 4.1):**
```bash
sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev pkg-config
```

**Build:**
```bash
git clone https://github.com/daktela/daktela-gui-tools.git
cd daktela-gui-tools

# Linux X11
wails build -o daktela-gui-tools

# Linux Wayland (WebKitGTK 4.1)
wails build -tags webkit2_41 -o daktela-gui-tools-wayland

# macOS
wails build -o daktela-gui-tools-darwin
```

The output binary will be in `build/bin/`.

### From GitHub Releases

Download a pre-built binary from [Releases](../../releases):

| File | Platform |
|------|----------|
| `daktela-gui-tools` | Linux X11 |
| `daktela-gui-tools-wayland` | Linux Wayland |
| `daktela-gui-tools-darwin.zip` | macOS (extract and run the `.app`) |

```bash
# Linux — make executable and move to PATH
chmod +x daktela-gui-tools
sudo mv daktela-gui-tools /usr/local/bin/
```

---

## Usage

```bash
# Normal launch (runs in the background in the tray)
daktela-gui-tools

# Open the Mount dialog immediately
daktela-gui-tools --show-mount

# Open the Connect SSH dialog immediately
daktela-gui-tools --show-connect
```

If the application is **already running**, `--show-mount` and `--show-connect` forward the command to the running instance via a Unix socket and the second instance exits immediately. This is useful for setting up system-level keyboard shortcuts (see below).

---

## Configuration

On first launch, a configuration file is created automatically:

| OS | Path |
|----|------|
| Linux | `~/.config/daktela-gui-tools/config.json` |
| macOS | `~/Library/Application Support/daktela-gui-tools/config.json` |

### Example `config.json`

```json
{
  "mount_base": "/home/user/Projects/Daktela",
  "terminal": "gnome-terminal --"
}
```

### Fields

| Field | Description |
|-------|-------------|
| `mount_base` | Local directory where server disks are mounted. Each server is mounted as a subdirectory — e.g. `mount_base/server-name`. |
| `terminal` | Command used to open a terminal. The SSH command is appended as arguments after this string. |

### `terminal` values by environment

| Terminal | Value |
|----------|-------|
| GNOME Terminal | `gnome-terminal --` |
| Kitty | `kitty --` |
| Alacritty | `alacritty -e` |
| Konsole (KDE) | `konsole -e` |
| xterm | `xterm -e` |
| macOS kitty | `kitty --` |

> **macOS:** Terminal.app and iTerm2 do not support receiving a command as a command-line argument. We recommend installing [kitty](https://sw.kovidgoyal.net/kitty/) via Homebrew (`brew install --cask kitty`) and setting `"terminal": "kitty --"`.

---

## Global Hotkeys

### Linux X11 / XWayland
Hotkeys work automatically via XGrabKey — no configuration needed.

| Shortcut | Action |
|----------|--------|
| `Ctrl+Shift+D` | Open Mount dialog |
| `Ctrl+Shift+S` | Open Connect SSH dialog |

### Linux — pure Wayland (no XWayland)
Shortcuts are registered via the `xdg-desktop-portal` GlobalShortcuts API. The desktop environment must support the portal (GNOME 43+, KDE Plasma 5.27+).

Alternatively, configure system shortcuts manually in your desktop settings:

```
# GNOME — Settings → Keyboard → Custom Shortcuts
Command: daktela-gui-tools --show-mount      Shortcut: Ctrl+Shift+D
Command: daktela-gui-tools --show-connect    Shortcut: Ctrl+Shift+S
```

### macOS
Hotkeys work automatically via a global event monitor (`golang.design/x/hotkey`).

---

## How Mount Works

After entering a server name (e.g. `customer`), the application:

1. Unmounts any existing mount: `fusermount -u ~/Projects/Daktela/customer`
2. Creates the target directory: `mkdir -p ~/Projects/Daktela/customer`
3. Mounts the remote directory via SSHFS:
   ```
   sshfs root@customer.daktela.com:/var/lib/daktela/custom/ ~/Projects/Daktela/customer
        -o ServerAliveInterval=10,ServerAliveCountMax=6
   ```

The output of all commands is displayed in the application window.

---

## Application Files

```
~/.config/daktela-gui-tools/
├── config.json    # configuration (mount_base, terminal)
└── history.json   # history of the last 50 servers
```
