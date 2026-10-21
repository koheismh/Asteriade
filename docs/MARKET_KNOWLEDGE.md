# Market Knowledge（最小カード形式）

Market Knowledgeは市場研究で得た知識、反証、限界を短く引き継ぐためのカードである。検索データベースやUIの仕様ではない。MarkdownとJSONのカードは同じ `id` を使い、詳細なレポート、凍結manifest、候補一覧、Holdout記録は既存成果物のIDで参照する。

## 記録規則

- 現に分かっている文脈・証拠範囲だけを書く。推測は仮説として記録し、未知は`untested_conditions`または`limitations`に残す。
- 反証、否定結果、無効結果、保留を有効な記録として扱う。モデルの主張やレポート上の解釈だけで市場メカニズムの確認済み状態にしない。
- `status`は状態、`evidence_scope`は根拠の種類・範囲を表す。どちらも実験開始前に固定した合否基準や成果物IDと矛盾してはならない。
- 公開用とprivateの可視性を明示する。secret、個人資金情報、private raw dataはカードへ複写しない。許可されない外部送信にも含めない。
- ここに示す合成例は説明用fixtureであり、市場優位性を評価・主張しない。

## カード項目

必須の形は [JSON例](examples/market-knowledge-v1.synthetic.json) を参照する。`schema_version`はカード形式の版、`id`はMarkdown/JSON共通の安定ID。`phenomenon`は観測対象、`mechanism`は検証対象の説明、`alternative_explanations`と`falsification_tests`は競合理論と反証方法を記す。`rejected_conditions`、`untested_conditions`、`next_hypotheses`は適用範囲を明確にする。

`research_refs`には詳細レポート等のIDを入れる。`evidence_scope`は例として`synthetic_fixture`、`exploratory`、`independent_historical`、`prospective_pending`、`prospective_observed`、`invalid`を使い、未実施を完了に見せない。`status`は`idea`、`investigating`、`rejected_in_tested_scope`、`promising_exploratory`、`holdout_checked`、`prospective_pending`、`prospectively_observed`、`invalid`の既存分類に沿う。`limitations`に不確実性、再現性、未解決の制約を記録する。`visibility`は`public_synthetic_example`、`public`、`private`のいずれかを使う。

## Markdown記入例

[JSON記入例](examples/market-knowledge-v1.synthetic.json)に対応する[Markdownカード](examples/market-knowledge-v1.synthetic.md)を保存する。`example-unexecuted-001`は形式説明用の参照IDであり、実施済みRunを指さない。将来の実研究では既存のRun/manifest/レポートIDへ置き換える。この例の生成データ・反証試験は未実施で、今回の会計テスト結果とは別である。

保存は既存artifactと同じ公開/非公開境界を使い、追記revisionと親IDを残す案とする。未知項目は空配列または明示的な未検証文として保持し、空欄を「反証なし・検証済み」と解釈しない。自動生成・整合validator・検索UIは今回は未実装。
