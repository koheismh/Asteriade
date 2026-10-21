# Asteriade 研究プロトコル設計案

- 状態: **設計案。実装・データ取得・実験実施は未確認**
- 適用対象: [現行v2.0開発基準](REQUIREMENTS.md) の研究サイクル。実装済み範囲はHANDOFFで確認
- 現在の作業境界: Stage 0A は合成データと参照計算だけを使う。Stage 1 の自律研究は未開放。

本書は、StrategySpec、固定検証カーネル、候補系譜/選抜履歴、Holdout、Prospective の研究契約を定める。市場固有のライブ接続や実注文を定義しない。ユーザーが最初の実データ優先市場を「暗号資産の現物（BTCなど）」と指定したが、venue、feed、時間足は未決定である。これらを推測して固定せず、実データを使う段階の前提条件として扱う。

## 1. 証拠と実行状態

次の状態は混同しない。

| 種類 | 意味 |
|---|---|
| Run status | Runnerが開始・終了した事実。例: `queued / preparing / running / succeeded / failed / interrupted / outcome_unknown` |
| Validation status | 出力・入力・検証カーネルの機械的検査。例: `unvalidated / valid / invalid / inconclusive / blocked` |
| Evidence class | 結果の位置付け: `exploratory / independent_historical / prospective` |
| Partition status | データの露出・使用状態。例: `unseen / preflight / evaluation_active / inspected / defect_review / consumed / corrected_analysis / retired` |

Runner成功だけでは検証済み結果にしない。機械ゲート、データ境界、成果物identityが確認されるまで結果は無効または未確定として扱う。優位性の有無は、実験完了・検証成功とは別に判定する。`unknown`、未取得、欠損、未実施をゼロや不利/有利な値に置き換えない。

Stage 0A ではネットワーク外部接続を行わず、合成データ、固定fixture、参照実装だけを扱う。実市場feed、外部データAPI、取引所/ブローカー、実注文、外部AI Provider送信、アプリによるGitHub送信、個人資金額は対象外とする。開発者による許可済みソース/文書のPushとは区別する。Stage 0Aの結果は実市場での性能・精度を示さない。

## 2. StrategySpecと検証カーネル

StrategySpecは仮説から実験可能な売買ルールへの、型付きで版管理された入力とする。最低限、以下を固定する。

- 市場/商品、方向、時間軸、必要な特徴量とその情報利用可能時刻。
- `Signal / Entry / Exit / Position / Sizing / Risk / Execution` の定義と依存関係。
- 変更可能なパラメータ、その型・範囲・探索方式。探索しない要素も明示する。
- 必要データ、注文・ポジション表現、ポジション併存/分割/決済の扱い。
- 仮説の予測、反証条件、事前に決めた評価指標と停止条件。

実行カーネルは、データ時計、注文イベント、約定順、ポジション、現金、費用、証拠金、損益、評価指標を共通に処理する固定版の実装とする。通常のStrategySpecや戦略プラグインから、カーネル内部、未許可データ、環境変数、秘密情報へアクセスできない設計を前提とする。

同じ入力、Spec、カーネル版、設定、seedで結果を再現できることを合成fixtureで確認する。カーネルの変更は戦略試行と分けた科学実装変更として記録し、差分・参照計算・回帰試験を経て新しい版を発行する。変更前後の結果を同一カーネルの結果として混ぜない。既存エンジンを使う場合も、その版、設定、約定仮定を固定して記録する。

## 3. 実験凍結とcanonical manifest

バックテスト開始前に、以下を含む凍結manifestを生成する。

1. Project、ResearchProgram、Hypothesis、ResearchCycle、ExperimentのIDと選抜系譜。
2. StrategySpecのcanonical内容とhash。
3. ベンチマーク仕様・実装hash、比較対象期間、同等化できない条件。
4. Dataset ID、content hash、source/license記録、期間、instrument、timezone/clock、partition role、可用時刻規則。
5. コードcommit/hash、依存環境、検証カーネル名・版・build hash、実行profile。
6. 全設定、固定パラメータ、探索範囲・全候補識別子、探索手段と停止条件。
7. 事前宣言したseed集合。seed追加は新しいmanifest版または別Experimentとして記録する。
8. 指標定義、評価閾値、コスト条件、約定モデル、必要品質/科学ゲートとfixture版。
9. Holdout/Prospectiveの割当、アクセス制御、公開する結果の範囲。

