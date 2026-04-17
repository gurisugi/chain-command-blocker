package shell

import (
	"strings"
	"testing"
)

func TestSplitCommands(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    []string
	}{
		{
			name:    "single command",
			command: "ls -la",
			want:    []string{"ls -la"},
		},
		{
			name:    "pipe",
			command: "git log --oneline | wc -l",
			want:    []string{"git log --oneline", "wc -l"},
		},
		{
			name:    "AND",
			command: "make build && make test",
			want:    []string{"make build", "make test"},
		},
		{
			name:    "OR",
			command: "test -f file || echo missing",
			want:    []string{"test -f file", "echo missing"},
		},
		{
			name:    "semicolon",
			command: "echo hello; echo world",
			want:    []string{"echo hello", "echo world"},
		},
		{
			name:    "compound chain",
			command: "git log --oneline | wc -l && echo done",
			want:    []string{"git log --oneline", "wc -l", "echo done"},
		},
		{
			name:    "three-stage pipe",
			command: "cat file | grep foo | wc -l",
			want:    []string{"cat file", "grep foo", "wc -l"},
		},
		{
			name:    "pipe inside quotes is not split",
			command: `jq '[.[] | select(.name)]' file.json`,
			want:    []string{`jq '[.[] | select(.name)]' file.json`},
		},
		{
			name:    "pipe with quoted pipe",
			command: `echo hello | jq '[.[] | select(.x)]'`,
			want:    []string{"echo hello", `jq '[.[] | select(.x)]'`},
		},
		{
			name:    "expands commands in command substitution",
			command: `echo "$(cat file)"`,
			want:    []string{`echo "$()"`, "cat file"},
		},
		{
			name:    "expands pipes in command substitution",
			command: `echo "$(echo foo | grep f)"`,
			want:    []string{`echo "$()"`, "echo foo", "grep f"},
		},
		{
			name:    "command substitution with chain",
			command: `echo "$(cat file)" && ls`,
			want:    []string{`echo "$()"`, "cat file", "ls"},
		},
		{
			name:    "empty string",
			command: "",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitCommands(tt.command)
			if err != nil {
				t.Fatalf("SplitCommands(%q) error: %v", tt.command, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("SplitCommands(%q) = %v (len=%d), want %v (len=%d)",
					tt.command, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if strings.TrimSpace(got[i]) != strings.TrimSpace(tt.want[i]) {
					t.Errorf("SplitCommands(%q)[%d] = %q, want %q",
						tt.command, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCommandNames(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    []string
	}{
		{
			name:    "single command",
			command: "ls -la",
			want:    []string{"ls"},
		},
		{
			name:    "pipe",
			command: "git log --oneline | wc -l",
			want:    []string{"git", "wc"},
		},
		{
			name:    "expands command substitution",
			command: `echo "$(cat file)" && ls`,
			want:    []string{"echo", "cat", "ls"},
		},
		{
			name:    "AND",
			command: "make build && make test",
			want:    []string{"make", "make"},
		},
		{
			name:    "empty string",
			command: "",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CommandNames(tt.command)
			if err != nil {
				t.Fatalf("CommandNames(%q) error: %v", tt.command, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("CommandNames(%q) = %v (len=%d), want %v (len=%d)",
					tt.command, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("CommandNames(%q)[%d] = %q, want %q",
						tt.command, i, got[i], tt.want[i])
				}
			}
		})
	}
}
