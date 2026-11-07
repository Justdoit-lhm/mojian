package app

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
)

// pane is one editor view of a note in a workspace.
type pane struct {
	ID    string
	Note  *note
	Asset *assetFile
	Term  *termTab
	state *editor.State

	// Preview shows Markdown rendered rather than as source; assetView
	// holds what a picture or a PDF viewer has loaded.
	Preview       bool
	previewScroll ui.ScrollState
	md            *markdown.Options
	assetView     *assetView
	dragging      bool
}

// assetView is what a picture or PDF pane has loaded, and its headings.
type assetView struct {
	pdfPages   []string
	pdfPage    int
	pdfDir     string
	pdfErr     string
	pdfLoading bool
	pdfStarted bool
	pdfImage   ui.ImageSource
	pdfImageAt int // the page pdfImage was decoded for
}

// node is a leaf pane, or two panes side by side or stacked, with the
// ratio the first takes. It is the reference's tree of splits.
type node struct {
	Vertical bool // the panes are stacked, one above the other
	Ratio    float32
	A, B     *node
	Pane     *pane

	// A split starts with the old pane almost filling the node, then
	// moves Ratio to animTo over the next frames.
	animFrom, animTo float32
	animAt           time.Time
	animating        bool
}

// startSplitAnim eases a fresh split from the old pane's full size to
// the proportion the two panes share.
func (n *node) startSplitAnim(to float32) {
	n.animFrom, n.animTo = 0.97, to
	n.animAt, n.animating = time.Now(), true
	n.Ratio = n.animFrom
}

// tab is a workspace: a tree of panes, the one with the focus, and the
// one zoomed, if any.
type tab struct {
	ID    string
	Name  string // a name of its own, else the focused note's title
	Root  *node
	Focus *pane
	Zoom  *pane
}

var idSeq int

func nextID(prefix string) string {
	idSeq++
	return fmt.Sprintf("%s%d", prefix, idSeq)
}

func newPane(n *note) *pane {
	return &pane{ID: nextID("p"), Note: n}
}

func newTermPane(t *termTab) *pane {
	return &pane{ID: nextID("p"), Term: t}
}

func newAssetPane(f *assetFile) *pane {
	return &pane{ID: nextID("p"), Asset: f, assetView: &assetView{}}
}

func newTab(n *note) *tab {
	p := newPane(n)
	return &tab{ID: nextID("t"), Root: &node{Pane: p}, Focus: p}
}

// panes returns the panes of the tab, in layout order.
func (t *tab) panes() []*pane {
	var out []*pane
	t.Root.walk(func(p *pane) { out = append(out, p) })
	return out
}

func (n *node) walk(fn func(*pane)) {
	if n == nil {
		return
	}
	if n.Pane != nil {
		fn(n.Pane)
		return
	}
	n.A.walk(fn)
	n.B.walk(fn)
}

// label is what the tab strip shows: the focused note's title, or the
// tab's own name, and how many panes it holds.
func (t *tab) label() (name, detail string) {
	name = t.Name
	if name == "" && t.Focus != nil {
		switch {
		case t.Focus.Note != nil:
			name = t.Focus.Note.Title
		case t.Focus.Asset != nil:
			name = t.Focus.Asset.Name
		case t.Focus.Term != nil:
			name = "Terminal"
		}
	}
	if name == "" {
		name = "Untitled"
	}
	if n := len(t.panes()); n > 1 {
		detail = fmt.Sprintf("· %d", n)
	}
	return name, detail
}

// editorPane returns the pane to focus when the tab appears: an editor
// if it has one, else the terminal.
func (t *tab) editorPane() *pane {
	if t.Focus != nil && (t.Focus.Note != nil || t.Focus.Asset != nil) {
		return t.Focus
	}
	if p := t.firstEditorPane(); p != nil {
		return p
	}
	return firstPane(t.Root)
}

// firstEditorPane returns the first pane that shows a file, not a shell.
func (t *tab) firstEditorPane() *pane {
	for _, p := range t.panes() {
		if p.Note != nil || p.Asset != nil {
			return p
		}
	}
	return nil
}

