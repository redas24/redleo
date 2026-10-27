package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/screensaver"
	"github.com/jezek/xgb/xproto"
)

// wm holds the window manager's connection and runtime state.
type wm struct {
	conn *xgb.Conn
	root xproto.Window

	atomWMProtocols    xproto.Atom
	atomWMDeleteWindow xproto.Atom

	// binds maps a grabbed keycode to the bindings registered for it
	// (there can be more than one combo sharing a physical key, e.g.
	// super-l and super-shift-l).
	binds map[xproto.Keycode][]runtimeBind

	lockCommand []string
	lockMu      sync.Mutex // held for the duration of a lock, to debounce

	// Drag state, valid only while a button grabbed by modKey is held down.
	dragWin            xproto.Window
	dragButton         xproto.Button
	dragStartRootX     int16
	dragStartRootY     int16
	dragWinX, dragWinY int16
	dragWinW, dragWinH uint16
}

// runtimeBind is a config bind resolved to a concrete keycode's modifier
// state and command.
type runtimeBind struct {
	mods    uint16
	command []string
}

// newWM connects the window manager's shortcuts to the root window and
// applies cfg. It fails if another window manager already owns the
// pointer/button grabs it needs.
func newWM(conn *xgb.Conn, cfg *config) (*wm, error) {
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
		binds:              make(map[xproto.Keycode][]runtimeBind),
		lockCommand:        cfg.lockCommand,
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

	for _, b := range cfg.binds {
		keycode, err := keycodeForKeysym(conn, b.keysym)
		if err != nil {
			log.Printf("bind for keysym 0x%x: %v (skipping)", b.keysym, err)
			continue
		}
		if err := xproto.GrabKeyChecked(conn, true, root, b.mods, keycode,
			xproto.GrabModeAsync, xproto.GrabModeAsync).Check(); err != nil {
			log.Printf("grabbing bind (keycode %d, mods 0x%x): %v (skipping)", keycode, b.mods, err)
			continue
		}
		w.binds[keycode] = append(w.binds[keycode], runtimeBind{mods: b.mods, command: b.command})
	}

	for _, cmd := range cfg.autostart {
		w.spawnCommand(cmd)
	}

	go w.watchIdle(cfg.idleSeconds)

	return w, nil
}

// run is the main event loop. It blocks forever handling X events.
func (w *wm) run() {
	log.Printf("simplewm running")
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

// onKeyPress dispatches to whichever configured bind matches both the
// keycode and the modifier state that were actually pressed.
func (w *wm) onKeyPress(e xproto.KeyPressEvent) {
	const relevantMods = xproto.ModMask4 | xproto.ModMaskShift
	for _, rb := range w.binds[e.Detail] {
		if uint16(e.State)&relevantMods == rb.mods {
			w.spawnCommand(rb.command)
			return
		}
	}
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

// watchIdle locks the screen after idleSeconds of no keyboard/mouse
// input, using the X MIT-SCREEN-SAVER extension to query idle time. A
// non-positive idleSeconds disables it entirely.
func (w *wm) watchIdle(idleSeconds int) {
	if idleSeconds <= 0 {
		return
	}
	if err := screensaver.Init(w.conn); err != nil {
		log.Printf("idle auto-lock disabled: %v", err)
		return
	}

	threshold := uint32(idleSeconds) * 1000
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		info, err := screensaver.QueryInfo(w.conn, xproto.Drawable(w.root)).Reply()
		if err != nil {
			log.Printf("querying idle time: %v", err)
			continue
		}
		if info.MsSinceUserInput >= threshold {
			go w.lockScreen()
		}
	}
}

// lockScreen runs the configured lock command, debounced so repeated
// triggers (idle polling, a repeated keypress) never start it twice at
// once.
func (w *wm) lockScreen() {
	if !w.lockMu.TryLock() {
		return
	}
	defer w.lockMu.Unlock()

	if len(w.lockCommand) == 0 {
		return
	}
	path, err := exec.LookPath(w.lockCommand[0])
	if err != nil {
		log.Printf("locking screen: %v", err)
		return
	}
	if err := exec.Command(path, w.lockCommand[1:]...).Run(); err != nil {
		log.Printf("running screen locker: %v", err)
	}
}

// spawnCommand runs a config command line. "$TERMINAL" and "$LOCK" are
// resolved specially; everything else is looked up in $PATH and run
// directly -- never through a shell, so config files can't be used for
// shell-injection tricks even though they're plain, user-editable text.
func (w *wm) spawnCommand(tokens []string) {
	if len(tokens) == 0 {
		return
	}
	switch tokens[0] {
	case "$TERMINAL":
		w.spawnTerminal(tokens[1:])
	case "$LOCK":
		go w.lockScreen()
	default:
		spawnGeneric(tokens[0], tokens[1:])
	}
}

// spawnTerminal launches the user's terminal of choice, passing extraArgs
// through to it. It tries $SIMPLEWM_TERMINAL first, then st (see
// redpac/), then xterm -- whichever is found first in $PATH.
func (w *wm) spawnTerminal(extraArgs []string) {
	candidates := []string{"st", "xterm"}
	if custom := os.Getenv("SIMPLEWM_TERMINAL"); custom != "" {
		candidates = append([]string{custom}, candidates...)
	}
	for _, name := range candidates {
		if spawnGeneric(name, extraArgs) {
			return
		}
	}
	log.Printf("no terminal found (tried $SIMPLEWM_TERMINAL, st, xterm)")
}

// spawnGeneric looks up name in $PATH and runs it, detached from
// simplewm so it keeps running after simplewm exits. It reports whether
// the command was found and started.
func spawnGeneric(name string, args []string) bool {
	path, err := exec.LookPath(name)
	if err != nil {
		return false
	}
	cmd := exec.Command(path, args...)
	if err := cmd.Start(); err != nil {
		log.Printf("spawning %s: %v", name, err)
		return false
	}
	go cmd.Wait() // reap the child without blocking the event loop
	return true
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
// mapping assigns to keysym, so a key grab survives whatever physical
// keyboard layout is active.
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
