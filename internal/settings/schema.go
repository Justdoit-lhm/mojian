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
	Shell
	Notifications
	Keyboard
	Hosts
	Git
	Advanced
	Updates
)

type Category struct {
	Title  string
	Detail string
	Glyph  string
	Words  string
}

// Categories is the rail, in order.
var Categories = []Category{
	{"General", "Startup and projects", "settings", "startup restore tabs project server"},
	{"Appearance", "Theme and interface text", "file-text", "theme light dark appearance look color font size"},
	{"Editor", "Type, writing and Markdown", "pencil", "editor code font line height tab wrap markdown active line indent brackets scroll"},
	{"Terminal", "Type, input and scrollback", "square-terminal", "terminal shell font cursor blink scrollback clipboard option alt"},
	{"Shell", "The program new terminals run", "square-terminal", "shell login command zsh bash fish"},
	{"Notifications", "When a terminal wants attention", "bell", "notify bell exit attention sound"},
	{"Keyboard", "Shortcuts this app uses", "command", "keyboard shortcut keys hotkeys keybindings commands"},
	{"Hosts", "SSH servers and recent projects", "server", "host server ssh remote project address"},
	{"Git", "The working-tree page", "git-branch", "git changes diff status refresh worktree"},
	{"Advanced", "Data and settings file", "settings", "advanced data settings file reset paths server"},
	{"Updates", "Signed releases and restart", "refresh-cw", "updates update release download install restart"},
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
// Advanced and Updates paint custom controls; their rows still feed search
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

	// Appearance
	{ID: "appearance", Cat: Appearance, Section: "THEME", Title: "Appearance",
		Words: "theme appearance light dark system look", Kind: Choice, Pref: "Appearance",
		Options: []string{"System", "Light", "Dark"}},
	{ID: "appearance-text-size", Cat: Appearance, Section: "TEXT", Title: "Editor text size",
		Detail: "The base size of source and prose.", Words: "font size editor text bigger smaller zoom",
		Kind: Slider, Pref: "EditorFontSize", Min: 9, Max: 28, Format: "%.1f pt"},
	{ID: "appearance-editor-font", Cat: Appearance, Section: "TEXT", Title: "Editor font",
		Detail: "The face used by editors and previews.", Words: "font family editor typeface monospace",
		Kind: Font, Pref: "EditorFontFamily", FontWhich: 0},

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
		Words: "tab size indent columns spaces", Kind: Choice, Pref: "EditorTabSize",
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

	// Shell
	{ID: "shell-path", Cat: Shell, Section: "LOGIN SHELL", Title: "Shell",
		Detail: "Program new local terminals run; empty uses $SHELL.", Words: "shell login program zsh bash fish environment",
		Kind: Text, Pref: "ShellPath", Placeholder: "$SHELL"},

	// Notifications
	{ID: "notify-bell", Cat: Notifications, Section: "TERMINAL", Title: "Notify on a bell",
		Detail: "Show a notification when a background terminal rings its bell.", Words: "bell notify attention sound terminal",
		Kind: Toggle, Pref: "NotifyBell"},
	{ID: "notify-exit", Cat: Notifications, Section: "TERMINAL", Title: "Notify when a terminal exits",
		Detail: "Show a notification when a background terminal finishes.", Words: "exit notify finish done terminal",
		Kind: Toggle, Pref: "NotifyExit"},

	// Keyboard (custom controls; rows feed search)
	{ID: "shortcuts", Cat: Keyboard, Section: "SHORTCUTS", Title: "SHORTCUTS",
		Words: "shortcuts keys keybindings hotkeys keyboard commands", Kind: Custom},
	{ID: "pane-navigation", Cat: Keyboard, Section: "PANE NAVIGATION", Title: "PANE NAVIGATION",
		Words: "pane navigation focus move keys", Kind: Custom},
	{ID: "move-focus", Cat: Keyboard, Section: "PANE NAVIGATION", Title: "Move focus",
		Detail: "Focus the neighboring pane.", Words: "pane navigation focus arrow keys", Kind: Custom},
	{ID: "move-pane", Cat: Keyboard, Section: "PANE NAVIGATION", Title: "Move the focused pane",
		Detail: "Put it on the neighbor's edge.", Words: "pane move reorder arrow keys", Kind: Custom},
	{ID: "tab-switching", Cat: Keyboard, Section: "TAB SWITCHING", Title: "TAB SWITCHING",
		Words: "tab switching next previous keys", Kind: Custom},
	{ID: "show-tab", Cat: Keyboard, Section: "TAB SWITCHING", Title: "Show tab",
		Detail: "Go to a tab by its position.", Words: "tab number position shortcut", Kind: Custom},
	{ID: "trash-shortcut", Cat: Keyboard, Section: "TAB SWITCHING", Title: "Move to Trash",
		Detail: "On a file in the file tree.", Words: "trash delete remove file tree shortcut", Kind: Custom},

	// Hosts (custom controls; rows feed search)
	{ID: "add-host", Cat: Hosts, Section: "ADD HOST", Title: "ADD HOST",
		Words: "add host ssh server address", Kind: Custom},
	{ID: "host-address", Cat: Hosts, Section: "ADD HOST", Title: "Address",
		Words: "host ssh server address", Kind: Custom},
	{ID: "servers", Cat: Hosts, Section: "SERVERS", Title: "SERVERS",
		Words: "servers hosts ssh local remote", Kind: Custom},
	{ID: "setup-server", Cat: Hosts, Section: "SERVERS", Title: "Set Up Server",
		Words: "server setup install upload binaries", Kind: Custom},
	{ID: "recent-remote", Cat: Hosts, Section: "RECENT REMOTE PROJECTS", Title: "RECENT REMOTE PROJECTS",
		Words: "recent remote projects ssh", Kind: Custom},
	{ID: "about-hosts", Cat: Hosts, Section: "ABOUT HOSTS", Title: "ABOUT HOSTS",
		Words: "hosts remote server upload ssh", Kind: Custom},

	// Git
	{ID: "git-refresh", Cat: Git, Section: "CHANGES PAGE", Title: "Refresh when the window returns",
		Detail: "Read the working tree again when the app regains focus.", Words: "git refresh focus auto worktree",
		Kind: Toggle, Pref: "NoGitAutoRefresh", Invert: true},
	{ID: "git-status", Cat: Git, Section: "ABOUT THE PAGE", Title: "Status",
		Words: "git status working tree head", Kind: Info},

	// Advanced (custom controls; rows feed search)
	{ID: "data", Cat: Advanced, Section: "DATA", Title: "DATA",
		Words: "data paths settings", Kind: Custom},
	{ID: "settings-file", Cat: Advanced, Section: "DATA", Title: "Settings file",
		Words: "settings file json path", Kind: Custom},
	{ID: "remote-project", Cat: Advanced, Section: "DATA", Title: "Remote project",
		Words: "remote project path server", Kind: Custom},
	{ID: "local-sessions", Cat: Advanced, Section: "DATA", Title: "Local sessions",
		Words: "local sessions socket terminal", Kind: Custom},
	{ID: "remote-server", Cat: Advanced, Section: "DATA", Title: "Remote server",
		Words: "remote server socket ssh", Kind: Custom},
	{ID: "settings-file-section", Cat: Advanced, Section: "SETTINGS FILE", Title: "SETTINGS FILE",
		Words: "settings file open reveal json", Kind: Custom},
	{ID: "reveal-settings", Cat: Advanced, Section: "SETTINGS FILE", Title: "Reveal",
		Words: "reveal show finder settings file", Kind: Custom},
	{ID: "open-settings", Cat: Advanced, Section: "SETTINGS FILE", Title: "Open",
		Words: "open settings file json editor", Kind: Custom},
	{ID: "reset-settings", Cat: Advanced, Section: "SETTINGS FILE", Title: "Reset all settings",
		Words: "reset defaults restore settings preferences", Kind: Custom},
	{ID: "about", Cat: Advanced, Section: "MOJIAN", Title: "MOJIAN",
		Words: "about mojian version credits", Kind: Custom},
	{ID: "about-version", Cat: Advanced, Section: "MOJIAN", Title: "Version",
		Words: "about version build", Kind: Custom},
	{ID: "about-system", Cat: Advanced, Section: "MOJIAN", Title: "System",
		Words: "about system os architecture", Kind: Custom},
	{ID: "about-project", Cat: Advanced, Section: "MOJIAN", Title: "Project",
		Words: "about project folder", Kind: Custom},
	{ID: "about-settings", Cat: Advanced, Section: "MOJIAN", Title: "Settings",
		Words: "about settings path", Kind: Custom},
	{ID: "about-fonts", Cat: Advanced, Section: "MOJIAN", Title: "Fonts",
		Words: "about fonts installed", Kind: Custom},

	// Updates (custom controls; rows feed search)
	{ID: "updates", Cat: Updates, Section: "UPDATES", Title: "UPDATES",
		Words: "updates release signed", Kind: Custom},
	{ID: "check-updates", Cat: Updates, Section: "UPDATES", Title: "Check for Updates",
		Detail: "Check the latest signed release on GitHub.", Words: "updates check release download", Kind: Custom},
	{ID: "install-update", Cat: Updates, Section: "UPDATES", Title: "Install",
		Detail: "Download and install the release it finds.", Words: "updates install download", Kind: Custom},
	{ID: "restart-update", Cat: Updates, Section: "UPDATES", Title: "Restart",
		Detail: "Restart into the installed update.", Words: "updates restart install apply", Kind: Custom},
}
