package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDesktopDataHomeFromEnv(t *testing.T) {
	if got, err := desktopDataHomeFromEnv("/tmp/sessionmixer-data", "/home/example"); err != nil || got != "/tmp/sessionmixer-data" {
		t.Errorf("absolute XDG_DATA_HOME must win: got %q, err %v", got, err)
	}
	want := filepath.Join("/home/example", ".local", "share")
	if got, err := desktopDataHomeFromEnv("", "/home/example"); err != nil || got != want {
		t.Errorf("empty XDG_DATA_HOME must fall back to home: got %q, err %v", got, err)
	}
	if got, err := desktopDataHomeFromEnv("relative", "/home/example"); err != nil || got != want {
		t.Errorf("relative XDG_DATA_HOME must fall back to home: got %q, err %v", got, err)
	}
	if _, err := desktopDataHomeFromEnv("", ""); err == nil {
		t.Error("the fallback requires HOME")
	}
}

func TestDesktopEntry(t *testing.T) {
	entry := desktopEntry(`/opt/mixer tools/session%mixer`)
	for _, want := range []string{
		"[Desktop Entry]\n",
		"Type=Application\n",
		"Name=sessionmixer\n",
		`Exec="/opt/mixer tools/session%%mixer" run` + "\n",
		"Icon=sessionmixer\n",
		"Terminal=false\n",
		"Categories=AudioVideo;Audio;Mixer;\n",
		"StartupWMClass=sessionmixer\n",
	} {
		if !strings.Contains(entry, want) {
			t.Errorf("entry lacks %q:\n%s", want, entry)
		}
	}
}

func TestIconIsAnSVG(t *testing.T) {
	if !strings.Contains(string(iconSVG), "<svg") {
		t.Error("icon.svg is not an svg")
	}
}

func TestInstallDesktopFiles(t *testing.T) {
	dataHome := t.TempDir()
	paths := newDesktopPaths(dataHome)
	if err := installDesktopFiles(paths, "/usr/local/bin/sessionmixer"); err != nil {
		t.Fatal(err)
	}
	for _, file := range paths.all() {
		if _, err := os.Stat(file); err != nil {
			t.Errorf("not installed: %v", err)
		}
	}
	if paths.entry != filepath.Join(dataHome, "applications", "sessionmixer.desktop") {
		t.Errorf("entry path %q", paths.entry)
	}
	if paths.svg != filepath.Join(dataHome, "icons", "hicolor", "scalable", "apps", "sessionmixer.svg") {
		t.Errorf("svg path %q", paths.svg)
	}
	// the freedesktop validator, where the machine has it
	if validator, err := exec.LookPath("desktop-file-validate"); err == nil {
		if out, err := exec.Command(validator, paths.entry).CombinedOutput(); err != nil {
			t.Errorf("desktop-file-validate: %v\n%s", err, out)
		}
	}
}

func TestRefreshIconCacheRebuildsAnExistingCache(t *testing.T) {
	if _, err := exec.LookPath("gtk-update-icon-cache"); err != nil {
		t.Skip("gtk-update-icon-cache is not installed")
	}
	dataHome := t.TempDir()
	paths := newDesktopPaths(dataHome)
	if err := installDesktopFiles(paths, "/usr/local/bin/sessionmixer"); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(paths.hicolor, "icon-theme.cache")
	// a directory without a cache is left without one
	refreshIconCache(paths.hicolor)
	if _, err := os.Stat(cache); err == nil {
		t.Fatal("a cache was created where none existed")
	}
	// a stale cache is rebuilt
	if err := os.WriteFile(cache, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale := time.Now().Add(-time.Hour)
	if err := os.Chtimes(cache, stale, stale); err != nil {
		t.Fatal(err)
	}
	refreshIconCache(paths.hicolor)
	info, err := os.Stat(cache)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().After(stale) || info.Size() == int64(len("stale")) {
		t.Errorf("cache not rebuilt: mtime %v, size %d", info.ModTime(), info.Size())
	}
}
