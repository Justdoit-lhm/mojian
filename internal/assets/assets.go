// Package assets holds the icons and fonts embedded in the app.
package assets

import "embed"

// Files is the app's static assets, by path: icons/*.svg,
// icons/material/*.svg and fonts/*.ttf.
//
//go:embed icons/*.svg icons/material/*.svg brand/*.svg brand/*.png fonts/*.ttf
var Files embed.FS
