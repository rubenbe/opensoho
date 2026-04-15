package main

import (
	"regexp"
	"strings"

	"github.com/mozillazg/go-unidecode"
)

var validHostnameLabelRegexp = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// normalizeHostname converts a name to a hostname-compatible string by
// transliterating Unicode characters to ASCII (via unidecode), replacing
// spaces and underscores with hyphens, stripping any remaining
// non-[a-zA-Z0-9-] characters, and trimming leading/trailing hyphens.
func normalizeHostname(name string) string {
	decoded := unidecode.Unidecode(name)
	decoded = strings.NewReplacer(" ", "-", "_", "-").Replace(decoded)

	var b strings.Builder
	for _, r := range decoded {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
		if r == '_' || r == ' ' {
			b.WriteRune('-')
		}
	}

	return strings.Trim(b.String(), "-")
}

// isValidHostname reports whether name is a valid single-label hostname.
// Non-ASCII characters are first transliterated to ASCII via unidecode.
// A valid hostname label must:
//   - be between 1 and 63 characters long
//   - contain only letters, digits, and hyphens
//   - not start or end with a hyphen
func isValidHostname(name string) bool {
	normalized := normalizeHostname(name)
	if len(normalized) == 0 || len(normalized) > 63 {
		return false
	}
	return validHostnameLabelRegexp.MatchString(normalized)
}
