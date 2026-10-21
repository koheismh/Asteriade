# BTC現物・初期市場プロファイル選定調査（2026-10-22）

**状態：公式サイトの通信制限により調査途中。API仕様の比較と条件付き推奨を保存した。規約・料金の確認前なので正式選定・取得開始の判断は保留。** Stage 0全体の合格、Stage 1移行を宣言しない。

## 結論と対象

比較候補を **bitbank BTC/JPY現物** と **GMOコイン BTC/JPY現物候補** の2件に絞る。旧quant-btc-labとの連続性があるGMOと、固定版の公式公開仕様で現物ペア・ローソク足・注文/feeメタデータの契約を確認できるbitbankを比べれば、今回の判断材料を得られる。海外venue・別quoteをさらに増やす前に、この2件の権利と履歴条件を確認する。日本語やタイムゾーンから利用者の居住国・サービス適格性を推定しない。

**暫定推奨はbitbank / BTC-JPY現物 / 1時間足 / 単一の公式public OHLCV source。** 初期の検証表現は借入なしlong-only、確定足由来の判断、availability後の適格な足のopenを参照する成行相当のシミュレーション、明示数量・現金・fee会計とする。理由は、確認できた公式仕様で `btc_jpy`、`1hour`日次要求、spot注文/fee/精度メタデータの参照先を説明でき、旧研究のGMO固有時計と仮定を無条件に持ち込まず小さなprofileを定義できるため。**料金や流動性が優れていると実証した順位ではない。** GMOの現行仕様・権利・料金が確認できれば順位は再評価する。現時点で取得可能・適法に保管可能と認定した候補はない。

ユーザーの今回の依頼を1市場・1ペア・**1時間足**の調査範囲として扱い、複数時間足の同時導入や新エンジンの実装はしない。利用者の選択事項と技術的な決定事項は後掲する。

## Draft PR #3と開始状態

