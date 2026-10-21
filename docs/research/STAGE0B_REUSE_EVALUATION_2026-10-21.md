# Stage 0B 既存研究・バックテストエンジンの再利用評価

記録日: 2026-10-21（JST）  
フェーズ: **Stage 0Bの調査・設計判断を完了。コード移植、エンジン選定実証、Stage 0Cの実行は未開始。**

**判断:** 旧`quant-btc-lab`は、研究境界、因果的なデータ処理、凍結・回復、合成テストの資産として価値がある。一方、研究専用の固定条件とsimple-return会計を、初期の現物BTC商品仕様へそのまま移せない。**部品単位の再利用評価を継続し、製品への丸ごとの組込みは行わない。** Freqtradeは初期ランタイムの採用対象から外す。Backtraderは軽量な独立比較器の候補として残すが、ランタイム採用は保留する。新エンジンの全面自作を決めたわけではない。

## 1. 調査の基準と証拠の範囲

Asteriadeの基準は`codex/investment-research-stage0`の`98f369344eb9e6e6eda71d72a9d35b98385a02f4`、[要件§7.4–7.5・§16・AT-39](../REQUIREMENTS.md)、[実装計画](../IMPLEMENTATION_PLAN.md)、[Stage 0A記録](STAGE0A_IDENTITY_2026-10-21.md)。初回優先市場は暗号資産現物（BTC等）だが、venue/pair/feed/timeframeは未決である。

| 比較対象 | 固定したソース | 確認範囲 |
|---|---|---|
| koheismh/quant-btc-lab | `9a0aee09965da55b5050af7a981ace3f5eec8dc7`、既定master | market/features/edges/interactions/genetics/robustness/m6c/strategy/monetization/higher_timeframe/htf_strategy/prospectiveと関連文書・テスト。M6の修正履歴も取得 |
| freqtrade/freqtrade | `0db79d391f646eafbdf3c0976dd55dc90b296522`、既定develop | ライセンス、依存、Windows導入、backtest・lookahead/recursive分析の一次文書と核コード。stableリリースの採用検証ではない |
| mementum/backtrader | `b853d7c90b6721476eb5a5ea3135224e33db1f14` | ライセンス、setup、broker/commission/filler/注文系。確認HEADの日時は2023年で、最近の保守・別forkは調査外 |

Gitのblob filterとsparse checkoutでコード・文書・テストを取得した。市場データ・実研究の数値成果物・Actions artifact・evidence branchは取得していない。外部READMEの実行手順や旧プロジェクトの次タスクは資料として扱った。

証拠は次の3種類を区別する。

- **今回の静的確認:** ソース、テストのassertion、commit差分、依存/ライセンス宣言を読んだ事実。
- **旧文書に記録された実績:** 過去のCI件数、実データ評価、checksum一致等。今回それらのartifactや結果を再検証してはいない。
- **今回の限定実行:** 人工入力で旧M6修正前後の純粋関数を抽出して比較した最小probeと、会計fixtureの独立Decimal計算だけ。旧package/pytest全体、外部エンジン、Windowsは実行していない。

## 2. 3候補の共通比較