// terminalPane returns the tab's terminal, if it has one.
func (t *tab) terminalPane() *pane {
	for _, p := range t.panes() {
		if p.Term != nil {
			return p
		}
	}
	return nil
}

// dirty reports whether any note of the tab has unsaved edits.
func (t *tab) dirty() bool {
	for _, p := range t.panes() {
		if p.Note != nil && p.Note.dirty() {
			return true
		}
	}
	return false
}

// split puts with beside p, or below it. It reports whether p is in this
// tab.
func (t *tab) split(p, with *pane, down bool) bool {
	if t.Root == nil {
		return false
	}
	if !t.Root.split(p, with, down) {
		return false
	}
	t.Focus = with
	t.Zoom = nil
	return true
}

// swapPanes exchanges two panes' places, keeping their contents and the
// tab's layout.
func (t *tab) swapPanes(a, b *pane) bool {
	if t == nil || a == nil || b == nil || a == b || t.Root == nil {
		return false
	}
	if !t.contains(a) || !t.contains(b) {
		return false
	}
	t.Root.swapPanes(a, b)
	return true
}

func (n *node) swapPanes(a, b *pane) {
	if n == nil {
		return
	}
	if n.Pane == a {
		n.Pane = b
		return
	}
	if n.Pane == b {
		n.Pane = a
		return
	}
	n.A.swapPanes(a, b)
	n.B.swapPanes(a, b)
}

func (n *node) split(p, with *pane, down bool) bool {
	if n.Pane != nil {
		if n.Pane != p {
			return false
		}
		old := n.Pane
		*n = node{
			Vertical: down,
			Ratio:    0.5,
			A:        &node{Pane: old},
			B:        &node{Pane: with},
		}
		n.startSplitAnim(0.5)
		return true
	}
	if n.A != nil && n.A.split(p, with, down) {
		return true
	}
	return n.B != nil && n.B.split(p, with, down)
}

// closePane removes p. It returns false when the tab is left with no
// pane.
func (t *tab) closePane(p *pane) bool {
	if t.Root == nil {
		return false
	}
	if t.Root.Pane == p {
		t.Root = nil
		t.Focus, t.Zoom = nil, nil
		return false
	}
	if !t.Root.remove(p) {
		return true
	}
	if t.Zoom == p {
		t.Zoom = nil
	}
	if t.Focus == p {
		t.Focus = firstPane(t.Root)
	}
	return true
}

func (n *node) remove(p *pane) bool {
	if n.A != nil && n.A.Pane == p {
		n.adopt(n.B)
		return true
	}
	if n.B != nil && n.B.Pane == p {
		n.adopt(n.A)
		return true
	}
	if n.A != nil && n.A.remove(p) {
		return true
	}
	return n.B != nil && n.B.remove(p)
}

// adopt makes n the node other, as a split collapses into its one
// remaining pane.
func (n *node) adopt(other *node) {
	if other == nil {
		*n = node{}
		return
	}
	*n = *other
}

func firstPane(n *node) *pane {
	if n == nil {
		return nil
	}
	if n.Pane != nil {
		return n.Pane
	}
	if p := firstPane(n.A); p != nil {
		return p
	}
	return firstPane(n.B)
}

// equalize makes every split of the tab half and half.
func (t *tab) equalize() {
	var walk func(*node)
	walk = func(n *node) {
		if n == nil || n.Pane != nil {
			return
		}
		n.Ratio = 0.5
		walk(n.A)
		walk(n.B)
	}
	walk(t.Root)
}

// layoutFile is what the app keeps of its tabs between launches.
type layoutFile struct {
	Tabs   []layoutTab `json:"tabs"`
	Active int         `json:"active"`
}

type layoutTab struct {
	Name  string      `json:"name,omitempty"`
	Root  *layoutNode `json:"root"`
	Focus string      `json:"focus,omitempty"`
	Zoom  string      `json:"zoom,omitempty"`
}

type layoutNode struct {
	Vertical bool        `json:"vertical,omitempty"`
	Ratio    float32     `json:"ratio,omitempty"`
	A        *layoutNode `json:"a,omitempty"`
	B        *layoutNode `json:"b,omitempty"`
	Pane     *layoutPane `json:"pane,omitempty"`
}

