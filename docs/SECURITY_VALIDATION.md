# Asteriade セキュリティ・研究検証計画

更新日: 2026-10-21  
状態: Stage 0 用の検証計画。Windows 実機検証と研究カーネルの実証は未完了。安全性の合格証明ではない。

## 1. 基準と現在許される範囲

本計画は[現行v2.0開発基準](REQUIREMENTS.md)と[レビュー結果](research/REQUIREMENTS_REVIEW_2026-10-21.md)に基づき、v1.1から引き継ぐWindows・Runner・保存の安全境界を研究向けに補う。Stage 0を開始するが安全ゲートの合格は未達である。

現在の実装証跡は [STAGE0_STATUS.md](STAGE0_STATUS.md) の記録に限る。そこではローカル API の loopback 制限と SQLite の保存・復旧試験を報告している。一方、Runner は全能力 `false`・`unverified` で、起動 API は拒否する。Windows ACL、Job、Firewall/WFP、Codex 認証と子孫プロセスの分離は未検証である。Linux テスト、Windows クロスビルド、mock、モデルやCLIの readiness は、Windows サンドボックスや実市場会計の証明にしない。

このStage 0の限定実装は、既存StrategySpec/freezeに加えて開発専用の合成時刻・現物会計・失敗検査を対象にできる。Python標準ライブラリの参照例はアプリ/Runnerから未接続であり、OSによるデータ隔離や市場仕様の実証ではない。[実測と限界](research/STAGE0C_SYNTHETIC_VALIDATION_2026-10-21.md)を参照。外部データ取得、Provider/AI 接続、ネットワーク送信、一般プロセス起動、バックテスト Runner 起動、アプリのGitHub送信、購入・課金は行わない。開発者による許可済みソース/文書のPushとは区別する。実注文、資金移動、口座変更、取引所・ブローカー連携、秘密鍵/API key の受け渡しは初期版に実装しない。

## 2. 保護対象と fail closed 規則

| 対象 | 境界 |
|---|---|
| Control Plane / SQLite | 研究状態、凍結済み仕様、候補系譜、費用状態、アクセス記録を管理する。未検証 Worker へ保存先や管理認証を渡さない。 |
| Provider Host | 将来の公式 Provider 認証を扱う専用側。秘密の値をLLM入力、Reviewer、Runner、Artifact、通常ログへ含めない。Host/Runner 分離が実証できなければ Provider を使う研究実行を閉じる。 |
| Runner / 研究コード | 未信頼コードとして扱う。実効的なファイル・プロセス・通信制約を Windows で証明する前は起動しない。 |
| Research / dataset identity | StrategySpec、コード、カーネル、データセット・分割、feed vintage、availability time、選抜系譜を識別する。記録が欠ける場合に `unseen` や独立確認へ昇格しない。 |
| Holdout | Final Holdout 値をResearch AI/GAへ自由に見せない。アクセス・結果公開・修正・再実行を記録し、露出または来歴不明なら `consumed/unknown/contaminated/inconclusive` とする。状態を未使用へ巻き戻さない。 |
| Private fields | 個人の投資額・口座情報・非公開研究データ・秘密は公開 GitHub へ送らない。外部 Provider への送信可否が不明な private field は送信前に停止し、既定は local-only / 最小化・秘匿化とする。 |
| Market feed | Feed の利用規約と raw 保存・再配布権が確認できるまで取得・保存・変換しない。権利不明の feed は取り込まない。現段階は合成データのみ。 |
| Live orders | 注文送信 capability、注文サービス、exchange/broker 資格情報を提供しない。将来の別要件・別安全審査なしに注文機能へ拡張しない。 |

権限、Provider 利用条件、料金、feed 権利、Holdout の連続した履歴、Runner の境界が不明なら該当する処理を止める。安全な合成データ上の設計・ローカル metadata 試験だけ続行可能とする。

## 3. 脅威範囲と境界候補

管理者・OS 所有者による完全な端末侵害、kernel 侵害、正当に許可された管理者の改変を防ぐとは主張しない。モデル出力、研究コード、リポジトリ指示、外部データ、成果物は信頼しない。

