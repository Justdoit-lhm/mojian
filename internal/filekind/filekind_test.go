package filekind

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		head []byte
		want Kind
	}{
		{"readme.md", nil, Text},
		{"main.go", nil, Text},
		{"notes.unknown", []byte("hello\n"), Text},
		{"empty.unknown", nil, Text},
		{"image.png", nil, Asset},
		{"paper.pdf", nil, Asset},
		{"archive.zip", nil, Other},
		{"blob.unknown", []byte{0, 1, 2}, Other},
		{"binary.unknown", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, Other},
	}
	for _, c := range cases {
		if got := Classify(c.name, c.head); got != c.want {
			t.Errorf("Classify(%q, %v) = %v, want %v", c.name, c.head, got, c.want)
		}
	}
}

func TestClassifyFileReadsOnlyUnknownHeads(t *testing.T) {
	dir := t.TempDir()
	text := filepath.Join(dir, "notes.unknown")
	if err := os.WriteFile(text, []byte("plain text\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if kind, err := ClassifyFile(text); err != nil || kind != Text {
		t.Fatalf("text: %v %v", kind, err)
	}
	bin := filepath.Join(dir, "blob.unknown")
	if err := os.WriteFile(bin, []byte{0, 1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	if kind, err := ClassifyFile(bin); err != nil || kind != Other {
		t.Fatalf("binary: %v %v", kind, err)
	}
}

func TestSkipDirHidesGeneratedTreesAndRemoteTrash(t *testing.T) {
	for _, name := range []string{".git", "node_modules", "dist", ".mojian-trash"} {
		if !SkipDir(name) {
			t.Errorf("%q is not skipped", name)
		}
	}
	if SkipDir("src") {
		t.Error("src is skipped")
	}
}
