package app

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestPreviewPaneKeepsHeaderButtonsWhenUnfocused(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 20), uint8(y * 20), 180, 255})
		}
	}
	path := filepath.Join(a.store.dir, "cover.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := a.store.refresh(); err != nil {
		t.Fatal(err)
	}
	asset := a.store.assetByRel("cover.png")
	if asset == nil {
		t.Fatal("the image was not indexed")
	}
	a.openAsset(asset)
	a.splitWith(a.store.byRelPath("Reading notes.md"))
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	tt.Frame()
	if a.tab().Focus.Asset != nil {
		t.Fatal("the asset pane still has the focus")
	}
	if got := countText(tt.Texts(), "Close Pane"); got != 2 {
		t.Fatalf("close buttons: %d, want 2 (preview and focused editor)", got)
	}
}

func TestMarkdownPreviewKeepsHeaderButtonsWhenUnfocused(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.openNote(a.store.byRelPath("Welcome.md"))
	p := a.tab().Focus
	p.Preview = true
	a.splitWith(a.store.byRelPath("Reading notes.md"))
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	tt.Frame()
	if a.tab().Focus == p {
		t.Fatal("the preview pane still has the focus")
	}
	if got := countText(tt.Texts(), "Close Pane"); got != 2 {
		t.Fatalf("close buttons: %d, want 2 (preview and focused editor)", got)
	}
	if !tt.HasText("Edit") {
		t.Fatalf("the preview's Edit button is gone: %q", tt.Texts())
	}
}

func countText(texts []string, want string) int {
	n := 0
	for _, text := range texts {
		if strings.HasPrefix(text, want) {
			n++
		}
	}
	return n
}

func TestEveryPaneKeepsHeaderButtonsWhenUnfocused(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	a.openNote(a.store.byRelPath("Reading notes.md"))
	first := a.tab().Focus
	a.splitWith(a.store.byRelPath("Welcome.md"))
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	tt.Frame()
	if a.tab().Focus == first {
		t.Fatal("the first pane still has the focus")
	}
	if got := countText(tt.Texts(), "Close Pane"); got != 2 {
		t.Fatalf("close buttons: %d, want 2", got)
	}
	if got := countText(tt.Texts(), "Split Right"); got != 2 {
		t.Fatalf("split buttons: %d, want 2", got)
	}
}
