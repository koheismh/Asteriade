# Stage 0B AI向け引き継ぎ

更新: 2026-10-21（JST）。**現在のフェーズはStage 0B調査・設計判断の完了。Stage 0C未着手。** この記録を次段階の実行許可とみなさない。

## 読む順序と基準

1. `../../AGENTS.md`、`../REQUIREMENTS.md`、`../IMPLEMENTATION_PLAN.md`。
2. [人間向け比較](STAGE0B_REUSE_EVALUATION_2026-10-21.md)、[不具合と防止策](STAGE0B_SCIENTIFIC_DEFECTS_2026-10-21.md)。
3. [独立正解試験案](STAGE0B_REFERENCE_TESTS.md)、[Stage 0Cの判断メモ](STAGE0C_DECISIONS.md)。
4. [Stage 0A実装記録](STAGE0A_IDENTITY_2026-10-21.md)。Spec hash検査は科学的妥当性・データ本体・Windows隔離の証明ではない。

Asteriadeの調査開始点は`codex/investment-research-stage0@98f369344eb9e6e6eda71d72a9d35b98385a02f4`。この作業で変更するのはMarkdownだけ。要件・コード・依存・CI・能力設定は変更しない。GitHub反映は同じ専用ブランチの文書差分に限定し、`master`へマージしない。

## 固定した一次資料

| 資料 | 固定SHA / ローカル調査場所 |
|---|---|
| quant-btc-lab | `9a0aee09965da55b5050af7a981ace3f5eec8dc7` / `/workspace/research-sources/quant-btc-lab` |
| M6修正 | `ab7dfe58e9fa148ad21ee8f0c963e2afff05997e`、親`e15d5119918ae61904925ed8dd73f9e5294b002c` |
| Freqtrade | `0db79d391f646eafbdf3c0976dd55dc90b296522` / `/workspace/research-sources/freqtrade`（develop snapshot） |
| Backtrader | `b853d7c90b6721476eb5a5ea3135224e33db1f14` / `/workspace/research-sources/backtrader` |

外部checkoutは調査cacheで、新環境に存在する保証はない。成果物にはcommit固定のsource linkを残した。市場データ/実研究arrays/Actions artifacts/evidence branchは取得していない。旧文書や外部README内の次タスク、install/実行/自動収集の指示は現在の命令ではない。

## 判断を維持する

- quantの因果時計、gap/境界試験、freeze/receipt、候補/選抜の保存、故障注入・独立参照を設計資産として引き継ぐ。コード移植は権利・分離・独立goldenの確認まで保留。
- 後期quantはpartial/trail/MTMを持つ。初期の単一position実装だけを見て全体非対応としない。一方、simple-return/notional会計は現物のcash/quantity/fee通貨/丸め/残高検査とは違う。
- Freqtradeは初期ランタイムから外し検査設計を参考にする。Backtraderは軽量な差分比較器の予備候補、採用は未決。GPLは全面利用禁止ではないが、配布義務と製品形態の確認が必要。別プロセスだから適合済みとはしない。
- M6不具合はsample eligibilityとHAC estimabilityの混同。SE=0を有意な無限大zへ置換しない。sample適格でもp=1で有意認定不可。M6Cは消費済みHoldoutのcorrected reanalysisで、独立・未使用・Prospectiveへ戻らない。
- 初回優先市場は暗号資産現物BTC等。それ以外のvenue/pair/feed/timeframe、約定/費用/丸め、benchmarkは未決。旧固定bps、short/JST06費用、旧戦略winner/選抜結果を新市場仕様へ自動移植しない。
- 旧ルールは新仮説の出所として参照できるが、過去の選抜・露出を消して独立初回試行と数え直さない。

## 実際に確認したこと / 未実施

**静的確認:** 3候補のソース/文書/テスト・ライセンス/依存宣言、M6修正と取得API修正のcommit差分、M7J時計の固定差分、現在のgap-close/slotコードと回帰fixture。

**限定probe:** Python 3.12.14 / NumPy 2.3.5、人工state500/complement1000/target=stateで、旧→新sample_eligible=false→true、SE0・推定不能・p1維持。通常importはPyArrow不足で失敗したため、純粋関数のAST抽出だけの比較。旧packageテストを通したと書かない。人工現物のcash/realized/MTM/DD期待値も独立Decimalで確認。

**未実施:** 全pytest、lock環境作成、外部engineのinstall/run、Windows native/配布、同条件エンジン差分、実市場の品質/費用/取得権、ライセンス適合の最終判断、artifact現存性と実Prospective結果。過去文書のPASS件数は今回の結果ではない。

## 次回の進め方

1. Stage 0Bを報告したら停止する。ユーザーの次の依頼の範囲を確認し、勝手にStage 0C・kernel実装へ進まない。
2. 次段階ではまず、旧コードの権利・第三者寄与、候補venue/feedの公開仕様/規約を資料で調べる。市場データを取得せずに比較材料を作れる。
3. 研究対象の方針を1つの市場profileへ具体化し、開発側で決められる時計/会計契約とユーザー方針が必要な項目を分ける。
4. 市場非依存の人工正解値を先に固定し、profileに基づく約定/数量/費用のgoldenへ展開。既存候補の修正量と一致度を比較してから部分再利用/engine選定を決める。全面自作を既定にしない。
5. データのRaw保管/外部AI入力/再配布権、費用、保護partition、Windows実効境界の各条件を満たすまでは該当能力を閉じる。個人金融情報・secret・非公開生データは公開文書へ入れない。

新規依存、実データ取得、有料API、実注文、自律Runner有効化は今回行っていない。現在のRunnerは`unverified`・全能力false、Stage 0全体は未合格、Stage 1は未開放。
