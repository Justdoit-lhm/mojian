package app

import (
	"mojian/internal/app/explorer"
	"mojian/internal/project"

	"github.com/egoist/mygo/ui/editor"
)

// These aliases keep the window's vocabulary — *store, *note, *assetFile,
// treeDraft — while the types live in the sibling packages. They are names,
// not logic: a one-line forwarder is the most this file should hold.

// The project model lives in internal/project; these are the names the
// window's code reads.
type store = project.Store
type note = project.Note
type assetFile = project.Asset
type fileNode = project.TreeNode

func isMarkdownPath(path string) bool { return project.IsMarkdownPath(path) }

func wrapFor(path string) bool { return project.WrapFor(path) }

func lineNumbersFor(path string) editor.LineNumbers { return project.LineNumbersFor(path) }

// The Explorer's draft type and constants keep the app's vocabulary.
type treeDraft = explorer.Draft

const (
	draftFile   = explorer.DraftFile
	draftFolder = explorer.DraftFolder
	draftRename = explorer.DraftRename
)