| 評価軸 | quant-btc-lab | Freqtrade | Backtrader |
|---|---|---|---|
| 主用途/機能 | BTC/JPY研究、特徴量・統計・GA・固定戦略。後期には部分決済/trail/MTM/前向き記録もある | 暗号資産の研究から実売買まで含むBot基盤。手数料・stop/ROI/trailing等 | 複数feed/時間軸の汎用イベント駆動シミュレーター。Market/Limit/Stop/StopLimit/StopTrail等 |
| Windows/依存 | Python>=3.12、PyYAML/PyArrow/filelock/NumPy、uv.lock。ネイティブwheelを含み、Windows配布は未実測 | Python>=3.12、多数の依存（CCXT/Pandas/SciPy/TA-Lib/PyArrow等）。公式はWindowsでDockerを強く推奨、WSL・native PowerShell手順も存在 | 主体は純Python、setupの通常runtime依存宣言は空。plottingはoptional。OS-independentの宣言はあるが、旧Python classifierだけで現行Python/Windows互換を保証できない |
| ライセンス/配布 | 確認treeにLICENSE/COPYING等がなくpyprojectにも宣言なし。所有・第三者寄与・再配布権の確認が必要 | GPLv3の宣言と本文。改変・同梱配布には適用条件、対応ソース/通知等の確認が必要 | GPLv3+の宣言と本文。同様に配布形態と義務を確認する |
| 会計/約定 | 版ごとに固定。初期は単一position/次足open/単純収益率、後期はpartial/trail/MTM。ただし汎用の現物cash/quantity/tick/fee通貨モデルではない | OHLC内の価格順序を仮定。公式backtestは原則指定価格・slippageなし。より細かい足で改善しても実約定を復元するものではない | commission/slippage/fillerを設定可能。既定Marketは次bar open。設定可能性自体は正解性の証明にならない |
| 戦略と固定kernelの境界 | 科学コード/条件をhash固定するが、PythonコードとrawファイルへのOS隔離は提供しない | 戦略PythonがDataFrame/callbackとengineにアクセス。検証器から未来情報を完全隔離する境界ではない | StrategyがCerebro/brokerへ広くアクセス。固定カーネルの不変性・ファイル/秘密/ネットワーク隔離は別途必要 |
| 検証資産 | 因果prefix、gap、target境界、独立HAC参照、freeze/receipt、GA容量・crash回帰等 | lookahead/recursive analysisと約定仮定の文書化。ただし未発火signal等の検出限界がある | 注文/commission/部分約定の独立比較候補。Asteriadeの時計/丸め/曖昧順序との一致は未検証 |
| 今回の採否 | **部分再利用の第一候補。コード移植・丸ごと採用は保留** | **初期製品ランタイムには非採用。検査設計は参考** | **参考・独立比較器候補。ランタイム採用は保留** |

根拠: [Q-deps]、[F-deps]、[F-win]、[F-fill]、[F-look]、[F-rec]、[B-deps]、[B-fill]、[B-volume]。

GPLだから利用できないという判断ではない。Asteriade自身の配布ライセンスと、改変・リンク・同梱の実態が未定である。**別プロセスにすれば当然に配布義務を回避できるとは扱わない。** [FreqtradeのGPL本文][F-license]と[BacktraderのGPL本文][B-license]の§4–6も参照する。同様に、ユーザー所有の旧リポジトリにライセンスがないことは再利用不能の確定ではないが、第三者の権利や配布条件を確認せずコードを転載しない。

## 3. 旧資産の部品別判断

区分: **再利用可能**＝考え方・受入観点を今後の仕様へ引き継げる。ソースの無条件コピーを意味しない。**要修正**＝条件を変えた部分再利用候補。**参考のみ**＝別目的/仮定を理解する資料。**非採用**＝初期製品へそのまま持ち込まないもの。

| 資産 | 区分 | 確認した内容 / 必要な対応 |
|---|---|---|
| 市場データschema/品質/Raw→normalized identity | 要修正 | UTC、OHLC整合、重複/修正/gap、Raw/checksum/manifest/checkpointは有用。GMOの日次label・銘柄・1h固定をsource adapterへ分離し、ライセンス・到着時刻・改訂の扱いを追加する [Q-market] |
| 過去のみのFeature計算、gap reset、prefix試験 | 再利用可能（設計・試験） | as-of境界、未来値変更で過去出力不変、EMA/Wilderの因果性、gap後再warm-upを受入観点にする。必要featureだけを選び、全カタログを必須依存にしない [Q-prefix] |
| 旧413特徴、固定target/horizonと高時間足集約 | 参考のみ | 構造や必要履歴の例。未来targetは利益・約定ではない。UTC4h/日足・固定lookbackを現物仕様として自動採択しない [Q-target]、[Q-htf] |
| train-only閾値、HAC/BH/Holm/block検証 | 要修正 | 独立行列参照・実時間lag・検定不能p=1を参考にする。標本適格性/推定可能性を分離し、研究ごとの前提とfamilyを固定。旧期間/lag/閾値/判定を汎用規則にしない [Q-inference] |
| GA grammar/capacity、inner/outer隔離、checkpoint | 要修正（Stage 2候補） | 容量を値より先に数える、全試行/失敗を保存、outerをfitnessへ戻さない設計は有用。旧GAはBoolean feature構造探索でEntry/Exit/Sizing全体の最適化ではない [Q-capacity]、[Q-outer] |
| 初期M7単一positionのexecution/指標 | 参考のみ | 単純な参照比較には使える。closed-trade equityのDDを保有中MTMのDDと混同しない [Q-simple]、[Q-simple-metric] |
| M7X/M7Jのpartial/trail/slot/MTM/clock | 要修正 | 旧資産がpartial/trail非対応とは判断しない。ただしsimple-return/notional型から、現物のquantity/cash/fee通貨・丸め・残高制約への変換と独立照合が必要 [Q-partial]、[Q-portfolio]、[Q-clock] |
| PRE seal、receipt、exact continuation | 要修正 | 読取り前の永続記録、同identityの継続と科学修正の分離を参考にする。単一出力directoryのfile gateから、全Project横断のcandidate/Holdout registryとOS境界へ拡張する [Q-phase]、[Q-loader] |
| Prospectiveのfreeze/first-seen/欠損記録 | 参考のみ | 過去backfill禁止、凍結後の観測、連続性・incidentは有用。特定の単一戦略、件数/期間のgateは引き継がない。実evidence branchは調査していない [Q-prospective] |
| 合成/独立oracle/故障注入のテスト設計 | 再利用可能（設計・試験） | 将来情報poison、実時刻HAC、receipt-before-read、gap後close event、atomic checkpoint等を独立fixtureへ再構成する。旧テストの存在は新実装合格の証明ではない [Q-regression] |
| 固定の旧費用・short financing・研究winner/成績 | 非採用（そのままの移植） | 固定bpsやJST06 financingを現物費用にしない。旧ルールは仮説の出所として参照できるが、選抜系譜を消して独立初回試行と数え直さない。旧成績を新kernelの正解・独立確認に使わない |
| 旧実データ/CI artifact/自動収集workflow | 非採用（製品への直接持込み） | 旧Actionsや取得CLIを実行せず、raw保管権・保持期限・復元を別途設計する。文書上のartifact期限は本調査日には経過しているが、現存性は未確認。compact metadataだけで完全再現可能としない [Q-retention] |

