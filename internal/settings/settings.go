// Package settings reads Claude Code's settings.json files and returns
// the permissions.allow entries union-merged across layers.
package settings

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// file is the minimal shape of Claude Code's settings.json we care about.
type file struct {
	Permissions struct {
		Allow []string `json:"allow"`
	} `json:"permissions"`
}

// ResolvePaths returns the settings.json paths to read, in merge order.
//
// envOverride, if non-empty, is split by ":" and used verbatim (intended
// for tests via CHAIN_COMMAND_BLOCKER_SETTINGS). Otherwise the default
// three-layer stack is returned:
//
//  1. <home>/.claude/settings.json
//  2. <projectDir>/.claude/settings.json
//  3. <projectDir>/.claude/settings.local.json
func ResolvePaths(envOverride, home, projectDir string) []string {
	if envOverride != "" {
		return strings.Split(envOverride, ":")
	}
	var out []string
	if home != "" {
		out = append(out, filepath.Join(home, ".claude", "settings.json"))
	}
	if projectDir != "" {
		out = append(out,
			filepath.Join(projectDir, ".claude", "settings.json"),
			filepath.Join(projectDir, ".claude", "settings.local.json"),
		)
	}
	return out
}

// CollectAllowRules reads each path in order and unions the
// permissions.allow arrays.
//
// Missing files and unparseable JSON are silently skipped to match the
// bash implementation's behavior (`jq ... 2>/dev/null`). Only hard OS
// errors other than "not exist" surface as errors.
func CollectAllowRules(paths []string) ([]string, error) {
	var out []string
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}
		var f file
		if err := json.Unmarshal(data, &f); err != nil {
			continue
		}
		out = append(out, f.Permissions.Allow...)
	}
	return out, nil
}
