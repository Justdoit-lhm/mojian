package main

import (
	"os"
	"path/filepath"

	"mojian/internal/app"
	"mojian/internal/welcome"

	"github.com/egoist/mygo/ui"
)

// fixtures build the disposable projects the harness serves. Welcome.md
// comes last so a fresh project opens on it.

func windowFixture() (func(*ui.Context), func(), error) {
	h, dir, err := newFixture()
	if err != nil {
		return nil, nil, err
	}
	return h.WindowView, func() { h.Close(); _ = os.RemoveAll(dir) }, nil
}

func settingsFixture() (func(*ui.Context), func(), error) {
	h, dir, err := newFixture()
	if err != nil {
		return nil, nil, err
	}
	return h.SettingsView, func() { h.Close(); _ = os.RemoveAll(dir) }, nil
}

func newFixture() (*app.Headless, string, error) {
	dir, err := os.MkdirTemp("", "mojian-harness-*")
	if err != nil {
		return nil, "", err
	}
	if err := welcome.WriteFiles(dir); err != nil {
		_ = os.RemoveAll(dir)
		return nil, "", err
	}
	s, err := app.OpenStoreAt(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, "", err
	}
	h := app.NewHeadless(s, filepath.Join(dir, "settings.json"), filepath.Join(dir, "layout.json"))
	return h, dir, nil
}
