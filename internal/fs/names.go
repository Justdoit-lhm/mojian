package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
)

// SplitExt separates a file name's extension for a numbered sibling. The
// extension keeps its leading dot.
func SplitExt(name string) (stem, ext string) {
	ext = filepath.Ext(name)
	stem = strings.TrimSuffix(name, ext)
	if stem == "" {
		stem = "untitled"
	}
	return stem, ext
}

// ShortHash names a cached file or a local socket by a short digest of s.
func ShortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:6])
}
