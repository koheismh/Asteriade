# Asteriade（アステリアーデ）

Windows 11 を優先した自律AI投資研究システムです。v2の開発基準をレビューし、現在は Stage 0 の基盤整備を進めています。既存の目標保存UIは旧v1.1の基盤で、研究ループはまだありません。**Runner の実効安全境界は未検証のため、自律実行は無効**です。Codex/Bedrock/Kiro のモデル実行や外部送信は行いません。

開発基準は [docs/REQUIREMENTS.md](docs/REQUIREMENTS.md)。[今回のレビューと未決ゲート](docs/research/REQUIREMENTS_REVIEW_2026-10-21.md)も参照してください。[基本設計](docs/ARCHITECTURE.md)、[実装計画](docs/IMPLEMENTATION_PLAN.md)、[現在の実装と検証状況](docs/STAGE0_STATUS.md) を参照してください。

## ビルドとテスト

Go 1.27.2 を使います。SQLite は Go に組み込まれ、実行時に Node.js/npm/Docker/外部 DB サーバーは不要です。

Windows PowerShell:

```powershell
go mod verify
go test ./...
$env:CGO_ENABLED = '0'
go build -trimpath -o bin/asteriade.exe ./cmd/asteriade
.\bin\asteriade.exe
```

Windows では通常のユーザーとして起動してください。既定の保存先は `%LOCALAPPDATA%\Asteriade`。必要なら `-data-dir` で専用の非共有ディレクトリを指定できます。所有者・ACL の検査に失敗する保存先、再解析点、共有先は拒否します。管理者権限の常用や、既存の ACL の自動的な緩和はしません。

初期セッションは10分・1回限りの起動リンクから取得します。既定ブラウザを開けない場合は、ログに示される `launch-*.html` を同じPCで開いてください。起動リンクを共有しないでください。アプリの終了時に起動ファイルを削除します。期限切れや別ブラウザへの切替時は再起動が必要です。

Linux のクラウド環境:

```bash
bash scripts/setup.sh
source scripts/env.sh
go test ./...
CGO_ENABLED=0 go build -trimpath -o bin/asteriade ./cmd/asteriade
./bin/asteriade -data-dir .local/data -no-browser
```

既存チェックアウトを使用し、新しい Git worktree は不要です。サーバーは IPv4 loopback のみに bind し、LAN やインターネットへ公開しません。クラウドの loopback は手元のブラウザとは異なり、今回の環境では Web プレビューを提供しません。

Windows 向けクロスビルド:

```bash
source scripts/env.sh
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o bin/asteriade.exe ./cmd/asteriade
```

## Stage 0 で利用できる機能

- 目標の保存と一覧、状態と監査イベントの表示。
- 全体/Project別の一時停止、個々の停止理由を解除する再開、取消・中断要求の記録。
- 操作内容・対象ハッシュ・期限に束縛した一回限りの承認記録。承認しても未検証 Runner は実行できません。
- SQLite WAL、マイグレーション、原子的なイベント/outbox/操作キー保存、クラッシュ後の結果確認待ち表示。
- 同じデータ領域の二重起動拒否と一貫したバックアップ。

v2向けの最小Spec envelope/凍結identityは `internal/research` に独立libraryとして追加しました。まだUIや実行系には接続していません。[検証と限界](docs/research/STAGE0A_IDENTITY_2026-10-21.md)を参照してください。

安全境界を検証した Worker、公式 Codex 接続、自律的な計画/ツール実行はまだありません。中断要求の記録はプロセス停止の実証ではありません。実行 API は常に拒否し、診断画面にも未検証理由を表示します。

## バックアップと復元

アプリを終了してから、専用の非共有フォルダへバックアップします。稼働中 DB 本体だけのファイルコピーは使いません。

```powershell
.\bin\asteriade.exe -backup "$env:LOCALAPPDATA\AsteriadeBackups\backup.db"
```

バックアップは SQLite の `VACUUM INTO` と整合性/外部キー検査を使い、既存の保存先ファイルを上書きしません。元と同じフォルダに置くより、アクセスを制限した別の保存先を推奨します。現在のバックアップは暗号化されていません。

復元はアプリを停止し、元のデータ領域を保全したまま、新しい非共有フォルダへバックアップを `asteriade.db` という名前でコピーし、そのフォルダを `-data-dir` に指定します。既存の DB/WAL/SHM を混在させないでください。起動時の整合性・マイグレーション検査が通ること、目標とイベントが読み出せることを確認してから切り替えます。破損時は元データを上書きせず停止します。

## Windows での引継ぎ（旧基盤の検証）

[実機検証手順](docs/WINDOWS_VALIDATION.md) に従い、まず `scripts/verify-windows.ps1` で基盤テストとビルドを記録してください。成功しても OS による禁止操作の拒否を証明したことにはならず、Stage 1 の自律研究は開放しません。旧配布ZIPは研究機能を含まない基盤検証版です。
