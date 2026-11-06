package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/egoist/mygo/ui/editor"
)

// pane is one editor view of a note in a workspace.
type pane struct {
	ID    string
	Note  *note
	state *editor.State
}

// node is a leaf pane, or two panes side by side or stacked, with the
// ratio the first takes. It is the reference's tree of splits.
type node struct {
	Vertical bool // the panes are stacked, one above the other
	Ratio    float32
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
	if name == "" && t.Focus != nil && t.Focus.Note != nil {
		name = t.Focus.Note.Title
	}
	if name == "" {
		name = "Untitled"
	}
	if n := len(t.panes()); n > 1 {
		detail = fmt.Sprintf("· %d", n)
	}
	return name, detail
}

// editorPane returns the pane to focus when the tab appears.
func (t *tab) editorPane() *pane {
	if t.Focus != nil {
		return t.Focus
	}
	return firstPane(t.Root)
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
	ID   string `json:"id"`
	Note string `json:"note"`
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
		return &layoutNode{Pane: &layoutPane{
			ID:   n.Pane.ID,
			Note: n.Pane.Note.Rel,
		}}
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
		n := a.store.byRelPath(ln.Pane.Note)
		if n == nil {
			return nil
		}
		p := &pane{ID: nextID("p"), Note: n}
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
