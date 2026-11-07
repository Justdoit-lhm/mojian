package motion

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// Motion is quiet and short: switching a tab or a page should explain
// where the last one went, not perform. Enter and exit both ease out, and
// MyGo drops the movement, keeping the fade, when the desktop asks for
// less motion.
const motionMs = 170 * time.Millisecond

var motionEase = ui.CubicBezier(0.23, 1, 0.32, 1)

// pageMotion swaps a tab's content, the editor and Git, and a project:
// the old page leaves upward, the new one arrives from below.
var Page = ui.ElementTransition{
	Duration: motionMs,
	Ease:     motionEase,
	Enter:    &ui.Motion{Y: 6, Opacity: 0},
	Exit:     &ui.Motion{Y: -4, Opacity: 0},
}

// tabMotion moves a tab into and out of the strip as tabs are added and
// closed; its colors follow the active tab separately.
var Tab = ui.ElementTransition{
	Duration: 150 * time.Millisecond,
	Ease:     motionEase,
	Position: true,
	Colors:   true,
	Enter:    &ui.Motion{Y: 3, Opacity: 0},
	Exit:     &ui.Motion{Y: -2, Opacity: 0},
}

// treeMotion opens and closes a folder's children.
var Tree = ui.ElementTransition{
	Duration: 130 * time.Millisecond,
	Ease:     motionEase,
	Enter:    &ui.Motion{Y: -3, Opacity: 0},
	Exit:     &ui.Motion{Y: -3, Opacity: 0},
}
