package app

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

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

	// W and H are the pane's place on the canvas, as fractions of the
	// viewport. A split to the right gives the new pane half a viewport;
	// a split down gives it a full viewport height. Unlike the old tree
	// ratios, these do not shrink the panes already there.
	W, H float32

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
}

// tab is a workspace: a tree of panes, the one with the focus, and the
// one zoomed, if any.
type tab struct {
	ID    string
	Name  string // a name of its own, else the focused note's title
	Root  *node
	Focus *pane
	Zoom  *pane
	// panX and panY are the top-left of the canvas shown in the
	// viewport. The viewport follows the focused pane; the pointer does
	// not scroll it.
	panX, panY float32
}

var idSeq int

func nextID(prefix string) string {
	idSeq++
	return fmt.Sprintf("%s%d", prefix, idSeq)
}

func newPane(n *note) *pane {
	return &pane{ID: nextID("p"), Note: n, W: paneColumnWidth, H: 1}
}

func newTermPane(t *termTab) *pane {
	return &pane{ID: nextID("p"), Term: t, W: paneColumnWidth, H: 1}
}

func newTermPaneSpec(spec termSpec, sid string) *termTab {
	return &termTab{Spec: spec, SID: sid}
}

func newAssetPane(f *assetFile) *pane {
	return &pane{ID: nextID("p"), Asset: f, assetView: &assetView{}, W: paneColumnWidth, H: 1}
}

func (p *pane) width() float32 {
	if p == nil || p.W <= 0 {
		return 1
	}
	return p.W
}

func (p *pane) height() float32 {
	if p == nil || p.H <= 0 {
		return 1
	}
	return p.H
}

// extent is the canvas size of a tree, in DIPs for a viewport of vw and
// vh. Splits keep their gap, and siblings larger than another leave the
// shorter one at its own size.
func (n *node) extent(vw, vh float32) (w, h float32) {
	if n == nil {
		return 0, 0
	}
	if n.Pane != nil {
		return n.Pane.width() * vw, n.Pane.height() * vh
	}
	aw, ah := n.A.extent(vw, vh)
	bw, bh := n.B.extent(vw, vh)
	if n.Vertical {
		return max(aw, bw), ah + gap + bh
	}
	return aw + gap + bw, max(ah, bh)
}

// rectOf returns p's place on the canvas, in DIPs.
func (n *node) rectOf(p *pane, vw, vh float32) (x, y, w, h float32, ok bool) {
	if n == nil {
		return
	}
	if n.Pane != nil {
		if n.Pane != p {
			return
		}
		return 0, 0, n.Pane.width() * vw, n.Pane.height() * vh, true
	}
	aw, ah := n.A.extent(vw, vh)
	if n.Vertical {
		if x, y, w, h, ok = n.A.rectOf(p, vw, vh); ok {
			return
		}
		if x, y, w, h, ok = n.B.rectOf(p, vw, vh); ok {
			return x, y + ah + gap, w, h, true
		}
		return
	}
	if x, y, w, h, ok = n.A.rectOf(p, vw, vh); ok {
		return
	}
	if x, y, w, h, ok = n.B.rectOf(p, vw, vh); ok {
		return x + aw + gap, y, w, h, true
	}
	return 0, 0, 0, 0, false
}

// followFocus pans as little as needed to show the focused pane.
func (t *tab) followFocus(vw, vh float32) {
	if t == nil || t.Root == nil || t.Focus == nil || vw <= 0 || vh <= 0 {
		return
	}
	cw, ch := t.Root.extent(vw, vh)
	x, y, w, h, ok := t.Root.rectOf(t.Focus, vw, vh)
	if !ok {
		return
	}
	t.panX = followAxis(t.panX, x, w, vw, cw)
	t.panY = followAxis(t.panY, y, h, vh, ch)
}

