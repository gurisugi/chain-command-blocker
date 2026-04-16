#!/bin/bash

SCRIPT_DIR="$(dirname "$0")"
SCRIPT="$SCRIPT_DIR/../hooks/block-chained-commands.sh"

# テスト用の一時設定ファイルを作成
TMPCONFIG=$(mktemp)
cat > "$TMPCONFIG" <<'JSON'
{
  "allow_list": [
    "Bash(jq *)",
    "Bash(git log *)",
    "Bash(wc *)"
  ]
}
JSON
export CHAIN_COMMAND_BLOCKER_CONFIG="$TMPCONFIG"

pass=0
fail=0

run_test() {
  local description="$1"
  local input="$2"
  local expect="$3"
  local env="${4:-}"

  if [ -n "$env" ]; then
    result=$(echo "$input" | env $env bash "$SCRIPT" 2>/dev/null)
  else
    result=$(echo "$input" | bash "$SCRIPT" 2>/dev/null)
  fi

  if [ -z "$result" ]; then
    actual="allow"
  else
    actual="ask"
  fi

  if [ "$actual" = "$expect" ]; then
    echo "PASS: $description"
    ((pass++))
  else
    echo "FAIL: $description (expected=$expect, actual=$actual)"
    echo "  output: $result"
    ((fail++))
  fi
}

run_test "単一コマンド" \
  '{"tool_input":{"command":"ls -la"}}' \
  "allow"

run_test "jq | jq (全てallow list内)" \
  '{"tool_input":{"command":"jq . file.json | jq .name"}}' \
  "allow"

run_test "echo | jq (echoがallow list外)" \
  '{"tool_input":{"command":"echo hello | jq ."}}' \
  "ask"

run_test "git status && git diff (gitがallow list外)" \
  '{"tool_input":{"command":"git status && git diff"}}' \
  "ask"

run_test "jq . a.json; jq . b.json (セミコロン、全てallow list内)" \
  '{"tool_input":{"command":"jq . a.json; jq . b.json"}}' \
  "allow"

run_test "空コマンド" \
  '{"tool_input":{}}' \
  "allow"

# jqクエリ内の | がシェルパイプとして誤検出されないことを確認
run_test "gh | jq (ghがallow list外)" \
  '{"tool_input":{"command":"gh api repos/o/r/pulls/1/reviews | jq '\''[.[] | select(.user.login==\"gurisugi\")]'\'' "}}' \
  "ask"

run_test "gh | jq (ghがallow list外・複数パイプ)" \
  '{"tool_input":{"command":"gh api repos/o/r/pulls | jq '\''[.[] | select(.draft==false) | .title]'\'' "}}' \
  "ask"

run_test "gh | jq (ghがallow list外・ダブルクォート)" \
  '{"tool_input":{"command":"gh api repos/o/r/pulls | jq \"[.[] | select(.state==\\\"APPROVED\\\")]\" "}}' \
  "ask"

# コマンド置換内のコマンドも展開されるため、allow list外のコマンドが検出される
run_test "git commit with HEREDOC containing pipe (catがallow list外)" \
  '{"tool_input":{"command":"git commit -m \"$(cat <<'"'"'EOF'"'"'\njqクエリ内の | が問題\nEOF\n)\""}}' \
  "ask"

run_test "コマンド置換内にパイプがある場合 (echo, grepがallow list外)" \
  '{"tool_input":{"command":"echo \"$(echo foo | grep f)\""}}' \
  "ask"

# 設定ファイルなしの場合: 許可リストが空なので全チェーンコマンドでask
run_test "設定ファイルなし: jq | jq でもask" \
  '{"tool_input":{"command":"jq . file.json | jq .name"}}' \
  "ask" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=/nonexistent"

# use_bundled_shs のテスト
TMPCONFIG_BUNDLED=$(mktemp)
cat > "$TMPCONFIG_BUNDLED" <<JSON
{
  "allow_list": ["Bash(jq *)"],
  "use_bundled_shs": true
}
JSON

# CLAUDE_PLUGIN_ROOT に同梱版shs（実際のshsへのシンボリックリンク）を用意
TMPBIN=$(mktemp -d)
mkdir -p "$TMPBIN/bin"
TEST_SHS_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
TEST_SHS_ARCH=$(uname -m)
case "$TEST_SHS_ARCH" in
  x86_64)  TEST_SHS_ARCH="amd64" ;;
  aarch64) TEST_SHS_ARCH="arm64" ;;
esac
ln -s "$(command -v shs)" "$TMPBIN/bin/shs-${TEST_SHS_OS}_${TEST_SHS_ARCH}"

