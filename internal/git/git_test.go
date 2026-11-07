package git

import "testing"

func TestParseDiff(t *testing.T) {
	diff := `diff --git a/x.py b/x.py
index 111..222 100644
--- a/x.py
+++ b/x.py
@@ -1,3 +1,4 @@
 old
-removed
+added
+more
 context
`
	lines := ParseDiff(diff)
	var kinds []byte
	for _, line := range lines {
		kinds = append(kinds, line.kind)
	}
	if string(kinds) != "@ -++ " {
		t.Fatalf("kinds: %q (lines=%+v)", kinds, lines)
	}
	if add, del := DiffStat(diff); add != 2 || del != 1 {
		t.Fatalf("stat: +%d -%d", add, del)
	}
	if lines[2].old != 2 || lines[2].new != 0 {
		t.Fatalf("deleted numbers: %+v", lines[2])
	}
	if lines[3].old != 0 || lines[3].new != 2 {
		t.Fatalf("added numbers: %+v", lines[3])
	}
	if lines[5].old != 3 || lines[5].new != 4 {
		t.Fatalf("context numbers: %+v", lines[5])
	}
}

func TestParseDiffKeepsContentThatLooksLikeHeaders(t *testing.T) {
	diff := `diff --git a/x b/x
--- a/x
+++ b/x
@@ -1,1 +1,1 @@
--- a/removed
+++ b/added
`
	lines := ParseDiff(diff)
	var kinds []byte
	for _, line := range lines {
		kinds = append(kinds, line.kind)
	}
	if string(kinds) != "@-+" {
		t.Fatalf("kinds: %q (lines=%+v)", kinds, lines)
	}
	if lines[1].text != "--- a/removed" || lines[1].old != 1 {
		t.Fatalf("deleted line: %+v", lines[1])
	}
	if lines[2].text != "+++ b/added" || lines[2].new != 1 {
		t.Fatalf("added line: %+v", lines[2])
	}
	if add, del := DiffStat(diff); add != 1 || del != 1 {
		t.Fatalf("stat: +%d -%d", add, del)
	}
}

func TestStatusLetterConflicts(t *testing.T) {
	for _, code := range []string{"UU", "AA", "DD", "AU", "UD", "UA", "DU"} {
		if got := statusLetter(code); got != "C" {
			t.Errorf("statusLetter(%q) = %q, want C", code, got)
		}
	}
}

func TestParseStatusKeepsRenameSource(t *testing.T) {
	changes := parseStatus("R  moved.go\x00pack.go\x00M  other.go\x00", "")
	if len(changes) != 2 {
		t.Fatalf("changes: %+v", changes)
	}
	if changes[0].Path != "moved.go" || changes[0].OldPath != "pack.go" || changes[0].Status != "R" {
		t.Fatalf("rename: %+v", changes[0])
	}
	if changes[1].Path != "other.go" || changes[1].OldPath != "" {
		t.Fatalf("edit: %+v", changes[1])
	}
}
