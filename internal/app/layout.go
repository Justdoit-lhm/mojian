package app

import (
	"encoding/json"
	"log"
	"os"

	"mojian/internal/fs"
)

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
	Share    float32     `json:"share,omitempty"`
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
	SID    string `json:"sid,omitempty"`
	Host   string `json:"host,omitempty"`
	Server string `json:"server,omitempty"`
}

func layoutOf(t *tab) layoutTab {
	lt := layoutTab{Name: t.Name, Root: layoutNodeOf(t.Root)}
	if t.Focus != nil && !t.Focus.closing {
		lt.Focus = t.Focus.ID
	}
	if t.Zoom != nil && !t.Zoom.closing {
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
		Share:    n.Share,
		A:        layoutNodeOf(n.A),
		B:        layoutNodeOf(n.B),
	}
}

func layoutPaneOf(p *pane) *layoutPane {
	if p == nil || p.closing {
		return nil
	}
	lp := &layoutPane{ID: p.ID, Preview: p.Preview}
	switch {
	case p.Term != nil:
		lp.Terminal = &layoutTerminal{SID: p.Term.SID, Host: p.Term.Spec.Host, Server: p.Term.Spec.Server}
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
	return &node{Vertical: ln.Vertical, Share: layoutShare(ln.Share), A: left, B: right}
}

// layoutShare reads a saved split share, giving layouts from before the
// dividers half.
func layoutShare(share float32) float32 {
	if share <= 0 || share >= 100 {
		return 50
	}
	return share
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
		root = &node{Vertical: vertical, Share: 50, A: root, B: n}
	}
	return root
}

func (a *app) restorePane(lp layoutPane, byID map[string]*pane) *pane {
	p := &pane{ID: nextID("p"), Preview: lp.Preview}
	switch {
	case lp.Terminal != nil:
		p.Term = newTermPaneSpec(termSpec{Host: lp.Terminal.Host, Server: lp.Terminal.Server}, lp.Terminal.SID)
	case lp.Note != "":
		n := a.store.ByRelPath(lp.Note)
		if n == nil || !a.loadNote(n) {
			return nil
		}
		p.Note = n
	case lp.Asset != "":
		f := a.store.AssetByRel(lp.Asset)
		if f == nil || !a.loadAsset(f) {
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
	if err := fs.WriteFileAtomic(a.layoutPath, data); err != nil {
		log.Print(err)
	}
}