func followAxis(pos, at, size, view, content float32) float32 {
	if content <= view {
		return 0
	}
	margin := float32(gap)
	switch {
	case size >= view:
		pos = at
	case at < pos+margin:
		pos = at - margin
	case at+size > pos+view-margin:
		pos = at + size - view + margin
	}
	return min(max(pos, 0), content-view)
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

func (n *node) panes() []*pane {
	var out []*pane
	n.walk(func(p *pane) { out = append(out, p) })
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
			name = t.Focus.Term.Spec.title()
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

// paneColumnWidth is just under half a viewport so two columns and their
// gap fit the window without a sliver being clipped.
const paneColumnWidth = 0.495

// splitSide is where a split puts the new pane relative to the pane it
// splits. The tree can grow in every direction, without a limit.
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
// middle swaps them, and an edge inserts it on that side.
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

type paneDirection int

const (
	paneLeft paneDirection = iota
	paneRight
	paneUp
	paneDown
)

// split puts with beside p, or below it.
func (t *tab) split(p, with *pane, down bool) bool {
	side := splitRight
	if down {
		side = splitDown
	}
	return t.splitSide(p, with, side)
}

// splitSide puts with to one side of p. It reports whether p is in this
// tab.
func (t *tab) splitSide(p, with *pane, side splitSide) bool {
	if t.Root == nil || p == nil || with == nil || p == with || t.contains(with) {
		return false
	}
	if !t.Root.splitSide(p, with, side) {
		return false
	}
	// New columns are half a viewport wide, and new rows are a full
	// viewport high. The canvas grows around them instead of squeezing
	// the panes that were already there.
	switch side {
	case splitRight, splitLeft:
		if p.width() > paneColumnWidth {
			p.W = paneColumnWidth
		}
		with.W, with.H = paneColumnWidth, 1
	case splitDown, splitUp:
		// A vertical extension keeps the half-screen column width and
		// grows to a full viewport height.
		with.W, with.H = p.width(), 1
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

// movePane moves from into the place a drop edge names around to, or
// swaps the two on a center drop. The rest of the tree is collapsed as
// from leaves it, and the new split grows from to's side.
func (t *tab) movePane(from, to *pane, zone paneDrop) bool {
	if t == nil || from == nil || to == nil || from == to || t.Root == nil {
		return false
	}
	if !t.contains(from) || !t.contains(to) {
		return false
	}
	if zone == dropNone {
		return false
	}
	if zone == dropCenter {
		if !t.swapPanes(from, to) {
			return false
		}
		t.Focus = from
		t.Zoom = nil
		return true
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

// movePaneInPlace moves from onto to and splits to's current block in
// half. The canvas does not push the pair outside the page: the dragged
// pane takes half of the target's old rectangle.
func (t *tab) movePaneInPlace(from, to *pane, zone paneDrop) bool {
	if t == nil || from == nil || to == nil || from == to || t.Root == nil {
		return false
	}
	if !t.contains(from) || !t.contains(to) || zone == dropNone {
		return false
	}
	if zone == dropCenter {
		return t.movePane(from, to, dropCenter)
	}
	toW, toH := to.width(), to.height()
	if t.Root.Pane == from {
		return false
	}
	if !t.Root.remove(from) {
		return false
	}
	if zone.vertical() {
		to.W, to.H = toW, max(toH/2, 0.1)
		from.W, from.H = toW, max(toH/2, 0.1)
	} else {
		to.W, to.H = max(toW/2, 0.1), toH
		from.W, from.H = max(toW/2, 0.1), toH
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
	split := &node{
		Vertical: zone.vertical(),
		Ratio:    0.5,
	}
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
// of splits that run the other way. It is how niri's focus commands move
// around a tiling.
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

// paneAtDirection picks the pane that visually lies in a direction. The
// split tree answers most cases, but full-screen rows and uneven canvas
// sizes make geometry the better answer for the keyboard.
func (t *tab) paneAtDirection(dir paneDirection, vw, vh float32) *pane {
	if t.Root == nil || t.Focus == nil || vw <= 0 || vh <= 0 {
		return nil
	}
	cx, cy, cw, ch, ok := t.Root.rectOf(t.Focus, vw, vh)
	if !ok {
		return nil
	}
	cxm, cym := cx+cw/2, cy+ch/2
	best, bestScore := (*pane)(nil), float32(1e30)
	consider := func(p *pane, primary, secondary float32) {
		score := primary + secondary*0.25
		if score < bestScore {
			best, bestScore = p, score
		}
	}
	for _, p := range t.panes() {
		if p == t.Focus {
			continue
		}
		x, y, w, h, ok := t.Root.rectOf(p, vw, vh)
		if !ok {
			continue
		}
		pxm, pym := x+w/2, y+h/2
		switch dir {
		case paneLeft:
			if x+w <= cx+0.5 {
				consider(p, cx-(x+w), absFloat(cym-pym))
			}
		case paneRight:
			if x >= cx+cw-0.5 {
				consider(p, x-(cx+cw), absFloat(cym-pym))
			}
		case paneUp:
			if y+h <= cy+0.5 {
				consider(p, cy-(y+h), absFloat(cxm-pxm))
			}
		case paneDown:
			if y >= cy+ch-0.5 {
				consider(p, y-(cy+ch), absFloat(cxm-pxm))
			}
		}
	}
	if best != nil {
		return best
	}
	// Fall back to a pane whose center lies that way, as a diagonal move
	// when the layouts do not share an edge.
	for _, p := range t.panes() {
		if p == t.Focus {
			continue
		}
		x, y, w, h, ok := t.Root.rectOf(p, vw, vh)
		if !ok {
			continue
		}
		pxm, pym := x+w/2, y+h/2
		switch dir {
		case paneLeft:
			if pxm < cxm {
				consider(p, cxm-pxm, absFloat(cym-pym))
			}
		case paneRight:
			if pxm > cxm {
				consider(p, pxm-cxm, absFloat(cym-pym))
			}
		case paneUp:
			if pym < cym {
				consider(p, cym-pym, absFloat(cxm-pxm))
			}
		case paneDown:
			if pym > cym {
				consider(p, pym-cym, absFloat(cxm-pxm))
			}
		}
	}
	return best
}

func absFloat(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// focusDirection moves the focus to the nearest pane in a direction.
func (t *tab) focusDirection(dir paneDirection) bool {
	target := t.paneInDirection(dir)
	if target == nil {
		return false
	}
	t.Focus = target
	return true
}

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

func (n *node) split(p, with *pane, down bool) bool {
	side := splitRight
	if down {
		side = splitDown
	}
	return n.splitSide(p, with, side)
}

func (n *node) splitSide(p, with *pane, side splitSide) bool {
	if n.Pane != nil {
		if n.Pane != p {
			return false
		}
		old := n.Pane
		left, right := &node{Pane: old}, &node{Pane: with}
		if side.before() {
			left, right = right, left
		}
		*n = node{
			Vertical: side.vertical(),
			Ratio:    0.5,
			A:        left,
			B:        right,
		}
		return true
	}
	if n.A != nil && n.A.splitSide(p, with, side) {
		return true
	}
	return n.B != nil && n.B.splitSide(p, with, side)
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

// equalize balances every split of the tab around its divider.
func (t *tab) equalize() {
	balanceAll(t.Root)
}

func balanceAll(n *node) {
	if n == nil || n.Pane != nil {
		return
	}
	n.balance()
	balanceAll(n.A)
	balanceAll(n.B)
}

// balance makes a split's two sides the same extent along its axis.
func (n *node) balance() {
	if n == nil || n.Pane != nil {
		return
	}
	aw, ah := n.A.extent(1, 1)
	bw, bh := n.B.extent(1, 1)
	if n.Vertical {
		total := ah + bh
		if total <= 0 {
			return
		}
		target := total / 2
		scaleHeights(n.A, safeScale(target, ah))
		scaleHeights(n.B, safeScale(target, bh))
		return
	}
	total := aw + bw
	if total <= 0 {
		return
	}
	target := total / 2
	scaleWidths(n.A, safeScale(target, aw))
	scaleWidths(n.B, safeScale(target, bw))
}

func safeScale(target, have float32) float32 {
	if have <= 0 {
		return 1
	}
	return target / have
}

func scaleWidths(n *node, factor float32) {
	if n == nil || factor <= 0 {
		return
	}
	if n.Pane != nil {
		n.Pane.W = max(0.12, n.Pane.W*factor)
		return
	}
	scaleWidths(n.A, factor)
	scaleWidths(n.B, factor)
}

func scaleHeights(n *node, factor float32) {
	if n == nil || factor <= 0 {
		return
	}
	if n.Pane != nil {
		n.Pane.H = max(0.12, n.Pane.H*factor)
		return
	}
	scaleHeights(n.A, factor)
	scaleHeights(n.B, factor)
}

// resize moves a split's divider by delta DIPs along its axis.
func (n *node) resize(delta, vw, vh float32) {
	if n == nil || n.Pane != nil || n.A == nil || n.B == nil {
		return
	}
	if n.Vertical {
		if vh <= 0 {
			return
		}
		delta /= vh
		ap, bp := n.A.panes(), n.B.panes()
		shiftHeights(ap, delta)
		shiftHeights(bp, -delta)
		return
	}
	if vw <= 0 {
		return
	}
	delta /= vw
	ap, bp := n.A.panes(), n.B.panes()
	shiftWidths(ap, delta)
	shiftWidths(bp, -delta)
}

func shiftWidths(ps []*pane, delta float32) {
	if len(ps) == 0 {
		return
	}
	d := delta / float32(len(ps))
	for _, p := range ps {
		p.W = min(max(p.W+d, 0.12), 4)
	}
}

func shiftHeights(ps []*pane, delta float32) {
	if len(ps) == 0 {
		return
	}
	d := delta / float32(len(ps))
	for _, p := range ps {
		p.H = min(max(p.H+d, 0.12), 4)
	}
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
	ID       string          `json:"id"`
	Note     string          `json:"note,omitempty"`
	Asset    string          `json:"asset,omitempty"`
	Preview  bool            `json:"preview,omitempty"`
	W        float32         `json:"w,omitempty"`
	H        float32         `json:"h,omitempty"`
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
		lp := &layoutPane{ID: n.Pane.ID, Preview: n.Pane.Preview, W: n.Pane.W, H: n.Pane.H}
		switch {
		case n.Pane.Term != nil:
			lp.Terminal = &layoutTerminal{SID: n.Pane.Term.SID, Host: n.Pane.Term.Spec.Host}
		case n.Pane.Note != nil:
			lp.Note = n.Pane.Note.Rel
		case n.Pane.Asset != nil:
			lp.Asset = n.Pane.Asset.Rel
		default:
			return nil
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
		t.assignLegacySizes()
		if p := byID[lt.Focus]; p != nil && t.contains(p) {
			t.Focus = p
		} else {
			t.Focus = firstPane(t.Root)
		}
		// A lone pane from an older layout is a file opened at the old
		// full-screen default; make it the new half-screen default.
		if len(t.panes()) == 1 && t.Focus != nil && t.Focus.W > paneColumnWidth {
			t.Focus.W = paneColumnWidth
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
		p := &pane{ID: nextID("p"), Preview: ln.Pane.Preview, W: ln.Pane.W, H: ln.Pane.H}
		switch {
		case ln.Pane.Terminal != nil:
			p.Term = newTermPaneSpec(termSpec{Host: ln.Pane.Terminal.Host}, ln.Pane.Terminal.SID)
		case ln.Pane.Note != "":
			n := a.store.byRelPath(ln.Pane.Note)
			if n == nil {
				return nil
			}
			p.Note = n
		case ln.Pane.Asset != "":
			f := a.store.assetByRel(ln.Pane.Asset)
			if f == nil {
				return nil
			}
			p.Asset = f
			p.assetView = &assetView{}
			p.Preview = false
		default:
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

// assignLegacySizes gives panes from layouts saved before the canvas
// their old proportions: one viewport wide and tall at the root, split
// by the saved tree ratios.
func (t *tab) assignLegacySizes() {
	if t.Root == nil {
		return
	}
	var walk func(*node, float32, float32) bool
	walk = func(n *node, w, h float32) bool {
		if n == nil {
			return false
		}
		if n.Pane != nil {
			if n.Pane.W <= 0 || n.Pane.H <= 0 {
				if n.Pane.W <= 0 {
					n.Pane.W = w
				}
				if n.Pane.H <= 0 {
					n.Pane.H = h
				}
				return true
			}
			return false
		}
		ratio := n.Ratio
		if ratio <= 0 || ratio >= 1 {
			ratio = 0.5
		}
		var migrated bool
		if n.Vertical {
			migrated = walk(n.A, w, h*ratio)
			if walk(n.B, w, h*(1-ratio)) {
				migrated = true
			}
		} else {
			migrated = walk(n.A, w*ratio, h)
			if walk(n.B, w*(1-ratio), h) {
				migrated = true
			}
		}
		return migrated
	}
	walk(t.Root, 1, 1)
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
