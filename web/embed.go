// Package web embeds the built SPA so the binary serves it with no external files.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Assets is the built SPA rooted at the dist directory.
var Assets fs.FS = mustSub(distFS, "dist")

func mustSub(f fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
