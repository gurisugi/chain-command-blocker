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
