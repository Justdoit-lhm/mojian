package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
)

// pane is one editor view of a file, a picture, or a terminal.
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
	closing       bool // fading out before the split closes
	closeStarted  bool
	hideStarted   bool    // a zoom hid it; its fade over the old place has begun
	lastBounds    ui.Rect // where the card showed, for its overlay
	enter         bool    // fades and slides in as a new pane, or back from a zoom
	enterX        float32
	enterY        float32
	emptyQuery    string // the file search in an empty pane
	emptySel      int
}

// assetView is what a picture or PDF pane has loaded, and its headings.
type assetView struct {
	pdfDir     string
	pdfCount   int // the document's pages, 0 until the count is read
	pdfPage    int
	pdfErr     string // why the document could not be opened
	pdfLoading bool
	pdfStarted bool
	pdfGen     int            // invalidates work whose pane was repurposed
	pdfImage   ui.ImageSource // the page pdfImageAt was decoded for
	pdfImageAt int
	pdfFiles   map[int]string // page (0-based) to the PNG drawn for it
	pdfPending map[int]bool   // pages being drawn now
	pdfPageErr map[int]string // why a page could not be drawn
}

// node is a leaf pane, or two panes side by side or stacked. Share is the
// first side's percentage of the room, from 0 to 100; the divider between
// the two sides sets it, and zero takes half.
type node struct {
	Vertical bool // the panes are stacked, one above the other
	Share    float32
	A, B     *node
	Pane     *pane
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

func newTermPaneSpec(spec termSpec, sid string) *termTab {
	return &termTab{Spec: spec, SID: sid}
}

func newAssetPane(f *assetFile) *pane {
	return &pane{ID: nextID("p"), Asset: f, assetView: &assetView{}}
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

func newTab(n *note) *tab {
	p := newPane(n)
	return &tab{ID: nextID("t"), Root: &node{Pane: p}, Focus: p}
}

// label is what the tab strip shows: the focused note's title, or the
// tab's own name, and how many panes it holds.
func (t *tab) label() (name, detail string) {
	name = t.Name
	if name == "" && t.Focus != nil {
		name = paneName(t.Focus)
	}
	if name == "" {
		name = "Welcome"
	}
	if n := len(t.panes()); n > 1 {
		detail = fmt.Sprintf("· %d", n)
	}
	return name, detail
}

// editorPane returns the pane to focus when the tab appears: an editor
// if it has one, else the terminal.
func (t *tab) editorPane() *pane {
	if t == nil {
		return nil
	}
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
		if p.Note != nil && p.Note.Dirty() {
			return true
		}
	}
	return false
}

// splitSide is where a split puts the new pane relative to the pane it
// splits.
type splitSide int

const (
	splitRight splitSide = iota
	splitDown
	splitLeft
	splitUp
)

func (s splitSide) vertical() bool { return s == splitDown || s == splitUp }
func (s splitSide) before() bool   { return s == splitLeft || s == splitUp }

// paneDrop is where a dragged pane is let go over another pane: the
// middle does nothing, an edge inserts it on that side.
type paneDrop int

const (
	dropNone paneDrop = iota - 1
	dropCenter
	dropAbove
	dropBelow
	dropLeft
	dropRight
)

func (d paneDrop) vertical() bool { return d == dropAbove || d == dropBelow }
func (d paneDrop) before() bool   { return d == dropAbove || d == dropLeft }

// splitSide puts with to one side of p. It reports whether p is in this
// tab.
func (t *tab) splitSide(p, with *pane, side splitSide) bool {
	if t.Root == nil || p == nil || with == nil || p == with || t.contains(with) {
		return false
	}
	if !t.Root.splitSide(p, with, side) {
		return false
	}
	t.Focus = with
	t.Zoom = nil
	return true
}

func (n *node) splitSide(p, with *pane, side splitSide) bool {
	if n == nil {
		return false
	}
	if n.Pane != nil {
		if n.Pane != p {
			return false
		}
		old := n.Pane
		left, right := &node{Pane: old}, &node{Pane: with}
		if side.before() {
			left, right = right, left
		}
		*n = node{Vertical: side.vertical(), Share: 50, A: left, B: right}
		return true
	}
	if n.A != nil && n.A.splitSide(p, with, side) {
		return true
	}
	return n.B != nil && n.B.splitSide(p, with, side)
}

// movePane takes from out of the tree and puts it on the edge of to.
func (t *tab) movePane(from, to *pane, zone paneDrop) bool {
	if t == nil || from == nil || to == nil || from == to || t.Root == nil {
		return false
	}
	if !t.contains(from) || !t.contains(to) || zone == dropNone || zone == dropCenter {
		return false
	}
	if t.Root.Pane == from {
		return false // from is the only pane; to cannot also be here
	}
	if !t.Root.remove(from) {
		return false
	}
	if !t.wrapPane(to, from, zone) {
		return false
	}
	t.Focus = from
	t.Zoom = nil
	return true
}

// wrapPane replaces to's leaf with a split holding from on the named
// side. to must still be in the tree.
func (t *tab) wrapPane(to, from *pane, zone paneDrop) bool {
	if t.Root == nil || !t.contains(to) || zone == dropNone || zone == dropCenter {
		return false
	}
	split := &node{Vertical: zone.vertical(), Share: 50}
	toLeaf, fromLeaf := &node{Pane: to}, &node{Pane: from}
	if zone.before() {
		split.A, split.B = fromLeaf, toLeaf
	} else {
		split.A, split.B = toLeaf, fromLeaf
	}
	if t.Root.Pane == to {
		t.Root = split
		return true
	}
	return t.Root.wrapLeaf(to, split)
}

// wrapLeaf replaces the leaf showing p with replacement.
func (n *node) wrapLeaf(p *pane, replacement *node) bool {
	if n == nil || n.Pane != nil {
		return false
	}
	if n.A != nil && n.A.Pane == p {
		n.A = replacement
		return true
	}
	if n.B != nil && n.B.Pane == p {
		n.B = replacement
		return true
	}
	if n.A != nil && n.A.wrapLeaf(p, replacement) {
		return true
	}
	return n.B != nil && n.B.wrapLeaf(p, replacement)
}

// paneInDirection finds the nearest pane in a direction, climbing out
// of splits that run the other way.
func (t *tab) paneInDirection(dir paneDirection) *pane {
	if t.Root == nil || t.Focus == nil {
		return nil
	}
	path := t.Root.pathTo(t.Focus)
	for i := len(path) - 2; i >= 0; i-- {
		n := path[i]
		child := path[i+1]
		var target *pane
		switch {
		case n.Vertical && child == n.B && dir == paneUp:
			target = lastPane(n.A)
		case n.Vertical && child == n.A && dir == paneDown:
			target = firstPane(n.B)
		case !n.Vertical && child == n.B && dir == paneLeft:
			target = lastPane(n.A)
		case !n.Vertical && child == n.A && dir == paneRight:
			target = firstPane(n.B)
		}
		if target != nil && target != t.Focus {
			return target
		}
	}
	return nil
}

// paneDirection is one of the four screen directions.
type paneDirection int

const (
	paneLeft paneDirection = iota
	paneRight
	paneUp
	paneDown
)

// pathTo returns the nodes from the root to p, or nil.
func (n *node) pathTo(p *pane) []*node {
	if n == nil {
		return nil
	}
	if n.Pane == p {
		return []*node{n}
	}
	if n.A != nil {
		if path := n.A.pathTo(p); path != nil {
			return append([]*node{n}, path...)
		}
	}
	if n.B != nil {
		if path := n.B.pathTo(p); path != nil {
			return append([]*node{n}, path...)
		}
	}
	return nil
}

func lastPane(n *node) *pane {
	if n == nil {
		return nil
	}
	if n.Pane != nil {
		return n.Pane
	}
	if p := lastPane(n.B); p != nil {
		return p
	}
	return lastPane(n.A)
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
	if n == nil {
		return false
	}
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
// remaining pane, or into the tree of the other side.
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

func (t *tab) contains(p *pane) bool {
	return t.Root.pathTo(p) != nil
}
