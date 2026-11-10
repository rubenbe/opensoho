package ui

import (
	"runtime/debug"
	"strings"
)

// Version is shown in the footer of the UI.
//
// Release builds set it from git with
//
//	-ldflags "-X github.com/rubenbe/opensoho/ui.Version=$(git describe --tags --always --dirty)"
//
// When it's empty, it's derived from the VCS info Go embeds in the binary.
var Version string

func version() string {
	if Version != "" {
		return Version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	// go build stamps the main module with a git based (pseudo-)version
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	var revision string
	var dirty bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if len(revision) > 7 {
		revision = revision[:7]
	}
	if revision != "" && dirty {
		revision += "-dirty"
	}
	return strings.TrimSpace(revision)
}
