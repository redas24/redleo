package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// xkReturn is the X keysym for the Return/Enter key (from <X11/keysymdef.h>).
const xkReturn xproto.Keysym = 0xff0d

// wm holds the window manager's connection state and the window currently
// being dragged (moved or resized), if any.
type wm struct {
	conn *xgb.Conn
	root xproto.Window

	atomWMProtocols    xproto.Atom
	atomWMDeleteWindow xproto.Atom

	// terminalKeycode is the Return key's current keycode, grabbed with
	// modKey to spawn a terminal. 0 if Return has no keycode on this
	// keyboard mapping (terminal spawning is then simply disabled).
	terminalKeycode xproto.Keycode

	// Drag state, valid only while a button grabbed by modKey is held down.
	dragWin            xproto.Window
	dragButton         xproto.Button
	dragStartRootX     int16
	dragStartRootY     int16
	dragWinX, dragWinY int16
	dragWinW, dragWinH uint16
}

// newWM connects the window manager's shortcuts to the root window. It
// fails if another window manager already owns the pointer/button grabs
// it needs.
func newWM(conn *xgb.Conn) (*wm, error) {
	root := xproto.Setup(conn).DefaultScreen(conn).Root

	wmProtocols, err := internAtom(conn, "WM_PROTOCOLS")
	if err != nil {
		return nil, err
	}
	wmDeleteWindow, err := internAtom(conn, "WM_DELETE_WINDOW")
	if err != nil {
		return nil, err
	}

	w := &wm{
		conn:               conn,
		root:               root,
		atomWMProtocols:    wmProtocols,
		atomWMDeleteWindow: wmDeleteWindow,
	}

	// Grab Super+Button1 (move), Super+Button2 (close) and Super+Button3
	// (resize) on the root window. owner_events=true and a target window
	// of None in the actual grab means these fire no matter which client
	// window is under the pointer.
	const eventMask = xproto.EventMaskButtonPress |
		xproto.EventMaskButtonRelease |
		xproto.EventMaskPointerMotion
	for _, button := range []byte{1, 2, 3} {
		err := xproto.GrabButtonChecked(
			conn, true, root, eventMask,
			xproto.GrabModeAsync, xproto.GrabModeAsync,
			xproto.WindowNone, xproto.CursorNone,
			button, modKey,
		).Check()
		if err != nil {
			return nil, fmt.Errorf("grabbing button %d: %w (is another window manager already running?)", button, err)
		}
	}

	keycode, err := keycodeForKeysym(conn, xkReturn)
	if err != nil {
		log.Printf("looking up Return keycode: %v (Super+Return will not spawn a terminal)", err)
	} else {
		if err := xproto.GrabKeyChecked(conn, true, root, modKey, keycode,
			xproto.GrabModeAsync, xproto.GrabModeAsync).Check(); err != nil {
			log.Printf("grabbing Super+Return: %v (Super+Return will not spawn a terminal)", err)
		} else {
			w.terminalKeycode = keycode
		}
	}

	return w, nil
}

// run is the main event loop. It blocks forever handling X events.
func (w *wm) run() {
	log.Printf("simplewm running (Super+drag to move/resize, Super+middle-click to close, Super+Return to open a terminal)")
	for {
		event, err := w.conn.WaitForEvent()
		if err != nil {
			log.Printf("X error: %v", err)
			continue
		}

		switch e := event.(type) {
		case xproto.ButtonPressEvent:
			w.onButtonPress(e)
		case xproto.MotionNotifyEvent:
			w.onMotionNotify(e)
		case xproto.ButtonReleaseEvent:
			w.onButtonRelease(e)
		case xproto.KeyPressEvent:
			w.onKeyPress(e)
		}
	}
}

// onButtonPress starts a drag (move/resize), closes a window, or just
// raises and focuses it, depending on which button was pressed.
func (w *wm) onButtonPress(e xproto.ButtonPressEvent) {
	target := e.Child
	if target == xproto.WindowNone {
		return
	}

	w.raiseAndFocus(target)

	if e.Detail == 2 { // Super + middle-click: close
		w.closeWindow(target)
		return
	}
	if e.Detail != 1 && e.Detail != 3 { // only left (move) and right (resize) drag
		return
	}

	geom, err := xproto.GetGeometry(w.conn, xproto.Drawable(target)).Reply()
	if err != nil {
		log.Printf("querying geometry of %d: %v", target, err)
		return
	}

	w.dragWin = target
	w.dragButton = e.Detail
	w.dragStartRootX = e.RootX
	w.dragStartRootY = e.RootY
	w.dragWinX, w.dragWinY = geom.X, geom.Y
	w.dragWinW, w.dragWinH = geom.Width, geom.Height
}

// onMotionNotify moves or resizes the window being dragged, following the
// pointer by however far it has traveled since the button went down.
func (w *wm) onMotionNotify(e xproto.MotionNotifyEvent) {
	if w.dragWin == xproto.WindowNone {
		return
	}

	dx := e.RootX - w.dragStartRootX
	dy := e.RootY - w.dragStartRootY

	switch w.dragButton {
	case 1: // move
		xproto.ConfigureWindow(w.conn, w.dragWin,
			xproto.ConfigWindowX|xproto.ConfigWindowY,
			[]uint32{
				uint32(int32(w.dragWinX) + int32(dx)),
				uint32(int32(w.dragWinY) + int32(dy)),
			},
		)
	case 3: // resize
		width := clampMin1(int32(w.dragWinW) + int32(dx))
		height := clampMin1(int32(w.dragWinH) + int32(dy))
		xproto.ConfigureWindow(w.conn, w.dragWin,
			xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
			[]uint32{uint32(width), uint32(height)},
		)
	}
}

