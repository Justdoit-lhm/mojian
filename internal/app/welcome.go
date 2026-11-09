package app

import (
	"mojian/internal/app/commands"
	"runtime"

	"mojian/internal/icons"

	"github.com/egoist/mygo/ui"
)

// emptyState is the welcome page of a tab with no file open: quick
// actions, then the files touched most recently.
func (a *app) emptyState(c *ui.Context, k *colors, vw, vh float32) {
	ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).FillWidth().MinHeight(vh).Justify(ui.Center).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(560).AlignSelf(ui.Center).
				Padding(28, 32, 36, 32).Gap(20).Children(func() {
				ui.Row(c).Gap(14).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(58, 48).Radius(radius).Background(k.CardFocused).
						Border(1, k.CardBorderFocused).Center().Children(func() {
						ui.Icon(c, brandMark(c)).Size(34, 34)
					})
					ui.Column(c).Gap(2).Children(func() {
						ui.Text(c, "Welcome to Mojian").FontSize(22).FontWeight(650).
							TextColor(k.Text)
						ui.Text(c, "A quiet editor for the files in your project.").
							FontSize(12.5).TextColor(k.TextMuted)
					})
				})

				actions := ui.Column(c).FillWidth().Background(k.Card).
					Border(1, k.CardBorder).Radius(radius).Clip()
				actions.Children(func() {
					a.welcomeAction(c, k, "file-plus", "New File…", cmdNewNote, func() { a.startNewFile() })
					ui.Divider(c)
					a.welcomeAction(c, k, "search", "Open File…", cmdNotes, func() { a.openQuickOpen() })
					if runtime.GOOS != "js" {
						ui.Divider(c)
						a.welcomeAction(c, k, "folder-open", "Open Project…", cmdOpenProject, func() { a.openProjectPicker() })
						if terminalsSupported() {
							ui.Divider(c)
							a.welcomeAction(c, k, "square-terminal", "Open Terminal", cmdTerminal, func() { a.toggleTerminal() })
						}
					}
				})

				notes := a.welcomeNotes()
				if len(notes) > 0 {
					ui.Text(c, "RECENT FILES").FontSize(10).FontWeight(650).
						TextColor(k.TextFaint).Padding(2, 2)
					recent := ui.Column(c).FillWidth().Background(k.Card).
						Border(1, k.CardBorder).Radius(radius).Clip()
					recent.Children(func() {
						for i, n := range notes {
							n := n
							if i > 0 {
								ui.Divider(c)
							}
							row := ui.ButtonBase(c.Key("welcome:"+n.Rel)).Height(40).Padding(0, 14).
								Gap(9).Justify(ui.Start).Cursor(ui.CursorPointer).
								Label(n.Title)
							if row.Hovered() {
								row.Background(k.Hover)
							}
							if row.Clicked() {
								a.later(func() { a.openNote(n) })
							}
							row.Children(func() {
								ui.Icon(c, icons.FileIcon(n.Path)).Size(15, 15).TextColor(k.TextMuted)
								ui.Text(c, n.Title).FontSize(12.5).TextColor(k.Text).
									SingleLine().Ellipsis("…").Grow(1).MinWidth(0)
								ui.Text(c, n.Rel).FontSize(11).TextColor(k.TextFaint).
									SingleLine().Ellipsis("…").Shrink(1).MinWidth(0)
							})
						}
					})
				}

				ui.Text(c, nativeKeys("⌘,")+" Settings     "+nativeKeys("⇧⌘P")+" Commands     "+nativeKeys("⌘O")+" Open Project").
					FontSize(11).TextColor(k.TextFaint).TextAlign(ui.Center)
			})
		})
	})
}

func (a *app) welcomeAction(c *ui.Context, k *colors, glyph, title string, cmd *commands.Command, run func()) {
	row := ui.ButtonBase(c).Height(42).Padding(0, 14).Gap(10).
		Justify(ui.Start).Cursor(ui.CursorPointer).Label(title)
	if row.Hovered() {
		row.Background(k.Hover)
	}
	if row.Clicked() {
		run()
	}
	row.Children(func() {
		ui.Icon(c, icons.Icon(glyph)).Size(15, 15).TextColor(k.TextMuted)
		ui.Text(c, title).FontSize(12.5).TextColor(k.Text).Grow(1)
		if hint := a.commandKeys(cmd); hint != "" {
			ui.Text(c, hint).FontSize(11).TextColor(k.TextFaint)
		}
	})
}

func (a *app) welcomeNotes() []*note {
	if a.store == nil {
		return nil
	}
	notes := a.store.Sorted()
	if len(notes) > 5 {
		notes = notes[:5]
	}
	return notes
}

// errorBanner shows what went wrong with the vault.
