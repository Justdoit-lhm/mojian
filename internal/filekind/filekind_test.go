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
		{"model.pt", nil, Other},
		{"data.mat", nil, Other},
		{"weights.safetensors", nil, Other},
		{"matrix.npy", nil, Other},
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

func TestClassifyFileSkipsHugeUnknownHeads(t *testing.T) {
	dir := t.TempDir()
	big := filepath.Join(dir, "dataset.unknown")
	if err := os.WriteFile(big, make([]byte, headLimit+1), 0o644); err != nil {
		t.Fatal(err)
	}
	if kind, err := ClassifyFile(big); err != nil || kind != Other {
		t.Fatalf("large unknown file: %v %v", kind, err)
	}
	log := filepath.Join(dir, "server.log")
	if err := os.WriteFile(log, make([]byte, headLimit+1), 0o644); err != nil {
		t.Fatal(err)
	}
	if kind, err := ClassifyFile(log); err != nil || kind != Text {
		t.Fatalf("large log: %v %v", kind, err)
	}
}

func TestClassifyFileKnownDataNeedsNoRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dataset.mat")
	if err := os.WriteFile(path, []byte("not really text"), 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0o644)
	if kind, err := ClassifyFile(path); err != nil || kind != Other {
		t.Fatalf("unreadable data file: %v %v", kind, err)
	}
}