run_test "use_bundled_shs: 同梱版shsで動作する" \
  '{"tool_input":{"command":"echo hello | jq ."}}' \
  "ask" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_BUNDLED CLAUDE_PLUGIN_ROOT=$TMPBIN"

run_test "use_bundled_shs: 同梱版shsが見つからない場合はskip" \
  '{"tool_input":{"command":"echo hello | jq ."}}' \
  "allow" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_BUNDLED CLAUDE_PLUGIN_ROOT=/nonexistent"

rm -f "$TMPCONFIG_BUNDLED"
rm -rf "$TMPBIN"

# merge_settings_allow のテスト
TMPCONFIG_MERGE=$(mktemp)
cat > "$TMPCONFIG_MERGE" <<'JSON'
{
  "allow_list": ["Bash(jq *)"],
  "merge_settings_allow": true
}
JSON

TMPSETTINGS=$(mktemp)
cat > "$TMPSETTINGS" <<'JSON'
{
  "permissions": {
    "allow": [
      "Bash(gh pr view *)",
      "Bash(gh search:*)",
      "Bash(git log)",
      "Bash(sed */foo/bar *)",
      "WebFetch(domain:example.com)"
    ],
    "ask": ["Bash(rm *)"],
    "deny": ["Bash(find *)"]
  }
}
JSON

run_test "merge_settings_allow: Bash(xxx *) 形式がマージされる" \
  '{"tool_input":{"command":"gh pr view 123 | jq ."}}' \
  "allow" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_MERGE CHAIN_COMMAND_BLOCKER_SETTINGS=$TMPSETTINGS"

run_test "merge_settings_allow: Bash(xxx:*) 形式がマージされる" \
  '{"tool_input":{"command":"gh search issues foo | jq ."}}' \
  "allow" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_MERGE CHAIN_COMMAND_BLOCKER_SETTINGS=$TMPSETTINGS"

run_test "merge_settings_allow: ワイルドカードなしの Bash(xxx) もマージされる" \
  '{"tool_input":{"command":"git log --oneline | jq ."}}' \
  "allow" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_MERGE CHAIN_COMMAND_BLOCKER_SETTINGS=$TMPSETTINGS"

run_test "merge_settings_allow: ask の Bash(rm *) はマージされない" \
  '{"tool_input":{"command":"rm foo | jq ."}}' \
  "ask" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_MERGE CHAIN_COMMAND_BLOCKER_SETTINGS=$TMPSETTINGS"

run_test "merge_settings_allow: 中間ワイルドカードはスキップされる" \
  '{"tool_input":{"command":"sed foo/bar baz | jq ."}}' \
  "ask" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_MERGE CHAIN_COMMAND_BLOCKER_SETTINGS=$TMPSETTINGS"

run_test "merge_settings_allow: Bash( 以外の prefix は無視される" \
  '{"tool_input":{"command":"WebFetch args | jq ."}}' \
  "ask" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_MERGE CHAIN_COMMAND_BLOCKER_SETTINGS=$TMPSETTINGS"

run_test "merge_settings_allow: settings.json が存在しなくてもエラーにならない" \
  '{"tool_input":{"command":"jq . a | jq . b"}}' \
  "allow" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_MERGE CHAIN_COMMAND_BLOCKER_SETTINGS=/nonexistent"

# 複数 settings.json のマージ（ユーザー・プロジェクト・プロジェクトローカル想定）
TMPSETTINGS2=$(mktemp)
cat > "$TMPSETTINGS2" <<'JSON'
{
  "permissions": {
    "allow": [
      "Bash(make build *)"
    ]
  }
}
JSON

TMPSETTINGS3=$(mktemp)
cat > "$TMPSETTINGS3" <<'JSON'
{
  "permissions": {
    "allow": [
      "Bash(cargo test *)"
    ]
  }
}
JSON

run_test "merge_settings_allow: 複数 settings を : 区切りで全てマージ (1つ目)" \
  '{"tool_input":{"command":"gh pr view 1 | jq ."}}' \
  "allow" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_MERGE CHAIN_COMMAND_BLOCKER_SETTINGS=$TMPSETTINGS:$TMPSETTINGS2:$TMPSETTINGS3"

run_test "merge_settings_allow: 複数 settings を : 区切りで全てマージ (2つ目)" \
  '{"tool_input":{"command":"make build foo | jq ."}}' \
  "allow" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_MERGE CHAIN_COMMAND_BLOCKER_SETTINGS=$TMPSETTINGS:$TMPSETTINGS2:$TMPSETTINGS3"