- PR: <https://github.com/koheismh/Asteriade/pull/3>。GitHubの公開HTMLで **Draft**、base `codex/investment-research-stage0`、head `codex/lean-mvp-synthetic-stage0c` と本文を確認した。Gitの `refs/pull/3/head` は `e77638eb644df7791f1997c2f8a84ad7970bb21d`、前回Pushと一致。
- PR本文は1A/1B段階化・24 Python test・40 Goトップレベル/105 subtest込みの既報、Python参照の製品未接続、Windows/市場/権利/Holdout未検証を明示している。今回コードを変更しないのでテストを再実行していない。これらの件数は前回の記録で、今回の新規実測ではない。
- 公開HTMLのreview欄は`No reviews`だが、動的領域に読み込みエラーがあり、認証付きレビュー・コメント全件の取得は未確認。APIは `api.github.com/graphql` への接続がForbidden。本文とGit差分を確認したことを、PR承認・CI独立再検証・全コメント読了と混同しない。
- 専用作業ブランチ `codex/btc-spot-profile-research` をPR #3 headから作成。元の開発ブランチはfetch時 `f721104b6adc6880e6d4bd8671cf78bc4aa5ff6c`。PR #2/#3をマージ・クローズ・編集せず、masterにも反映しない。
- [PR #2に基づくスコープ方針](MVP_SCOPE_REVISION_2026-10-21.md)、[0C判断メモ](STAGE0C_DECISIONS.md)、[合成参照の実測/限界](STAGE0C_SYNTHETIC_VALIDATION_2026-10-21.md)、[実装計画](../IMPLEMENTATION_PLAN.md)に従う。要件正本を選定済みへ書き換えない。

## 根拠の取得状況

日付は2026-10-22（Asia/Tokyo）。確認済みは読めた文書の記載だけであり、実市場データの品質・連続性・取得可否を検証していない。

| ID | 資料 | 状態 |
|---|---|---|
| B1 | [bitbank公式API文書repo](https://github.com/bitbankinc/bitbank-api-docs/tree/0badd68019646171826625b074cfef4235c3e713) | 文書repoのみ取得。HEAD `0badd68019646171826625b074cfef4235c3e713`、記録上のcommit時刻2026-09-11。ライセンスファイルは当該treeにない。文書閲覧やソース公開は市場データの利用許諾を意味しない |
| B2 | [pairs.md](https://github.com/bitbankinc/bitbank-api-docs/blob/0badd68019646171826625b074cfef4235c3e713/pairs.md#L15) | `btc_jpy`、base `btc`、quote `jpy`、delisted=falseの記載を確認。実際の現在稼働状態は未照会 |
| B3 | [public-api.md: Candlestick](https://github.com/bitbankinc/bitbank-api-docs/blob/0badd68019646171826625b074cfef4235c3e713/public-api.md#L293) | 粒度・要求形式・OHLCV/timestamp定義を確認 |
| B4 | [rest-api.md: 注文](https://github.com/bitbankinc/bitbank-api-docs/blob/0badd68019646171826625b074cfef4235c3e713/rest-api.md#L363)、[約定fee](https://github.com/bitbankinc/bitbank-api-docs/blob/0badd68019646171826625b074cfef4235c3e713/rest-api.md#L946)、[pair設定](https://github.com/bitbankinc/bitbank-api-docs/blob/0badd68019646171826625b074cfef4235c3e713/rest-api.md#L1653) | API仕様だけを閲覧。例示curl・API呼出しは実行していない |
| B5 | [bitbank公式サイト](https://bitbank.cc/)、[サーキットブレーカー制度](https://bitbank.cc/docs/circuit-breaker-mode/) | 未閲覧。公式サイトから現行手数料表・利用規約・API/情報利用条件を辿り、適用日/条項を確認する必要がある。具体的な現行条項は未取得 |
| G1 | [GMO公式API文書](https://api.coin.z.com/docs/)、[公式サイト](https://coin.z.com/) | API文書への通常HTTPSがproxy CONNECT 403。文書本文を読めておらず、現行の履歴仕様・注文・fee/規約は未確認。データAPIへは接続していない |
| Q1 | [旧研究のMARKET_DATA.md](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/docs/MARKET_DATA.md)、[market/client.py](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/market/client.py) | 固定 `9a0aee09965da55b5050af7a981ace3f5eec8dc7` を静的読取。過去の研究実装/記録であり、取引所の現行仕様の一次証明ではない |
| Q2 | [Stage 0B比較](STAGE0B_REUSE_EVALUATION_2026-10-21.md)、[科学的不具合](STAGE0B_SCIENTIFIC_DEFECTS_2026-10-21.md) | 旧資産の制約と訂正内容を再確認。旧raw/Parquet/CI数値artifactは取得していない |

この通信制約のため、**保管権・現行料金・GMOの公式仕様を含む最終比較は未完了**。既存の`api.github.com`許可案を保ち、`api.coin.z.com`、`coin.z.com`、`bitbank.cc`、`www.bitbank.cc`を環境設定下書きに追加した。設定保存は実行環境への適用・公開ではない。既定proxy/TLSを維持し、未許可先を迂回しない。

## 2候補の比較

凡例：**公式確認**＝固定版の公式文書、**旧研究のみ**＝Q1/Q2の記録、**未確認**＝今回根拠を取得できない項目。空欄や未確認をゼロ・無制限・許諾済みへ置換しない。

| 比較項目 | bitbank BTC/JPY現物 | GMOコイン BTC/JPY現物候補 |
|---|---|---|
| 市場/ペア | **公式確認** B2の`btc_jpy`。marginの設定/注文も同じ文書にあるが、今回使うのは借入なし現物だけ | **旧研究のみ** 主系列`BTC`、副系列`BTC_JPY`。文字列だけで同じ商品と同定しない。現行公式資料で現物/レバレッジ、参照価格の出所とJPY quoteを確認するまで未確定 |
| OHLCV粒度・要求単位 | **公式確認** B3は1min/5min/15min/30min/1hour/4hour/8hour/12hour/1day/1week/1month。`1hour`は **YYYYMMDD指定**（日付単位の要求）。暦日全体の網羅や常に24本の返却は仕様で保証されていない。4hour以上はYYYY。例示パスは文書であり呼び出していない | **旧研究のみ** 1hour KLineをYYYYMMDD日次要求。GMO固有のJST06境界を使用。現行公式契約として未確認。他venueへこの境界を流用しない |
| 過去履歴の期間 | B3に日付指定契約はあるが、**最古日・何年分の保証・保持期限の明記は確認できない**。年/月/日の指定ができることと、その全期間を取得可能なことは別 | Q1は研究開始を**2021-04-15T00:00:00Z**に設定。これは研究上の開始日であり取引所の提供開始日・現行最古履歴保証ではない。現行保証・連続性は未確認 |
| 時刻・確定足 | B3のOHLCV末尾はUnix ms、responseのtimestampはpublished-atと記載。ただし足start/end・日付bucketのtimezone・revision・確定の条件は確認できない。response時刻を各足の当時の公開時刻へコピーしない | Q1はopenTimeを正本、`[open,end)`、理論availability=end、実fetched_atを分離。現在の確定/訂正/公開遅延は未確認。実受領を理論時刻で代用しない |
| 細粒度の将来拡張 | B3には日付指定transactionsと現在のdepth/tickerもある。履歴tradesの最古日・過去の板/BBO・完全性は未保証。**今回取得しない** | 現行公式のtrades/板履歴範囲は未確認。旧OHLCデータだけから板やspreadを生成しない |
| 注文表現 | B4にspot注文、buy/sell、market/limit/stop/stop_limit等、post_only、部分約定状態の仕様。文書にはmargin専用要素もあるため、全typeをBTC現物へ無条件適用しない。circuit-breaker中はmarket制限、post_only扱いも変わる | 現行の現物market/limit、注文上限、停止/特殊取引の仕様は未確認。旧研究の次足openは取引所の注文/約定仕様ではない |
| 価格・数量条件 | B4に`unit_amount`（最小注文量）、market/limit最大量、price/amount digits、停止flagの参照フィールド。ただし実値・quantity step/tick・丸め規則は未確認。最小量や桁数だけからstepを推測しない | 現行の最小量/tick/lot/上限/丸め/最小代金は未確認 |
| 明示手数料 | B4はbase/quote別maker/taker rateと約定fee_amount_base/quoteを定義。**BTCJPYの実率・適用日・徴収通貨・キャンペーン/段階条件は未確認**。publicなpairメタデータAPIも今回は呼ばない。fee_typeにはDYNAMIC等の制度もあり固定率だけと断定しない | 現行fee表・tier・通貨・rebate・丸めは未確認。旧20bps/short financingを料金表の代用にしない |
| spread/slippage/impact | 1hour OHLCVだけでは当時のBBO、板、queue/遅延を復元できない。足のhigh-lowはspreadではない。文書だけで通常値・分布を推定できない | 同じ限界。旧固定総費用はGMO現物の基準spread/滑りを証明しない |
| 保管・AI送信・公開権 | 規約未閲覧で**3権利とも未確認**。公開API/公式GitHubがあることはRawの長期保管・再配布許可ではない | 規約未閲覧で**3権利とも未確認**。旧研究が保存した事実はAsteriadeの保存許諾を証明しない |
| Windows組込み | 文書上はHTTPS/JSONで、専用SDK・Dockerを必須にしない収集adapterを設計可能。floatへ丸めず文字列から扱える。Go CP＋独立Python Runnerも維持。実装/Windows実証は未実施 | 旧PythonクライアントのHTTP/JSON・checkpoint設計は参考。新規依存なしのadapter候補だが、現行schema確認・権利/配布・Windows実証は別途必要 |
| 初期選定への評価 | **現時点の調査優先候補**。監査できる現物ペアと1h contractを確認できた点で条件付き推奨。履歴・費用・権利ゲートはblocked | 旧経験との連続性が強い対抗候補。時計/品質の失敗知見を活用できる。ただし今回公式仕様を読めておらず正式採否を決める根拠が不足 |

## 推奨profileの最小案（未凍結）

実装用schema/新DB/新Runnerは作らず、以下を次回確認用のprofile cardとする。**未確認パラメータは埋めず、実行可能profileとして公開しない。**

| 要素 | 案と固定前の条件 |
|---|---|
| venue/product/pair/source | bitbank、借入なしBTC/JPY spot、source pair `btc_jpy`、公式public OHLCV単一source。ユーザーがvenue/JPY案を選び、利用条件とデータ権利が確認できた後に凍結 |
| timeframe/期間 | 今回の対象は1hour。研究対象期間・データのcutoff・partitionはまだ決めない。提供履歴/研究目的から候補結果を見る前に固定し、旧2021-04-15を自動流用しない |
| position/sizing | 初期はbase quantityとquote cash、借入/shortなし。任意の個人資金額は不要。追加買い・部分売却は会計参照で表現できるが、fill生成と時刻制御は別のgoldenが必要 |
| 判断時刻 | bar start/end、provider publication、system received、feature/strategy available、signalを分離する。取得後に得た改訂値や未受領の足は判断入力にしない。最初の履歴取得時刻を、過去の当時の受領時刻と偽らない。履歴に当時の到着記録がなければ、latencyは明示的なシミュレーション仮定とし、実際のas-of可用性を実証済みと呼ばない |
| 初期fillの仮定 | **signalより後の最初の適格な足open**を参照する成行相当モデル。11:00終了→11:00:02公開→11:00:05受領/判断なら、11:00のopenは過去なので利用不可。1h境界だけを使う初期モデルでは12:00以降が候補。実際の次の約定時刻が12:00という事実ではなく、検証用の粗い近似。生成したsignalと実約定能力を混同しない |
| 評価/欠損 | 確定close等のmark時計とfill時計を分け、保持中equity/観測点DDを保存。連続時間DDとは呼ばない。gap・revision・同一足の到達順序不明を補完せず、unresolvedで停止またはそのRunの条件不足を表示 |
| fee/丸め | taker相当の**確認済み公開条件**を基本候補とし、base/quote・fee時点・桁/丸め/最低量をprofile版へ固定。現時点は値不明でblocked。Stage 0C参照はquoteの非負feeだけなので、base fee/negative rebate/venue丸めは未対応であり、確認後に独立goldenが必要 |
| spread/slippage/impact | 明示feeと分ける。根拠ある基準値が未確立のまま「通常費用」や「現実に執行可能」と表示しない。将来は根拠と観測粒度を記録した好条件/基準/ストレスと感度を比較し、推定不能なら仮定依存・執行妥当性未確認とする。固定最悪費だけで仮説を棄却しない |
| benchmark | 少なくともcash/no-positionと費用込みbuy-and-holdを候補に、研究目的と同じ時計/期間/費用/資金制約で事前固定。候補成績を見て比較対象を選ばない。いま優位性の評価をしない |
| 記録 | profile版/根拠条項/取得時の仕様snapshot、dataset identity、修正履歴、全候補・探索family・partition、検証条件/未検証範囲を既存manifest/MD/JSONへ関連付ける。今回profile freeze、取得、研究DB統合はしない |

### 初期モデルで扱えないもの

**取引所で機能が提供されることと、1h OHLCモデルで妥当に評価できることは別。** 初期のunsupportedは、戦略に優位性がないという判定ではない。

- 板/約定フロー、高頻度、秒・分内の発注遅延、queue優先順位、maker約定率、market impactを主要因とする戦略。
- 高安touchだけでfillを確定するlimit/post-only、板がない状況での部分約定、VWAP/TWAP分割執行、約定価格の改善を仮定する手法。
- 同一足のstop/target到達順、stop-limit未約定、gap-through、足内trailing/複数決済の順序。Stage 0Cの両barrier touch拒否だけでこれらを実装済みとしない。
- maker rebateを収益源とする方式、feeがbase assetや複数通貨になる方式、venue固有のfee丸め。Stage 0C会計の対応範囲を超える部分は独立試験待ち。
- short/借入/レバレッジ、funding、先物、裁定の複数venue/pair、別時間足の同期。現物1市場1hという今回の範囲外。

各仮説について、entry/exit/sizing/分割/trailingなどの`tested / untested / unsupported`、理由（解像度/仕様/権利/安全/段階）、必要な追加データやgoldenを記録する。テストできないexitを含む仮説全体を不採用へ置き換えない。後から表現を広げる際は新profile/Spec/kernel revisionとし、同一Holdoutを未使用へ戻さない。Market Knowledgeにも未検証条件を残す。

## 旧quant-btc-labから活用する経験

- Q1の日次checkpoint・原本/checksum・取得済み世代の確定・gap/重複/訂正・API failureと欠損の区別は設計知識として使う。既存の実データやadapterコードを未検証で移植しない。
- 年指定→日次指定の旧不具合、JST06境界はsource固有。bitbankの1hourも日次だが、GMOと同じ日付timezoneとは推論しない。
- 旧`available_at=end`は理論上の最早時刻。published/received/feature readyの証拠ではない。今回のStage 0Cで分けた時計を維持する。
- 旧固定20bpsやshortのJST06 financingは、現在のBTC現物のfee/spread/slippageの根拠にならない。旧simple-return会計をcash/quantity/fee会計の正解にしない。
- 旧gap時の決済event脱落、MTMとclosed-trade DDの違い、会計/時計の独立oracleを新profileのgoldenへ引き継ぐ。旧成績が新profileでも再現するとは主張しない。

## 選択・技術判断・証拠待ちを分ける

| 区分 | 必要な事項 | 今回の推奨/扱い |
|---|---|---|
| **ユーザーの選択** | 初回のvenue/quoteと研究目的 | 条件付きのbitbank BTC/JPY案を使うか、旧経験優先のGMOを先に検証するか。1hour/BTC現物/1市場1ペアという今回の前提は再質問しない。JPY以外を優先するなら比較範囲を改めて決める |
| **ユーザーの選択** | 研究対象期間・目的/仮説の優先順位、将来のデータ取得範囲/保存・外部送信の許可 | 全期間無制限取得や外部AI入力を仮定しない。新たな支出やサービス契約が必要なら対象/費用を決める。今回取得・契約はしない。個人資金/口座は尋ねない |
| **技術的に決定可** | 選定・確認済み仕様の正確な符号化 | pair mapping、UTC正規化、日付bucket変換、有限retry/429処理、重複/改訂/gap、decimal表現、manifest/hash、fixture設計、同一条件の比較手順。最小のHTTP/JSON adapter候補、既存Go/Python境界を維持する。要件を変更する新依存やUI刷新は不要 |
| **技術的に決定可（条件付き）** | profile/kernelの最小表現 | long-only現物、適格openモデル、cash/quantity/fee/mark/DDとunsupportedの記録を提案。ユーザーの研究目的・必要なentry/exit表現を勝手に変更しない。actual fee/tick/roundingは公式条件を確認後にだけ固定 |
| **第三者資料/証拠待ち（承認で代替不可）** | Raw/正規化データ保管権、外部AI入力、再配布、利用者/地域/用途適格性 | 各候補の**別々の権利欄**として管理。無料・無認証・公開・既存保存実績を許諾と見なさない。利用者の承認だけで第三者の権利を補わない。曖昧なら取得不可、必要なら先方へ照会。ただし今回は外部への照会送信をしない |
| **第三者資料/実測待ち** | 履歴最古日/保持期限、足境界/確定/訂正、fee tier/通貨/丸め、tick/lot/minimum、実際のspread/slippage | まず公式文書を調べる。文書だけで分からない実品質は、後日明示的に許可された小標本の取得で初めて確認する。今は無通信fixtureで埋められる範囲だけを設計し、架空値で正式profileを作らない |

## 次に必要な確認と停止条件

1. 環境設定の許可案を保存・反映し、GMO/bitbank公式サイトの現行API仕様・料金・規約を読む。記録するのはURL/適用日/条項/対象用途/制約と短い要約。資料閲覧であり市場データ取得ではない。
2. 比較表の未確認欄を補い、保管/AI送信/公開権の3欄と履歴範囲・注文/費用を根拠付きで再比較する。必要条件が満たされなければ初期構成を確定しない。別quote/有料feedを自動追加しない。
3. ユーザーが初期venue案と研究範囲を選択後、確認済みの仕様でprofile固有の**合成golden**を設計する。例えば日付/足境界、確定と遅延、最小量/丸め、base/quote fee、不足cash/在庫、rebate対応外拒否、gap/stop同時touch、MTMとDD。今回はテストやエンジンを実装しない。
4. 実データ取得には別途対象/期間/権利/保存先/上限の許可が必要。Runner有効化とStage 1はWindows実効境界・Holdout・残るStage 0ゲート・Provider条件等と別の開始許可が必要で、本調査から進めない。

## 実施範囲と品質確認

アプリコード・依存・CI・合成参照コードは変更しない。取得したのはPRのHTMLと文書repo/Markdownだけ。実価格・約定・ローソク足・板・pair設定データAPI、archive/CSV/ZIP市場データ、アカウント/注文APIは呼んでいない。有料サービス/外部照会/Runnerを使わず、既存quantのデータ取得CLIも実行していない。

文書の相対リンク、出典の固定版、記述と公式ソースの突合、差分/公開範囲を確認する。今回の成果は机上比較・未確定項目・暫定profileで、数値性能テストではない。

`gpt-6-luna`指定の3担当にbitbank公式API、旧GMO研究/不具合、要件と判断境界の読取を委任した。ツールの指定受理は確認したが実行基盤内の独立モデル証明はない。親がPR #3本文、bitbankの最小量/精度フィールド等の追加根拠、availability/fillの矛盾と条件付き推奨を統合・確認した。別の従量APIへ切り替えていない。