**特に時刻に注意する。** 旧仕様は`available_at=bar_end_time`を理論上の最早時刻と明記し、`fetched_at`を別記している [Q-market]。これはデータが実際にその時点で到着した証拠ではない。Asteriadeではsource公開、ローカル受領、計算完了、判断、約定の時計を明示し、後日再取得したバーを当時観測済みと装わない。

**統計にも限界がある。** M3→M4→GA等の適応・選抜は条件付きの研究履歴であり、各段階のBH/Holmやseed数が全研究横断の選抜偏りを自動解消するわけではない。LLMの既知市場情報も時間分割だけでは除去できない。全候補・family・露出/選抜を保存する現在要件を優先する。

## 4. 不具合から引き継ぐ判断

[科学的実装不具合と防止策](STAGE0B_SCIENTIFIC_DEFECTS_2026-10-21.md)に、M6の標本適格性とHAC推定可能性の混同、GA容量矛盾、checkpoint出力、GMO日次API、gap時の決済event、source envelopeとslot分母の問題を整理した。

M6の修正は[ab7dfe58](https://github.com/koheismh/quant-btc-lab/commit/ab7dfe58e9fa148ad21ee8f0c963e2afff05997e)で確認した。HACの計算式を全面置換した修正ではなく、標本の条件と推定不能の理由を分けた修正である。M6Cは消費済みHoldoutのcorrected reanalysisであり、未使用・独立・Prospectiveへ戻す根拠にはしない [Q-corrected]。

今回の最小probeは、state=500、complement=1,000、target=stateの人工系列で、修正前の`sample_eligible=false`が修正後`true`となり、SE=0・推定不能・p=1は維持されることを確認した。旧packageの通常importは`pyarrow`未導入で失敗したため、**純粋関数をAST抽出した限定比較**である。Python 3.12.14 / NumPy 2.3.5を使い、プロジェクトlock環境や既存pytestの合格とは扱わない。依存は新規導入していない。

## 5. 次の実装方針と着手条件

1. **引き継ぐ:** データ/研究identity、因果時計、欠損・未来参照拒否、候補/選抜の保存、標本/推定/有効性/経済性の分離、故障注入・独立参照の試験観点。
2. **再利用の第一候補:** quantの研究条件に依存しない部品。権利・依存・入出力・副作用を確認し、合成参照値に合わせてから限定adapter/Runner化の可否を判断する。旧sourceをAsteriadeへコピーしていない。
3. **比較の予備候補:** Backtraderは現物数量/commission等の差分照合用候補。採用版、Python/Windows適合とライセンス、同じ約定契約の設定を確認する。必須依存にしない。
4. **初期に採らない:** Freqtrade全体の組込み、ライブBot機能・取引資格情報、全特徴/GAの先行導入、旧費用/科学gateの無条件踏襲。理由は初期scopeとの広さ、配布/依存負担、固定kernel/権限制御の追加工事であり、機能不足やGPLによる全面禁止ではない。
5. **選定の出口:** [正解試験案](STAGE0B_REFERENCE_TESTS.md)で同じ時計・入力・fee・quantity/cashを固定し、既存候補が参照約定と残高へ一致するか比較する。差分と修正量を見ずに既存採用/新規自作を確定しない。

[Stage 0Cの判断メモ](STAGE0C_DECISIONS.md)は、調査で解ける項目、ユーザーの研究方針が必要な項目、Windows実測が必要な項目を分ける。venue/feedの権利、時計、限られた注文/会計profileと正解値を具体化するのが次の作業である。本Stage 0Bでは実市場データ取得、有料API、実注文、Runner有効化、大規模実装へ進まない。

## 6. 完了と未検証

**完了:** 固定ソースの確認、部品分類、3候補の比較、主要不具合/修正の照合、人工probe、正解試験とStage 0C判断準備、引き継ぎ。AT-39の設計上の比較記録を作成した。

**未検証:** 旧/外部engineの全test suite・インストール・性能、同条件の実行差分、Windows配布/OS安全境界、実市場の品質・費用・保管権、ライセンス適合、長期artifactの現存性、実Prospective証拠。AT-39を実行妥当性やWindows配布まで全面合格と数えず、Stage 0全体も未合格のまま。

Luna指定(`gpt-6-luna`)の3サブエージェントへデータ/特徴量/統計/GA、約定/会計、外部比較を委任した。親Codexが差分・重要ソースを読み、人工probeと会計算術を確認し、採否と未検証の境界を統合した。内部モデル再ルーティングは観測できない。

[Q-deps]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/pyproject.toml#L1
[Q-market]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/docs/MARKET_DATA.md#L8
[Q-prefix]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/tests/test_feature_foundation.py#L30
[Q-target]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/edges/targets.py#L26
[Q-htf]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/higher_timeframe/market.py#L28
[Q-inference]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/robustness/inference.py#L70
[Q-capacity]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/genetics/capacity.py#L16
[Q-outer]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/tests/test_m5_phase_firewall.py#L84
[Q-simple]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/strategy/execution.py#L66
[Q-simple-metric]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/strategy/metrics.py#L23
[Q-partial]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/monetization/execution.py#L117
[Q-portfolio]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/htf_strategy/portfolio.py#L136
[Q-clock]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/htf_strategy/clock.py#L6
[Q-phase]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/robustness/phase.py#L25
[Q-loader]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/robustness/loader.py#L70
[Q-prospective]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/prospective/engine.py#L25
[Q-regression]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/tests/test_m5_checkpoint_atomic.py#L13
[Q-retention]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/docs/M5_ENGINE.md#L121
[Q-corrected]: https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/docs/M6C_GOVERNANCE_NOTE.md#L5
[F-deps]: https://github.com/freqtrade/freqtrade/blob/0db79d391f646eafbdf3c0976dd55dc90b296522/pyproject.toml#L16
[F-win]: https://github.com/freqtrade/freqtrade/blob/0db79d391f646eafbdf3c0976dd55dc90b296522/docs/installation.md#L18
[F-fill]: https://github.com/freqtrade/freqtrade/blob/0db79d391f646eafbdf3c0976dd55dc90b296522/docs/backtesting.md#L556
[F-look]: https://github.com/freqtrade/freqtrade/blob/0db79d391f646eafbdf3c0976dd55dc90b296522/docs/lookahead-analysis.md#L7
[F-rec]: https://github.com/freqtrade/freqtrade/blob/0db79d391f646eafbdf3c0976dd55dc90b296522/docs/recursive-analysis.md#L3
[B-deps]: https://github.com/mementum/backtrader/blob/b853d7c90b6721476eb5a5ea3135224e33db1f14/setup.py#L62
[B-fill]: https://github.com/mementum/backtrader/blob/b853d7c90b6721476eb5a5ea3135224e33db1f14/backtrader/brokers/bbroker.py#L48
[B-volume]: https://github.com/mementum/backtrader/blob/b853d7c90b6721476eb5a5ea3135224e33db1f14/backtrader/fillers.py#L30

[F-license]: https://github.com/freqtrade/freqtrade/blob/0db79d391f646eafbdf3c0976dd55dc90b296522/LICENSE#L195
[B-license]: https://github.com/mementum/backtrader/blob/b853d7c90b6721476eb5a5ea3135224e33db1f14/LICENSE#L195
