# simplewm

A minimal floating X11 window manager written in Go, using the pure-Go
[`jezek/xgb`](https://github.com/jezek/xgb) X protocol bindings (no Xlib/cgo
required). It does not reparent clients or draw decorations — it just lets
you move, resize, raise, focus, close, and open windows with the keyboard
and mouse. The move/resize/raise behavior is a direct port of the classic
[tinywm](https://github.com/mackstann/tinywm).

## Controls

All mouse shortcuts are held together with Super (Mod4, the "Windows" key):

| Shortcut                | Action                          |
|--------------------------|----------------------------------|
| `Super` + Left-drag      | Move the window under the pointer |
| `Super` + Right-drag     | Resize the window under the pointer |
| `Super` + click          | Raise and focus the window under the pointer |
| `Super` + middle-click   | Close that window (`WM_DELETE_WINDOW`, or kill) |
| `Super` + `Return`       | Open a terminal |

## Running

```sh
go build .
./simplewm
```

Run it as your window manager (e.g. from `.xinitrc`), or inside a nested
server such as Xephyr for testing:

```sh
Xephyr -screen 1280x800 :1 &
DISPLAY=:1 ./simplewm
```

## Terminal

`Super+Return` looks for, in order: `$SIMPLEWM_TERMINAL`, `st`, then
`xterm`, and runs whichever it finds first in `$PATH`. `st` (the suckless
terminal) is the intended default — install it with
[`redpac`](../redpac), the installer in this repository.
