package admin

import (
	"embed"
	"io/fs"
)

// use this to embed entire dist to go bin so it can be more portable.
//go:embed all:dist
var dist embed.FS

func spaFS() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
