package main

import "testing"

func TestFileIconAssets(t *testing.T) {
	cases := map[string]string{
		"tasks.py":         "python",
		"app.js":           "javascript",
		"worker.tsx":       "react",
		"main.go":          "go",
		"lib.rs":           "rust",
		"README.md":        "readme",
		"LICENSE":          "license",
		"package.json":     "npm",
		"Dockerfile":       "docker",
		".gitignore":       "git",
		"notes.unknown":    "document",
		"scripts/tasks.py": "python",
	}
	for name, want := range cases {
		if got := fileIconAsset(name); got != want {
			t.Errorf("%s: %q, want %q", name, got, want)
		}
	}
}

func TestMappedIconsExist(t *testing.T) {
	for _, name := range []string{
		"folder-base", "document", "python", "javascript", "react",
		"typescript", "go", "rust", "c", "cpp", "swift", "kotlin",
		"java", "ruby", "php", "csharp", "console", "powershell",
		"json", "yaml", "toml", "xml", "database", "html", "css",
		"sass", "less", "vue", "svelte", "astro", "docker", "git",
		"makefile", "cmake", "license", "readme", "markdown", "mdx",
		"npm", "settings", "lock",
	} {
		if icon("material/"+name) == nil {
			t.Errorf("missing Material icon %q", name)
		}
	}
}
