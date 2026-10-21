# Asteriade 研究レポート・引き継ぎ設計案

- 状態: **設計案。レポート生成・GitHub送信は未実装**
- 初期対象: 外部接続なし・合成データのみのStage 0（今回の参照例は0C）。実市場データ、個人の資金情報、認証情報は含めない。
- 公開先: 初期GitHubリポジトリはpublicとして扱い、明示allowlistに入った内容だけを送信対象にする。
- 初期の仮想試験では個人資金値を入力せず、合成/正規化した資本だけを使う。

本書はMarkdown/JSONの研究レポート、セッション間の引き継ぎ、外部送信時の公開範囲を定める。添付文書や会話に書かれた情報を暗黙にGitへ転載しない。研究・報告の実行結果は実際に生成され、検証された後だけ「完了」と記録する。

Market Knowledgeは[最小カード仕様](MARKET_KNOWLEDGE.md)に従い、Markdown/JSON共通IDで研究レポートを参照する。現象・想定機構・反証・否定条件・未検証・次仮説を残す。今回のJSONは説明用であり、記載した反証試験は未実施。

## 1. 研究記録の共通原則

- 決定、提案、未決事項、技術仮説、実測結果を明示的に区別する。
- 否定的結果、無効Run、未検証項目、棄却候補、失敗理由を消さない。
- AIの完了/有望という発言ではなく、保存済み状態、Runner event、manifest、データ/コードhash、独立検査を根拠にする。
- 金額、結果、欠損、未実施、データ未取得を推測で埋めない。利用量やコストが不明なら`unknown`と記録する。
- 正本のmachine-readable JSONと人間向けMarkdownを同じrecord ID・revision・manifest hashで結ぶ。
- レポート生成は要約/表示処理であり、研究の採否、データの再分類、承認、外部送信を自動的に許可しない。

## 2. レポート・引き継ぎパッケージ

1研究cycleまたは実験の記録パッケージには、必要に応じて次を含める。

- `record_id`, `schema_version`, 作成/更新UTC時刻、record revision、visibility classification。
- Product/requirements version、Project/ResearchProgram/Hypothesis/Cycle/Experiment/Run IDs。
- 仮説、因果説明、反証条件、優先順位の根拠と不確実性。
- Manager roleと判断。Stage 1Aは`review_status=not_enabled_stage1a`と空のレビュー配列を記録し、完了を捏造しない。Stage 1BではReviewer role、blind Session ID、論点/回答/再反論/最終判断も保存。
- Decision、未解決点、次の具体的行動、停止理由と解除条件。
- frozen manifest ID/hash、StrategySpec hash、kernel/build identity、code/config/seed。
- Market/instrument/timeframe/period、dataset IDsとpartition role、source/license/retention status。
- benchmark definition/results、cost scenarios、tested/untested variants、selection family、全trialの参照。
- machine gates/test results、run status、validation status、evidence class、holdout/prospective status。
- 結果、費用内訳/unknown、limitations、failure reason、reproducibility status、artifact refs/expiry。
- visibility決定、送信先とscope、除外/redaction記録、必要な人間承認の状態。

完全な会話本文を常に保存・共有する必要はない。外部向け記録は、研究を再開/監査するために必要な最小の根拠を含む。公開できないartifactはURI/hash/保存期限だけを記録し、参照先が消えたときは完全再現不能と表示する。

## 3. MarkdownとJSONの契約

Markdownは人間に、JSONは機械処理と照合に使う。双方に同じ`record_id`, `revision`, `manifest_hash`, `generated_at`, `visibility`を含める。JSON schemaはversionを持ち、必須キー欠落、無効enum、manifest不一致があればexportを失敗させる。JSON値とMarkdownの主要数値が一致しない場合、送信可能な成果物として扱わない。

推奨するMarkdownの並びは次のとおり。

1. 現在地と状態（完了/進行中/停止/未開始）。
2. 仮説・対象市場・根拠・反証条件。
3. 実施した工程、検証/Runner状態、次の工程。
4. 凍結条件・ベースライン・コスト/データ境界。
5. 結果とEvidence class、全trialへの索引。
6. レビュー論点・Manager判断・異論。
7. 検証済み事項、未検証/失敗、制約、未解決事項。
8. Artifact参照・再現手順・次の行動。

機械可読JSONの上位key案は以下とする。実装時は正式schemaとmigrationを定義し、ここにない詳細を暗黙必須とはしない。

```json
{
  "schema_version": "1",
  "record_id": "",
  "revision": 1,
  "generated_at": "",
  "visibility": "local_private",
  "requirements_version": "",
  "project": {},
  "research_program": {},
  "hypothesis": {},
  "cycle": {},
  "reviews": [],
  "decision": {},
  "frozen_manifest": {},
  "benchmark": {},
  "runs": [],
  "evidence": {},
  "holdout_and_prospective": {},
  "costs": {},
  "artifacts": [],
  "limitations": [],
  "open_questions": [],
  "next_actions": [],
  "export_audit": {}
}
```

上の例はschema設計用であり、実データ、実行済みRun、実験結果を示すものではない。金融資金、口座、secret、private raw dataを通常のresearch export schemaへ入れない。必要な個人設定は別のローカル保護設定として扱い、公開/外部schemaにフィールドを追加しない。

