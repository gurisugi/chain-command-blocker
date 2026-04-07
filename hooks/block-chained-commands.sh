#!/bin/bash

# shs (github.com/gurisugi/shs) でシェルコマンドを分解し、
# allow list外のコマンドが含まれていたらユーザーに確認を求める。

# jq の存在チェック
if ! command -v jq &>/dev/null; then
  echo "chain-command-blocker: 'jq' is not installed. Skipping hook." >&2
  exit 0
fi

# 許可リストと設定の読み込み
CONFIG_FILE="${CHAIN_COMMAND_BLOCKER_CONFIG:-$HOME/.claude/chain-command-blocker.json}"
ALLOW_LIST=()
USE_BUNDLED_SHS=false

if [ -f "$CONFIG_FILE" ]; then
  while IFS= read -r entry; do
    [ -n "$entry" ] && ALLOW_LIST+=("$entry")
  done < <(jq -r '.allow_list[]' "$CONFIG_FILE" 2>/dev/null)

  if [ "$(jq -r '.use_bundled_shs // false' "$CONFIG_FILE" 2>/dev/null)" = "true" ]; then
    USE_BUNDLED_SHS=true
  fi
fi

# shs のパス解決
if [ "$USE_BUNDLED_SHS" = true ]; then
  SHS="$CLAUDE_PLUGIN_ROOT/bin/shs"
else
  SHS="shs"
fi

if ! command -v "$SHS" &>/dev/null; then
  echo "chain-command-blocker: '$SHS' is not found. Skipping hook." >&2
  exit 0
fi

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

if [ -z "$COMMAND" ]; then
  exit 0
fi

# 単一コマンドならチェーンではないので許可
CMD_COUNT=$(echo "$COMMAND" | "$SHS" -n)
if [ "$CMD_COUNT" -le 1 ]; then
  exit 0
fi

# shs でコマンドを分解し、allow listと前方一致で照合
all_cmds=()
disallowed_cmds=()

while IFS= read -r cmd; do
  [ -z "$cmd" ] && continue
  all_cmds+=("$cmd")

  allowed=false
  for a in "${ALLOW_LIST[@]}"; do
    if [[ "$cmd" == "$a" || "$cmd" == "$a "* ]]; then
      allowed=true
      break
    fi
  done

  if [ "$allowed" = false ]; then
    disallowed_cmds+=("$cmd")
  fi
done < <(echo "$COMMAND" | "$SHS")

if [ "${#disallowed_cmds[@]}" -eq 0 ]; then
  exit 0
fi

# 表示用のコマンドリストを構築
cmd_list=""
for cmd in "${all_cmds[@]}"; do
  marker="  "
  for d in "${disallowed_cmds[@]}"; do
    if [ "$cmd" = "$d" ]; then
      marker="* "
      break
    fi
  done
  cmd_list="${cmd_list}
${marker}${cmd}"
done

reason="Chained command contains non-allowlisted command(s) (* marked):
${cmd_list}"

jq -n --arg reason "$reason" '{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "ask",
    "permissionDecisionReason": $reason
  }
}'

exit 0
