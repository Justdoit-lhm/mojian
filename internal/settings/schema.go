// Package settings holds the Settings window's pages as data: the rail,
// the rows and the words search matches. The app renders it and applies
// the preferences; this package has no dependency on the window.
package settings

type Kind uint8

const (
	Toggle Kind = iota
	Choice
	Slider
	Text
	Font
	Info
	Custom
)

const (
	General = iota
	Appearance
	Editor
	Terminal
	Keyboard
	Hosts
	Advanced
	About
)

type Category struct {
	Title  string
	Detail string
	Glyph  string
	Words  string
}

// Categories is the rail, in order.
var Categories = []Category{
	{"General", "Startup, project and working tree", "settings", "startup restore tabs project server git working tree refresh"},
	{"Appearance", "How Mojian looks", "palette", "theme light dark appearance look colors mode auto palette"},
	{"Editor", "Type, writing and Markdown", "pencil", "editor code font line height tab wrap markdown active line indent brackets scroll"},
	{"Terminal", "Type, input, notifications and the shell", "square-terminal", "terminal shell font cursor blink scrollback clipboard option alt notifications bell exit"},
	{"Keyboard", "Shortcuts this app uses", "command", "keyboard shortcut keys hotkeys keybindings commands"},
	{"Hosts", "SSH servers and remote projects", "server", "host server ssh remote project address"},
	{"Advanced", "Shell command, data, settings file and reset", "sliders-horizontal", "advanced shell command zsh bash alias install data settings file reset paths server session socket"},
	{"About", "Version, updates and system", "info", "about mojian version updates release install restart system fonts project"},
}

// Row is one setting, info line or action. Pref names an exported field
// of the app's prefs struct; the app reads and writes it.
type Row struct {
	ID      string
	Cat     int
	Section string
	Title   string
	Detail  string
	Words   string
	Kind    Kind
	Pref    string
	Invert  bool
	Min     float64
	Max     float64
	Format  string
	Options []string
	Values  []int
	// FontWhich is 0 for the editor face, 1 for the terminal face.
	FontWhich int
	// Placeholder is the empty text of a Text row.
	Placeholder string
}

