package permissions

import (
	"reflect"
	"testing"
)

func TestParseBashPattern(t *testing.T) {
	tests := []struct {
		name  string
		entry string
		want  string
		ok    bool
	}{
		{"exact match", "Bash(git log)", "git log", true},
		{"trailing space wildcard", "Bash(gh pr view *)", "gh pr view", true},
		{"trailing family wildcard", "Bash(gh search:*)", "gh search", true},
		{"bare command with no args", "Bash(jq)", "jq", true},
		{"middle wildcard is unsupported", "Bash(git * main)", "", false},
		{"middle wildcard with trailing stripped", "Bash(sed */foo/* *)", "", false},
		{"non-Bash rule is rejected", "Read(./.env)", "", false},
		{"empty entry is rejected", "", "", false},
		{"missing closing paren", "Bash(git log", "", false},
		{"missing Bash prefix", "(git log)", "", false},
		{"Bash(*) is rejected as empty", "Bash(*)", "", false},
		{"Bash() is rejected as empty", "Bash()", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseBashPattern(tt.entry)
			if ok != tt.ok {
				t.Fatalf("ParseBashPattern(%q) ok=%v, want %v", tt.entry, ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("ParseBashPattern(%q) = %q, want %q", tt.entry, got, tt.want)
			}
		})
	}
}

func TestParseBashPatterns(t *testing.T) {
	entries := []string{
		"Bash(jq *)",
		"Bash(git log)",
		"Bash(gh search:*)",
		"Bash(git * main)", // skipped (middle wildcard)
		"Read(./.env)",     // skipped (non-Bash)
		"",                 // skipped (empty)
		"Bash(wc *)",
	}
	want := []string{"jq", "git log", "gh search", "wc"}
	got := ParseBashPatterns(entries)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseBashPatterns = %v, want %v", got, want)
	}
}
