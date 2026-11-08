package app

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func writeTestPNG(t *testing.T, path string) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 10, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 20), uint8(y * 20), 120, 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestRefreshKeepsAssetPane(t *testing.T) {
	a := newProjectApp(t)
	path := filepath.Join(a.store.Dir, "cover.png")
	first := writeTestPNG(t, path)
	if err := a.store.Refresh(); err != nil {
		t.Fatal(err)
	}
	asset := a.store.AssetByRel("cover.png")
	if asset == nil {
		t.Fatal("the image was not indexed")
	}
	a.openAsset(asset)
	tab := a.tab()
	if tab == nil || tab.Focus == nil || tab.Focus.Asset != asset {
		t.Fatalf("asset pane: %+v", tab)
	}

	a.refreshProject()
	if a.tab() != tab {
		t.Fatal("refreshing the project closed the asset tab")
	}
	p := a.tab().Focus
	if p == nil || p.Asset == nil || p.Asset.Rel != "cover.png" {
		t.Fatalf("asset pane after refresh: %+v", p)
	}

	// A changed image is rebound to a fresh entry, but the pane stays.
	second := append([]byte(nil), first...)
	second[len(second)-1] ^= 0xff
	if err := os.WriteFile(path, second, 0o644); err != nil {
		t.Fatal(err)
	}
	future := asset.Mod.Add(2e9)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
	a.refreshProject()
	if a.tab() != tab {
		t.Fatal("refreshing a changed asset closed its tab")
	}
	if got := a.tab().Focus.Asset; got == nil || got == asset || got.Rel != "cover.png" {
		t.Fatalf("changed asset was not rebound: %+v", got)
	}
}

func TestOpeningAssetKeepsTerminalPane(t *testing.T) {
	a := newProjectApp(t)
	path := filepath.Join(a.store.Dir, "cover.png")
	writeTestPNG(t, path)
	if err := a.store.Refresh(); err != nil {
		t.Fatal(err)
	}
	asset := a.store.AssetByRel("cover.png")
	if asset == nil {
		t.Fatal("the image was not indexed")
	}

	a.newTab()
	terminalTab := a.tab()
	a.addTerminalPane(&termTab{})
	term := a.tab().Focus
	if term == nil || term.Term == nil {
		t.Fatal("terminal pane was not created")
	}

	a.openAsset(asset)
	if a.tab() == terminalTab {
		t.Fatal("opening an asset reused the terminal tab")
	}
	if p := a.tab().Focus; p == nil || p.Asset != asset || p.Term != nil {
		t.Fatalf("asset pane: %+v", p)
	}
	if !terminalTab.contains(term) || term.Term == nil || term.Asset != nil {
		t.Fatalf("opening the asset corrupted the terminal pane: %+v", term)
	}
}

func TestOpeningNoteKeepsUnfocusedTerminalPane(t *testing.T) {
	a := newProjectApp(t)
	n := a.store.Sorted()[0]
	a.newTab()
	terminalTab := a.tab()
	a.addTerminalPane(&termTab{})
	term := a.tab().Focus
	terminalTab.Focus = nil

	a.openNote(n)
	if a.tab() == terminalTab {
		t.Fatal("opening a note reused the terminal tab")
	}
	if p := a.tab().Focus; p == nil || p.Note != n || p.Term != nil {
		t.Fatalf("note pane: %+v", p)
	}
	if !terminalTab.contains(term) || term.Term == nil || term.Note != nil {
		t.Fatalf("opening the note corrupted the terminal pane: %+v", term)
	}
}

func TestPaletteSelectsLateTab(t *testing.T) {
	a := newProjectApp(t)
	for i := 0; i < 12; i++ {
		a.newTab()
		a.tab().Name = fmt.Sprintf("Pane %d", i)
	}
	target := 9 // "Pane 8", before the last tab
	a.paletteQuery = "Pane 8"
	var found bool
	for _, item := range a.paletteItems() {
		if item.title != "Pane 8" {
			continue
		}
		found = true
		item.run()
		break
	}
	if !found {
		t.Fatal("the palette did not list the tab")
	}
	if a.active != target {
		t.Fatalf("palette selected tab %d, want %d", a.active, target)
	}
}

func TestLaterRunsFromSettingsWindow(t *testing.T) {
	a := &app{workspace: &workspace{}}
	ran := false
	a.later(func() { ran = true })
	tt := ui.NewTester(a.settingsWindowView, 900, 640)
	tt.Frame()
	if !ran {
		t.Fatal("a deferred settings action did not run")
	}
}
