# MVPをStage 1A/1Bへ段階化し、Stage 0C合成参照検証を追加

初期MVPで討論と汎用基盤を同時に作る負担を減らすため、PR #2を参照し、単一Manager＋機械ゲートの1Aを先行、独立Reviewer・最大2ラウンドと比較を1Bへ分けます。全42AT、費用上限、安全/科学ゲートは維持します。Market Knowledgeの最小Markdown/JSON仕様を追加しました。

Python標準ライブラリだけの開発用参照例で、受領前の判断・早過ぎる約定・曖昧なOHLCを拒否し、現物の現金/数量/手数料/部分売却/損益/equity/DDを固定正解値と独立Decimal oracleへ照合します。Goアプリ・StrategySpec/freeze・DB/UI・Runner能力は変更していません。汎用engineや製品Runnerの採用ではありません。

検証: Python 24件成功、Go race 40トップレベル（サブテスト込み105）成功、vet/module検証、LinuxとWindows x64クロスビルド成功。独立回帰で見つけたDD率履歴と売却fee不足の2不具合は修正済み。Windows実行/実効Sandbox・実市場/market profile・Holdout統合・Provider・自律研究は未検証/未実装です。

base: `codex/investment-research-stage0`（開始時f721104）、head: `codex/lean-mvp-synthetic-stage0c`。PR #2は参照のみ。masterへマージせず、今回の合成増分で停止します。実データ・有料API・取引所接続・実注文・自律Runnerは有効化していません。

詳細: [合成検証記録](STAGE0C_SYNTHETIC_VALIDATION_2026-10-21.md)、[スコープ改訂案](MVP_SCOPE_REVISION_2026-10-21.md)、[実装計画](../IMPLEMENTATION_PLAN.md)。
