// Package permissions parses Claude Code permission rule entries and
// matches shell commands against them.
package permissions

import "strings"

// ParseBashPattern converts a single Bash(...) rule entry into a prefix
// pattern suitable for chain-command-blocker's match logic.
//
// Supported forms:
//
//	Bash(cmd)         -> "cmd"       (exact match)
//	Bash(cmd *)       -> "cmd"       (trailing space wildcard)
//	Bash(cmd:*)       -> "cmd"       (trailing family wildcard)
//
// Returns ok=false for entries that are not Bash(...) rules, that contain
// a middle wildcard (e.g. Bash(git * main)), or that resolve to an empty
// inner string.
func ParseBashPattern(entry string) (string, bool) {
	if !strings.HasPrefix(entry, "Bash(") || !strings.HasSuffix(entry, ")") {
		return "", false
	}
	inner := entry[len("Bash(") : len(entry)-1]
	inner = strings.TrimSuffix(inner, " *")
	inner = strings.TrimSuffix(inner, ":*")
	if strings.Contains(inner, "*") {
		return "", false
	}
	if inner == "" {
		return "", false
	}
	return inner, true
}

// ParseBashPatterns converts multiple entries, dropping unsupported ones.
func ParseBashPatterns(entries []string) []string {
	result := make([]string, 0, len(entries))
	for _, e := range entries {
		if p, ok := ParseBashPattern(e); ok {
			result = append(result, p)
		}
	}
	return result
}
