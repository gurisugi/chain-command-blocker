// chain-command-blocker is a Claude Code PreToolUse hook that inspects
// Bash tool invocations containing chained commands (e.g. `a && b`) and
// asks the user for confirmation when any chained sub-command is not
// covered by the configured allow list.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gurisugi/chain-command-blocker/internal/config"
	"github.com/gurisugi/chain-command-blocker/internal/permissions"
	"github.com/gurisugi/chain-command-blocker/internal/settings"
	"github.com/gurisugi/chain-command-blocker/internal/shell"
)

type hookInput struct {
	ToolInput struct {
		Command string `json:"command"`
	} `json:"tool_input"`
}

type hookOutput struct {
	HookSpecificOutput hookSpecificOutput `json:"hookSpecificOutput"`
}

type hookSpecificOutput struct {
	HookEventName            string `json:"hookEventName"`
	PermissionDecision       string `json:"permissionDecision"`
	PermissionDecisionReason string `json:"permissionDecisionReason"`
}

// env exposes just the environment variables run needs, so tests can
// supply fakes without touching the real process environment.
type env struct {
	ConfigPath       string // override for CHAIN_COMMAND_BLOCKER_CONFIG
	SettingsOverride string // CHAIN_COMMAND_BLOCKER_SETTINGS (":" separated)
	Home             string // HOME
	ProjectDir       string // CLAUDE_PROJECT_DIR (falls back to cwd)
}

func main() {
	home, _ := os.UserHomeDir()
	wd, _ := os.Getwd()
	e := env{
		ConfigPath:       firstNonEmpty(os.Getenv("CHAIN_COMMAND_BLOCKER_CONFIG"), filepath.Join(home, ".claude", "chain-command-blocker.json")),
		SettingsOverride: os.Getenv("CHAIN_COMMAND_BLOCKER_SETTINGS"),
		Home:             home,
		ProjectDir:       firstNonEmpty(os.Getenv("CLAUDE_PROJECT_DIR"), wd),
	}
	if err := run(os.Stdin, os.Stdout, e); err != nil {
		fmt.Fprintf(os.Stderr, "chain-command-blocker: %v\n", err)
		os.Exit(0) // hook failures should not block tool use
	}
}

func run(stdin io.Reader, stdout io.Writer, e env) error {
	cfg, err := config.Load(e.ConfigPath)
	if err != nil {
		return err
	}

	patterns := permissions.ParseBashPatterns(cfg.AllowList)

	if cfg.MergeSettingsAllow {
		paths := settings.ResolvePaths(e.SettingsOverride, e.Home, e.ProjectDir)
		rules, err := settings.CollectAllowRules(paths)
		if err != nil {
			return err
		}
		patterns = append(patterns, permissions.ParseBashPatterns(rules)...)
	}

	var in hookInput
	if err := json.NewDecoder(stdin).Decode(&in); err != nil {
		return nil // malformed or empty stdin: stay out of the way
	}
	if in.ToolInput.Command == "" {
		return nil
	}

	cmds, err := shell.SplitCommands(in.ToolInput.Command)
	if err != nil {
		return emitAsk(stdout, fmt.Sprintf(
			"chain-command-blocker could not parse the command (%v); asking for confirmation as a precaution.",
			err,
		))
	}
	if len(cmds) <= 1 {
		return nil
	}

	var disallowed []string
	for _, c := range cmds {
		if !permissions.IsAllowed(c, patterns) {
			disallowed = append(disallowed, c)
		}
	}
	if len(disallowed) == 0 {
		return nil
	}

	return emitAsk(stdout, buildReason(cmds, disallowed))
}

// emitAsk writes a PreToolUse ask decision with the given reason.
func emitAsk(stdout io.Writer, reason string) error {
	out := hookOutput{
		HookSpecificOutput: hookSpecificOutput{
			HookEventName:            "PreToolUse",
			PermissionDecision:       "ask",
			PermissionDecisionReason: reason,
		},
	}
	return json.NewEncoder(stdout).Encode(&out)
}

// buildReason reproduces the bash implementation's reason message:
//
//	Chained command contains non-allowlisted command(s) (* marked):
//	  <allowed cmd>
//	* <disallowed cmd>
//
// Note the blank line between the header and the first command, which
// the bash version produced by starting cmd_list with an empty string
// before appending newline-prefixed entries.
func buildReason(cmds, disallowed []string) string {
	disallowedSet := make(map[string]struct{}, len(disallowed))
	for _, d := range disallowed {
		disallowedSet[d] = struct{}{}
	}
	var sb strings.Builder
	sb.WriteString("Chained command contains non-allowlisted command(s) (* marked):")
	for _, c := range cmds {
		marker := "  "
		if _, ok := disallowedSet[c]; ok {
			marker = "* "
		}
		sb.WriteByte('\n')
		sb.WriteString(marker)
		sb.WriteString(c)
	}
	return sb.String()
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
