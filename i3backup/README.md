# i3backup

A single-file backup/restore tool for an i3 window manager environment.
It packages the config files for i3 itself and the usual programs
that run alongside it into one tarball, and can put them back later —
on the same machine, or a fresh one.

## What it backs up

Whichever of these exist under `$HOME`:

- `~/.config/i3`, `~/.i3` — the window manager config
- `~/.config/i3status`, `~/.i3status.conf`, `~/.config/i3status-rust` — status bar
- `~/.config/polybar` — status bar (alternative to i3status)
- `~/.config/picom`, `~/.config/picom.conf`, `~/.picom.conf` — compositor
- `~/.config/rofi`, `~/.config/dmenu` — application launcher
- `~/.config/dunst` — notifications
- `~/.config/nitrogen`, `~/.fehbg` — wallpaper
- `~/.Xresources`, `~/.xprofile`, `~/.xinitrc` — X session setup

Paths that don't exist on the host are skipped, so it works whether or
not you use polybar vs. i3status, picom vs. no compositor, etc.

## Usage

```sh
./i3backup create             # write i3-backup-<host>-<timestamp>.tar.gz here
./i3backup create ~/backups   # ...or into a chosen directory

./i3backup list i3-backup-myhost-20260101-120000.tar.gz

./i3backup restore i3-backup-myhost-20260101-120000.tar.gz           # keeps any files already in $HOME
./i3backup restore i3-backup-myhost-20260101-120000.tar.gz --force   # overwrites them
```

Paths inside the archive are stored relative to `$HOME`, so `restore`
extracts them straight back into place. By default it never clobbers a
file that's already there — pass `--force` to overwrite.
