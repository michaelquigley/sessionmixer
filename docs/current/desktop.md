# The desktop

`sessionmixer desktop integrate` makes the binary that runs it an application the Linux desktop knows. It installs a freedesktop entry at `~/.local/share/applications/sessionmixer.desktop` whose `Exec` is the absolute path of that binary followed by `run`, and the sessionmixer mark as the scalable icon at `~/.local/share/icons/hicolor/scalable/apps/sessionmixer.svg`. Then it refreshes the desktop database when `update-desktop-database` is installed, and rebuilds the user's hicolor icon cache when one already exists there, since GTK and gnome-shell trust an existing cache over the directory and a cache written before the mark would hide it. From then on sessionmixer is in the application grid under its mark, launching the window against the configured card. `sessionmixer desktop remove` deletes exactly those two files and refreshes the same caches.

Because the entry carries the binary's path, it is for a specific binary: each machine builds its own into its project GOBIN, off `PATH`, and that is what the entry launches. Rebuilding in place needs nothing; moving the binary means running integrate again. A temporary build (`go run`) is refused, since its path is gone by the next launch. `XDG_DATA_HOME` is honored, so the install lands wherever the user's data home is. Integration is Linux only.

The entry is named `sessionmixer`, not reverse-DNS: nothing ships a `sessionmixer.desktop` to shadow, and the desktop pairs a running window with its entry by matching the window's app id (Wayland) or class (X11) to the entry's name, so the id has to be the one the window reports. The window reports `sessionmixer` through dfx's `Config.AppID`, set in `cmd/sessionmixer/run.go` from the same `desktopID` constant the entry is written from, and `StartupWMClass=sessionmixer` completes the match on X11. So a running window pairs with the entry however it was launched, from the grid or from a terminal.

The mark is `cmd/sessionmixer/icon.svg`, embedded in the binary: a stereo meter over four fader channels, in the window's own hues. Only the scalable icon is shipped, which GNOME renders directly; no rasterized sizes are committed. No window icon is set: GLFW sets window icons on X11 only, and the desktop supplies the icon from the entry.

```
sessionmixer desktop integrate     # install the entry and icon for this binary
sessionmixer desktop remove        # remove them
```