// onButtonRelease ends the current drag, if any.
func (w *wm) onButtonRelease(e xproto.ButtonReleaseEvent) {
	w.dragWin = xproto.WindowNone
}

// onKeyPress spawns a terminal when Super+Return is pressed.
func (w *wm) onKeyPress(e xproto.KeyPressEvent) {
	if w.terminalKeycode != 0 && e.Detail == w.terminalKeycode {
		spawnTerminal()
	}
}

// spawnTerminal launches the user's terminal of choice, detached from
// simplewm so it keeps running after simplewm exits. It tries
// $SIMPLEWM_TERMINAL first, falling back to st (see redpac/) and then
// xterm, whichever is found first in $PATH.
func spawnTerminal() {
	candidates := []string{"st", "xterm"}
	if custom := os.Getenv("SIMPLEWM_TERMINAL"); custom != "" {
		candidates = append([]string{custom}, candidates...)
	}

	for _, name := range candidates {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		cmd := exec.Command(path)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
		if err := cmd.Start(); err != nil {
			log.Printf("spawning %s: %v", name, err)
			continue
		}
		go cmd.Wait() // reap the child without blocking the event loop
		return
	}
	log.Printf("no terminal found (tried $SIMPLEWM_TERMINAL, st, xterm)")
}

// raiseAndFocus brings win to the top of the stack and gives it input focus.
func (w *wm) raiseAndFocus(win xproto.Window) {
	xproto.ConfigureWindow(w.conn, win,
		xproto.ConfigWindowStackMode,
		[]uint32{uint32(xproto.StackModeAbove)},
	)
	xproto.SetInputFocus(w.conn, xproto.InputFocusPointerRoot, win, xproto.TimeCurrentTime)
}

// closeWindow asks win to close via the ICCCM WM_DELETE_WINDOW protocol if
// it supports it, otherwise it forcibly kills the client connection.
func (w *wm) closeWindow(win xproto.Window) {
	if w.supportsDeleteWindow(win) {
		event := xproto.ClientMessageEvent{
			Format: 32,
			Window: win,
			Type:   w.atomWMProtocols,
			Data: xproto.ClientMessageDataUnionData32New([]uint32{
				uint32(w.atomWMDeleteWindow),
				uint32(xproto.TimeCurrentTime),
				0, 0, 0,
			}),
		}
		xproto.SendEvent(w.conn, false, win, xproto.EventMaskNoEvent, string(event.Bytes()))
		return
	}

	xproto.KillClient(w.conn, uint32(win))
}

// supportsDeleteWindow reports whether win advertises WM_DELETE_WINDOW in
// its WM_PROTOCOLS property.
func (w *wm) supportsDeleteWindow(win xproto.Window) bool {
	reply, err := xproto.GetProperty(
		w.conn, false, win, w.atomWMProtocols,
		xproto.AtomAtom, 0, 64,
	).Reply()
	if err != nil || reply == nil {
		return false
	}

	for i := 0; i+4 <= len(reply.Value); i += 4 {
		atom := xproto.Atom(xgb.Get32(reply.Value[i:]))
		if atom == w.atomWMDeleteWindow {
			return true
		}
	}
	return false
}

// internAtom looks up (creating if necessary) the X atom for name.
func internAtom(conn *xgb.Conn, name string) (xproto.Atom, error) {
	reply, err := xproto.InternAtom(conn, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0, fmt.Errorf("interning atom %q: %w", name, err)
	}
	return reply.Atom, nil
}

// keycodeForKeysym looks up the keycode the X server's current keyboard
// mapping assigns to keysym (e.g. Return), so a key grab survives
// whatever physical keyboard layout is active.
func keycodeForKeysym(conn *xgb.Conn, keysym xproto.Keysym) (xproto.Keycode, error) {
	setup := xproto.Setup(conn)
	count := byte(setup.MaxKeycode - setup.MinKeycode + 1)

	mapping, err := xproto.GetKeyboardMapping(conn, setup.MinKeycode, count).Reply()
	if err != nil {
		return 0, fmt.Errorf("getting keyboard mapping: %w", err)
	}

	perKeycode := int(mapping.KeysymsPerKeycode)
	if perKeycode == 0 {
		return 0, fmt.Errorf("keyboard mapping reports 0 keysyms per keycode")
	}

	for i := 0; i < int(count); i++ {
		base := i * perKeycode
		for j := 0; j < perKeycode; j++ {
			if mapping.Keysyms[base+j] == keysym {
				return xproto.Keycode(int(setup.MinKeycode) + i), nil
			}
		}
	}
	return 0, fmt.Errorf("keysym 0x%x is not mapped on this keyboard", keysym)
}

// clampMin1 keeps a window dimension from shrinking to zero or below.
func clampMin1(v int32) int32 {
	if v < 1 {
		return 1
	}
	return v
}
