package icons

import (
	"path/filepath"
	"strings"

	"mojian/internal/assets"

	"github.com/egoist/mygo/ui"
)

var svgCache = map[string]*ui.SVG{}

// Icon returns an embedded Lucide action icon by name.
func Icon(name string) *ui.SVG {
	if s, ok := svgCache[name]; ok {
		return s
	}
	data, err := assets.Files.ReadFile("icons/" + name + ".svg")
	if err != nil {
		panic("mojian: icon " + name + ": " + err.Error())
	}
	s := ui.MustParseSVG(data)
	svgCache[name] = s
	return s
}

// fileIconAsset names the Material Icon Theme asset for a file: by its
// exact name for special files, else by its extension.
func fileIconAsset(name string) string {
	base := strings.ToLower(filepath.Base(name))
	switch base {
	case "dockerfile", "containerfile":
		return "docker"
	case "makefile":
		return "makefile"
	case "cmakelists.txt":
		return "cmake"
	case ".gitignore", ".gitattributes", ".gitmodules":
		return "git"
	case "package.json", "package-lock.json", "npm-shrinkwrap.json":
		return "npm"
	case "tsconfig.json":
		return "typescript"
	case "go.mod", "go.sum":
		return "go"
	case "cargo.toml":
		return "rust"
	case "license", "license.md", "license.txt", "copying":
		return "license"
	case "readme", "readme.md", "readme.txt", "readme.rst":
		return "readme"
	case ".env", ".env.local", ".env.example":
		return "settings"
	}
	switch strings.TrimPrefix(strings.ToLower(filepath.Ext(base)), ".") {
	case "py", "pyw", "pyi":
		return "python"
	case "js", "mjs", "cjs":
		return "javascript"
	case "jsx", "tsx":
		return "react"
	case "ts", "mts", "cts":
		return "typescript"
	case "go":
		return "go"
	case "rs":
		return "rust"
	case "c", "h":
		return "c"
	case "cc", "cpp", "cxx", "hpp", "hh", "hxx":
		return "cpp"
	case "swift":
		return "swift"
	case "kt", "kts":
		return "kotlin"
	case "java":
		return "java"
	case "rb", "erb":
		return "ruby"
	case "php":
		return "php"
	case "cs":
		return "csharp"
	case "sh", "bash", "zsh", "fish":
		return "console"
	case "ps1", "psm1":
		return "powershell"
	case "json", "jsonc", "json5":
		return "json"
	case "yaml", "yml":
		return "yaml"
	case "toml":
		return "toml"
	case "xml", "plist":
		return "xml"
	case "sql", "sqlite", "db":
		return "database"
	case "html", "htm":
		return "html"
	case "css":
		return "css"
	case "scss", "sass":
		return "sass"
	case "less":
		return "less"
	case "vue":
		return "vue"
	case "svelte":
		return "svelte"
	case "astro":
		return "astro"
	case "md", "markdown":
		return "markdown"
	case "mdx":
		return "mdx"
	case "lock":
		return "lock"
	default:
		return "document"
	}
}

// fileIcon returns the file-type icon from Material Icon Theme.
func FileIcon(name string) *ui.SVG {
	return Icon("material/" + fileIconAsset(name))
}

// folderIcon returns the Material Icon Theme folder.
func FolderIcon() *ui.SVG {
	return Icon("material/folder-base")
}

// Brand returns a Mojian logo asset: "mark", "mark-reversed",
// "wordmark" or "wordmark-reversed". The files come from logos/svg.
func Brand(name string) *ui.SVG {
	key := "brand/mojian-" + name
	if s, ok := svgCache[key]; ok {
		return s
	}
	data, err := assets.Files.ReadFile(key + ".svg")
	if err != nil {
		panic("mojian: brand " + name + ": " + err.Error())
	}
	s := ui.MustParseSVG(data)
	svgCache[key] = s
	return s
}
