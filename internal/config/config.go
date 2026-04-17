// Package config loads chain-command-blocker's plugin-level config file.
package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
)

// Config mirrors the user-editable chain-command-blocker.json schema.
// Unknown fields (e.g. the deprecated use_bundled_shs) are ignored.
type Config struct {
	AllowList          []string `json:"allow_list"`
	MergeSettingsAllow bool     `json:"merge_settings_allow"`
}

// Load reads the config at path. Returns an empty Config if the file does
// not exist. Returns an error for any other read or parse failure.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
