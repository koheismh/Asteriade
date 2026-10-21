# Asteriade 開発引き継ぎ入口

`AGENTS.md` と `docs/REQUIREMENTS.md` を読み、`docs/research/HANDOFF.md` で現在地と未決ゲートを確認する。

v2は投資研究が対象。旧v1.1の汎用Codex開発MVPは現行のStage 1ではない。ユーザーは要件確認・問題解消後の既存ファイル/添付のGitHub保存と開発開始を依頼済み。現在はStage 0のみを進め、実データ・有料Provider・Runner実行は各ゲートを満たすまで閉じる。

- 最初の優先市場は暗号資産の現物（BTC等）。venue/feed/時間足は未決。
- Stage 0の合成契約試験と、市場会計・Windows実効Sandboxの合格は別。
- 原本は `docs/research/inputs/`、旧設計は `docs/archive/v1.1/`。
- `docs/ARCHITECTURE.md`、`IMPLEMENTATION_PLAN.md`、`SECURITY_VALIDATION.md`、`RESEARCH_PROTOCOL.md`、`REPORTING_AND_HANDOFF.md` に従い、実装済みか計画かを区別する。
- Lunaへの独立作業委任は最大3件。重要判断・最終品質は親が担う。
- 外部への新しい送信、課金、リリース、デプロイ、既定ブランチへのマージを開発開始の許可から推定しない。