// Rows are the pages' contents, in display order. Keyboard, Hosts,
// Advanced and About paint custom controls; their rows still feed search
// and their section headers.
var Rows = []Row{
	// General
	{ID: "restore-tabs", Cat: General, Section: "SESSIONS", Title: "Restore tabs on launch",
		Detail: "Bring back the last project's tabs and panes.", Words: "startup restore layout tabs panes launch",
		Kind: Toggle, Pref: "NoRestoreLayout", Invert: true},
	{ID: "current-project", Cat: General, Section: "PROJECT", Title: "Current project",
		Words: "project folder path current", Kind: Info},
	{ID: "server", Cat: General, Section: "PROJECT", Title: "Server",
		Words: "server ssh local remote host", Kind: Info},
	{ID: "git-refresh", Cat: General, Section: "WORKING TREE", Title: "Refresh when the window returns",
		Detail: "Read the working tree again when the app regains focus.", Words: "git refresh focus auto worktree",
		Kind: Toggle, Pref: "NoGitAutoRefresh", Invert: true},

	// Appearance (the page paints the mode row and the theme grid; these
	// rows feed search)
	{ID: "appearance", Cat: Appearance, Section: "THEME", Title: "Mode",
		Detail: "System, Light or Dark.",
		Words:  "appearance mode system light dark auto follow", Kind: Choice, Pref: "Appearance",
		Options: []string{"System", "Light", "Dark"}},
	{ID: "appearance-theme", Cat: Appearance, Section: "THEME", Title: "Theme",
		Detail: "The window's colors.", Words: "theme color colors colour palette " +
			"petal mist meadow dusk tide apricot aurora dune paper plain",
		Kind: Text, Pref: "Theme"},

	// Editor
	{ID: "editor-font", Cat: Editor, Section: "TYPE", Title: "Editor font",
		Detail: "The family used by source and prose.", Words: "font family editor typeface monospace",
		Kind: Font, Pref: "EditorFontFamily", FontWhich: 0},
	{ID: "editor-font-size", Cat: Editor, Section: "TYPE", Title: "Font size",
		Detail: "In points, from 9 to 28.", Words: "font size editor text bigger smaller",
		Kind: Slider, Pref: "EditorFontSize", Min: 9, Max: 28, Format: "%.1f"},
	{ID: "editor-line-height", Cat: Editor, Section: "TYPE", Title: "Line height",
		Detail: "Room for each line, as a multiple of the size.", Words: "line height leading spacing editor",
		Kind: Slider, Pref: "EditorLineHeight", Min: 1.1, Max: 2.2, Format: "%.2f"},
	{ID: "editor-tab-size", Cat: Editor, Section: "TYPE", Title: "Tab size",
		Detail: "Columns per tab.", Words: "tab size indent columns spaces", Kind: Choice, Pref: "EditorTabSize",
		Options: []string{"2", "4", "8"}, Values: []int{2, 4, 8}},
	{ID: "line-numbers", Cat: Editor, Section: "WRITING", Title: "Line numbers",
		Detail: "Number the lines of code files; Markdown stays clean.", Words: "line numbers gutter code markdown",
		Kind: Toggle, Pref: "HideLineNumbers", Invert: true},
	{ID: "soft-wrap", Cat: Editor, Section: "WRITING", Title: "Soft wrap code",
		Detail: "Wrap long code lines at the pane's width.", Words: "soft wrap long lines code width",
		Kind: Toggle, Pref: "EditorSoftWrap"},
	{ID: "markdown-preview", Cat: Editor, Section: "WRITING", Title: "Preview Markdown by default",
		Detail: "Open .md files rendered, with the Edit button to switch back.", Words: "markdown preview render md default",
		Kind: Toggle, Pref: "MarkdownPreview"},
	{ID: "active-line", Cat: Editor, Section: "WRITING", Title: "Highlight active line",
		Detail: "Show a quiet tint behind the caret's line.", Words: "active line caret highlight tint",
		Kind: Toggle, Pref: "NoActiveLine", Invert: true},
	{ID: "indent-guides", Cat: Editor, Section: "WRITING", Title: "Indent guides",
		Detail: "Lines that show each level of indentation.", Words: "indent guides levels lines",
		Kind: Toggle, Pref: "NoIndentGuides", Invert: true},
	{ID: "auto-close", Cat: Editor, Section: "WRITING", Title: "Auto-close brackets",
		Detail: "Close quotes and brackets as you type.", Words: "brackets quotes auto close typing",
		Kind: Toggle, Pref: "NoAutoClose", Invert: true},
	{ID: "scroll-past-end", Cat: Editor, Section: "WRITING", Title: "Scroll past the last line",
		Detail: "Keep a little air below the end of the file.", Words: "scroll past end air bottom line",
		Kind: Toggle, Pref: "NoScrollPastEnd", Invert: true},

	// Terminal
	{ID: "terminal-font", Cat: Terminal, Section: "TYPE", Title: "Terminal font",
		Detail: "The family used by new shells.", Words: "font family terminal shell typeface monospace",
		Kind: Font, Pref: "TerminalFontFamily", FontWhich: 1},
	{ID: "terminal-font-size", Cat: Terminal, Section: "TYPE", Title: "Font size",
		Detail: "In points, from 9 to 28.", Words: "font size terminal text",
		Kind: Slider, Pref: "TerminalFontSize", Min: 9, Max: 28, Format: "%.1f"},
	{ID: "terminal-line-height", Cat: Terminal, Section: "TYPE", Title: "Line height",
		Detail: "Room for each terminal row.", Words: "line height terminal rows spacing",
		Kind: Slider, Pref: "TerminalLineHeight", Min: 0.8, Max: 2, Format: "%.2f"},
	{ID: "cursor-shape", Cat: Terminal, Section: "CURSOR", Title: "Cursor shape",
		Words: "cursor shape block bar underline caret", Kind: Choice, Pref: "TerminalCursor",
		Options: []string{"Block", "Bar", "Underline"}},
	{ID: "cursor-blink", Cat: Terminal, Section: "CURSOR", Title: "Blink the cursor",
		Detail: "Let the terminal's cursor blink when the program allows it.", Words: "cursor blink caret",
		Kind: Toggle, Pref: "NoCursorBlink", Invert: true},
	{ID: "option-alt", Cat: Terminal, Section: "INPUT", Title: "Option as Alt",
		Detail: "Send Option to programs as Meta instead of typing accented letters.", Words: "option alt meta keyboard macos",
		Kind: Toggle, Pref: "TerminalOptionAsAlt"},
	{ID: "copy-select", Cat: Terminal, Section: "INPUT", Title: "Copy on select",
		Detail: "Copy a selection to the clipboard as it ends.", Words: "copy select clipboard selection",
		Kind: Toggle, Pref: "TerminalCopySelect"},
	{ID: "history-size", Cat: Terminal, Section: "SCROLLBACK", Title: "History size",
		Detail: "Megabytes kept above the screen.", Words: "scrollback history size memory megabytes mb",
		Kind: Slider, Pref: "TerminalScrollback", Min: 1, Max: 100, Format: "%.0f MB"},
	{ID: "notify-bell", Cat: Terminal, Section: "NOTIFICATIONS", Title: "Notify on a bell",
		Detail: "Show a notification when a background terminal rings its bell.", Words: "bell notify attention sound terminal",
		Kind: Toggle, Pref: "NotifyBell"},
	{ID: "notify-exit", Cat: Terminal, Section: "NOTIFICATIONS", Title: "Notify when a terminal exits",
		Detail: "Show a notification when a background terminal finishes.", Words: "exit notify finish done terminal",
		Kind: Toggle, Pref: "NotifyExit"},
	{ID: "shell-path", Cat: Terminal, Section: "SHELL", Title: "Shell",
		Detail: "Program new local terminals run; empty uses $SHELL.", Words: "shell login program zsh bash fish environment",
		Kind: Text, Pref: "ShellPath", Placeholder: "$SHELL"},

	// Keyboard (custom controls; rows feed search)
	{ID: "show-tab", Cat: Keyboard, Section: "TAB SWITCHING", Title: "Show tab",
		Detail: "Go to a tab by its position.", Words: "tab number position shortcut", Kind: Custom},

	// Hosts (custom controls; rows feed search)
	{ID: "setup-server", Cat: Hosts, Section: "SERVERS", Title: "Set Up Server",
		Words: "server setup install upload binaries start stop restart persistent", Kind: Custom},

	// Advanced (custom controls; rows feed search)
	{ID: "shell-command", Cat: Advanced, Section: "COMMAND LINE", Title: "Shell command",
		Detail: "Make mojian run from zsh or bash.", Words: "shell command line terminal alias zsh bash cli path install launcher",
		Kind: Custom, Pref: "ShellCommand"},
	{ID: "settings-file", Cat: Advanced, Section: "SETTINGS FILE", Title: "Settings file",
		Words: "settings file json path", Kind: Custom},
	{ID: "reveal-settings", Cat: Advanced, Section: "SETTINGS FILE", Title: "Reveal",
		Words: "reveal show finder settings file", Kind: Custom},
	{ID: "open-settings", Cat: Advanced, Section: "SETTINGS FILE", Title: "Open",
		Words: "open settings file json editor", Kind: Custom},
	{ID: "reset-settings", Cat: Advanced, Section: "RESET", Title: "Reset all settings",
		Words: "reset defaults restore settings preferences", Kind: Custom},

	// About (custom controls; rows feed search)
	{ID: "software-update", Cat: About, Section: "SOFTWARE UPDATE", Title: "Software Update",
		Detail: "Check GitHub for a newer signed release.", Words: "updates check release download github", Kind: Custom},
	{ID: "install-update", Cat: About, Section: "SOFTWARE UPDATE", Title: "Install",
		Detail: "Download and install the release it finds.", Words: "updates install download", Kind: Custom},
	{ID: "restart-update", Cat: About, Section: "SOFTWARE UPDATE", Title: "Restart",
		Detail: "Restart into the installed update.", Words: "updates restart install apply", Kind: Custom},
	{ID: "about-system", Cat: About, Section: "SYSTEM", Title: "System",
		Words: "about system os architecture", Kind: Custom},
	{ID: "about-project", Cat: About, Section: "SYSTEM", Title: "Project",
		Words: "about project folder path", Kind: Custom},
	{ID: "about-fonts", Cat: About, Section: "SYSTEM", Title: "Fonts",
		Words: "about fonts installed families", Kind: Custom},
}
