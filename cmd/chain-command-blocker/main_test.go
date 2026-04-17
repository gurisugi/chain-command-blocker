package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeConfig creates a chain-command-blocker.json with the given allow
// list entries and optional merge flag.
func writeConfig(t *testing.T, dir string, allow []string, mergeSettings bool) string {
	t.Helper()
	body := map[string]any{
		"allow_list":           allow,
		"merge_settings_allow": mergeSettings,
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "chain-command-blocker.json")
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// runHook invokes the hook against the given command string and returns
// the stdout output (empty when the hook decides to stay silent).
func runHook(t *testing.T, command string, e env) string {
	t.Helper()
	in := map[string]any{
		"tool_input": map[string]any{"command": command},
	}
	if command == "" {
		in = map[string]any{"tool_input": map[string]any{}}
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run(bytes.NewReader(data), &out, e); err != nil {
		t.Fatalf("run error: %v", err)
	}
	return out.String()
}

func TestRun_AllowList(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, []string{
		"Bash(jq *)",
		"Bash(git log *)",
		"Bash(wc *)",
	}, false)
	e := env{ConfigPath: cfgPath}

	cases := []struct {
		name    string
		command string
		wantAsk bool
	}{
		{"single command passes through", "ls -la", false},
		{"all allow-listed chain is silent", "jq . a.json | jq . b.json", false},
		{"all allow-listed with semicolon", "jq . a.json; jq . b.json", false},
		{"jq + wc chain is silent", "jq . file.json | wc -l", false},
		{"echo piped into jq triggers ask", "echo hello | jq .", true},
		{"git status && git diff triggers ask", "git status && git diff", true},
		{"empty command stays silent", "", false},
		{"pipe inside jq query is not treated as shell pipe",
			`jq '[.[] | select(.name)]' file.json`, false},
		{"gh | jq chain triggers ask when gh not allowed",
			`gh api repos/o/r/pulls | jq '[.[] | select(.draft==false) | .title]'`, true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := runHook(t, tt.command, e)
			if tt.wantAsk && got == "" {
				t.Errorf("expected ask output, got nothing")
			}
			if !tt.wantAsk && got != "" {
				t.Errorf("expected no output, got:\n%s", got)
			}
			if tt.wantAsk {
				var resp hookOutput
				if err := json.Unmarshal([]byte(got), &resp); err != nil {
					t.Fatalf("output not JSON: %v\nraw: %s", err, got)
				}
				if resp.HookSpecificOutput.HookEventName != "PreToolUse" {
					t.Errorf("hookEventName = %q", resp.HookSpecificOutput.HookEventName)
				}
				if resp.HookSpecificOutput.PermissionDecision != "ask" {
					t.Errorf("permissionDecision = %q", resp.HookSpecificOutput.PermissionDecision)
				}
				if !strings.Contains(resp.HookSpecificOutput.PermissionDecisionReason,
					"non-allowlisted") {
					t.Errorf("reason missing marker text: %q",
						resp.HookSpecificOutput.PermissionDecisionReason)
				}
			}
		})
	}
}

