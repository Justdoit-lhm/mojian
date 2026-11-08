// Package explorer draws the project tree and runs its file actions. The
// per-project tree state stays on the app's workspace; the Host interface
// gives the view access to it and to window actions.
package explorer

import (
	"mojian/internal/icons"
	"mojian/internal/motion"
	"mojian/internal/project"
	"mojian/internal/theme"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

type colors = theme.Colors

const (
	radius  = theme.Radius
	headerH = 33
)

type fileNode = project.TreeNode
type note = project.Note
type assetFile = project.Asset

// ClipState is the app's own file clipboard.
type ClipState struct {
	Paths  []string
	Cut    bool
	Remote bool
	Host   string
	Root   string
}

// Host is what the Explorer asks of the window.
type Host interface {
	Store() *project.Store
	CurrentNote() *project.Note
	OpenRel(rel string)
	OpenNote(*project.Note)
	OpenNoteInTab(*project.Note)
	OpenAssetInTab(*project.Asset)
	Later(func())
	Fail(error)
	Invalidate()
	Changed()

	CloseNotes(func(*project.Note) bool)
	CloseAssets(func(*project.Asset) bool)
	AfterRefresh()
	RefreshGit()
	GitOpen() bool
	AutoRefreshGit() bool

	Tip(label, commandID string) string
	FilteredNotes() []*project.Note
	ActiveWorkspaceKey() string
	NativeWindow() bool
	RemotePath() string
	ServerAlias() string
	ResetRenamedStates([]*project.Note, map[*project.Note]string)

	TreeOpen() bool
	SetTreeOpen(bool)
	TreeQuery() string
	TreeQueryPtr() *string
	SetTreeQuery(string)
	TreeSel() string
	SetTreeSel(string)
	TreeReveal() string
	SetTreeReveal(string)
	OpenDirs() map[string]bool
	SetOpenDirs(map[string]bool)
	TreeDraft() *Draft
	SetTreeDraft(*Draft)
	NextDraftSeq() int
	TreeSearch() *ui.Handle

	Clip() ClipState
	SystemClipboardFiles() []string
	WriteClipboardFiles(paths ...string) error
	SetClip(ClipState)
}

// UI is the Explorer view; the app sets Host before using it.
type UI struct{ Host }

func (u *UI) store() *project.Store                       { return u.Store() }
func (u *UI) currentNote() *project.Note                  { return u.CurrentNote() }
func (u *UI) fail(err error)                              { u.Fail(err) }
func (u *UI) later(fn func())                             { u.Later(fn) }
func (u *UI) invalidate()                                 { u.Invalidate() }
func (u *UI) changed()                                    { u.Changed() }
func (u *UI) openRel(rel string)                          { u.OpenRel(rel) }
func (u *UI) openNote(n *project.Note)                    { u.OpenNote(n) }
func (u *UI) openNoteInTab(n *project.Note)               { u.OpenNoteInTab(n) }
func (u *UI) openAssetInTab(f *project.Asset)             { u.OpenAssetInTab(f) }
func (u *UI) closeNotes(match func(*project.Note) bool)   { u.CloseNotes(match) }
func (u *UI) closeAssets(match func(*project.Asset) bool) { u.CloseAssets(match) }
func (u *UI) afterRefresh()                               { u.AfterRefresh() }
func (u *UI) refreshGit()                                 { u.RefreshGit() }
func (u *UI) gitOpen() bool                               { return u.GitOpen() }
func (u *UI) autoRefreshGit() bool                        { return u.AutoRefreshGit() }
func (u *UI) filteredNotes() []*project.Note              { return u.FilteredNotes() }
func (u *UI) activeWorkspaceKey() string                  { return u.ActiveWorkspaceKey() }
func (u *UI) nativeWindow() bool                          { return u.NativeWindow() }
func (u *UI) remotePath() string                          { return u.RemotePath() }
func (u *UI) serverAlias() string                         { return u.ServerAlias() }
func (u *UI) resetRenamedStates(notes []*project.Note, before map[*project.Note]string) {
	u.ResetRenamedStates(notes, before)
}
func (u *UI) tip(label, id string) string { return u.Tip(label, id) }

func editorLanguage(path string) *editor.Language { return editor.LanguageFor(path) }

func editorShape(path string) string {
	lang := ""
	if l := editorLanguage(path); l != nil {
		lang = l.Name
	}
	return lang + "/" + boolString(project.IsMarkdownPath(path))
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// iconButton is a borderless icon button, shared with the rest of the
// window's chrome.
func iconButton(c *ui.Context, k *colors, name, label string, size, iconSize float32) ui.Element {
	b := ui.Box(c).Size(size, size).Center().Radius(radius).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(label)
	if b.Pressed() {
		b.Background(k.Pressed)
	} else if b.Hovered() {
		b.Background(k.Hover)
	}
	b.Transition(motion.Colors)
	col := k.IconMuted
	if b.Hovered() {
		col = k.Text
	}
	b.Children(func() {
		ui.Icon(c, icons.Icon(name)).Size(iconSize, iconSize).TextColor(col)
	})
	return b
}
