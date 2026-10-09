package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/michaelquigley/df/dl"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// the desktop id names the entry and the icon. it is the plain binary name, not reverse-DNS: nothing ships a
// sessionmixer.desktop to shadow, and the desktop matches a running window to its entry by this id, so it must be
// whatever the window reports as its app id (wayland) or class (x11); run.go sets the same string on dfx.Config.AppID.
const desktopID = "sessionmixer"

// the mark, as the scalable icon. gnome renders an svg from the hicolor theme directly, so no rasterized sizes are
// shipped.
//
//go:embed icon.svg
var iconSVG []byte

func init() {
	rootCmd.AddCommand(newDesktopCommand())
}

func newDesktopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "desktop",
		Short: "Manage the linux desktop entry for this binary",
	}
	cmd.AddCommand(newDesktopIntegrateCommand(), newDesktopRemoveCommand())
	return cmd
}

func newDesktopIntegrateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "integrate",
		Short: "Install a desktop entry and icon that launch this binary",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if runtime.GOOS != "linux" {
				return errors.Errorf("desktop integration is only supported on linux (got '%s')", runtime.GOOS)
			}
			executable, err := desktopExecutable()
			if err != nil {
				return err
			}
			dataHome, err := desktopDataHome()
			if err != nil {
				return err
			}
			paths := newDesktopPaths(dataHome)
			if err := installDesktopFiles(paths, executable); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "installed desktop entry '%s' for '%s'\n", paths.entry, executable)
			fmt.Fprintf(out, "installed icon '%s'\n", paths.svg)
			refreshDesktopDatabase(filepath.Dir(paths.entry))
			refreshIconCache(paths.hicolor)
			return nil
		},
	}
}

func newDesktopRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove the desktop entry and icon",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dataHome, err := desktopDataHome()
			if err != nil {
				return err
			}
			paths := newDesktopPaths(dataHome)
			out := cmd.OutOrStdout()
			for _, file := range paths.all() {
				if err := os.Remove(file); err != nil {
					if errors.Is(err, os.ErrNotExist) {
						continue
					}
					return errors.Wrapf(err, "remove '%s'", file)
				}
				fmt.Fprintf(out, "removed '%s'\n", file)
			}
			refreshDesktopDatabase(filepath.Dir(paths.entry))
			refreshIconCache(paths.hicolor)
			return nil
		},
	}
}

// desktopExecutable is the absolute path of the running binary, which the entry launches. a binary under the temp
// directory is `go run`'s and would be gone by the next launch, so it is refused.
func desktopExecutable() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", errors.Wrap(err, "resolve executable")
	}
	if executable, err = filepath.Abs(executable); err != nil {
		return "", errors.Wrap(err, "resolve executable path")
	}
	if strings.HasPrefix(executable, os.TempDir()+string(filepath.Separator)) {
		return "", errors.Errorf("'%s' is a temporary build; install sessionmixer (make) and integrate the installed binary", executable)
	}
	return executable, nil
}

type desktopPaths struct {
	entry   string // the freedesktop entry
	hicolor string // the user's hicolor theme directory
	svg     string // the scalable icon, under hicolor
}

func (p desktopPaths) all() []string {
	return []string{p.entry, p.svg}
}

func desktopDataHome() (string, error) {
	return desktopDataHomeFromEnv(os.Getenv("XDG_DATA_HOME"), os.Getenv("HOME"))
}

func desktopDataHomeFromEnv(xdgDataHome, home string) (string, error) {
	if filepath.IsAbs(xdgDataHome) {
		return xdgDataHome, nil
	}
	if home == "" {
		return "", errors.New("HOME is required when XDG_DATA_HOME is unset or relative")
	}
	return filepath.Join(home, ".local", "share"), nil
}

func newDesktopPaths(dataHome string) desktopPaths {
	hicolor := filepath.Join(dataHome, "icons", "hicolor")
	return desktopPaths{
		entry:   filepath.Join(dataHome, "applications", desktopID+".desktop"),
		hicolor: hicolor,
		svg:     filepath.Join(hicolor, "scalable", "apps", desktopID+".svg"),
	}
}

func installDesktopFiles(paths desktopPaths, executable string) error {
	for _, file := range paths.all() {
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			return errors.Wrapf(err, "create '%s'", filepath.Dir(file))
		}
	}
	if err := os.WriteFile(paths.entry, []byte(desktopEntry(executable)), 0o644); err != nil {
		return errors.Wrap(err, "write desktop entry")
	}
	if err := os.WriteFile(paths.svg, iconSVG, 0o644); err != nil {
		return errors.Wrap(err, "write icon")
	}
	return nil
}

// desktopEntry is the freedesktop entry. the window is the `run` subcommand, so the entry launches that; StartupWMClass
// is the id the window reports, so the desktop pairs the running window with this entry rather than a generic one.
func desktopEntry(executable string) string {
	return strings.Join([]string{
		"[Desktop Entry]",
		"Type=Application",
		"Name=" + desktopID,
		"Comment=a cue mix control surface for audio interfaces with onboard mixing",
		"Exec=" + desktopExecPath(executable) + " run",
		"Icon=" + desktopID,
		"Terminal=false",
		"Categories=AudioVideo;Audio;Mixer;",
		"StartupWMClass=" + desktopID,
		"",
	}, "\n")
}

// desktopExecPath quotes the executable for the Exec key: the entry's own escaping over a shell-style quoted string,
// with `%` doubled so a path containing one is not read as a field code.
func desktopExecPath(path string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"`", "\\`",
		"$", "\\$",
		"%", "%%",
	)
	return `"` + replacer.Replace(path) + `"`
}

// refreshDesktopDatabase rebuilds the cache the desktop reads entries from, when the tool is installed; without it
// the entry is still found, only later.
func refreshDesktopDatabase(applications string) {
	if _, err := exec.LookPath("update-desktop-database"); err != nil {
		dl.Warnf("desktop database not refreshed: 'update-desktop-database' is not installed")
		return
	}
	if out, err := exec.Command("update-desktop-database", applications).CombinedOutput(); err != nil {
		dl.Warnf("desktop database not refreshed: %v: '%s'", err, strings.TrimSpace(string(out)))
	}
}

// refreshIconCache rebuilds the hicolor cache when the directory already has one. gtk and gnome-shell trust an
// existing cache that is newer than the theme directory, and writing an icon into a subdirectory does not touch the
// theme directory, so a stale cache would hide the mark. a directory without a cache needs none.
func refreshIconCache(hicolor string) {
	if _, err := os.Stat(filepath.Join(hicolor, "icon-theme.cache")); err != nil {
		return
	}
	if _, err := exec.LookPath("gtk-update-icon-cache"); err != nil {
		dl.Warnf("icon cache not refreshed: 'gtk-update-icon-cache' is not installed")
		return
	}
	// -i: a user theme directory has no index.theme; the cache is still consulted
	if out, err := exec.Command("gtk-update-icon-cache", "-f", "-t", "-i", hicolor).CombinedOutput(); err != nil {
		dl.Warnf("icon cache not refreshed: %v: '%s'", err, strings.TrimSpace(string(out)))
	}
}
