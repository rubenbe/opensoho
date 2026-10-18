// Package ui contains the OpenSOHO extension of the PocketBase superuser UI.
package ui

import (
	"bytes"
	"embed"
	"io/fs"
	"time"
)

//go:embed main dashboard css images libs
var extensionDir embed.FS

// mainScripts are concatenated, in this order, into the main.js of the extension.
//
// The upstream UI runs main.js before starting its router, but it doesn't
// wait for asynchronous work such as dynamic imports. Everything that must
// be in place before the first page render is therefore loaded this way.
// Each script is wrapped in its own block scope.
var mainScripts = []string{
	"main/branding.js",
	"main/prng.js",
	"main/helpers.js",
	"main/dashboard.js",
	"main/sidebar.js",
	"main/records.js",
	"main/upsert.js",
}

// ExtensionFS contains the files of the "opensoho" superuser UI extension.
var ExtensionFS fs.FS = extensionFS{FS: extensionDir, main: buildMain()}

func buildMain() []byte {
	var buf bytes.Buffer
	for _, name := range mainScripts {
		script, err := extensionDir.ReadFile(name)
		if err != nil {
			panic(err)
		}
		buf.WriteString("// " + name + "\n{\n")
		buf.Write(script)
		buf.WriteString("\n}\n")
	}
	return buf.Bytes()
}

// extensionFS serves the embedded files, plus the generated main.js.
type extensionFS struct {
	fs.FS
	main []byte
}

func (f extensionFS) Open(name string) (fs.File, error) {
	if name == "main.js" {
		return &mainFile{Reader: bytes.NewReader(f.main), size: int64(len(f.main))}, nil
	}
	return f.FS.Open(name)
}

type mainFile struct {
	*bytes.Reader
	size int64
}

func (f *mainFile) Stat() (fs.FileInfo, error) { return f, nil }
func (f *mainFile) Close() error               { return nil }

// fs.FileInfo
func (f *mainFile) Name() string       { return "main.js" }
func (f *mainFile) Size() int64        { return f.size }
func (f *mainFile) Mode() fs.FileMode  { return 0o444 }
func (f *mainFile) ModTime() time.Time { return time.Time{} }
func (f *mainFile) IsDir() bool        { return false }
func (f *mainFile) Sys() any           { return nil }
