package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeHostname(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Already clean ASCII
		{"router", "router"},
		{"my-device", "my-device"},
		{"device01", "device01"},
		{"Router", "Router"},
		{"MY-DEVICE", "MY-DEVICE"},
		// Spaces and underscores become hyphens
		{"my device", "my-device"},
		{"my_device", "my-device"},
		// Leading/trailing hyphens trimmed
		{"-router-", "router"},
		{"--router--", "router"},
		// Unicode transliteration (case preserved)
		{"büro", "buro"},
		{"Ñoño", "Nono"},
		{"café", "cafe"},
		{"naïve", "naive"},
		{"中文", "Zhong-Wen"},
		// Strips non-[a-z0-9-] chars
		{"router!", "router"},
		{"rout.er", "router"},
		// Normalizes to empty
		{"---", ""},
		{"!!!", ""},
		{"   ", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.expected, normalizeHostname(tc.input))
		})
	}
}

func TestIsValidHostname(t *testing.T) {
	valid := []string{
		"router",
		"my-device",
		"device01",
		"a",
		"x1",
		"a1b2c3",
		"node-1",
		"-router",               // leading hyphen stripped → "router"
		"router-",               // trailing hyphen stripped → "router"
		"Router",                // case preserved → "Router"
		"büro",                  // normalizes to "buro"
		"Ñoño",                  // normalizes to "Nono"
		"café",                  // normalizes to "cafe"
		strings.Repeat("a", 63), // exactly 63 chars
	}

	for _, name := range valid {
		t.Run("valid/"+name, func(t *testing.T) {
			assert.True(t, isValidHostname(name), "expected %q to be valid", name)
		})
	}

	invalid := []string{
		"",                      // empty
		"-",                     // only hyphen → empty after normalize
		"---",                   // only hyphens → empty after normalize
		"!!!",                   // only special chars → empty after normalize
		strings.Repeat("a", 64), // 64 chars, too long
		"   ",                   // only spaces → empty after normalize
	}

	for _, name := range invalid {
		t.Run("invalid/"+name, func(t *testing.T) {
			assert.False(t, isValidHostname(name), "expected %q to be invalid", name)
		})
	}
}
