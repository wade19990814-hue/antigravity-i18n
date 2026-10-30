package main

import (
	"embed"
	"io/fs"
)

// Locale packs, patch templates and the translation engine are compiled into the
// binary. That is what removes the runtime dependency on Node: nothing is read
// from the project directory at run time.
//
//go:embed all:assets/locales all:assets/patches
var assetsFS embed.FS

// frontendFS holds the hand-written static window: HTML, CSS and JS only. Wails
// serves it as-is, which is why no Node toolchain is involved at build time.
//
//go:embed all:frontend/dist
var frontendFS embed.FS

// assetFS returns the embedded assets rooted at dir.
func assetFS(dir string) fs.FS {
	sub, err := fs.Sub(assetsFS, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