type layoutPane struct {
	ID      string `json:"id"`
	Note    string `json:"note,omitempty"`
	Asset   string `json:"asset,omitempty"`
	Preview bool   `json:"preview,omitempty"`
}

func layoutOf(t *tab) layoutTab {
	lt := layoutTab{Name: t.Name, Root: layoutNodeOf(t.Root)}
	if t.Focus != nil {
		lt.Focus = t.Focus.ID
	}
	if t.Zoom != nil {
		lt.Zoom = t.Zoom.ID
	}
	return lt
}

func layoutNodeOf(n *node) *layoutNode {
	if n == nil {
		return nil
	}
	if n.Pane != nil {
		if n.Pane.Note == nil && n.Pane.Asset == nil {
			return nil // shells are not saved; the file panes around them are
		}
		lp := &layoutPane{ID: n.Pane.ID, Preview: n.Pane.Preview}
		if n.Pane.Note != nil {
			lp.Note = n.Pane.Note.Rel
		} else {
			lp.Asset = n.Pane.Asset.Rel
		}
		return &layoutNode{Pane: lp}
	}
	return &layoutNode{
		Vertical: n.Vertical,
		Ratio:    n.Ratio,
		A:        layoutNodeOf(n.A),
		B:        layoutNodeOf(n.B),
	}
}

func (a *app) snapshotLayout() layoutFile {
	f := layoutFile{Active: a.active}
	for _, t := range a.tabs {
		f.Tabs = append(f.Tabs, layoutOf(t))
	}
	return f
}

// restoreLayout builds the tabs of the saved layout, dropping panes whose
// note is no longer in the vault. It reports whether any tab came back.
func (a *app) restoreLayout() bool {
	if a.layoutPath == "" {
		return false
	}
	data, err := os.ReadFile(a.layoutPath)
	if err != nil {
		return false
	}
	var f layoutFile
	if err := json.Unmarshal(data, &f); err != nil {
		return false
	}
	byID := map[string]*pane{}
	for _, lt := range f.Tabs {
		t := &tab{ID: nextID("t"), Name: lt.Name}
		t.Root = a.restoreNode(lt.Root, byID)
		if t.Root == nil {
			continue
		}
		if p := byID[lt.Focus]; p != nil && t.contains(p) {
			t.Focus = p
		} else {
			t.Focus = firstPane(t.Root)
		}
		if p := byID[lt.Zoom]; p != nil && t.contains(p) {
			t.Zoom = p
		}
		a.tabs = append(a.tabs, t)
	}
	if len(a.tabs) == 0 {
		return false
	}
	a.active = min(max(f.Active, 0), len(a.tabs)-1)
	return true
}

func (a *app) restoreNode(ln *layoutNode, byID map[string]*pane) *node {
	if ln == nil {
		return nil
	}
	if ln.Pane != nil {
		p := &pane{ID: nextID("p"), Preview: ln.Pane.Preview}
		if n := a.store.byRelPath(ln.Pane.Note); n != nil {
			p.Note = n
		} else if f := a.store.assetByRel(ln.Pane.Asset); f != nil {
			p.Asset = f
			p.assetView = &assetView{}
			p.Preview = false
		} else {
			return nil
		}
		byID[ln.Pane.ID] = p
		return &node{Pane: p}
	}
	left := a.restoreNode(ln.A, byID)
	right := a.restoreNode(ln.B, byID)
	switch {
	case left == nil && right == nil:
		return nil
	case left == nil:
		return right
	case right == nil:
		return left
	}
	ratio := ln.Ratio
	if ratio <= 0 || ratio >= 1 || math.IsNaN(float64(ratio)) {
		ratio = 0.5
	}
	return &node{Vertical: ln.Vertical, Ratio: ratio, A: left, B: right}
}

func (t *tab) contains(p *pane) bool {
	for _, q := range t.panes() {
		if q == p {
			return true
		}
	}
	return false
}

// saveLayout writes the tabs to the app's data directory.
func (a *app) saveLayout() {
	if a.layoutPath == "" {
		return
	}
	data, err := json.MarshalIndent(a.snapshotLayout(), "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(a.layoutPath), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(a.layoutPath, data, 0o644)
}
