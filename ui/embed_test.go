package ui

import (
	"io"
	"io/fs"
	"strings"
	"testing"
)

func TestExtensionMainJs(t *testing.T) {
	f, err := ExtensionFS.Open("main.js")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	last := -1
	for _, name := range mainScripts {
		pos := strings.Index(string(content), "// "+name+"\n{\n")
		if pos <= last {
			t.Fatalf("expected %s after the previous script, got position %d", name, pos)
		}
		last = pos
	}

	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), info.Size())
	}
}

func TestExtensionStaticFiles(t *testing.T) {
	for _, name := range []string{
		"css/opensoho.css",
		"css/dashboard.css",
		"dashboard/page.js",
		"images/logo.svg",
		"libs/chart.umd.min.js",
	} {
		if _, err := fs.Stat(ExtensionFS, name); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}
