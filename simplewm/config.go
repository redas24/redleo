package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jezek/xgb/xproto"
)

// xkReturn is the X keysym for the Return/Enter key (from <X11/keysymdef.h>).
// Letters and digits need no such table: X11 keysyms for 'a'-'z' and '0'-'9'
// are defined to equal their ASCII codes.
const xkReturn = xproto.Keysym(0xff0d)

// bind is one "bind COMBO COMMAND..." line: grab COMBO (Super, optionally
// with Shift, plus one key) and run COMMAND when it's pressed.
type bind struct {
	keysym  xproto.Keysym
	mods    uint16
	command []string
}

// config is simplewm's parsed configuration.
type config struct {
	autostart   [][]string // "exec COMMAND..." lines, run once at startup
	binds       []bind
	idleSeconds int      // "idle SECONDS"; 0 disables auto-lock
	lockCommand []string // "lock COMMAND..."; default {"slock"}
}

const defaultConfigTemplate = `# simplewm config — generated automatically on first run. simplewm never
# overwrites this file once it exists, so edit freely.
#
#   exec COMMAND [ARGS...]         run once when simplewm starts
#   bind COMBO COMMAND [ARGS...]   grab a key combo, run COMMAND on press
#     COMBO is super-KEY or super-shift-KEY (KEY is a-z, 0-9, or return)
#   idle SECONDS                   auto-lock after this many idle seconds
#                                   (0 disables idle locking)
#   lock COMMAND [ARGS...]         command that locks the screen (default: slock)
#
# $TERMINAL resolves at runtime to $SIMPLEWM_TERMINAL, st, or xterm --
# whichever is found first in $PATH. $LOCK runs whatever "lock" above
# resolves to (slock by default).
#
# Commands run directly (never through a shell), so there's no quoting
# and no shell-injection risk, but also no "a || b" style fallback except
# for the built-in $TERMINAL and $LOCK tokens.

bind super-return $TERMINAL
bind super-shift-l $LOCK

idle 300

# Try the devops tools redpac (../redpac) can install, by uncommenting:
# bind super-b $TERMINAL btm
# bind super-d $TERMINAL lazydocker
# bind super-v $TERMINAL nvim
# exec nyxt
`

// defaultConfigPath returns $XDG_CONFIG_HOME/simplewm/config, falling back
// to ~/.config/simplewm/config.
func defaultConfigPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "simplewm", "config"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".config", "simplewm", "config"), nil
}

// loadConfig reads simplewm's config file at path, creating it from
// defaultConfigTemplate first if it doesn't exist yet.
func loadConfig(path string) (*config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(defaultConfigTemplate), 0o644); err != nil {
			return nil, fmt.Errorf("writing default config to %s: %w", path, err)
		}
		data = []byte(defaultConfigTemplate)
	} else if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return parseConfig(bytes.NewReader(data))
}

// parseConfig parses simplewm's small config language. Unknown directives
// or malformed lines are reported as errors rather than silently ignored.
func parseConfig(r *bytes.Reader) (*config, error) {
	cfg := &config{
		idleSeconds: 300,
		lockCommand: []string{"slock"},
	}

	scanner := bufio.NewScanner(r)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)

		switch fields[0] {
		case "exec":
			if len(fields) < 2 {
				return nil, fmt.Errorf("line %d: exec needs a command", lineNo)
			}
			cfg.autostart = append(cfg.autostart, fields[1:])

		case "bind":
			if len(fields) < 3 {
				return nil, fmt.Errorf("line %d: bind needs a combo and a command", lineNo)
			}
			keysym, mods, err := parseCombo(fields[1])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo, err)
			}
			cfg.binds = append(cfg.binds, bind{keysym: keysym, mods: mods, command: fields[2:]})

		case "idle":
			if len(fields) != 2 {
				return nil, fmt.Errorf("line %d: idle needs exactly one number of seconds", lineNo)
			}
			n, err := strconv.Atoi(fields[1])
			if err != nil || n < 0 {
				return nil, fmt.Errorf("line %d: invalid idle seconds %q", lineNo, fields[1])
			}
			cfg.idleSeconds = n

		case "lock":
			if len(fields) < 2 {
				return nil, fmt.Errorf("line %d: lock needs a command", lineNo)
			}
			cfg.lockCommand = fields[1:]

		default:
			return nil, fmt.Errorf("line %d: unknown directive %q", lineNo, fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// parseCombo parses a key combo like "super-return" or "super-shift-l".
func parseCombo(s string) (keysym xproto.Keysym, mods uint16, err error) {
	parts := strings.Split(strings.ToLower(s), "-")
	if len(parts) < 2 || parts[0] != "super" {
		return 0, 0, fmt.Errorf("combo %q must start with \"super-\"", s)
	}
	mods = modKey
	rest := parts[1:]
	if len(rest) == 2 && rest[0] == "shift" {
		mods |= xproto.ModMaskShift
		rest = rest[1:]
	}
	if len(rest) != 1 {
		return 0, 0, fmt.Errorf("combo %q: unrecognized modifier(s)", s)
	}

	key := rest[0]
	switch {
	case key == "return":
		keysym = xkReturn
	case len(key) == 1 && ((key[0] >= 'a' && key[0] <= 'z') || (key[0] >= '0' && key[0] <= '9')):
		keysym = xproto.Keysym(key[0])
	default:
		return 0, 0, fmt.Errorf("combo %q: unsupported key %q (want a-z, 0-9, or return)", s, key)
	}
	return keysym, mods, nil
}
