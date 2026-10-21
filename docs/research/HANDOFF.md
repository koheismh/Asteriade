# Asteriade 現在地と引き継ぎ

更新: 2026-10-21。方向: Windows 11ローカルの自律AI投資研究。**現在はStage 0Cの小規模合成参照増分まで完了し、報告で停止。Stage 0全体は未合格、Stage 1A/1B未着手。実注文なし。**

開発基準は[REQUIREMENTS.md](../REQUIREMENTS.md)。今回の専用ブランチは `codex/lean-mvp-synthetic-stage0c`、開始元はfetch済み `codex/investment-research-stage0` の `f721104b6adc6880e6d4bd8671cf78bc4aa5ff6c`。masterへマージしていない。PR #2は参照のみで、マージ/クローズしない。受領Draft 2/判断原本と過去記録は保全する。

## 完了した範囲

- 旧Go/SQLite/loopback UI/API・承認・停止/復旧基盤を維持。Runnerは全能力false、unverified、start API拒否。
- 0A: `internal/research`の最小Spec envelope/凍結identity。[記録](STAGE0A_IDENTITY_2026-10-21.md)。今回改変せず、製品RunnerやDB研究schemaへ未接続。
- 0B: quant-btc-lab/Freqtrade/Backtraderの固定ソース調査・再利用判断。[比較報告](STAGE0B_REUSE_EVALUATION_2026-10-21.md)、[引き継ぎ](STAGE0B_HANDOFF.md)。実データ・旧成績・コードを移植していない。
- 0C今回分: Python標準ライブラリの独立合成参照例。時刻/未来情報、現物会計、曖昧性/失敗拒否、独立oracle。**新規24テスト成功**、既存Go **40トップレベル/105サブテスト込み成功**。DD率履歴と売却fee不足の欠陥を独立テストで発見・修正。[実測と限界](STAGE0C_SYNTHETIC_VALIDATION_2026-10-21.md)、[実行手順](../../reference/README.md)。
- PR2本文とユーザー指示を反映した[Stage 1A/1Bスコープ案](MVP_SCOPE_REVISION_2026-10-21.md)。1Aは単一Manager＋機械ゲート、1Bで既存有限討論を追加し品質/費用/時間比較。全42製品AT・費用/科学/安全ゲートを保持する。
- [Market Knowledge最小仕様](../MARKET_KNOWLEDGE.md)。MD/JSON例は未実施研究の記入例であり、今回の合成会計テストや市場優位性の証拠ではない。

## 未完了と固定する境界

初回市場種別はユーザー選択の**暗号資産現物（BTC等）**。venue/pair/feed/timeframe、利用/保管/外部送信権、注文/費用/tick/lot/丸めprofileは未決。合成参照値を市場仕様へ流用しない。0C市場部分、engine同条件実行比較/配布、0D Holdout registry/Windows実効境界、Provider/研究loop/UI統合は未完了。

時計/会計は開発専用プリミティブで、将来値を物理的に隠すデータAPIやOS隔離は未実装。Pythonを製品Runnerとして採用した判断でもない。Go CP＋独立Python Runner、旧資産の権利確認後の部分再利用は比較候補として維持する。旧M6等の科学バグと消費済みHoldout修正を独立確認と呼べない点は0B記録を参照する。

Managerは強制科学/安全/予算ゲートを解除できない。Stage 1Aは`review_status=not_enabled_stage1a`を明記し、割当済みReviewerの失敗を1A切替で回避しない。Stage 1BはReviewer発言最大2回、採否理由/最終判断を残す。全候補/選抜系譜/benchmark/Holdout露出を保全し、個人金融情報・secret・private raw dataは公開Git/外部AIへ送らない。

## 再開順序と開始条件

1. 今回の差分と[検証記録](STAGE0C_SYNTHETIC_VALIDATION_2026-10-21.md)、[実装計画](../IMPLEMENTATION_PLAN.md)、[0C市場判断メモ](STAGE0C_DECISIONS.md)をレビューする。今回の許可範囲を超えて自動継続しない。
2. 次の限定作業はvenue等の資料確認とprofileの独立golden計画。実市場データの取得/費用発生は別の許可を要する。
3. ユーザー手元のWindows 11で必要な境界・Holdout・復旧否定試験を行う。クロスビルドや既存基盤検証スクリプト成功だけでSandboxを合格にしない。
4. Stage 0全体の必要証拠・市場仕様・Provider利用条件/料金・明示的な開始許可が揃うまで1Aへ進まない。

GitHub APIは環境ポリシーのCONNECT 403で拒否され、PR2追加コメント/レビュー状態と新PR作成は確認できていない。必要ホスト`api.github.com`の環境設定下書きを保存済みだが、保存は実行環境への反映ではない。[準備済みPR本文](STAGE0C_PR_DESCRIPTION.md)からbase/headを指定してレビュー用PRを作る。PushとPR作成の実結果を混同しない。

`source scripts/env.sh`で固定Go、Python testは追加依存不要。古い配布ZIPはv1.1基盤検証版で投資研究機能を含まない。期限付きリンクだけを再現成果物と見なさない。
