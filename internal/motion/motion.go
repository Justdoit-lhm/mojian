package motion

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// The app moves on two clocks. Springs carry everything that can change
// again on the way — a page, a tab, the tree — so switching tabs in quick
// succession reverses from wherever the last switch got to instead of
// restarting a curve. Critically damped springs (Bounce 0) settle without
// overshooting: a quiet editor never bounces. Colors keep a short curve,
// because they only have to catch up with the pointer.
//
// The responses are Apple's idea of spring duration: smaller is snappier.
// They stay short enough that a switch never reads as a performance.
const (
	pageResponse  = 240 * time.Millisecond
	tabResponse   = 200 * time.Millisecond
	treeResponse  = 220 * time.Millisecond
	paneResponse  = 220 * time.Millisecond
	colorDuration = 120 * time.Millisecond
)

// EaseOut is a strong ease-out for short fades and color changes. It
// starts fast, so feedback never lags behind the pointer.
var EaseOut = ui.CubicBezier(0.23, 1, 0.32, 1)

// spring is a critically damped spring that arrives in about d.
func spring(d time.Duration) *ui.Animation {
	return &ui.Animation{Spring: ui.SpringOptions{Duration: d}}
}

// The tree's rows and its unfolding children share one spring, so they
// move in step as a folder opens or closes.
var treeSpring = spring(treeResponse)

// paneSpring carries a pane the layout moves, and a new pane sliding in,
// at the same pace.
var paneSpring = spring(paneResponse)

var (
	// Page swaps a tab's content, the editor and Git, and a project: the
	// old page leaves upward, the new one arrives from below.
	Page = ui.ElementTransition{
		Animation: spring(pageResponse),
		Enter:     &ui.Motion{Y: 4, Opacity: 0},
		Exit:      &ui.Motion{Y: -4, Opacity: 0},
	}

	// Tab moves a tab into and out of the strip as tabs are added and
	// closed; its colors follow the active tab.
	Tab = ui.ElementTransition{
		Animation: spring(tabResponse),
		Position:  true,
		Colors:    true,
		Enter:     &ui.Motion{Y: 2, Opacity: 0},
		Exit:      &ui.Motion{Y: -2, Opacity: 0},
	}

	// Row moves a tree row when a folder above it opens or closes, so the
	// rows below glide with the unfolding children instead of jumping.
	Row = ui.ElementTransition{
		Animation: treeSpring,
		Position:  true,
	}

	// Colors makes hover faces catch up with the pointer.
	Colors = ui.ElementTransition{
		Duration: colorDuration,
		Ease:     EaseOut,
		Colors:   true,
	}

	// Pane moves a pane card that the layout puts elsewhere — into a new
	// split, or back to full width when one closes — from where it showed.
	// Set Match to the pane's ID at the call site; without it (terminals,
	// whose grid would reflow at every size on the way) the card only
	// fades its colors as focus moves.
	Pane = ui.ElementTransition{
		Animation: paneSpring,
		Colors:    true,
	}

	// PaneSlide moves a new pane card from the edge of its split into
	// place, with the spring the pane beside it resizes with.
	PaneSlide = *paneSpring

	// FadeIn brings a new pane card in, which the engine cannot enter with
	// a Motion because its split is new too: the view animates its Opacity
	// with AnimateTo instead.
	FadeIn = ui.Animation{Duration: 160 * time.Millisecond, Ease: EaseOut}

	// Close fades a pane out before its split closes, moving it back
	// toward the edge it entered from. The view removes it when the
	// animation rests, since the split node goes with the pane and the
	// engine cannot keep an Exit copy.
	Close = ui.Animation{Duration: 160 * time.Millisecond, Ease: EaseOut}
)

// Tree is the transition of a folder's children: they unfold while the
// rows below glide with Row. A collapse moves its siblings, which MyGo
// drops when the desktop asks for less motion, so there the children only
// fade in place and nothing jumps.
func Tree(reduceMotion bool) ui.ElementTransition {
	if reduceMotion {
		return ui.ElementTransition{
			Animation: treeSpring,
			Enter:     &ui.Motion{Opacity: 0},
			Exit:      &ui.Motion{Opacity: 0},
		}
	}
	return ui.ElementTransition{
		Animation: treeSpring,
		Enter:     &ui.Motion{Opacity: 0, Collapse: true},
		Exit:      &ui.Motion{Opacity: 0, Collapse: true},
	}
}
