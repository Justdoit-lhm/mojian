package app

import (
	"mojian/internal/project"

	"github.com/egoist/mygo/ui/editor"
)

// The project model lives in internal/app/project. These aliases keep the
// window's code reading as `*store`, `*note` and `*assetFile`, the names
// the rest of the app has always used.

type store = project.Store
type note = project.Note
type assetFile = project.Asset
type fileNode = project.TreeNode

func isMarkdownPath(path string) bool { return project.IsMarkdownPath(path) }

func wrapFor(path string) bool { return project.WrapFor(path) }

func lineNumbersFor(path string) editor.LineNumbers { return project.LineNumbersFor(path) }
