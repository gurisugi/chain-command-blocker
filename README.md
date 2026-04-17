# chain-command-blocker

A Claude Code `PreToolUse` hook plugin that asks the user to confirm a Bash
invocation whenever a chained command (`&&`, `||`, `;`, `|`, etc.) contains
a sub-command not covered by the allow list.

## How it works

The plugin ships prebuilt Go binaries for `darwin/linux` × `amd64/arm64`.
When the hook fires, `hooks/block-chained-commands.sh` resolves the right
binary for the current OS/arch and `exec`s it. **No external dependencies
are required.**

## Allow list

Create `~/.claude/chain-command-blocker.json` to mark commands that should
not require confirmation even when they appear inside a chain:

```json
{
  "allow_list": [
    "Bash(jq *)",
    "Bash(git log)",
    "Bash(wc *)",
    "Bash(grep *)"
  ]
}
```

Entries use Claude Code's own `Bash(...)` syntax, the same format as
`permissions.allow`:

| Entry                 | Matches                             |
|-----------------------|-------------------------------------|
| `Bash(git log)`       | Exactly `git log`                   |
| `Bash(gh pr view *)`  | Anything starting with `gh pr view` |
| `Bash(gh search:*)`   | Same as `Bash(gh search *)`         |

Middle wildcards (e.g. `Bash(git * main)`) are **not** supported in v1 and
are silently skipped.

If the config file is missing, the allow list is empty and every chained
command prompts for confirmation.

## Merging `permissions.allow` from settings.json

Set `merge_settings_allow: true` to pull `permissions.allow` entries from
the following three layers into the allow list at runtime:

1. `~/.claude/settings.json`
2. `$CLAUDE_PROJECT_DIR/.claude/settings.json`
3. `$CLAUDE_PROJECT_DIR/.claude/settings.local.json`

```json
{
  "allow_list": ["Bash(jq *)"],
  "merge_settings_allow": true
}
```

Claude Code typically writes "always allow" decisions to
`settings.local.json`, so merging lets day-to-day approvals flow in
without having to maintain two separate allow lists. Note that
`permissions.ask` and `permissions.deny` are **not** merged in v1 — only
`allow` is.

## Development

```bash
make build   # Build binaries for all 4 platforms into bin/
make test    # go test ./...
make clean
```

### Why binaries are committed to the repo

Claude Code plugins are distributed as Git repositories, so shipping
prebuilt binaries means users don't need a Go toolchain installed.

## v1 limitations

The following aspects of Claude Code's `permissions.allow` semantics are
**not** yet implemented (planned for follow-up releases):

- wrapper stripping (e.g. `timeout npm test` matching `Bash(npm test *)`)
- middle wildcards (`Bash(git * main)`)
- implicit read-only commands (`ls`, `cat`, etc. passing without an
  explicit allow entry)
- no-boundary wildcards (`Bash(ls*)` matching `lsof`)
- `deny` / `ask` rule handling