初期の凍結identityは固定フィールド順の型付きJSON（`asteriade/freeze-json-v1`、UTF-8）に限定し、正規化方式の版を含めてSHA-256を計算する。これはRFC 8785互換を主張するものではない。汎用JSONや別言語の実装へ拡張する時点でcanonicalization方式を比較・版更新する。重複key、NaN/Infinity、規則外数値は拒否する。時刻はUTC RFC 3339、金額/数量はスキーマで定める整数または10進文字列とし、実装言語ごとの浮動小数点表現に依存させない。Manifestの凍結後は内容を上書きしない。変更は理由と親manifest hashを付けた新revision/new Experimentを作る。単なる再実行でも新しいRun IDを発行する。

データ期間、benchmark、partition、コード、kernel、metrics、cost assumptions、seed、gatesのいずれかが結果を見た後で変わった場合、従前の実験を成功に読み替えない。変更条件での再実験は新identityであり、同じ最終Holdoutを独立未使用データと呼ばない。

## 4. ベースラインと相対比較

候補結果を見る前に、研究目的に適したベースラインを少なくとも1つ登録し、選定理由と比較可能な範囲を記録する。ベースラインと候補は可能な範囲で同じデータ、時刻規則、費用シナリオ、資金拘束、レバレッジ/リスク量を使う。

適切な比較対象を定められない場合は比較不能として表示し、相対的な優位を確認済みとしない。候補の成績を見てベースラインを差し替えない。比較条件が不一致なら差分とその影響を記録する。

## 5. 共有candidate / selection-family registry

Project内に閉じず、同一・重複する市場、instrument、データ期間、特徴量、または研究目的を共有するResearchProgramの間で、次の履歴を検索可能な共有registryとして扱う。

- Candidate/Hypothesis ID、生成元、親候補、関連/類似候補、派生理由。
- 使用したdata partition/期間/銘柄とそのregistry identity。
- 試行した全variant、seed、パラメータ、失敗/未完了/無効を含むRun結果。
- 選抜family ID、選抜した時点、選抜集合、選抜基準、後段へ渡した候補と棄却理由。
- Project間の再利用・変更・結果参照のeventと、受け取った結果の範囲。

パラメータ変更、失敗後の再設計、別AIによる再提案、市場/足種の移動を独立な初回仮説として数え直さない。どの集合から何が選ばれたか後から復元できるようにする。類似性の自動判定は候補探索の補助であり、検出できないことを独立性の証明に使わない。

## 6. 共有Holdout registryと露出防止

Holdoutの割当と露出履歴も全Project/ResearchProgramで共有する。Registryは少なくともDataset ID/hash、source/version、instrument、期間、partition role、割当・開始・終了時刻、Experiment/manifest hash、アクセス主体、閲覧可能だった出力、結果の意思決定への露出、バグ/修正、現在のpartition statusを記録する。

Preflight時にデータidentity/期間の重複・部分重複を照会する。照会結果が不明、registryが欠落、復元したDBに履歴がない、partition対応を検出できない場合は`unseen`や独立確認として表示しない。Holdoutを使用する独立評価を停止し、`exposure_unknown`相当の要対応状態にする。

値へのアクセスを開始したら、結果表示が空でも開始・閲覧・エラー・中断の事実を追記する。結果や集計値をAI/人が見た後に、単なるDB復元、履歴削除、partition作り直しで`unseen`へ戻さない。アクセス前のpreflight失敗だけは、値/結果が一切読み出されなかった証拠がある場合に限り、その証拠を記録して未使用状態を維持できる。

科学実装バグの発見後は、露出した値、意思決定への影響、修正差分、再実行の必要性をレビューし、`corrected_analysis / contaminated / inconclusive / prospective_pending`等へ分類する。インフラ再試行、結果に影響する科学バグ修正、結果に合わせた戦略変更を分離する。新しいデータがなければDevelopmentの研究は続けてもよいが、独立最終確認の証拠クラスへ昇格させない。

複数検定・PBO/CSCV・Deflated Sharpe等は、標本数、粒度、候補依存、研究目的を踏まえて適用を選ぶ。前提が満たされない場合は補正済みと主張せず、使った方法/使わなかった理由/限界を保存する。

## 7. 有限のReviewer討論

標準Cycleは最大5段階とする: (1) Managerの仮説・計画提示、(2) Reviewerの独立初回反論、(3) Managerの指摘別回答、(4) 必要時に限るReviewerの再反論1回、(5) Managerの最終判断。再反論省略も理由とともに記録する。

