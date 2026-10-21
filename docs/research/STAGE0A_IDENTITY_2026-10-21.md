# Stage 0A 最小Spec envelope / 凍結identity 実装記録

日付: 2026-10-21。状態: **最初のlibrary増分を実装・Linuxで検証。Stage 0全体とStage 1は未合格。**

## 実装

`internal/research/identity.go` はGo標準libraryだけを使う。`StrategySpec` は完全な戦略言語ではなく、synthetic環境、instrument、timeframe、Signal/Entry/Exit/Sizing/Riskの内容hashを固定するenvelopeである。参照されたコードや式は実行しない。

`FrozenManifest` はProject/ResearchProgram/Hypothesis/Cycle/Experiment/selection-familyのID、Specとそのhash、kernel/code/dataset/partition/benchmark/protocol/cost/metrics/required-gatesのSHA-256参照、seed、UTC凍結時刻を含む。`Freeze` は値をコピーしてidentityを計算し、`Verify` は対応版・形式と両hashの整合性を再計算する。

方式は固定フィールド順の `encoding/json`（`asteriade/freeze-json-v1`）で、hash計算時の `identity_hash` は空文字。SHA-256はlowercase hex。RFC 8785互換は主張しない。現在の入力はGoの型のみで、汎用JSON decoder、map、可変slice、浮動小数点式は扱わない。文字列の不正UTF-8を拒否し、JSON置換による異なる入力の同一化を避ける。

synthetic + development以外の宣言は拒否する。ID/必須hashの欠落、未対応版、zero/不正hash、不正時刻やcomponentも拒否する。初期入力制約はASCII ID 3〜128 byte、instrumentはUTF-8かつ128 byte以下、正のtimeframe、UTCの2000〜2100年。これは検査用schemaの制約であり、市場の取引仕様ではない。

API、UI、SQLite、Provider、data feed、Runnerには接続していない。DB migration、依存関係、既存の実行拒否設定は変更していない。

## 実測

| 検査 | 結果 |
|---|---|
| `go test ./internal/research` | pass |
| `go test -race -json ./...` | 40 top-level / subtest込み105件pass、fail 0、test skip 0 |
| `go vet ./...` | pass |
| `go mod verify` | all modules verified |
| CGOなしLinux/Windows x64ビルド | pass。Windowsでの実行は未実施 |
| 独立したPython JSON/SHA-256計算 | 固定fixtureのdigestがGoのgoldenと一致 |
| 文書リンク・対象差分 | 現行設計のローカルリンクを確認。原本は受領時のhashを保持 |

新libraryの試験は、同じ型入力の再現性、呼出元の値変更からの分離、Spec・seed・benchmark・protocol・時計・hash改変の拒否、必須identity欠落、実データ/Holdout/Prospective宣言や不正UTF-8の拒否を含む。固定vectorのidentity: `92cc31a04b9290993141084d570652e365992099b9feb9e896afa87ddfc9fa51`。固定値は合成fixtureでありBTC市場データではない。

親がコード全文を確認し、欠けていたprotocol参照とUTF-8検査を追加修正させた。実装は既存 `gpt-6-luna` 指定のサブエージェントに委任、最終判断と統合検証は親Codexが担当した。内部のモデル再ルーティングは観測できない。

## この検査が証明しないもの

- hashは署名・認証・時計のattestationではない。変更してhashを再計算できる主体に対する改竄防止にはならない。
- IDの実在、親子/選抜系譜、参照hashに対応するファイル本体、データ源の真偽や情報利用可能時刻、科学ゲートの実施/合格を検査しない。
- データ宣言の形式拒否は、実データ・HoldoutへのOSアクセス制御の証明ではない。未知の外部入力を受けるJSON APIとして使う前に別の境界検査が必要。
- 科学的な実験凍結、追記台帳、Holdout独立性、Prospective開始の証明、固定会計カーネル、Windows Sandboxを完成したとは扱わない。
- 研究成果、優位性、収益性を判定しない。`Verify`の成功を`validated`へ昇格する条件として単独利用しない。

## 次の作業と停止条件

Stage 0の残りは、既存engine/旧BTC資産の再利用評価、合成約定・会計の独立参照fixture、kernel固定と候補/Holdout台帳、実市場仕様、Windows否定試験である。暗号資産現物を優先するがvenue/feed/timeframe・保管権・注文範囲は未決。実データ取込と有料サービスを開始しない。Windows能力は未検証、Runnerは全能力false/start拒否を維持する。
