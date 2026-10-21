# Windows 11 実機検証の引継ぎ

更新日: 2026-10-20（Asia/Tokyo）。ユーザーの手元の Windows 11 で後から検証する方針。クラウド側は Linux であり、クロスビルドの成功を Windows 実機の合格としない。

EXEだけで行う最初の確認は [WINDOWS_QUICKSTART.md](WINDOWS_QUICKSTART.md) を参照する。以下のGoテスト・ビルド手順はソース一式とGoが必要な追加検証であり、EXEをダウンロードしただけで全手順を実行できるわけではない。

## 1. 準備

- Windows 11 x64 の通常ユーザーで実施する。管理者アカウントの常用、ACL/Firewall の緩和、他ユーザー作成はこの手順では行わない。
- リポジトリ一式と公式 Go 1.27.2 Windows amd64 を用意する。Go の取得時は公式配布ページのチェックサムと照合する。
- 専用のテスト用保存領域を使い、個人の資格情報、本番リポジトリ、実サービス投稿先を試験材料にしない。
- SQLite は EXE に組み込まれる。アプリの起動だけなら Go/Node/npm/Docker/外部 DB は不要。Go はソースからのビルドとテストのために必要。

## 2. 基盤テストと記録

リポジトリ直下で PowerShell から実行する。

```powershell
.\scripts\verify-windows.ps1
```

ローカルの実行ポリシーでスクリプトが拒否された場合は、ポリシーを無効化せず、管理者/組織の許可済み手順を使う。スクリプト内の Go コマンドを通常の端末で順に実行してもよい。

結果は `.local\validation\go-test.jsonl` と `manifest.json`。OS/build、Go版、EXE SHA-256、テストの passed/skip 件数を記録する。**manifest は Runner=BLOCKED、Windows sandbox/Codex実経路=unrun を維持する。** Go のユニットテストは OS による禁止操作の拒否を検証したものではない。

## 3. 単一 EXE と管理画面

```powershell
.\bin\asteriade.exe
```

1. 管理画面の診断が Stage 0、実行無効、Runner未検証、Provider未検証/未対応を示すことを確認する。
2. 目標を1件保存し、一覧とイベントを確認する。別 Origin や認証なしの API が拒否される自動テスト結果も確認する。
3. 全体と Project の一時停止を両方作り、Project分だけを再開しても全体の停止理由が残ることを確認する。これは scheduler 抑制理由の保存テストであり、実 Worker の一時停止試験ではない。
4. 即時停止/取消の画面が要求と完了を混同しないことを確認する。この版に実行中 Worker は存在せず、実際のプロセス停止は未検証。
5. 正常終了・再起動後に目標・イベント・停止理由が残ることを確認する。
6. 同じ保存領域で二重起動し、2つ目が DB ロックで終了することを確認する。元のアプリを終了後は再起動できることも確かめる。
7. Node/npm/Docker/外部DBがない端末でも、Goなしで同じEXEを起動できることを確認する。

既定の保存先は `%LOCALAPPDATA%\Asteriade`。新規フォルダは現在ユーザー/SYSTEM/Administrators に限定する DACL を設定し、既存フォルダやDB/起動ファイルは所有者とACLを検査する実装。ACL不明・他主体の許可・再解析点・共有先は起動を拒否する。拒否された場合、ACLを広くして回避せず、エラーと対象フォルダの権限情報（秘密や起動リンクを除く）を記録する。既存データやACLを自動修正しない。

クラウドで生成した `bin\asteriade.exe` は Windows 向けビルド確認済みだが、コード署名と Windows 実行確認は未実施。配布/署名/信頼判断は別のリリース作業である。

## 4. 保存と復旧

README のバックアップ手順で、一意な新規バックアップを作り、元データを保全したまま新しい専用フォルダへ復元する。DB/WAL/SHMを混在させない。破損/容量不足の試験は使い捨て領域だけで行い、個人のデータを破損させない。

クラッシュ復旧は Linux のStore試験で擬似的な生存Execution/送信済みjournalをDBへ作って検証している。Windowsで実Provider/実Workerが動いた後の副作用照合は別の未実施試験であり、DB fixtureの合格で代用しない。

## 5. Stage 0 安全ゲートの残件

[SECURITY_VALIDATION.md](SECURITY_VALIDATION.md) SEC-01〜10/15/16/19 に従い、特に以下の実証を必要とする。

| 項目 | 現在 | 次に必要な証拠 |
|---|---|---|
| 限定された Worker 起動、identity、Job/子孫制限 | 未実装・未検証 | Windows方式を選び、起動競合/breakaway/helperも含めた拒否と陽性対照 |
| Workspace外/他Task/秘密へのアクセス制限 | 未検証 | 実際の Worker と Codex 子孫で canary の read/write がOS拒否される証拠 |
| 任意通信/PowerShell/Git push の遮断 | 未検証 | 管理下sinkへIPv4/IPv6/DNS/proxy/loopback/子孫で送信0、許可経路の対照 |
| Codex Host の認証とツールの秘密/通信分離 | 未実装・未検証 | 公式固定版で認証・承認・中断・usageと実ツール経路を確認 |
| 認証IPC、Named Pipe、fingerprint採取/失効 | 未実装 | 最小プロトタイプ、相手認証・squatting/replay・構成変更時拒否 |
| 状態遷移・claim/epoch・時間上限の実実行への適用 | 状態表/復旧骨格のみ | 検証済みWorkerに結合し、競合・時間・停止の試験 |

この版には実行を開放するフラグや、JSONの「検証済み」を読み込んで有効化する経路はない。Windowsの基盤テストが通っても、これらの方式の実装・実測を終えるまで自律実行不可である。結果を共有する際は manifest と秘匿化した失敗情報を使い、起動リンク・Cookie・CSRF token・Provider秘密を含めない。

Stage 1 の着手は安全ゲートを満たした後の別承認を要する。手元Windowsでの基盤検証結果に応じ、Stage 0 の残件を継続する。
