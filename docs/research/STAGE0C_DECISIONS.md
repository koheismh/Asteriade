# Stage 0C前の市場・検証仕様判断メモ

記録日: 2026-10-21  
状態: **Stage 0C開始前の判断準備。候補案と未決事項の整理であり、Stage 0Cの開始承認、データ取得許可、費用発生の許可ではない。**

## この文書の境界

本メモの市場profileに関して確定扱いするのは、初回優先市場が暗号資産現物（BTC等）、対象OSがWindows 11、実注文を扱わないこと、Runnerの実行能力が未検証で既存capabilityがすべてfalseであること、実市場データと有料APIを取得していないことだけである。実データの利用権、外部送信、Runner有効化、実注文をこの文書から許可しない。個人の資金額や口座情報は研究仕様に含めない。

venue、取引ペア、feed、時間足、データ保管・外部送信・再配布条件、約定・費用会計、評価手順、Windowsでの配布方法は未決である。下記の「推奨案」は検討を進めるための暫定案で、ユーザー決定や検証済み仕様として扱わない。

旧 `quant-btc-lab` は固定コミット `9a0aee09965da55b5050af7a981ace3f5eec8dc7` のコード・テストを設計参考として読んだ。単純収益率、固定bps費用、short側のJST06固定financingは当該研究の仮定であり、暗号資産現物の価格・費用・会計規則として採用しない。旧コードの時計、部分決済、MTM、gap fail-closed等は独立した合成golden fixtureを作る題材に限る（[`strategy/execution.py` at fixed commit](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/strategy/execution.py#L165)、[`htf_strategy/clock.py`](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/htf_strategy/clock.py#L6)、[`monetization/portfolio.py`](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/monetization/portfolio.py#L111)、[`prospective/engine.py`](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/prospective/engine.py#L310)）。

## 判断事項とゲート

| 項目 | 選択肢・推奨案 | 判断・依存関係 |
|---|---|---|
| **venue・pair・feed** | 選定venueのBTC/quote現物feed、または権利条件を満たす独立feedを候補にする。初期候補を1 venue・1 pairに絞る案は、データ契約と仕様を限定しやすい。 | **事前調査可:** 公開仕様、API条件、履歴範囲、価格・数量単位、提供地域、過去データの有無を調べる。**ユーザー決定:** 初期venue/pair/feedと対象範囲。**実機検証:** API/CSV等から受領したデータの時計・値・欠損の実態。決定までは外部feedへ接続しない。 |
| **利用権・保管・AI入力・再配布** | Raw responseのローカル保管権、feature化した値の保管、外部AI入力、GitHubへの再配布を別々の権利として確認する。保管権が明記されないsourceはRaw保全可能と仮定しない。 | **事前調査可:** providerの規約・ライセンス・履歴データ条件を記録する。**ユーザー決定:** 利用するsourceと許諾範囲、外部AIへ送る許可対象。GitHub公開は別途明示許可。**実機検証:** 選んだ方法で原本・metadata・checksumを保全し、禁止された内容が公開成果物へ混入しないこと。権利不明なら取込を止める。 |
| **bar時計・到着・改訂・欠損** | bar open/close、provider時刻、実受領時刻、strategyが参照できた時刻を別フィールドで記録する案。訂正・遅着は履歴を上書きせずrevisionとして記録し、重複・欠損・順不同を区別する。 | **事前調査可:** feed仕様のtimestampの意味、確定足、訂正挙動、timezoneを確認する。**設計で確定:** 再現する到着時点とlate/revised barの扱い。研究目的が変わる場合のみユーザー確認。**実機検証:** 到着遅延・訂正・gap・重複の実測と、未来情報が利用可能時刻より前に見えない試験。時刻の意味が曖昧なら研究を開始しない。 |
| **時間足・市場profile** | 1時間足は初期実験の候補案に留める。15m/1h/4h等の選択は、sourceの解像度、到着時計、計算量、仮説と検証力を確認して決める。 | **事前調査可:** sourceで提供される解像度、合成時計の仕様、候補時間足ごとのデータ要件を比較する。**方針選定:** 初回profileの時間足はユーザーまたは許可範囲内のManagerが選び、UTC/venue時刻規約は実仕様を根拠に設計で固定。**実機検証:** feedからbarを同一規則で構成できること。1hや特定venueを確定事項として記録しない。 |
| **注文種別・約定・intrabar時計** | 注文はシミュレーション上の表現に限定する。最初は単純なmarket-at-next-eligible-openを候補にし、limit/stopは価格到達、gap、手数料区分、同一bar内の順序が仕様化できるまで対象外とする案。OHLCだけで高値・安値の到達順が分からない場合は、保守規則を固定するか結果を曖昧として拒否する。 | **事前調査可:** 現物venueの注文・約定規則、barで観測可能な情報、細粒度履歴の有無を調べる。**設計で確定:** MVPで表現する注文種別と保守的な曖昧順序規則。研究の対象拡大は別判断。**実機検証:** 独立golden上でsignal availability→entry→partial/exit時刻を照合。実注文接続や発注は含めない。 |
| **quantity・現金・tick/lot丸め** | 単純リターンではなく、base quantity・quote cash・在庫・取引ごとのcash flowを明示する案。現物のlong-only 1xは候補であり確定仕様ではない。価格tick、quantity step/minimum、fee currency、丸め方向と丸め時点をprofileに固定する。 | **事前調査可:** 選定venueのtick/lot/minimum・fee currencyの仕様を確認する。**設計で確定:** 初期kernelのposition/sizing表現、fractional quantity、venue準拠の丸め。研究のリスク方針を変える場合のみユーザー確認。**実機検証:** 既知入力のcash/quantity保存則、fee控除、丸め境界のgolden照合。個人資金額を要求しない。short・借入・証拠金・liquidationは独立した市場要件なしに入れない。 |
| **費用モデル** | 手数料、bid/ask spread、slippage、market impactを別項目にする。根拠のある基準ケースと好条件・ストレスケースを分けて評価する案。旧固定20bpsは現物のfeeやspread根拠として使わない。 | **事前調査可:** 公開fee schedule、過去のbid/askまたは適切な費用資料、sourceの粒度限界を調べる。**設計で確定:** 公開fee tierと注文方法の比較条件、基準・stressの根拠。個人の非公開fee/残高を既定入力にしない。**実機検証:** fee currency/丸め/entry-exit控除位置をsynthetic ledgerで照合。impactや実行条件を根拠づけられない場合は推定不能・仮定依存と表示する。 |
| **MTM・open/close・DD** | 「価格を評価に使う時点」と「約定した時点」を分離し、realized cash、open position、unrealized PnL、equity、feesを時系列で記録する。1h open/close評価は試験しやすい候補だが、バー内の損益・DDを完全に表すとの主張はしない。 | **事前調査可:** venue/feedのbar semanticsと必要なintrabar解像度を調べる。**設計で確定:** 初期のmark時刻と出力するDD/損益の定義。現金残高とequityを区別。**実機検証:** partial、保有中のprice move、exit、gapを含む独立参照計算。closed-trade DDだけをMTM DDとして扱わない。 |
| **benchmark・採否指標** | 候補結果を見る前に、cash/no-position、spot buy-and-hold、または目的に合う単純ルールから比較対象を登録する。最低出力・費用内訳・標本数・worst period・tail risk・推定誤差を、採否gateと分けて定義する案。 | **事前調査可:** 目的別benchmarkと、データ粒度で計算可能なmetricを比較する。**計画で固定:** 候補成績を見る前にbenchmarkと採否軸を決める。研究目的・リスク許容はユーザー方針を用いる。**実機検証:** 同じ期間、時計、費用・cash制約の下で再現可能に比較できるか。結果を見てbenchmarkや閾値を選ばない。 |
| **統計・証拠クラス・registry** | Development/Validation/Final Holdoutを区別し、candidate/selection-familyと同期間・同銘柄の利用/露出履歴を共有registryで追跡する。Final値を探索AIへ渡さず、Prospectiveは凍結後の未観測データを追記する案。方法選択は標本・候補依存など前提の確認後に行う。 | **事前調査可:** 対象データが持つ粒度・期間・候補間依存と、候補統計手法の前提を整理する。**計画で固定:** 初期partition、benchmark、実験開始前のgateを、データ範囲と研究目的から根拠付きで定める。保護データへのアクセスは許可範囲を確認する。**実機検証:** storage/OS境界、アクセス監査、restore時の欠落検出、Holdoutを探索側から読めない否定試験。該当ゲート未実装なら昇格しない。 |
| **Python/native・Windows配布・ライセンス・保全** | Go native kernel、Python同梱/独立Runner、既存Python資産の限定再利用を比較する。RuntimeでDocker/Node/external DBを必須にせず、Windows Sandboxが実証されない自動実行能力は有効化しない案。データ・依存・研究成果の保存期間と復元可能性は分けて定義する。 | **事前調査可:** 旧sourceのライセンス・依存・配布方式、Windows上の候補実行方式、必要な長期保全要件を調べる。**設計で選定:** 配布/保全profileと依存方式。新しい費用や追加セットアップ等の製品条件が変わる場合はユーザー判断。利用許諾とライセンス適合を確認する。**実機検証:** Windows 11で起動、パス/権限/network拒否、Runner停止、バックアップ/restoreを検査。Linux buildやcross-buildだけでWindows安全性を合格にしない。 |

## 次の作業順

1. Stage 0B比較報告と旧sourceのライセンス・依存情報を根拠として保全し、venue/feed候補とその規約を調査する。この調査は接続・取得を含めない。
2. Source候補ごとにRaw保管、外部AI入力、公開再配布の許可を別欄で確認し、未確認条件を可視化する。
3. ユーザーに必要な市場profile上の決定事項を、選択肢・影響・根拠とともに提示する。本Stage 0Bの依頼は取得を禁止しているため実データへは接続しない。次回依頼の許可範囲と、課金・非公開送信等に必要な条件を確認して進める。
4. まず市場に依存しない人工会計・時計の正解値を文書で固定し、profile決定後に市場固有の合成golden fixtureへ具体化し、手計算のquantity/cash/fees/equity/時計を参照値と照合する。旧エンジンからコードを移植せず、新engineの大規模実装にも進まない。
5. Golden比較、kernel版固定、データ境界/registry、Windows実機否定試験の各結果をレビュー用にまとめる。実データ、有料API、Runner有効化、Stage 1移行は、それぞれ別の要件・ゲート・許可を要する。

本メモは、上記の準備順を提示するものであり、Stage 0Cの開始決定や実データ取得の申請・承認を発生させない。
