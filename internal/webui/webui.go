package webui

import (
	"embed"
	"io/fs"
)

// The build copies web/dist here so the server ships as a single binary.
//
//go:embed all:dist
var content embed.FS

func FS() fs.FS {
	sub, err := fs.Sub(content, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
