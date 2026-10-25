# Nyxt keybinding cheatsheet

Nyxt ships with multiple "keyschemes." Out of the box it defaults to
**CUA** (standard Ctrl/Alt bindings), but it also supports **Emacs** and
**VI-normal** schemes. All three inherit a shared `default` scheme, so
some bindings (marked *default*) work the same in every scheme.

| Action | CUA | Emacs | VI-normal |
|---|---|---|---|
| New / focus buffer | `C-t` *(default)* | `C-t` | `B` |
| Close buffer | `C-w` | `C-x k` / `C-x C-k` | `d` / `D` |
| Switch buffer | `M-down`, `C-[` / `C-]` | `C-x b`, `M-p` / `M-n` | `g b`, `[` / `]` |
| Open URL / prompt bar | `C-l` *(default)* | `C-l` | `o` / `O` (new buffer) |
| Execute command (dispatcher) | `C-space` *(default)* | `C-space`, also `M-x` | `C-space`, also `:` |
| Search in page | `C-f` / `F3` | `C-s s` | `/` |
| Add bookmark | `C-d`, `C-m s` | `C-x r m` / `C-x r l` | `m m` / `m M` |
| Reload page | `C-r` / `F5` *(default)* | `C-r` | `R` / `r` |
| Go back / forward | `M-left` / `M-right` *(default)*  | `C-b` / `C-f` | `H` / `L` |
| Quit | `C-q` | `C-x C-c` | `Z Z` |
| Open manual | `f1 r` | `f1 r` | `f1 r` |
| Open tutorial | `f1 t` | `f1 t` | `f1 t` |

Notes:

- CUA has no execute-command binding of its own — it relies on the
  inherited `C-space`, which works no matter which keyscheme is active.
- Switch keyscheme at runtime with `C-space` → `set-keyscheme`.

Source: Nyxt's own `source/keyscheme.lisp`, `source/mode/base.lisp`,
`source/mode/history.lisp`, `source/mode/search-buffer.lisp`, and
`source/mode/bookmark.lisp` (atlas-engineer/nyxt, master branch), cross-checked
against the [ArchWiki Nyxt page](https://wiki.archlinux.org/title/Nyxt).