func TestRun_MergeSettingsAllow(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	proj := filepath.Join(dir, "proj")
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(proj, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{
		"permissions": {"allow": ["Bash(git *)"]}
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, ".claude", "settings.local.json"), []byte(`{
		"permissions": {"allow": ["Bash(npm test)"]}
	}`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfgPath := writeConfig(t, dir, []string{"Bash(jq *)"}, true)
	e := env{
		ConfigPath: cfgPath,
		Home:       home,
		ProjectDir: proj,
	}

	t.Run("settings-allow merged, chain is silent", func(t *testing.T) {
		got := runHook(t, "git status && jq . file.json", e)
		if got != "" {
			t.Errorf("expected silent, got:\n%s", got)
		}
	})

	t.Run("not in any allow list still asks", func(t *testing.T) {
		got := runHook(t, "git status && rm -rf tmp", e)
		if got == "" {
			t.Error("expected ask output")
		}
	})

	t.Run("npm test from settings.local.json is merged", func(t *testing.T) {
		got := runHook(t, "npm test && jq . file.json", e)
		if got != "" {
			t.Errorf("expected silent, got:\n%s", got)
		}
	})
}

func TestRun_SettingsOverrideEnv(t *testing.T) {
	dir := t.TempDir()
	override := filepath.Join(dir, "override.json")
	if err := os.WriteFile(override, []byte(`{
		"permissions": {"allow": ["Bash(rg *)"]}
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfgPath := writeConfig(t, dir, []string{"Bash(jq *)"}, true)
	e := env{
		ConfigPath:       cfgPath,
		SettingsOverride: override,
	}

	got := runHook(t, "rg foo && jq . file.json", e)
	if got != "" {
		t.Errorf("expected silent, got:\n%s", got)
	}
}

func TestRun_MergeDisabledIgnoresSettings(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{
		"permissions": {"allow": ["Bash(git *)"]}
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	// merge disabled: the settings-allow entry for git should be ignored.
	cfgPath := writeConfig(t, dir, []string{"Bash(jq *)"}, false)
	e := env{ConfigPath: cfgPath, Home: home, ProjectDir: dir}

	got := runHook(t, "git status && jq . file.json", e)
	if got == "" {
		t.Error("expected ask because git should NOT be allowed when merge disabled")
	}
}

func TestRun_AskDenyRulesNotMergedAsAllow(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Put rm under "ask" / "deny" - should never be treated as allowed.
	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{
		"permissions": {
			"allow": ["Bash(jq *)"],
			"ask":   ["Bash(rm *)"],
			"deny":  ["Bash(curl *)"]
		}
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfgPath := writeConfig(t, dir, []string{}, true)
	e := env{ConfigPath: cfgPath, Home: home, ProjectDir: dir}

	got := runHook(t, "rm -rf tmp && jq . file.json", e)
	if got == "" {
		t.Error("expected ask (rm should not have been merged from ask-list)")
	}
}

func TestRun_MissingConfigFile(t *testing.T) {
	dir := t.TempDir()
	e := env{ConfigPath: filepath.Join(dir, "does-not-exist.json")}

	// No allow list at all: every chain command should trigger ask.
	got := runHook(t, "jq . a.json | jq . b.json", e)
	if got == "" {
		t.Error("expected ask when config file is missing (empty allow list)")
	}
}

func TestRun_SettingsOverrideThreeFiles(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.json")
	b := filepath.Join(dir, "b.json")
	c := filepath.Join(dir, "c.json")
	if err := os.WriteFile(a, []byte(`{"permissions":{"allow":["Bash(a *)"]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(`{"permissions":{"allow":["Bash(b *)"]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c, []byte(`{"permissions":{"allow":["Bash(c *)"]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfgPath := writeConfig(t, dir, []string{}, true)
	e := env{
		ConfigPath:       cfgPath,
		SettingsOverride: strings.Join([]string{a, b, c}, ":"),
	}

	for _, tc := range []string{"a x && b y", "b y && c z", "a x && c z"} {
		if got := runHook(t, tc, e); got != "" {
			t.Errorf("expected silent for %q, got:\n%s", tc, got)
		}
	}
}

func TestRun_ParseErrorTriggersAsk(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, []string{"Bash(jq *)"}, false)
	e := env{ConfigPath: cfgPath}

	// Unclosed command substitution: mvdan/sh cannot parse this.
	out := runHook(t, "echo $(foo && bar", e)
	if out == "" {
		t.Fatal("expected ask when shell parse fails, got nothing")
	}
	var resp hookOutput
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("output not JSON: %v\nraw: %s", err, out)
	}
	if resp.HookSpecificOutput.PermissionDecision != "ask" {
		t.Errorf("permissionDecision = %q, want ask", resp.HookSpecificOutput.PermissionDecision)
	}
	if !strings.Contains(resp.HookSpecificOutput.PermissionDecisionReason, "could not parse") {
		t.Errorf("reason missing parse-error explanation: %q",
			resp.HookSpecificOutput.PermissionDecisionReason)
	}
}

func TestRun_DisallowedListedInReason(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, []string{"Bash(jq *)"}, false)
	e := env{ConfigPath: cfgPath}

	out := runHook(t, "echo foo | jq .", e)
	if out == "" {
		t.Fatal("expected output")
	}
	var resp hookOutput
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatal(err)
	}
	reason := resp.HookSpecificOutput.PermissionDecisionReason
	if !strings.Contains(reason, "* echo foo") {
		t.Errorf("expected reason to mark 'echo foo' as disallowed, got:\n%s", reason)
	}
	if !strings.Contains(reason, "  jq .") {
		t.Errorf("expected reason to list 'jq .' as allowed, got:\n%s", reason)
	}
}
