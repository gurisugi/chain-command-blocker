#!/bin/bash

SCRIPT_DIR="$(dirname "$0")"
SCRIPT="$SCRIPT_DIR/../hooks/block-chained-commands.sh"

# テスト用の一時設定ファイルを作成
TMPCONFIG=$(mktemp)
cat > "$TMPCONFIG" <<'JSON'
{
  "allow_list": [
    "jq",
    "git log",
    "wc"
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
  "allow_list": ["jq"],
  "use_bundled_shs": true
}
JSON

# CLAUDE_PLUGIN_ROOT に同梱版shs（実際のshsへのシンボリックリンク）を用意
TMPBIN=$(mktemp -d)
mkdir -p "$TMPBIN/bin"
ln -s "$(command -v shs)" "$TMPBIN/bin/shs"

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

rm -f "$TMPCONFIG"

echo ""
echo "--- 結果: $pass passed, $fail failed ---"
[ "$fail" -eq 0 ] && exit 0 || exit 1
