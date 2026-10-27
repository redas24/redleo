# simplewm

A minimal floating X11 window manager written in Go, using the pure-Go
[`jezek/xgb`](https://github.com/jezek/xgb) X protocol bindings (no Xlib/cgo
required). It does not reparent clients or draw decorations — it just lets
you move, resize, raise, focus, and close windows with the mouse, plus a
small configurable keybinding/autostart system and a couple of security
features (screen locking, idle auto-lock). The move/resize/raise behavior
is a direct port of the classic [tinywm](https://github.com/mackstann/tinywm).

## Mouse controls

All held together with Super (Mod4, the "Windows" key):

| Shortcut                | Action                          |
|--------------------------|----------------------------------|
| `Super` + Left-drag      | Move the window under the pointer |
| `Super` + Right-drag     | Resize the window under the pointer |
| `Super` + click          | Raise and focus the window under the pointer |
| `Super` + middle-click   | Close that window (`WM_DELETE_WINDOW`, or kill) |

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

## Configuration

On first run, simplewm writes a default config to
`$XDG_CONFIG_HOME/simplewm/config` (`~/.config/simplewm/config` by
default) and never touches it again — edit it freely. It's a small
line-based format:

```
exec COMMAND [ARGS...]         # run once at startup
bind COMBO COMMAND [ARGS...]   # grab a key combo, run COMMAND on press
                                # COMBO is super-KEY or super-shift-KEY
                                # (KEY is a-z, 0-9, or return)
idle SECONDS                   # auto-lock after this many idle seconds
                                # (0 disables idle locking)
lock COMMAND [ARGS...]         # the screen-lock command (default: slock)
```

`$TERMINAL` and `$LOCK` are special tokens: `$TERMINAL` resolves to
`$SIMPLEWM_TERMINAL`, `st`, or `xterm` (whichever is found first in
`$PATH`), and `$LOCK` runs whatever `lock` is set to. The default config:

```
bind super-return $TERMINAL
bind super-shift-l $LOCK

idle 300
```

Commands run directly via `execve`, never through a shell — the config
file can't be used for shell-injection tricks, but there's also no
shell-style quoting or `a || b` fallback beyond the two built-in tokens.

To wire up the devops tools [`redpac`](../redpac) can install, uncomment
(or add) lines like:

```
bind super-b $TERMINAL btm
bind super-d $TERMINAL lazydocker
bind super-v $TERMINAL nvim
exec nyxt
```

## Security features

- **Screen lock** — `Super+Shift+L` locks the screen by running `slock`
  (or whatever `lock` is set to in the config).
- **Idle auto-lock** — after `idle` seconds (300 by default) with no
  keyboard/mouse input, measured via the X `MIT-SCREEN-SAVER` extension,
  simplewm locks the screen the same way. Set `idle 0` to disable.
- **No shell in the loop** — every command simplewm spawns (autostart,
  keybindings, the locker) runs directly via `execve`, never `/bin/sh -c`,
  so config content can never be interpreted as shell syntax.
- simplewm logs a warning (non-fatal) if started as root, since a window
  manager running as root launches everything else as root too.

Install `slock` with `redpac install slock` ([`../redpac`](../redpac)).
