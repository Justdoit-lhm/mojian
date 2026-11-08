package app

// prefs is the set of preferences the Settings window edits. It is
// embedded in app and in settingsFile, so loading, saving, resetting and
// the JSON keys all come from this one declaration.
type prefs struct {
	Appearance          int     `json:"appearance"`
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
}

func defaultPrefs() prefs {
	return prefs{
		EditorFontSize: 12.5, EditorLineHeight: 1.55, EditorTabSize: 4,
		TerminalFontSize: 12.5, TerminalLineHeight: 1, TerminalScrollback: 10,
		NotifyBell: true, NotifyExit: true,
	}
}

func clampFloat(v, lo, hi, def float64) float64 {
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

func clampInt(v, lo, hi, def int) int {
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

// sanitize keeps a hand-edited settings.json in the ranges the Settings
// window can show.
func (p *prefs) sanitize() {
	if p.Appearance < 0 || p.Appearance > 2 {
		p.Appearance = 0
	}
	p.EditorFontSize = clampFloat(p.EditorFontSize, 9, 28, 12.5)
	p.EditorLineHeight = clampFloat(p.EditorLineHeight, 1.1, 2.2, 1.55)
	p.EditorTabSize = clampInt(p.EditorTabSize, 1, 8, 4)
	p.TerminalFontSize = clampFloat(p.TerminalFontSize, 9, 28, 12.5)
	p.TerminalLineHeight = clampFloat(p.TerminalLineHeight, 0.8, 2, 1)
	if p.TerminalCursor < 0 || p.TerminalCursor > 2 {
		p.TerminalCursor = 0
	}
	p.TerminalScrollback = clampInt(p.TerminalScrollback, 1, 100, 10)
}
