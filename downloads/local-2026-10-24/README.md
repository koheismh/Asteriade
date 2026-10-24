# Asteriade — Windows 11へのローカル移行

2026-10-24（Asia/Tokyo）。Codex Cloudのアーティファクトを取得できないため、GitHub経由で既存の移行パッケージを配布する専用フォルダーです。

## ダウンロードするもの

- [移行ZIP](asteriade-local-2026-10-24.zip)
- [SHA-256](asteriade-local-2026-10-24.zip.sha256)

ZIPのファイル画面を開いた場合は、GitHubのDownload raw fileから保存してください。このブランチ全体のCode → Download ZIPは、移行ZIPそのものとは異なります。

移行ZIPをWindowsの「ダウンロード」フォルダーへ、`asteriade-local-2026-10-24.zip`の名前で保存してください。

## 対象と保存範囲

- 元の作業ブランチ: `codex/model-task-plan`
- 開始元HEAD: `6d01d08897b4ed53de01138ee737cc6ef9deb119`
- 現在のbranchの祖先Git履歴と、未commit変更を含む115ソースファイルを同梱しています。
- 資格情報、Gitの個人設定、アプリDB、クラウドのproxy設定、Go/cache、任意のBacktrader sourceは同梱しません。
- この配布ブランチのcommitを開発の開始HEADとして使わず、ZIP内の`restore.py`で新規フォルダーへ復元してください。

## Windows PowerShellで復元

Git、Python 3.13、Go 1.27.2を用意したPC向けです。ZIP内部の旧手順にある`py -3.12`は、このPCでは`py -3.13`へ置き換えてください。Python 3.13/Windows実機での成功はまだ未確認で、復元後の試験で確認します。

### 1. ツールとZIPを確認

```powershell
git --version
py -3.13 --version
go version

$asteriadeZip = "$env:USERPROFILE\Downloads\asteriade-local-2026-10-24.zip"
$asteriadeExpected = "d990ee901266fc3b628cf66bcae1c21085a23bbc3c45001aaf55ae6207b22bb4"
if ((Get-FileHash -LiteralPath $asteriadeZip -Algorithm SHA256 -ErrorAction Stop).Hash -ne $asteriadeExpected) {
    throw "ZIPの確認に失敗しました。ここで停止してください。"
}
Write-Output "ZIPの確認OK"
```

Goの期待値は`go version go1.27.2 windows/amd64`です。各段階でエラーが出たら、それ以降のコマンドは実行せず原因を確認してください。

### 2. 展開して復元

```powershell
$asteriadePackage = "$env:USERPROFILE\Downloads\asteriade-local-2026-10-24"
Expand-Archive -LiteralPath $asteriadeZip -DestinationPath $asteriadePackage
New-Item -ItemType Directory -Force "$env:USERPROFILE\source" | Out-Null
py -3.13 "$asteriadePackage\restore.py" --destination "$env:USERPROFILE\source\Asteriade-local"
```

すでに同じ場所に展開済みなら、`Expand-Archive`だけを省略します。復元先は未作成のフォルダーを指定してください。復元器は既存フォルダーを上書きせず、途中で失敗した場合の新規フォルダーも調査用に残します。

成功表示は`Verified source files: 115`と`HEAD, file hashes and uncommitted Git status match the cloud handoff.`です。

### 3. 状態を確認

```powershell
Set-Location "$env:USERPROFILE\source\Asteriade-local"
git branch --show-current
git rev-parse HEAD
git status --short
```

期待するbranch/HEADは上記の元ブランチと開始元HEADです。未commit変更が多数表示されるのは正常です。

### 4. 復元後の試験

以下は新たにPC上で実行する確認です。各コマンドの成功を確認して次へ進み、エラー時は止めてください。

```powershell
go mod download
go mod verify
go test -count=1 -p 4 ./...
$env:PYTHONPATH = 'reference'
$env:PYTHONDONTWRITEBYTECODE = '1'
py -3.13 -m unittest discover -s reference/tests -v
$env:CGO_ENABLED = '0'
go build -trimpath -o bin/asteriade.exe ./cmd/asteriade
```

過去のLinux復元検証はGo 164成功・helper 1skip、Python 78成功・任意比較4skipです。PCで実行した結果と区別してください。Windowsでの製品安全境界の実証は、これらの単体試験やビルドの成功とは別に必要です。

## ローカルCodexに渡す依頼

このクラウドチャットからPCを直接操作する接続はありません。ローカルで動くCodex CLIまたはIDE拡張で、復元した`Asteriade-local`を開き、次を渡してください。

```text
Asteriadeのローカル開発を引き継いでください。
最初にAGENTS.md、tasks.md、docs/research/HANDOFF.md、
docs/research/LOCAL_DEVELOPMENT_HANDOFF_2026-10-24.mdを読んでください。
Git branch/HEADと既存の未commit変更を確認し、reset/clean/上書き/無断のbranch切替をしないでください。
Pythonは手元の3.13を使い、旧手順のpy -3.12指定を置き換えてください。
Go 1.27.2/Gitを確認し、Windows上で基準試験を実行して結果を記録してください。
このPCのネットワーク方針と許可を確認して、通常の許可経路でMKT-01の公式文書調査を進めてください。
クラウドの403とPCの接続可否を区別し、拒否を迂回しないでください。
価格・足・板・約定・pair設定などの市場データAPIや問い合わせ送信は行わず、公式文書だけを調査してください。
保存・外部AI送信・公開の権利、料金と適用日、時刻・足確定・訂正・履歴・数量条件を別々に記録してください。
市場選定・科学仕様・engine採否・最終安全承認は担当上位モデルへ渡してください。
Runner能力やGATEDを開放せず、push/公開/追加課金は行わないでください。
過去のLinux結果をWindowsの実測として扱わず、tasks.mdとHANDOFF.mdに今回の結果を記録してください。
```

この配布は移行のためのもので、Stage 0全体・製品AT・Windows安全ゲートの合格を示しません。
