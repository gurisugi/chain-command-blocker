#!/bin/bash

# shs (github.com/gurisugi/shs) でシェルコマンドを分解し、
# allow list外のコマンドが含まれていたらユーザーに確認を求める。

# jq の存在チェック
if ! command -v jq &>/dev/null; then
  echo "chain-command-blocker: 'jq' is not installed. Skipping hook." >&2
  exit 0
fi

# Bash(...) 形式のエントリを前方一致パターンに変換する。
# stdin から 1 行 1 エントリを受け取り、変換後のパターンを stdout へ出す。
# Bash(...) 以外の形式、中間ワイルドカード、空エントリはスキップする。
parse_bash_patterns() {
  local entry inner
  while IFS= read -r entry; do
    [ -z "$entry" ] && continue
    if [[ "$entry" =~ ^Bash\((.*)\)$ ]]; then
      inner="${BASH_REMATCH[1]}"
      # 末尾の " *" または ":*" を除去
      inner="${inner% \*}"
      inner="${inner%:\*}"
      # 中間に * が残る場合は前方一致に変換できないのでスキップ
      [[ "$inner" == *\** ]] && continue
      [ -z "$inner" ] && continue
      printf '%s\n' "$inner"
    fi
  done
}

# 許可リストと設定の読み込み
CONFIG_FILE="${CHAIN_COMMAND_BLOCKER_CONFIG:-$HOME/.claude/chain-command-blocker.json}"
ALLOW_LIST=()
USE_BUNDLED_SHS=false
MERGE_SETTINGS_ALLOW=false

if [ -f "$CONFIG_FILE" ]; then
  while IFS= read -r entry; do
    [ -n "$entry" ] && ALLOW_LIST+=("$entry")
  done < <(jq -r '.allow_list[]? // empty' "$CONFIG_FILE" 2>/dev/null | parse_bash_patterns)

  if [ "$(jq -r '.use_bundled_shs // false' "$CONFIG_FILE" 2>/dev/null)" = "true" ]; then
    USE_BUNDLED_SHS=true
  fi

  if [ "$(jq -r '.merge_settings_allow // false' "$CONFIG_FILE" 2>/dev/null)" = "true" ]; then
    MERGE_SETTINGS_ALLOW=true
  fi
fi

# settings.json の permissions.allow を取り込む
if [ "$MERGE_SETTINGS_ALLOW" = true ]; then
  SETTINGS_FILE="${CHAIN_COMMAND_BLOCKER_SETTINGS:-$HOME/.claude/settings.json}"
  if [ -f "$SETTINGS_FILE" ]; then
    while IFS= read -r entry; do
      [ -n "$entry" ] && ALLOW_LIST+=("$entry")
    done < <(jq -r '.permissions.allow[]? // empty' "$SETTINGS_FILE" 2>/dev/null | parse_bash_patterns)
  fi
fi

# shs のパス解決
if [ "$USE_BUNDLED_SHS" = true ]; then
  SHS_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  SHS_ARCH=$(uname -m)
  case "$SHS_ARCH" in
    x86_64)  SHS_ARCH="amd64" ;;
    aarch64) SHS_ARCH="arm64" ;;
  esac
  SHS="$CLAUDE_PLUGIN_ROOT/bin/shs-${SHS_OS}_${SHS_ARCH}"
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
