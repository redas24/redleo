# redpac

The software installer for this repository's minimal desktop stack. It
installs the three "main programs" meant to run under
[`simplewm`](../simplewm):

- **st** — the suckless terminal (opened by `simplewm`'s `Super+Return`)
- **fish** — the shell
- **nyxt** — the web browser

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
./redpac install all         # everything
./redpac list                # show what's already installed
```

## Notes on nyxt

Nyxt isn't in the main package repository on either distro:

- **Arch**: it's AUR-only. `redpac` uses an AUR helper (`yay` or `paru`)
  if one is installed, preferring the prebuilt `nyxt-bin` package over
  building `nyxt` from source. Without a helper, it prints the manual
  `makepkg` steps.
- **Gentoo**: it's not in the main Portage tree. `redpac` tries a couple
  of likely category paths, and if neither is found, prints the steps to
  enable the community GURU overlay and retry.

## Supported systems

Arch Linux and Gentoo only. On anything else (including Debian/Ubuntu),
`redpac` exits with a clear error rather than guessing at a different
package manager.
