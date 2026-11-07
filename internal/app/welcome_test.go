package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestWelcomeOpenFileFocusesSearch(t *testing.T) {
	for _, treeOpen := range []bool{false, true} {
		t.Run("tree="+map[bool]string{false: "closed", true: "open"}[treeOpen], func(t *testing.T) {
			registerFonts()
			a := newProjectApp(t)
			a.treeOpen = treeOpen
			tt := ui.NewTester(a.view, 1100, 700)
			tt.Frame()
			if err := tt.Click("New Tab"); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click("Open File…"); err != nil {
				t.Fatal(err)
			}
			if !a.treeOpen {
				t.Fatal("Open File did not show the Explorer")
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
		})
	}
}
