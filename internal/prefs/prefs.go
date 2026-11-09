// Package prefs is the preference schema Mojian keeps between launches.
// The Settings window edits these fields, and settings.json stores them.
package prefs

// Prefs is the set of preferences the Settings window edits. The app and
// its settings file embed it, so loading, saving, resetting and the JSON
// keys all come from this one declaration.
type Prefs struct {
	Appearance          int     `json:"appearance"`
	Theme               string  `json:"theme,omitempty"`
	EditorFontFamily    string  `json:"editorFontFamily,omitempty"`
	EditorFontSize      float64 `json:"editorFontSize,omitempty"`
	EditorLineHeight    float64 `json:"editorLineHeight,omitempty"`
	EditorTabSize       int     `json:"editorTabSize,omitempty"`
	EditorSoftWrap      bool    `json:"editorSoftWrap,omitempty"`
	HideLineNumbers     bool    `json:"hideLineNumbers,omitempty"`
	MarkdownPreview     bool    `json:"markdownPreview,omitempty"`
	NoActiveLine        bool    `json:"noActiveLine,omitempty"`
	NoIndentGuides      bool    `json:"noIndentGuides,omitempty"`
	NoAutoClose         bool    `json:"noAutoClose,omitempty"`
	NoScrollPastEnd     bool    `json:"noScrollPastEnd,omitempty"`
	TerminalFontFamily  string  `json:"terminalFontFamily,omitempty"`
	TerminalFontSize    float64 `json:"terminalFontSize,omitempty"`
	ShellPath           string  `json:"shellPath,omitempty"`
	TerminalCursor      int     `json:"terminalCursor,omitempty"`
	NotifyBell          bool    `json:"notifyBell,omitempty"`
	NotifyExit          bool    `json:"notifyExit,omitempty"`
	TerminalLineHeight  float64 `json:"terminalLineHeight,omitempty"`
	TerminalOptionAsAlt bool    `json:"terminalOptionAsAlt,omitempty"`
	TerminalCopySelect  bool    `json:"terminalCopySelect,omitempty"`
	NoCursorBlink       bool    `json:"noCursorBlink,omitempty"`
	TerminalScrollback  int     `json:"terminalScrollback,omitempty"`
	NoRestoreLayout     bool    `json:"noRestoreLayout,omitempty"`
	NoGitAutoRefresh    bool    `json:"noGitAutoRefresh,omitempty"`
	ShellCommand        string  `json:"shellCommand,omitempty"`
}

// defaults is the settings of a new install. Default and the effective
// value getters of the app both read it, so a default lives in one place.
var defaults = Prefs{
	Theme:          "petal",
	EditorFontSize: 12.5, EditorLineHeight: 1.55, EditorTabSize: 4,
	TerminalFontSize: 12.5, TerminalLineHeight: 1, TerminalScrollback: 10,
	NotifyBell: true, NotifyExit: true,
}

// Default is the settings of a new install.
func Default() Prefs { return defaults }

// clamp keeps a hand-edited number inside the range the Settings window
// can show, and restores def when it is unset.
func clamp(v, lo, hi, def float64) float64 {
	if v <= 0 {
		return def
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Sanitize keeps a hand-edited settings.json in the ranges the Settings
// window can show.
func (p *Prefs) Sanitize() {
	if p.Appearance < 0 || p.Appearance > 2 {
		p.Appearance = 0
	}
	// Unknown ids are resolved by theme.Resolve, so only an unset one
	// needs a value here.
	if p.Theme == "" {
		p.Theme = "petal"
	}
	p.EditorFontSize = clamp(p.EditorFontSize, 9, 28, defaults.EditorFontSize)
	p.EditorLineHeight = clamp(p.EditorLineHeight, 1.1, 2.2, defaults.EditorLineHeight)
	p.EditorTabSize = int(clamp(float64(p.EditorTabSize), 1, 8, float64(defaults.EditorTabSize)))
	p.TerminalFontSize = clamp(p.TerminalFontSize, 9, 28, defaults.TerminalFontSize)
	p.TerminalLineHeight = clamp(p.TerminalLineHeight, 0.8, 2, defaults.TerminalLineHeight)
	if p.TerminalCursor < 0 || p.TerminalCursor > 2 {
		p.TerminalCursor = 0
	}
	p.TerminalScrollback = int(clamp(float64(p.TerminalScrollback), 1, 100, float64(defaults.TerminalScrollback)))
	switch p.ShellCommand {
	case "", "zsh", "bash":
	default:
		p.ShellCommand = ""
	}
}
