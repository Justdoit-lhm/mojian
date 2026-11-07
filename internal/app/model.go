package app

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

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
	emptyQuery    string // the file search in an empty pane
	emptySel      int
}

// assetView is what a picture or PDF pane has loaded, and its headings.
type assetView struct {
	pdfPages   []string
	pdfPage    int
	pdfDir     string
	pdfErr     string
	pdfLoading bool
	pdfStarted bool
	pdfGen     int // invalidates a preview whose pane was repurposed
	pdfImage   ui.ImageSource
	pdfImageAt int // the page pdfImage was decoded for
}

// node is a leaf pane, or two panes side by side or stacked. There are
// no pixel sizes: each side of a split takes the same share of the space
// it is given.
type node struct {
	Vertical bool // the panes are stacked, one above the other
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
		if p.Note != nil && p.Note.dirty() {
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
		*n = node{Vertical: side.vertical(), A: left, B: right}
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
	split := &node{Vertical: zone.vertical()}
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

// layoutFile is what the app keeps of its tabs between launches.
type layoutFile struct {
	Tabs   []layoutTab `json:"tabs"`
	Active int         `json:"active"`
}

type layoutTab struct {
	Name  string      `json:"name,omitempty"`
	Root  *layoutNode `json:"root,omitempty"`
	Focus string      `json:"focus,omitempty"`
	Zoom  string      `json:"zoom,omitempty"`
	// Columns and Panes are the grid formats written just before the
	// split tree came back; they are read so their panes survive.
	Columns [][]layoutPane `json:"columns,omitempty"`
	Panes   []layoutPane   `json:"panes,omitempty"`
}

type layoutNode struct {
	Vertical bool        `json:"vertical,omitempty"`
	Ratio    float32     `json:"ratio,omitempty"`
	A        *layoutNode `json:"a,omitempty"`
	B        *layoutNode `json:"b,omitempty"`
	Pane     *layoutPane `json:"pane,omitempty"`
}

type layoutPane struct {
	ID       string          `json:"id"`
	Note     string          `json:"note,omitempty"`
	Asset    string          `json:"asset,omitempty"`
	Preview  bool            `json:"preview,omitempty"`
	Terminal *layoutTerminal `json:"terminal,omitempty"`
}

type layoutTerminal struct {
	SID  string `json:"sid,omitempty"`
	Host string `json:"host,omitempty"`
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
		lp := layoutPaneOf(n.Pane)
		if lp == nil {
			return nil
		}
		return &layoutNode{Pane: lp}
	}
	return &layoutNode{
		Vertical: n.Vertical,
		A:        layoutNodeOf(n.A),
		B:        layoutNodeOf(n.B),
	}
}

func layoutPaneOf(p *pane) *layoutPane {
	if p == nil {
		return nil
	}
	lp := &layoutPane{ID: p.ID, Preview: p.Preview}
	switch {
	case p.Term != nil:
		lp.Terminal = &layoutTerminal{SID: p.Term.SID, Host: p.Term.Spec.Host}
	case p.Note != nil:
		lp.Note = p.Note.Rel
	case p.Asset != nil:
		lp.Asset = p.Asset.Rel
	default:
		return nil
	}
	return lp
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
		switch {
		case lt.Root != nil:
			t.Root = a.restoreNode(lt.Root, byID)
		case len(lt.Columns) > 0:
			t.Root = a.restoreColumns(lt.Columns, byID)
		case len(lt.Panes) > 0:
			// The old one-row format: every pane was a column.
			var cols [][]layoutPane
			for _, lp := range lt.Panes {
				cols = append(cols, []layoutPane{lp})
			}
			t.Root = a.restoreColumns(cols, byID)
		}
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
		p := a.restorePane(*ln.Pane, byID)
		if p == nil {
			return nil
		}
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
	return &node{Vertical: ln.Vertical, A: left, B: right}
}

// restoreColumns turns the old grid into a split tree: columns side by
// side, each column's panes stacked.
func (a *app) restoreColumns(cols [][]layoutPane, byID map[string]*pane) *node {
	var columns []*node
	for _, lc := range cols {
		var panes []*node
		for _, lp := range lc {
			if p := a.restorePane(lp, byID); p != nil {
				panes = append(panes, &node{Pane: p})
			}
		}
		if len(panes) > 0 {
			columns = append(columns, joinNodes(panes, true))
		}
	}
	return joinNodes(columns, false)
}

func joinNodes(ns []*node, vertical bool) *node {
	if len(ns) == 0 {
		return nil
	}
	root := ns[0]
	for _, n := range ns[1:] {
		root = &node{Vertical: vertical, A: root, B: n}
	}
	return root
}

func (a *app) restorePane(lp layoutPane, byID map[string]*pane) *pane {
	p := &pane{ID: nextID("p"), Preview: lp.Preview}
	switch {
	case lp.Terminal != nil:
		p.Term = newTermPaneSpec(termSpec{Host: lp.Terminal.Host}, lp.Terminal.SID)
	case lp.Note != "":
		n := a.store.byRelPath(lp.Note)
		if n == nil {
			return nil
		}
		p.Note = n
	case lp.Asset != "":
		f := a.store.assetByRel(lp.Asset)
		if f == nil {
			return nil
		}
		p.Asset = f
		p.assetView = &assetView{}
		p.Preview = false
	default:
		return nil
	}
	byID[lp.ID] = p
	return p
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
	if err := writeFileAtomic(a.layoutPath, data); err != nil {
		log.Print(err)
	}
}
