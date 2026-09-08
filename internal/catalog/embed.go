package catalog

import (
	"embed"
	"io/fs"
)

//go:embed all:exercises
var embedded embed.FS

// Embedded returns the catalog compiled into the binary. Using all: means
// files are included even if a future exercise directory starts with a dot or
// an underscore, which go:embed would otherwise skip.
func Embedded() (*Catalog, error) {
	return Load(embedded)
}

// EmbeddedFS exposes the raw filesystem, for tests that need to check what
// was actually compiled in.
func EmbeddedFS() fs.FS { return embedded }
