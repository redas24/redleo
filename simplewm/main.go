// Command simplewm is a minimal floating X11 window manager.
//
// It does not reparent clients or draw decorations — it just lets you
// move, resize, raise and close whatever windows are already on screen.
// Controls (held together with the Super/Mod4 key):
//
//	Super + Left-drag   move the window under the pointer
//	Super + Right-drag  resize the window under the pointer
//	Super + click       raise and focus the window under the pointer
//	Super + Shift + C   close the focused window (WM_DELETE_WINDOW, or kill)
package main

import (
	"log"
	"os"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// modKey is the modifier held down for every window-manager shortcut.
const modKey = xproto.ModMask4 // Super / "Windows" key

func main() {
	if os.Geteuid() == 0 {
		log.Printf("warning: running simplewm as root is not recommended")
	}

	configPath, err := defaultConfigPath()
	if err != nil {
		log.Fatalf("locating config file: %v", err)
	}
	cfg, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("loading config %s: %v", configPath, err)
	}

	conn, err := xgb.NewConn()
	if err != nil {
		log.Fatalf("connecting to X server: %v", err)
	}
	defer conn.Close()

	wm, err := newWM(conn, cfg)
	if err != nil {
		log.Fatalf("starting window manager: %v", err)
	}
	wm.run()
}
