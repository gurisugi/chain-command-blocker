package permissions

import "strings"

// IsAllowed reports whether cmd is allowed by any of the given prefix
// patterns. A pattern matches when cmd equals it exactly, or when cmd
// starts with the pattern followed by a space (word boundary).
//
// This mirrors the bash implementation's check:
//
//	[[ "$cmd" == "$a" || "$cmd" == "$a "* ]]
func IsAllowed(cmd string, patterns []string) bool {
	for _, p := range patterns {
		if cmd == p || strings.HasPrefix(cmd, p+" ") {
			return true
		}
	}
	return false
}
