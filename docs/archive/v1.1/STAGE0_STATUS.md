# Stage 0 基盤実装・検証状況

更新日: 2026-10-20（Asia/Tokyo）。**Stage 0 の基盤を実装したが、安全ゲートは未達。Stage 1 は未着手。** Windows実機検証はユーザーの手元で後から行う。

## 実装した範囲

| 領域 | ファイル | 現在の挙動 |
|---|---|---|
| 正式要件 | REQUIREMENTS.md | 元v1.1を内容同一で移動。SHA-256 `6249fb89e5dde5a33a1292c0f1f3189c5db15065f826ad9933c2b1a95571197b` |
| ドメイン | internal/core | Project/Task/Execution等の型、許可遷移表、既定上限、操作内容hashと承認の整合検査 |
| 保存 | internal/store | SQLite WAL/排他ロック、schema版/checksum、状態・監査event・outbox・冪等receiptの同一tx保存 |
| 停止・復旧 | internal/store | holdの合成、個別解除、取消/中断要求、不確定実行/送信のunknown化、Project/Task復旧待ち、無条件再実行なし |
| 承認 | core/store/httpapi/webui | 操作hash/期限/一回scope/版の確認、取消時失効。記録だけで、実際の外部操作を送信しない |
| 管理API | internal/httpapi | 127.0.0.1のみ、正確なHost/Origin、session/CSRF、単回bootstrap、サイズ/未知項目/重複JSON拒否 |
| 管理画面 | internal/webui | Go埋込みの日本語UI、目標・停止理由・イベント・承認・診断。応答不明時は同じ内容の操作keyを保持 |
| データ保護 | internal/privatefs | Unix権限/リンク検査、Windowsの保護DACL作成と所有者/ACE/reparse検査のコード。Windows実効性は未検証 |
| バックアップ | internal/store、CLI | VACUUM INTO、整合性/FK検査、既存保存先を上書きしない設置、別領域への復元確認 |
| Runner | internal/runner | 全能力false、常にunverified。起動・外部送信は実装せず、start APIは必ず拒否 |
| 起動・再利用 | cmd/asteriade、scripts | 単一EXE、既定ブラウザへのprivateな引渡し、Go環境準備、Windows記録、Linux実プロセスsmoke |

## 今回実施した検証

Linux x86_64 / Go 1.27.2。Go本体は公式リリースメタデータのSHA-256を照合して導入。依存は通常のGo checksum検証と `go mod verify` を維持した。確認を無効化してインストールしていない。

- `go test -race -json ./...`: **35トップレベルテスト、サブテスト込み65 passed、failed 0、テストskip 0**。main/webuiのテスト未定義packageを「テスト合格件数」に含めない。
- Host/Origin/session/CSRF拒否、単回bootstrapの競合、未知/重複JSON、内容変更時の冪等キー拒否、並行再送で1件だけ保存を確認。
- 期限切れ承認の再送、取消時承認失効、他Projectへの不整合参照、event失敗時のrollbackを確認。
- DB fixtureによるクラッシュ/送信成否不明、停止要求と終了確認の区別、二重Open拒否、破損原本保全、バックアップ復元を確認。SQLiteのmax_page_countで容量不足を模擬し、失敗時の原子的rollbackと既存データ保全を確認（実ディスク枯渇は未試験）。
- Linuxの実行ファイルを別プロセスとして起動し、認証→目標保存→実行拒否→Origin拒否→終了→再起動→バックアップ→別領域復元を確認。`python scripts/smoke.py` で再現可能。
- `go vet ./...`、JSの `node --check internal/webui/app.js`、LinuxのCGO不要ビルド、Windows x64のCGO不要クロスビルドを確認。
- `scripts/setup.sh` を既存ツール/依存上で実行し、再実行とmodule検証を確認。

Windows向け成果物は `bin/asteriade.exe`。SHA-256 `b24358b30bb775a67392f73abf49a50fdc2c49c947ab1c3fbb097d08e3dbed27`。このEXEは未署名・Windows実行未確認で、正式リリースではない。生成物と作業用データはGit対象外。

## 方式選定と制約

SQLite driverは `github.com/ncruces/go-sqlite3 v0.35.6`。公式READMEでCGO不要/Go化SQLite/Windows対応の記載とライセンスを確認し、依存版を固定した。外部Wasmランタイムをインストールする方式ではない。初期候補moderncはこの環境で配布zipのredirect先が通信制限に拒否されたため採用せず、チェックサム保護を維持できる選定済みdriverを使った。追加の通信許可やTLS/署名検証の無効化はしていない。ライセンスは [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md)。

この実装のマイグレーションはStage0で実際に使う保存表だけであり、基本設計の全段階schemaではない。自律的Plan/Phase生成、Agent再帰、Provider Sessionの実接続、実行claim/epoch/lease/時間計測の実Worker結合、成果物の実行検証、型付き外部操作Broker、秘密保管との接続はまだ実装していない。DBにあるexecutionは復旧試験用fixtureであり、実Workerを起動した結果ではない。

WindowsのWorker identity/Job/NTFS/Firewall/WFP、Codex Hostと子孫の資格情報/通信分離、認証IPC/fingerprintの方式は実機での成立確認を要する。安全性が不明なまま汎用プロセス/通信経路を作らず、全能力を閉じた。設計上の状態表だけで、実際の依存・実行上限・安全性が強制できるとは主張しない。

Windows ACLのコードとブラウザ起動はクロスコンパイルのみ。Windows基盤テスト、sandbox否定試験、実Codex認証/承認/停止、ブラウザ上のUI操作E2E、ディスク枯渇の実機試験、通知/Edge/Bedrock/料金試験は未実施。ログ/DB/バックアップの暗号化、保持・削除の実装も未着手で、現時点はprivateな保存領域と秘密の未導入に限定している。

## 委任と次の作業

`gpt-6-luna` 指定で3つの独立担当を起動し、core/Runner・SQLite・UIを委任した。core担当は後続でprivatefsも担当。親がAPI/起動/統合/追加の異常系試験と最終判断を担当した。起動ツールは指定を受理し、結果を受領したが、実行基盤内部の独立したモデル識別情報は確認できない。

ユーザーは手元Windows11での後日検証を選択済み。[WINDOWS_VALIDATION.md](WINDOWS_VALIDATION.md) と `scripts/verify-windows.ps1` で基盤の結果を記録し、それを踏まえてStage0のWindows境界・実Worker/IPC・実実行への状態/上限適用の残件を継続する。Windowsのユニットテストが成功してもsandbox合格やStage1承認にはならない。commit/push、公開、デプロイ、課金API、GitHub Actions設定変更は今回行っていない。
