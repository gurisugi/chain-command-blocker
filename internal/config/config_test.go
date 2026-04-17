package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()

	t.Run("missing file returns empty config", func(t *testing.T) {
		cfg, err := Load(filepath.Join(dir, "nope.json"))
		if err != nil {
			t.Fatalf("Load(missing) error: %v", err)
		}
		if len(cfg.AllowList) != 0 || cfg.MergeSettingsAllow {
			t.Errorf("Load(missing) = %+v, want zero value", cfg)
		}
	})

	t.Run("valid config is parsed", func(t *testing.T) {
		path := filepath.Join(dir, "cfg.json")
		if err := os.WriteFile(path, []byte(`{
			"allow_list": ["Bash(jq *)", "Bash(git log)"],
			"merge_settings_allow": true
		}`), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load error: %v", err)
		}
		want := &Config{
			AllowList:          []string{"Bash(jq *)", "Bash(git log)"},
			MergeSettingsAllow: true,
		}
		if !reflect.DeepEqual(cfg, want) {
			t.Errorf("Load = %+v, want %+v", cfg, want)
		}
	})

	t.Run("unknown fields are ignored", func(t *testing.T) {
		path := filepath.Join(dir, "cfg_with_unknown.json")
		if err := os.WriteFile(path, []byte(`{
			"allow_list": ["Bash(wc *)"],
			"use_bundled_shs": true,
			"legacy_flag": "whatever"
		}`), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load error: %v", err)
		}
		if !reflect.DeepEqual(cfg.AllowList, []string{"Bash(wc *)"}) {
			t.Errorf("AllowList = %v", cfg.AllowList)
		}
		if cfg.MergeSettingsAllow {
			t.Errorf("MergeSettingsAllow = true, want false")
		}
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		path := filepath.Join(dir, "bad.json")
		if err := os.WriteFile(path, []byte(`{not json`), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Error("Load(malformed) expected error, got nil")
		}
	})
}