## 4. HANDOFFの内容

`docs/research/HANDOFF.md`相当の現在地記録には、最低限次を含める。

- 正式なrequirementsの所在・version・承認状態。Draftと採用済みの区別。
- 現在のStage/承認範囲、着手済み/未着手/停止/blockedの状態。
- ユーザーが確定した製品判断と、未決の製品判断。
- 現在のmarket/data scopeと許可範囲。今回のStage 0Cではsynthetic-only、実データvenue/feed/timeframe未確定を明示。
- 最後に完了した実作業と参照可能な証拠。未実施の検証を明記。
- Frozen manifest/registryのIDと状態、Holdoutの露出/未知状態。
- 有効なsandbox/provider/data permission profileと閉じている能力。
- 費用・時間・trialの消費/残量、unknown値、次に必要な承認。
- 具体的な次の作業、合否条件、依存、担当、artifact expiry。
- 再現上の制約、既知不具合、失効した成果物と理由。

HANDOFFは過去の記録を上書きせずrevisionを進める。人間の決定には日時、actor、根拠、対象要件版を紐付ける。AIは未決事項を決定済みとして書き換えない。

## 5. Privacy・外部AI・GitHubの出力制御

### ローカルprivate領域

ユーザーの資金額、口座/取引所情報、保有状況、許容損失、API key/secret、非公開データ、private raw market dataはlocal-private扱いとする。公開レポート、通常ログ、エラー文、commit messageに含めない。資金方針が必要な場合は権限制御されたローカル設定へ分離する。

### 外部AIへ送る内容

外部AIには、そのReviewer/Managerのタスクに必要な最小限の仮説・仕様・集計/feature情報だけを送る。口座、ユーザー資金額/許容損失、credential、private raw dataは送信しない。公開済みまたは適法に送信可能なsource dataも、data-source termsと送信scopeを確認する。利用権限、受信先、送信対象のどれかが不明なら送信を拒否する。

各requestに送信対象の分類、provider/model、目的、データ範囲/manifest、料金見積りと承認状態を結び付ける。Reviewer blind inputには他Reviewerの回答を含めない。拒否/不明時は`blocked_review`等で停止し、内容を他Providerへ自動転送しない。

### GitHub公開allowlist

Public exportはdeny-listだけで判定せず、field単位のallowlistから新しいexport projectionを生成する。既定allowlistは、一般化された設計・protocol、合成fixture/結果、ソフトウェア版・hash、公開可能と確認された方法論、redactedな合否/研究状態である。実験ごとの戦略/市場結果、候補群、非公開data metadata、private findingsは明示的な公開承認なしに含めない。

必ず除外するもの:

- 個人の資金額、資産、口座、取引履歴、損失許容、個別投資方針。
- API keys、tokens、cookies、認証情報、secret reference/path、環境変数値。
- private/raw market data、契約上再配布不可のdata、dataから再構成可能な未許可内容。
- 非公開研究仮説、結果、戦略仕様、レビュー内容、ユーザーや第三者を識別できる情報。

Export処理はallowlist外の値を落とした後にDLP/secret scan、JSON/Markdown間の一致、artifactリンク範囲、git diffの人間レビューを行う。検査不能・分類不明・allowlist不一致は送信拒否とする。値を隠したつもりの単純な文字列置換や、AIによる自己申告だけで安全とみなさない。

GitHubはpublic repository前提とする。専用branch/PRを基本とし、既定branchへ無断push/mergeしない。外部送信は具体的な対象と公開範囲の承認が必要で、研究文書の一般的な承認をデータ・結果・資金情報の一括公開許可に拡張しない。

## 6. 更新、再現性、保持

研究判断、manifest凍結、Run完了、重大な失敗/バグ、Holdout露出、研究方向変更、作業中断の各節目で記録を作成する。追記前に元データ/コードのhash、export scope、visibilityを記録し、変更理由と親record IDを保持する。

保持期限はdata source licenseとユーザー設定に従う。raw dataや成果物を保持できない場合は参照、hash、期限、消失時点、再現可能性への影響を記録する。期限切れを隠さず、データが失われたRunを完全再現可能と表示しない。

## 7. 現時点の引き継ぎ前提

- 現行v2.0開発基準はREQUIREMENTS.mdと要件レビュー記録。受領原本の未承認表記と、今回のStage 0開始判断を区別する。
- ユーザーが指定した最初の実データ優先対象は暗号資産現物（BTC等）。venue/feed/timeframeは未決定。
- Stage 0Aはネットワーク外部接続なしのsynthetic-only。実市場data取得、実Provider送信、アプリからのGit操作、実験の成功主張はしない。開発者による許可済みソース/文書のPushとは区別する。
- Stage 0Aの試験資本は合成/正規化値のみとし、個人の資金額・許容損失を入力しない。
- Stage 1A/1B自律研究は未開放。prospectiveはStage 1で凍結manifestと手動記録契約、Stage 2で自動観測/昇格機能を追加する設計。
- private/raw data、個人資金、口座、secretを外部AI/GitHubへ送らない。許可が不明ならfail closed。
