# redpac

The software installer for this repository's minimal desktop stack. It
installs the "main programs" meant to run under [`simplewm`](../simplewm):

- **st** — the suckless terminal (bound to `Super+Return` by default)
- **fish** — the shell
- **nyxt** — the web browser
- **lazyvim** — Neovim set up with the [LazyVim](https://www.lazyvim.org/) config — the editor

Plus a couple of devops/security tools, wired into `simplewm`'s example
keybindings (see [`../simplewm/README.md`](../simplewm/README.md)):

- **bottom** — a system/process monitor TUI (binary: `btm`)
- **lazydocker** — a Docker/Compose management TUI
- **slock** — the suckless screen locker, used by `simplewm`'s
  `Super+Shift+L` and its idle auto-lock

`redpac` is a thin wrapper around your distro's native package manager —
**pacman** on Arch Linux, or **emerge**/Portage on Gentoo — rather than a
generic apt/source-tarball installer. Because the real package manager
does the work, required libraries (X11, Xft, WebKitGTK, SBCL, ...) come in
automatically as ordinary dependencies; there's no separate "install libs"
step.

## Usage

```sh
./redpac install fish        # just the shell
./redpac install st          # just the terminal
./redpac install nyxt        # just the browser
./redpac install lazyvim     # just the editor
./redpac install bottom      # just the system monitor
./redpac install lazydocker  # just the Docker TUI
./redpac install slock       # just the screen locker
./redpac install all         # everything
./redpac list                # show what's already installed
```

## Notes on lazyvim

`redpac` installs Neovim, git, ripgrep and fd, then clones the official
[LazyVim starter](https://github.com/LazyVim/starter) into
`$XDG_CONFIG_HOME/nvim` (`~/.config/nvim` by default) — but only if that
directory doesn't already exist. It never overwrites an existing Neovim
config; if one is there, it prints the manual steps to back it up and
switch over yourself. The first `nvim` run after install bootstraps
LazyVim's plugins.

## Notes on nyxt

Nyxt isn't in the main package repository on either distro:

- **Arch**: it's AUR-only. `redpac` uses an AUR helper (`yay` or `paru`)
  if one is installed, preferring the prebuilt `nyxt-bin` package over
  building `nyxt` from source. Without a helper, it prints the manual
  `makepkg` steps.
- **Gentoo**: it's not in the main Portage tree. `redpac` tries a couple
  of likely category paths, and if neither is found, prints the steps to
  enable the community GURU overlay and retry.

## Notes on lazydocker

`lazydocker` is an official Arch package, installed the same way as
`fish` or `st`. On Gentoo it currently only lives in third-party
overlays: `redpac` tries the main tree first, and on failure prints the
steps to try the community GURU overlay, with a `go install` one-liner
as a last resort that works on either distro.

## Cheatsheets

Default keybinding references for the programs `redpac` installs:

- [`cheatsheets/nyxt.md`](cheatsheets/nyxt.md)
- [`cheatsheets/fish.md`](cheatsheets/fish.md)

## Supported systems

Arch Linux and Gentoo only. On anything else (including Debian/Ubuntu),
`redpac` exits with a clear error rather than guessing at a different
package manager.
