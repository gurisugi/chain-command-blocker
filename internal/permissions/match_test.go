package permissions

import "testing"

func TestIsAllowed(t *testing.T) {
	patterns := []string{"jq", "git log", "gh search"}

	tests := []struct {
		cmd  string
		want bool
	}{
		{"jq", true},
		{"jq .", true},
		{"jq -r .name file.json", true},
		{"jqx", false},               // word boundary: "jq" must not match "jqx"
		{"git log", true},            // exact
		{"git log --oneline", true},  // prefix + space
		{"git log--oneline", false},  // no space, no match
		{"git", false},               // shorter than pattern
		{"git status", false},        // different subcommand
		{"gh search repos", true},    // prefix + space
		{"gh searcher", false},       // word boundary
		{"", false},                  // empty cmd never matches a non-empty pattern
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := IsAllowed(tt.cmd, patterns)
			if got != tt.want {
				t.Errorf("IsAllowed(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestIsAllowedEmptyPatterns(t *testing.T) {
	if IsAllowed("anything", nil) {
		t.Error("IsAllowed with nil patterns should be false")
	}
	if IsAllowed("", nil) {
		t.Error("IsAllowed(\"\", nil) should be false")
	}
}
