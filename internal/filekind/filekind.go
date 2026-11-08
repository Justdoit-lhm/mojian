// Package filekind classifies a project file as text, a picture or PDF
// the viewers show, or something the tree leaves out, and names the
// generated folders a project hides. The app and the session server both
// use it, so a remote project lists the same files as a local one.
package filekind

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui/editor"
)

// Kind is what a file opens as.
type Kind uint8

const (
	// Other is a binary the app does not show: archives, media, objects.
	Other Kind = iota
	// Text opens in the editor.
	Text
	// Asset opens in the picture or PDF viewer.
	Asset
)

// HeadSize is how much of an unknown file is read to tell text from
// binary.
const HeadSize = 8192

// SkipDir keeps generated trees and version control out of the project
// view; a code project's own sources still show.
func SkipDir(name string) bool {
	switch name {
	case ".git", ".svn", ".hg", "node_modules", "__pycache__", ".venv", "venv",
		"dist", "build", ".next", ".cache", ".turbo", "target", "vendor",
		".idea", ".gradle", "Pods", "DerivedData", ".mojian-trash":
		return true
	}
	return false
}

// textExts are text files MyGo has no highlighter for, or the editor
// shows as plain text.
var textExts = map[string]bool{
	".txt": true, ".text": true, ".log": true, ".csv": true, ".tsv": true,
	".rst": true, ".org": true, ".tex": true, ".bib": true, ".nfo": true,
	".md": true, ".markdown": true,
}

// binaryExts are never read as text, even if a prefix could pass.
var binaryExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
	".ico": true, ".icns": true, ".pdf": true, ".zip": true, ".gz": true,
	".tar": true, ".7z": true, ".rar": true, ".dmg": true, ".app": true,
	".mp3": true, ".mp4": true, ".mov": true, ".woff": true, ".woff2": true,
	".ttf": true, ".otf": true, ".exe": true, ".dylib": true, ".so": true,
	".a": true, ".o": true, ".class": true, ".jar": true, ".wasm": true,
	".sqlite": true, ".db": true, ".mojian-tmp": true,
}

// assetExts are the files the viewer shows: pictures MyGo can decode,
// and PDFs it previews through the system.
var assetExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
	".bmp": true, ".tif": true, ".tiff": true, ".ico": true, ".svg": true,
	".pdf": true,
}

// Classify returns what path opens as, using head (its first HeadSize
// bytes, or nil when not read) for extensions nothing else knows.
func Classify(path string, head []byte) Kind {
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case assetExts[ext]:
		return Asset
	case binaryExts[ext]:
		return Other
	case editor.LanguageFor(path) != nil || textExts[ext]:
		return Text
	case len(head) == 0:
		return Text
	case bytes.IndexByte(head, 0) >= 0:
		return Other
	}
	controls := 0
	for _, b := range head {
		if b < 0x09 || b > 0x0d && b < 0x20 {
			controls++
		}
	}
	if controls*100 >= len(head) {
		return Other
	}
	return Text
}

// ClassifyFile returns what a file on disk opens as, reading only the
// head of a file whose extension nothing knows.
func ClassifyFile(path string) (Kind, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case assetExts[ext]:
		return Asset, nil
	case binaryExts[ext]:
		return Other, nil
	case editor.LanguageFor(path) != nil || textExts[ext]:
		return Text, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return Other, err
	}
	defer f.Close()
	head := make([]byte, HeadSize)
	n, _ := f.Read(head)
	return Classify(path, head[:n]), nil
}