Windows 11 の候補は、専用標準ユーザー/プロファイルと NTFS ACL、Restricted Token、Job Object、AppContainer、Firewall/WFP、Codex の公式 Windows sandbox、必要なら Sandbox/VM の組合せである。Job 単体はファイル・通信を隔離せず、Restricted Token 単体は通常ユーザーの既存 ACL 権限を消さない。具体 backend の採用は子孫・資格情報・ネットワークを含む実測後に限る。

Microsoft の [AssignProcessToJobObject](https://learn.microsoft.com/en-us/windows/win32/api/jobapi2/nf-jobapi2-assignprocesstojobobject) 公開ソースでは、Job 継承に breakaway 条件がある。資料中の割当前メモリ操作の記述はメモリ制限に関するものとして扱う。設計上、後付けの Job 割当てを起動前の隔離や既発生の副作用の巻き戻しとして頼らず、起動窓を実測する。[CreateRestrictedToken](https://learn.microsoft.com/en-us/windows/win32/api/securitybaseapi/nf-securitybaseapi-createrestrictedtoken) はSID/特権等を制限できるが、資格情報アクセス全般を保証しない。

Codex App Server の Windows backend、承認、認証、子プロセス、通信について、README の readiness 表示だけで実効安全性を判定しない。App Server/Host の Provider 通信と、そこから起動する任意ツールの通信・資格情報を区別できるかは未検証である。

## 4. Stage と許可能力

| Stage/状態 | 許される範囲 | 開始条件・失敗時 |
|---|---|---|
| 現在の Stage 0 限定実装 | 最小 StrategySpec、freeze identity と SQLite 保存・照合。process start、Provider/AI、network、feed import、external write は不可。 | identity 変更や欠落を検出できない場合は凍結/再開を拒否。Runner は全能力 false のまま。 |
| Stage 0 研究基盤の後続技術検証 | synthetic data、known-answer reference、会計/kernel テスト、候補/Holdout metadata、境界試験を検証計画として準備する。 | Stage 0の計画範囲であり、今回は最小Spec契約から開始する。各試験の前提と実行環境を整えてから進める。実市場データは対象 market/feed/timeframe と feed 条件・費用が確認されるまで取得しない。Windowsの実効境界未達なら Runner を閉じる。 |
| Stage 1A/1B 研究 MVP | 1Aは1市場・単一Manager・機械ゲート、1Bで有限討論を追加。共通で凍結、検証、バックテスト、benchmark、記録。実取引なし。Prospective は凍結した仮想シグナルを手動/低負荷で追記記録し、状態を pending と表示可能。 | 必要profileのWindows試験と機械ゲートが合格するまで execution不可。Prospectiveが未完了でも MVP は `pending/inconclusive` を明示できるが、確認済みに昇格不可。 |
| Stage 2 | 自動Prospective追跡と完全な証拠昇格、研究横断の高度な診断・効率評価。 | 新たな実行経路、予算、データ利用を別途評価。前向き期間の短さを成功とみなさない。 |
| `BLOCKED` | 診断・設計閲覧・既存のローカル保存だけ。 | 必須境界、データ来歴または許可状態が不明。model/tool/Runner execution 不可。 |
| `VERIFIED(profile)` | 試験済みの特定 profile/capability のみ。 | 陽性・否定試験、実効設定、鮮度と証跡が揃う。別Windows版/構成への一般化不可。 |

## 5. 試験環境と証跡

Windows 試験を開始する段階では、使い捨て Windows 11 x64 環境、build/edition、実行ユーザーと SID、管理操作、Go/Codex/Runner版とhash、profile fingerprint を記録する。実データや実注文先ではなく、使い捨て Workspace、合成価格、ダミー秘密、管理下 sink を使う。禁止操作試験では、fixture が壊れているだけの拒否を合格にしないため陽性対照も用いる。

各結果は `test_id, run_id, stage, result(passed/failed/blocked/skipped/unrun), time, OS/build, tool versions/hashes, profile fingerprint, identity/privilege summary, policy revision, fixture, expected/observed, exit code/OS error, evidence refs` を記録する。秘密値、私的な研究データ、Holdout 値そのものは証跡へコピーしない。ログ、SQLite、Artifact、backup の暗号化・保持・削除は現時点で実装済みと主張しない。

Identity は StrategySpec schema/hash、code SHA、kernel/version/hash、synthetic dataset ID/hash、partition role、feed vintage/availability time（実feed利用時）、parameter space、benchmark、acceptance criteria、seed を束ねる。freeze 後の変更は新しい identity/candidate とし、元記録を上書きしない。過去研究 import、backup 復元、registry 移行で選抜/露出履歴の連続性を証明できない場合、候補/partition は `unknown` 扱いとし、未使用や独立確認へ昇格しない。

## 6. 受け入れ試験

以下は計画であり、Windows sandbox、Provider、market feed、研究 kernel の試験はまだ実施されていない。Stage 欄は実際の実行許可ではなく、依存を示す。
今回の実装増分はStage 0Cの小規模合成参照例に限定し、既存Spec envelope/freezeおよびRunner全能力falseを維持する。残りの受入試験は未実施として `unrun` を維持し、Stage 0の計画に沿って順次進める。試験計画を実施済み証拠に読み替えない。実データ・課金・非公開情報の外部送信・権限設定変更にはそれぞれ具体的な前提/許可が必要。

| ID / Stage | 試験 | 合格条件 / 失敗時 |
|---|---|---|
| SV-01 / 0 Spec | 最小 StrategySpec を保存・凍結・再起動後に照合。値・parameter space・schema を変更する。 | hash mismatch で旧freezeを再利用しない。変更は新identityとし、旧成果/証拠クラスとの関係を保持。テストはローカル保存だけで完結。 |
| SV-02 / 0 scope | 現行限定ビルドで Provider/AI client、feed fetch/import、一般 process/Runner、network egress、external write 呼出し経路を試みる。 | 呼び出し機能/credential/capability がなく、model/Runner/API/networkへの送信0。内部の拒否試験またはstubで示し、外部先へ試験リクエストしない。 |
| SV-03 / 0 order prohibition | 注文、取消、送金、口座変更を要求する架空tool call/APIを投入。 | 対応API/service/credentialが存在せず、Policy/Runnerが拒否。実取引所へ接続せず、network sink受信0。sandboxの外部送信未達でもorder capabilityは追加しない。 |
| SV-04 / 0 privacy | synthetic markerで個人投資額、口座、private feed、credentialを模擬し、Prompt/Runner/Artifact/log/export/GitHub対象へ混入させる。 | private field は既定でlocal-only。Provider送信可否が不明なら送信前に停止。secret値はLLM/Reviewer/Runner/Artifact/logへ出ない。public exportは明示許可と内容検査なしに行わない。 |
| SV-05 / 0 data rights | feed権利・raw保存・再配布可否不明、または保存不可のdatasetを指定。 | fetch/import/raw retention/feature化を開始せず、停止理由を記録。最初はsynthetic only。権利のないデータをGitHubやProviderへ送らない。 |
| SV-06 / 0 identity continuity | freeze 後のSpec/code/kernel/data/acceptance変更、欠損したmanifest、backup/importから履歴の欠落、未知のpartitionを投入。 | 新identity/`unknown`/`inconclusive`へ移し、旧Holdoutをunseen/independentと再表示しない。履歴が回復できなければ独立昇格不可。 |
| SV-07 / 0–1 lineage | 複数候補、variant、派生、失敗、未採用、基準変更、同じ市場/データ期間を使う別ResearchProgramを投入。 | 親子・選抜元候補集合・全試行・不採用理由・使ったデータ/metricを追跡可能。欠落した系譜は多重選抜補正済みと表示しない。 |
| SV-08 / 0–1 Holdout preflight | holdout preflight失敗、値要求、結果公開、再実行要求、別Projectから同一partition/重複区間へのアクセスを投入。 | 値/結果がResearch AI/GAへ出ず、アクセス記録とpartition stateが残る。重複アクセスは共通registryで露出扱い。stateを巻き戻さない。未実装/未証明ならFinal Holdout実行と独立確認表示を無効化。 |
| SV-09 / 後続0–1 scientific gate | (当該Stage 0の前提を整えた後) future leak、誤ったfill order、費用/建玉/分割決済のbugをsynthetic fixtureに注入。Managerが続行/昇格を要求。 | 機械検査が重大欠陥を拒否し、Managerで上書き不能。結果はinvalid/contaminated等に分類し、必要な回帰fixtureを残す。AI review単独合格は不可。 |
| SV-10 / 後続0 kernel synthetic | (当該Stage 0の前提を整えた後) known-answer reference と synthetic OHLC/eventを使い、同一Spec/kernel/version/seedを再実行。 | deterministic結果、約定順、fees、positions、PnL、missing/timezoneの期待値と一致し、差異を記録。これは合成検証のみで市場feed/実約定の正確性を証明しない。 |
| SV-11 / 0 market accounting | crypto spot のvenue/feed/timeframeが決定後、利用条件を確認した履歴でavailability time、欠損、修正、費用、約定を比較。 | 条件/許容誤差を事前に固定し、referenceとの差を説明。ユーザーがvenue/feed/timeframeを選ぶまで`blocked/unrun`。synthetic passを実市場会計passへ転用しない。 |
| SV-12 / 1 benchmark | 研究前登録した基準戦略と同期間・同kernel・明示費用条件で比較。比較条件を結果確認後に差し替える。 | ベンチ未設定/比較不能を表示し、相対優位と断言しない。risk/cost/capital差を明示。最低1 benchmarkをStage1前に事前固定。 |
| SV-13 / 1–2 Prospective | freeze後に生じるデータのシグナル/仮想約定、no-trade、欠損、停止、bug修正を追記。freeze前の過去期間を追加。 | 過去の遡及埋め込みはProspectiveでない。変更後は新identity、既存系列は汚染/修正扱い。Stage1は手動/低負荷記録と`pending`表示、Stage2で自動追跡/昇格。実注文なし。 |
| SV-14 / 1B–2 reviewer and benchmark comparison | A/B/C方式を同等課題・制約・data boundary・予算・試行数で比較。Reviewer初回に他Reviewer応答を混入、無限再反論、usage欠落を試す。 | Reviewerは最大2発言機会、機械ゲートは全方式で同じ。品質/欠陥検出/費用/時間を記録し、小標本・不一致条件では優劣を断言しない。model identityやSessionの分離を記録するが統計的独立を主張しない。高度な評価はStage2。 |
| SV-15 / 0–2 cost | 未知の料金/usage、予約超過、provider価格変更、data fee不明、欠落usage、再送を投入。 | 不明額を0にせず、超過/未見積の新規有料実行を止め、project/experiment別に推定/予約/確定を区別。月3,000円は初期上限目標であり、provider側請求総額保証とは表示しない。 |
| SV-16 / 0 Windows file boundary | (Windows実機ゲート後のみ) Workspace外、別Task、DB、secret storeへのread/write/delete、junction/symlink/reparse/ADS/UNC、競合renameをWorker/Codex子孫から試す。 | OS ACL/identity/handle boundaryで拒否し、陽性対照はWorkspace内のみ成功。fixture hash不変。失敗なら該当Runner能力を閉じる。 |
| SV-17 / 0 Windows process boundary | (Windows試験構成を整えた後) child/grandchild、PowerShell/別exe、Job breakaway、startup assignment race、inherited handles、helper/service/COM/RPC、終了競合を試す。 | 不許可process/媒介操作を拒否し、未管理の子孫が残らない。Job membershipだけでsystem-mediated action拒否を代替しない。失敗ならprocess_execを無効化。 |
| SV-18 / 0 Windows network boundary | (管理下sinkとWindows試験構成を整えた後) direct socket、IPv4/6、DNS/UDP、proxy、loopback/LAN、curl/Git/Browser、child processから管理sinkへ送る。 | 未許可送信をOS層で拒否し、sink受信0。Provider Host経路は将来の別陽性対照で、子孫へ権限が漏れないこと。現段階では外部通信自体を行わない。失敗時はnetwork/external_writeを閉じる。 |
| SV-19 / 0 Host secret and auth | (将来Provider検証時のみ) Host専用のdummy authでProvider動作を確認し、LLM/Reviewer/Runner/Artifact/logにsecret markerを探索する。 | Host以外にsecret値が現れず、Host/Worker同一identity等で分離不能ならProvider連携を停止。認証・料金・利用条件不明の時は代替有料Providerを使わない。今のStage0はProvider呼出しなし。 |
| SV-20 / 0 IPC | (Windows Runner実証時のみ) pipe先行作成/squatting、別SID/session、remote client、古いExecution secretのreplay、巨大/不正frameを試す。 | 認証前の副作用と管理APIアクセス0。正規Runner/Executionだけ通す。IPC未検証ならRunnerを起動しない。 |
| SV-21 / 0–1 recovery | DB/registry backup、外部既存研究import、強制終了、途中のholdout access/journal欠落を試す。 | registry continuityが証明できないpartition/candidateは`unknown/inconclusive`。unseen/independentへ昇格せず、元記録を上書きしない。復旧完了前のRunner起動なし。 |

## 7. Stage Gate 判定

- **段階化:** Stage 1Aは単一Manager＋機械ゲート、Stage 1Bで独立Reviewerを追加する。ゲートは同一で、Reviewerの未実装を許可/安全性の代替にしない。
- **現在:** Runner は `BLOCKED/unverified`。実行開始 API の拒否を維持。現行の loopback/SQLite 証跡をWindows sandboxの証拠へ拡張しない。
- **限定 Spec 段階:** StrategySpec/frozen identityのローカル保存テストだけを行い、Provider、AI、data feed、ネットワーク、Runner/process、order capabilityを追加しない。
- **Windows境界未達:** 一般 process起動、バックテスト、外部通信、Holdout評価を許可しない。承認画面や人の監督だけで合格にしない。
- **実市場データ未選定/権利未確認:** 合成データのみ。ユーザーがcrypto spotのvenue/feed/timeframeを選ぶまで取得しない。feed条件・価格が不明なら取込と課金を止める。
- **Holdout来歴欠落/露出:** Holdoutによる独立確認を閉じ、研究を `exploratory / unknown / contaminated / inconclusive` のいずれかに留める。別Projectや既存研究をimportして系譜が不明な場合も同じ。
- **kernel欠陥/検査未実施:** 結果をvalidated/promotedへ昇格しない。修正は新kernel/spec identityと回帰検証を要する。
- **外部/private data送信・order:** private fieldの許可不明時はProviderへ送らない。初期版には実注文、取引API key、口座連携、注文サービスを設けない。Providerの正当なHost認証とは区別する。
- **Prospective不足:** `pending/inconclusive` として表示し、所定の将来観測が揃うまで確認済みとしない。Prospectiveは仮想シグナル記録であり注文を伴わない。

## 8. 後段で決める事項

現時点でユーザーが指定した優先市場は暗号資産の現物（BTC等）。venue/feed/timeframe は未定で、当面は合成データのみである。市場固有のavailability time、費用、約定、許容誤差はvenue/feed/timeframe決定後に根拠を示して固定する。raw保存権がない/不明なfeedの取得は行わない。

PBO/Deflated Sharpe/Reality Check等の採否、selection familyの統計補正、bench課題数・優劣の閾値、Prospectiveの追跡期間と昇格条件、実データ費用は対象データと標本を見て選ぶ。どの統計手法も全状況で適用可能とはしない。未定は推定・成功ゼロ・優位性確認へ置き換えず、限界を表示する。

この文書の手順は試験計画であり、未実施ケースの合格を主張しない。Stage 0の限定Spec実装・保存試験とWindows実機のsandbox試験は別の証跡として記録する。Stage 1の実行は必要なWindows profileの検証と別途の承認後に限る。
