package ui

import "testing"

func TestVersionOverride(t *testing.T) {
	old := Version
	defer func() { Version = old }()

	Version = "v1.2.3-4-gabcdef0"
	if got := version(); got != "v1.2.3-4-gabcdef0" {
		t.Fatalf("expected the ldflags version, got %q", got)
	}
}
