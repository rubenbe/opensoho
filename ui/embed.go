// Package ui contains the OpenSOHO extension of the PocketBase superuser UI.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed main.js js css images
var extensionDir embed.FS

// ExtensionFS contains the files of the "opensoho" superuser UI extension.
var ExtensionFS fs.FS = extensionDir
