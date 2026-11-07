package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestWelcomeOpenFileFocusesSearch(t *testing.T) {
	registerFonts()
	a := newProjectApp(t)
	tt := ui.NewTester(a.view, 1100, 700)
	tt.Frame()
	if err := tt.Click("New Tab"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Open File…"); err != nil {
		t.Fatal(err)
	}
	if !tt.Focused("Search files") {
		t.Fatal("Open File did not focus the search field")
	}
	tt.Type("tasks")
	tt.Frame()
	if a.treeQuery != "tasks" {
		t.Fatalf("tree query: %q", a.treeQuery)
	}
	if !tt.HasText("tasks.py") {
		t.Fatalf("search results: %q", tt.Texts())
	}
}