run_test "merge_settings_allow: 複数 settings を : 区切りで全てマージ (3つ目)" \
  '{"tool_input":{"command":"cargo test --all | jq ."}}' \
  "allow" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_MERGE CHAIN_COMMAND_BLOCKER_SETTINGS=$TMPSETTINGS:$TMPSETTINGS2:$TMPSETTINGS3"

rm -f "$TMPSETTINGS2" "$TMPSETTINGS3"

# CLAUDE_PROJECT_DIR を使ったプロジェクト設定ファイルの自動検出
TMPPROJDIR=$(mktemp -d)
mkdir -p "$TMPPROJDIR/.claude"
cat > "$TMPPROJDIR/.claude/settings.json" <<'JSON'
{
  "permissions": {
    "allow": ["Bash(make test *)"]
  }
}
JSON
cat > "$TMPPROJDIR/.claude/settings.local.json" <<'JSON'
{
  "permissions": {
    "allow": ["Bash(pytest *)"]
  }
}
JSON

# CHAIN_COMMAND_BLOCKER_SETTINGS 未設定時は HOME/.claude + CLAUDE_PROJECT_DIR/.claude の3層を見る
TMPHOME=$(mktemp -d)
mkdir -p "$TMPHOME/.claude"
cat > "$TMPHOME/.claude/settings.json" <<'JSON'
{
  "permissions": {
    "allow": ["Bash(gh pr view *)"]
  }
}
JSON

run_test "merge_settings_allow: CLAUDE_PROJECT_DIR の settings.json を拾う" \
  '{"tool_input":{"command":"make test unit | jq ."}}' \
  "allow" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_MERGE HOME=$TMPHOME CLAUDE_PROJECT_DIR=$TMPPROJDIR"

run_test "merge_settings_allow: CLAUDE_PROJECT_DIR の settings.local.json を拾う" \
  '{"tool_input":{"command":"pytest tests/ | jq ."}}' \
  "allow" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_MERGE HOME=$TMPHOME CLAUDE_PROJECT_DIR=$TMPPROJDIR"

run_test "merge_settings_allow: ユーザー設定 (HOME/.claude/settings.json) も拾う" \
  '{"tool_input":{"command":"gh pr view 123 | jq ."}}' \
  "allow" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_MERGE HOME=$TMPHOME CLAUDE_PROJECT_DIR=$TMPPROJDIR"

rm -rf "$TMPPROJDIR" "$TMPHOME"

# デフォルト（merge_settings_allow: false）ではマージされない
TMPCONFIG_NOMERGE=$(mktemp)
cat > "$TMPCONFIG_NOMERGE" <<'JSON'
{
  "allow_list": ["Bash(jq *)"]
}
JSON

run_test "merge_settings_allow: デフォルト無効ではマージされない" \
  '{"tool_input":{"command":"gh pr view 123 | jq ."}}' \
  "ask" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_NOMERGE CHAIN_COMMAND_BLOCKER_SETTINGS=$TMPSETTINGS"

rm -f "$TMPCONFIG_MERGE" "$TMPCONFIG_NOMERGE" "$TMPSETTINGS"

# allow_list の書式パースのテスト
TMPCONFIG_FMT=$(mktemp)
cat > "$TMPCONFIG_FMT" <<'JSON'
{
  "allow_list": [
    "Bash(jq *)",
    "Bash(gh search:*)",
    "Bash(git log)",
    "Bash(sed */foo/bar *)",
    "plain_text_entry"
  ]
}
JSON

run_test "allow_list 書式: Bash(xxx *) が認識される" \
  '{"tool_input":{"command":"jq . a | jq . b"}}' \
  "allow" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_FMT"

run_test "allow_list 書式: Bash(xxx:*) が認識される" \
  '{"tool_input":{"command":"gh search issues foo | jq ."}}' \
  "allow" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_FMT"

run_test "allow_list 書式: Bash(xxx) が認識される" \
  '{"tool_input":{"command":"git log --oneline | jq ."}}' \
  "allow" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_FMT"

run_test "allow_list 書式: 中間ワイルドカードはスキップ" \
  '{"tool_input":{"command":"sed foo/bar baz | jq ."}}' \
  "ask" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_FMT"

run_test "allow_list 書式: Bash(...) 以外の生文字列は無視される" \
  '{"tool_input":{"command":"plain_text_entry arg | jq ."}}' \
  "ask" \
  "CHAIN_COMMAND_BLOCKER_CONFIG=$TMPCONFIG_FMT"

rm -f "$TMPCONFIG_FMT"

rm -f "$TMPCONFIG"

echo ""
echo "--- 結果: $pass passed, $fail failed ---"
[ "$fail" -eq 0 ] && exit 0 || exit 1
