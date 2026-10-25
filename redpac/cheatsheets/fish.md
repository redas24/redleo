# fish shell keybinding cheatsheet

These are fish's interactive defaults (`fish_default_key_bindings`, i.e.
emacs-style editing) — not `fish_vi_key_bindings`.

| Action | Binding |
|---|---|
| Move char forward / back | `ctrl-f` / → ; `ctrl-b` / ← |
| Move word forward / back | `alt-f` / `alt-right` ; `alt-b` / `alt-left` |
| Start / end of line | `ctrl-a` / Home ; `ctrl-e` / End |
| Delete char forward / back | `ctrl-d` / Delete ; `ctrl-h` / Backspace |
| Delete word backward | `alt-backspace` |
| Delete to end of line | `ctrl-k` |
| Delete to start of line | `ctrl-u` |
| Accept full autosuggestion | `ctrl-f` / → or `ctrl-e` / End (at end of line) |
| Accept one word of suggestion | `alt-f` / `alt-right` |
| Accept one token | `ctrl-right` |
| History search (prev / next) | `ctrl-p` / ↑ ; `ctrl-n` / ↓ |
| History pager (press again for older) | `ctrl-r` |
| Tab completion / cycle | `tab`; `shift-tab` opens pager search mode |
| Clear screen | `ctrl-l` |
| Cancel current input | `escape` or `ctrl-g` |
| Interrupt running command (SIGINT) | `ctrl-c` |
| Edit command line in `$EDITOR`/`$VISUAL` | `alt-e` or `alt-v` |
| Expand abbreviation | automatic on `space` / `enter` |
| Insert a literal space (no abbreviation expansion) | `ctrl-space` |

Source: [fishshell.com — Interactive use](https://fishshell.com/docs/current/interactive.html)
(mirrors `doc_src/interactive.rst` in fish-shell/fish-shell).
