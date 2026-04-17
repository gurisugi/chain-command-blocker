package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolvePaths(t *testing.T) {
	t.Run("env override wins", func(t *testing.T) {
		got := ResolvePaths("/a/s.json:/b/s.json", "/home/u", "/proj")
		want := []string{"/a/s.json", "/b/s.json"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ResolvePaths override = %v, want %v", got, want)
		}
	})

	t.Run("default 3 layer", func(t *testing.T) {
		got := ResolvePaths("", "/home/u", "/proj")
		want := []string{
			"/home/u/.claude/settings.json",
			"/proj/.claude/settings.json",
			"/proj/.claude/settings.local.json",
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ResolvePaths default = %v, want %v", got, want)
		}
	})

	t.Run("empty home and project yields empty list", func(t *testing.T) {
		got := ResolvePaths("", "", "")
		if len(got) != 0 {
			t.Errorf("ResolvePaths empty = %v, want empty", got)
		}
	})
}

func TestCollectAllowRules(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	proj := filepath.Join(dir, "proj")
	mustMkdir(t, filepath.Join(home, ".claude"))
	mustMkdir(t, filepath.Join(proj, ".claude"))

	mustWrite(t, filepath.Join(home, ".claude", "settings.json"), `{
		"permissions": {"allow": ["Bash(jq *)", "Bash(git log)"]}
	}`)
	mustWrite(t, filepath.Join(proj, ".claude", "settings.json"), `{
		"permissions": {"allow": ["Bash(npm test)"]}
	}`)
	mustWrite(t, filepath.Join(proj, ".claude", "settings.local.json"), `{
		"permissions": {"allow": ["Bash(wc *)"]}
	}`)

	t.Run("unions across 3 layers", func(t *testing.T) {
		paths := ResolvePaths("", home, proj)
		got, err := CollectAllowRules(paths)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"Bash(jq *)", "Bash(git log)", "Bash(npm test)", "Bash(wc *)"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("missing files are skipped", func(t *testing.T) {
		paths := []string{filepath.Join(dir, "nope.json")}
		got, err := CollectAllowRules(paths)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("got %v, want empty", got)
		}
	})

	t.Run("malformed JSON is silently skipped", func(t *testing.T) {
		bad := filepath.Join(dir, "bad.json")
		mustWrite(t, bad, `{not json`)
		good := filepath.Join(home, ".claude", "settings.json")
		got, err := CollectAllowRules([]string{bad, good})
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"Bash(jq *)", "Bash(git log)"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("missing permissions.allow is ignored", func(t *testing.T) {
		empty := filepath.Join(dir, "empty.json")
		mustWrite(t, empty, `{"other": "stuff"}`)
		got, err := CollectAllowRules([]string{empty})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("got %v, want empty", got)
		}
	})
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, p, content string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
