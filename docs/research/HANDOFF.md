# Asteriade 現在地と引き継ぎ

更新: 2026-10-21。方向: Windows 11ローカルの自律AI投資研究。**実注文なし。現在はStage 0Bの調査・設計判断を完了し、報告段階で停止。Stage 0Cは未着手。**

開発基準は [REQUIREMENTS.md](../REQUIREMENTS.md)。受領Draft 2と判断記録は `inputs/` に原本保存。[レビューと整合修正](REQUIREMENTS_REVIEW_2026-10-21.md)で、ユーザー依頼に基づくStage 0開始と未決ゲートを記録した。

## 完了・進行中・未開始

- 完了: 要件本文/添付とGitHub設計ブランチの同一性確認、3件の独立レビュー、Holdout監査/A-B-C条件/Prospective段階等の整合修正、v2設計5文書への更新。旧v1.1は `../archive/v1.1/` に保全。
- 保存対象: 既存Go/SQLite/loopback API・UI・承認/復旧の基盤ソース、テスト、設定/スクリプト、今回の設計/原本。既存の設計ブランチの履歴を継承した開発ブランチ `codex/investment-research-stage0` へPush済み（基準コミット `849c29f`）。既定 `master` は更新せず、`main` は存在しない。
- 実装済み増分: Stage 0Aの最小Spec envelope/凍結identity library（`internal/research`）。検証結果と限界は [Stage 0A記録](STAGE0A_IDENTITY_2026-10-21.md)。市場戦略言語、固定会計カーネル、DB研究schema、UI、Runner実行へは未接続。
- Stage 0B完了: quant-btc-lab / Freqtrade / Backtraderの固定ソースを読み取り調査し、部品別再利用判断、主要不具合/修正、正解試験案、Stage 0C未決事項を記録。[比較報告](STAGE0B_REUSE_EVALUATION_2026-10-21.md)と[Stage 0B引き継ぎ](STAGE0B_HANDOFF.md)。
- 未開始/未達: コード移植/engine実行比較、実市場データ取得、市場会計goldenの実行、Holdout共通台帳、Windows実効Sandbox、実Codex/Reviewer接続、課金台帳、Stage 1自律研究。
- Windows: ユーザー手元で後から検証する予定。クロスビルド成功はOS安全境界の証明ではない。Runnerは `unverified`、全能力false、start APIは拒否のまま。

## 確定と未決

初回優先市場はユーザー選択の**暗号資産現物（BTC等）**。venue/pair/feed/timeframe、利用規約/保管権、費用、約定範囲は未決。決まるまでは合成のみ。旧 `quant-btc-lab` の固定ソースは調査済み。部分再利用は権利/独立正解値/実行境界の確認待ちで、実データや旧成績の移植は行っていない。

Manager最終判断と独立Reviewerの最大2発言、必須科学/安全ゲート、事前benchmark、全候補/選抜系譜、Holdoutの露出・復元時の不確実性を保全する。個人金融情報、secret、非公開市場データを公開Git/外部AIへ送らない。無断の有料Providerへの切替は禁止。

## 再開順序

1. [Stage 0B引き継ぎ](STAGE0B_HANDOFF.md)と[Stage 0C判断メモ](STAGE0C_DECISIONS.md)を読み、次のユーザー依頼の範囲を確認する。本作業では報告後に停止する。
2. `source scripts/env.sh` で固定Goを使用し、意味のある変更に対応するテストと既存の必要チェックを実行する。
3. 次段階では権利/venue/feed仕様を資料で確認し、人工の会計/時計oracleを固定、必要な方針を市場profileへ具体化する。未実施/blockedを合格へ置換しない。
4. Stage 0全体の必要能力と会計/データ条件が実証されるまでStage 1を開放しない。課金・権限設定変更等の具体的な外部操作は対象ごとの許可が必要。

古い `codex/windows-stage0-download` のZIPはv1.1基盤検証用であり、投資研究機能を含まない。短期リンクの保存だけで再現可能としない。Stage 0Aの最新実測はリンク先の記録を参照する。
