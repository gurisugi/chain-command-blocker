# chain-command-blocker

チェーンされたBashコマンド（`&&`, `||`, `;`, `|` など）のうち、許可リスト外のコマンドが含まれる場合にユーザー確認を求めるPreToolUseフックプラグイン。

## 必要な依存コマンド

| コマンド | 用途 | インストール |
|---------|------|-------------|
| [shs](https://github.com/gurisugi/shs) | シェルコマンドの分解・解析 | `brew install gurisugi/tap/shs` or `go install github.com/gurisugi/shs/cmd/shs@latest` |
| [jq](https://jqlang.github.io/jq/) | JSON の解析・生成 | `brew install jq` |

依存コマンドが見つからない場合、フックはスキップされます（エラーにはなりません）。

### 同梱版 shs の利用

プラグインにはビルド済みの shs バイナリが同梱されています（対応プラットフォーム: darwin-arm64, darwin-amd64, linux-amd64, linux-arm64）。

`use_bundled_shs` を有効にすると、shs を別途インストールせずに利用できます：

```json
{
  "allow_list": ["jq", "git log"],
  "use_bundled_shs": true
}
```

デフォルトでは無効（`false`）です。ローカルにインストール済みの shs がある場合はそちらが使用されます。

## 許可リスト

`~/.claude/chain-command-blocker.json` を作成することで、チェーンに含まれていても確認不要なコマンドをカスタマイズできます（前方一致）：

```json
{
  "allow_list": [
    "jq",
    "git log",
    "wc",
    "grep"
  ]
}
```

設定ファイルが存在しない場合、許可リストは空となり、すべてのチェーンコマンドで確認が求められます。

### `settings.json` の `permissions.allow` と連携

`merge_settings_allow` を有効にすると、`~/.claude/settings.json` の `permissions.allow` にある `Bash(...)` 形式のエントリを自動的に許可リストへマージします：

```json
{
  "allow_list": ["jq"],
  "merge_settings_allow": true
}
```

変換ルール：

| `settings.json` の記法 | マージされる前方一致パターン |
|-----------------------|--------------------------------|
| `Bash(gh pr view *)`  | `gh pr view`                   |
| `Bash(gh search:*)`   | `gh search`                    |
| `Bash(git log)`       | `git log`                      |
| `Bash(sed */foo/* *)` | （中間ワイルドカードはスキップ） |
| `WebFetch(...)` 等     | （`Bash(...)` 以外は無視）      |

`permissions.ask` / `permissions.deny` や、他レイヤーの settings（プロジェクト側 `.claude/settings.json` など）は対象外です。必要に応じて `chain-command-blocker.json` に個別エントリを追加してください。

設定ファイルパスは環境変数 `CHAIN_COMMAND_BLOCKER_SETTINGS` で上書きできます（テスト用途）。

## 開発

### 同梱版 shs のバージョン更新

1. `bin/SHS_VERSION` を新しいバージョンに書き換える
2. バイナリを再ダウンロードする
3. 動作確認してコミット

```bash
echo "v0.1.0" > bin/SHS_VERSION
make clean && make
bash tests/test_block-chained-commands.sh
```

`Makefile` は [gurisugi/shs](https://github.com/gurisugi/shs) の GitHub Releases からプラットフォーム別のバイナリをダウンロードします。`gh` CLI が必要です。
