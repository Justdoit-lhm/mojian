package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// repoRoot is the directory holding go.mod, found from this file's path.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test's own file")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("%s is not the repository root: %v", root, err)
	}
	return root
}

// parseFile parses one Go file with its comments.
func parseFile(t *testing.T, path string) (*ast.File, *token.FileSet) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	return f, fset
}

// appFiles lists the non-test Go files of the window package, including
// platform-tagged ones.
func appFiles(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "internal", "app")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}
	sort.Strings(files)
	return files
}

// walkGo lists every non-test Go file under internal and cmd, plus the
// root main.go.
func walkGo(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	skip := map[string]bool{
		"testdata": true, "build": true, ".mygo": true, "node_modules": true,
	}
	var files []string
	for _, tree := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, tree), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if skip[d.Name()] {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	files = append(files, filepath.Join(root, "main.go"))
	sort.Strings(files)
	return files
}

// imports is the set of import paths a file names.
func imports(f *ast.File) map[string]bool {
	out := make(map[string]bool, len(f.Imports))
	for _, im := range f.Imports {
		path, err := strconv.Unquote(im.Path.Value)
		if err != nil {
			continue
		}
		out[path] = true
	}
	return out
}

// call is one os.<Name> call at a position.
type call struct {
	Pos  token.Position
	Name string
}

// osCalls finds every call to an os package function in a file.
func osCalls(f *ast.File, fset *token.FileSet) []call {
	var calls []call
	ast.Inspect(f, func(n ast.Node) bool {
		ce, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := ce.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "os" {
			return true
		}
		calls = append(calls, call{Pos: fset.Position(ce.Pos()), Name: sel.Sel.Name})
		return true
	})
	return calls
}

// TestAppImportsStayBehindTheBridges keeps the window's coupling to the
// packages it reaches through one bridge file.
func TestAppImportsStayBehindTheBridges(t *testing.T) {
	allowed := map[string]map[string]bool{
		"mojian/internal/theme": {
			"settings_host.go": true, "theme_bridge.go": true,
		},
		"mojian/internal/platform": {
			"platform_bridge.go": true,
		},
		"mojian/internal/remote": {
			"remote_bridge.go": true, "remote_picker.go": true, "remote_pool.go": true,
			"remote_project.go": true, "remotehost_host.go": true, "settings_host.go": true,
		},
	}
	var violations []string
	for _, path := range appFiles(t) {
		f, _ := parseFile(t, path)
		file := filepath.Base(path)
		for imp := range imports(f) {
			files, bridged := allowed[imp]
			if !bridged || files[file] {
				continue
			}
			violations = append(violations, file+": imports "+imp+
				"; app reaches it through its bridge (see docs/subsystems/ui-shell.md)")
		}
	}
	report(t, violations)
}

// TestAppWritesAndCommandsStayBehindStorage keeps file writes and child
// processes out of the window package.
func TestAppWritesAndCommandsStayBehindStorage(t *testing.T) {
	writeCalls := map[string]bool{"WriteFile": true, "Create": true, "Rename": true, "Remove": true}
	var violations []string
	for _, path := range appFiles(t) {
		f, fset := parseFile(t, path)
		file := filepath.Base(path)
		if imports(f)["os/exec"] && file != "pdfrender.go" {
			violations = append(violations, file+
				": imports os/exec; commands belong to internal/platform or a lower package")
		}
		for _, c := range osCalls(f, fset) {
			if !writeCalls[c.Name] {
				continue
			}
			violations = append(violations, file+":"+strconv.Itoa(c.Pos.Line)+": os."+c.Name+
				"; use fs.WriteFileAtomic or the project/session layer (see docs/subsystems/projects-and-storage.md)")
		}
	}
	report(t, violations)
}

// TestDirectWritesStayInTheAllowlist keeps os.WriteFile to the three
// places that legitimately bypass the project layer, and os.Create out of
// the repository entirely.
func TestDirectWritesStayInTheAllowlist(t *testing.T) {
	allowedWrite := map[string]bool{
		"internal/fs/server.go":         true, // the local asset cache
		"internal/welcome/welcome.go":   true, // the starter project's files
		"internal/testutil/testutil.go": true, // test fixtures
	}
	root := repoRoot(t)
	var violations []string
	for _, path := range walkGo(t) {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatal(err)
		}
		rel = filepath.ToSlash(rel)
		f, fset := parseFile(t, path)
		for _, c := range osCalls(f, fset) {
			at := rel + ":" + strconv.Itoa(c.Pos.Line) + ": os." + c.Name
			switch c.Name {
			case "WriteFile":
				if !allowedWrite[rel] {
					violations = append(violations, at+
						"; direct writes belong to fs.WriteFileAtomic, and os.WriteFile is limited to internal/fs/server.go, internal/welcome/welcome.go and internal/testutil/testutil.go")
				}
			case "Create":
				violations = append(violations, at+"; os.Create is not allowed; use fs.WriteFileAtomic")
			}
		}
	}
	report(t, violations)
}

// report prints the violations sorted and fails the test once.
func report(t *testing.T, violations []string) {
	t.Helper()
	if len(violations) == 0 {
		return
	}
	sort.Strings(violations)
	t.Errorf("repository structure guard failed:\n\t%s", strings.Join(violations, "\n\t"))
}