各Reviewerは独立したblind Sessionで初回評価する。初回入力に他Reviewerの回答や他者の結論を含めない。これは情報隔離上の要件であり、モデル間の統計的独立性や認知的独立性を保証しない。UI/レポートで統計的独立性を主張しない。Reviewerが失敗/未設定/期限超過の場合は割当済みレビューを完了扱いにせず、事前許可された代替か`blocked_review`で停止する。

Managerは各指摘を採用/一部採用/却下/実験で確認に分類し、最終判断と理由を残す。Managerは、データ漏洩、科学ゲート、秘密/アクセス制御、権限、予算ゲートを上書きできない。最大発言回数はOrchestrator側で制御し、総合意・無期限討論を要求しない。

## 8. A/B/C方式比較の共通ゲート

方法比較の3群すべてに、同一の必須安全ゲート・科学ゲート・データ境界・Run上限・事前定義評価手順を適用する。A群だけ検証を免除する設計にはしない。

必須ゲートは必要な実行前・実行中の時点でも共通に強制し、各群の助言条件が終わった後も同じ機械評価器で全出力を検査する。A群の「feedbackなし」は、Manager提案時に助言的機械診断を入力として渡さない意味に限る。必須のデータ境界、カーネル検査、Leak/会計/安全確認は全群で実施し、違反結果を有効な実験として比較しない。

| 群 | 提案段階の条件 | 全群共通の条件 |
|---|---|---|
| A | Managerが機械診断の助言的feedbackを受けずに仮説/StrategySpecを提案 | 同じ実行前・中・後の機械的安全/科学ゲートを実施し、未達は無効/停止 |
| B | Managerが同じ種類の機械診断feedbackを受けてから提案を確定 | Aと同じゲート、データ、費用・Run上限、評価規則 |
| C | Bと同じ機械診断feedbackに加え、独立Reviewerの討論を行う | A/Bと同じゲート、データ、費用・Run上限、評価規則 |

比較課題、入力、データ境界、budget、trial数、評価者/採点基準、実施順を事前登録する。条件差・脱落・介入を記録する。結果を見た後の群定義変更は禁止する。小標本や課題難易度差が大きい場合は優劣を断言しない。記録する指標には有効実験率、既知欠陥の検出、研究時間、モデル/計算/データ費用、人手介入、独立評価成績を含める。

## 9. Prospectiveの凍結・観測契約

Stage 1は、実市場入力の利用許可・接続安全性・市場仕様を別ゲートで満たした後に限り、Prospective開始前の凍結manifest、ルール/データfeed identityの保存、時刻付きの手動観測記録を開始できる。Stage 1は自動feed接続、連続監視、証拠クラスの自動昇格を必須にしない。現在のStage 0Aは合成データのみのため、このProspective契約は設計のみで実データ記録を開始しない。

Stage 1の手動記録は、凍結後に受領したシグナル、対象instrument、観測時刻/利用可能時刻、使用した価格/データ参照、仮想約定仮定、欠損・無取引・追跡停止を追記専用で保存する。過去データを後から埋めてProspectiveと呼ばない。記録の完全性が確認できない場合は`inconclusive`または`prospective_pending`とする。

Stage 2で自動観測を追加する場合、data-feed identity、取得時計、欠損検知、連続性、再接続/重複排除、停止/復旧、監査を別途検証する。自動的なEvidence promotionは事前登録済みの観測期間・件数・品質/科学ゲートを全て満たす場合のみ許す。将来収益を保証せず、期待観測数未達と否定的結果を区別する。

実注文、取引所/ブローカー口座、秘密鍵、資金移動はこのプロトコルの対象外であり、別途正式要件・安全レビュー・明示承認なしに追加しない。

## 10. データ許可と取込

Raw responseを保持できるのは、source terms/licenseが複製・保管を明示的に許す場合に限る。保存権利がない、条件が曖昧、または再配布/保持条件を確認できないfeedは取込不可とする。許諾範囲で必要に応じてmetadata、source参照、取得時刻、検証可能なhashのみ保存するが、それをRaw dataの代替や完全再現可能性の証明と呼ばない。

利用目的、保持期限、ローカル/非公開保管先、外部AIへ渡す要約範囲、GitHub公開可否をsource単位で記録する。許可が不明なら取得・外部送信を停止する。Stage 0Aはこれらの照会・合成fixtureの設計に限り、market feedへの接続はしない。
