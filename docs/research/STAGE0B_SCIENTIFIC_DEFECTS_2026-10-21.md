# Stage 0B 不具合・仕様修正の確認と再発防止

記録日: 2026-10-21（JST）。対象は`quant-btc-lab@9a0aee09965da55b5050af7a981ace3f5eec8dc7`と指定修正履歴。旧研究の再計算や、消費済みHoldoutの再評価は行っていない。科学的な判定不具合、取得/保存不具合、結果を見る前の仕様明確化を分ける。

## D-01: M6で標本適格性とHAC推定可能性を混同

**確認した原因:** 修正前の`robustness.inference.evaluate`は、必要な件数/比率/分散の違反と、HACの`zero_hac_standard_error`等を同じ`reasons`へ追加していた。結果として、標本条件を満たしていてもSE=0なら`sample_eligible=false`になり、点効果等に基づくF0判定まで遮断された。これは係数やHAC分散の計算式を全面変更した事例ではない。

**修正:** [ab7dfe58](https://github.com/koheismh/quant-btc-lab/commit/ab7dfe58e9fa148ad21ee8f0c963e2afff05997e)（親`e15d5119918ae61904925ed8dd73f9e5294b002c`）は、sample条件と`inference_estimable`を分離した。SE=0でも標本が適格なら点効果を保持し、pは保守的に1、zは未定義、有意なF1へは進めない。構造のrank deficiencyは別途sample不適格に残す。[現行の分離](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/robustness/inference.py#L71)と[追加回帰テスト](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/tests/test_m6_inference.py#L68)を確認した。

**今回の限定probe:** state=500、complement=1,000、target=state、1,500個の等間隔人工観測を与えた。全体target分散は正、群内残差はゼロ。

| 観測 | 修正前 | 現行 |
|---|---|---|
| sample_eligible | false | true |
| invalid_reasons | zero_hac_standard_error | 空 |
| raw effect / HAC SE | 1 / 0 | 1 / 0 |
| inference_estimable | 独立fieldなし | false |
| raw p | 1 | 1 |

純粋関数`evaluate/factorial_hac`を修正前と現行のASTから抽出し、既存の純粋な`hac`と定数HOUR=3600/HAC_LAG=168で比較、上記assertionはPASS。Python 3.12.14 / NumPy 2.3.5。**通常のpackage importはPyArrow不足で失敗した。** 依存を入れずに行った関数レベルの再現であり、package/既存pytest/lock環境/全研究の成功を証明しない。型・定数・関数の抽出範囲外をmockで合格させたとは報告しない。

**Holdoutの扱い:** [消費を記録して停止したcommit ac97f884](https://github.com/koheismh/quant-btc-lab/commit/ac97f8846b5629b33d62440e2b4e19c7cddbdf24)、[Consumption Addendum](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/docs/M6_HOLDOUT_CONSUMPTION_ADDENDUM_V1.md#L8)、[M6C governance](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/docs/M6C_GOVERNANCE_NOTE.md#L5)を確認した。旧M6は消費済み・科学不具合でSTOPPEDのまま。修正版M6Cは別identityのcorrected reanalysisで、pristine/未使用/完全ProspectiveやL3へ復元するものではない。旧結果の再読込みやcorrected runは今回行っていない。

**Asteriadeでの防止:** sample eligibility、inference estimability、出力の妥当性、証拠クラス、経済性を別field/別gateにする。SE=0、全体分散0、rank不足、件数の境界、p=1でfamilyに残る場合を独立oracleで試験する。LLM判断や点効果の大きさで統計ゲートを上書きしない。

## D-02: GAの合法空間34件とpopulation64の矛盾

[旧blockerとv1.1修正](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/docs/M5_GENOME_CAPACITY_BLOCKER.md#L1)では、特定targetの合法な方向付きGenomeが34件しかないのにpopulation64かつ重複禁止を求めていた。**探索前の仕様矛盾**であり、好成績を作るための結果後変更ではない。

修正は値を見る前のcapacity列挙を導入し、容量<=64なら全数1回、>64ならGAとするもの。全数方式で偽の8 seed convergenceを作らず`seed_support=null`とする。[capacity実装](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/genetics/capacity.py#L16)と[34件の回帰試験](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/tests/test_m5_genome.py#L127)を静的確認した。

Asteriadeはgrammar、重複判定、合法容量、探索予算の整合を事前検査する。容量や個体数の旧数値を製品既定にせず、全候補・未選択・失敗とouter評価の一方向性を保存する。GAがEntry以外のExit/Sizingまで最適化したという表示はしない。

## D-03: compact checkpoint出力でコピーdictの更新が保存されない

[旧M5実装記録](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/docs/M5_ENGINE.md#L119)では、`mapping()`が返すコピーへの変更が保存用dictに戻っていなかった。`export_compact`の戻し代入等を修正し、数値producerとmetadata consumerのidentityを区別した。数値関数/元checkpoint不変を照合したという旧記録はあるが、今回その全artifactを検証していない。

[compactの回帰試験](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/tests/test_m5_artifact_boundaries.py#L43)、[atomic publish直前crashと再開byte一致の試験](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/tests/test_m5_checkpoint_atomic.py#L13)を確認した。

Asteriadeはserialize→decodeの必須項目/参照整合、compactとfullの別contract、journal→cache→checkpoint公開の各故障点を試験する。metadataだけの修正で研究のproducer identityを書き換えず、異なるruntimeへの黙示resumeや新規探索へのfallbackを拒否する。

## D-04: GMO 1h KLineの年指定と日次API契約の不一致

[14609d75の差分](https://github.com/koheismh/quant-btc-lab/commit/14609d751bd0e7ec19cc463fd0dd10f371914a30)で、`fetch_year(..., YYYY)`から日次`fetch_date(..., YYYYMMDD)`への修正と、JST06起点のbucket、日次checkpointの導入を確認した。[旧データ仕様](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/docs/MARKET_DATA.md#L29)は旧schemaの無断流用を拒否する。

これは取得契約の不具合であり、HTTP 404を市場の欠損やネットワーク遮断と同一視しない。今回GMOへは接続していない。Asteriadeはsource-specificなtimestamp/日付bucket/範囲切出しの人工fixture、HTTP/APIエラー分類、原本/schema version、確定済み足のrevision検出を用意する。過去のGMO修正を他venueの正解にしない。

## D-05: gap直後にportfolioの決済eventが落ちる

[旧記録DECISIONS:445](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/DECISIONS.md#L445)は、intrabar exitを記録する時刻の次market hourが欠けると、event axisからcloseが落ちる問題を実市場PnL前に発見したと記録する。現在の[axis生成](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/monetization/portfolio.py#L94)は観測済みsource hourのclose時刻も含める。[回帰試験](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/tests/test_monetization_portfolio.py#L117)は、exit時刻の価格rowがなくても建玉を閉じ、gap価格を作らないことを検査している。

Asteriadeはprice eventとfill/accounting eventの時計を分離する。観測済み約定を欠損足と一緒に消さず、未観測価格をforward-fillしてMTM/損益を補わない。position/cash/在庫保存則とevent exactly-onceを独立に検査する。

## D-06: direct-entry時計とslot上限の定義

[66b2d115の差分](https://github.com/koheismh/quant-btc-lab/commit/66b2d115a92d209b4da8ffc4fe095e9172752330)は、closed 4h signalのavailability時刻とdirect entryを分け、厳密にその後のH1 openを使う規則をPnL前に固定した。これは**事前の仕様明確化**で、すべての市場に1h遅延を強制する根拠ではない。

[DECISIONS:489](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/DECISIONS.md#L489)はStage Bでcost控除後MTMをslot分母にした追加拒否を修正したと記録する。[現在のintended slot cap](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/src/quant_btc_lab/htf_strategy/portfolio.py#L213)を確認した。intended比率と実時点のnotional/equity比は違う概念であり、同じ「1x制限」と表示しない。

Asteriadeはdecision可用時刻、注文受付、約定、fee、mark、残高検査の順序をprofileに固定する。実際の現物cash/quantityと、意図した資金配分/露出指標を分ける。手数料後残高不足や丸め不足を、slot指標だけで見逃さない。

## D-07: 保存形式envelopeとraw recordを取り違える

[DECISIONS:416–418](https://github.com/koheismh/quant-btc-lab/blob/9a0aee09965da55b5050af7a981ace3f5eec8dc7/DECISIONS.md#L416)は、元M3の`record/sha256` envelopeをloaderが見落とし、価格/Featureのprojection前に停止した事例と、実形式fixtureでの修正を記録している。これはsource ingestの失敗であり、約定の正解性やHoldoutでの成績を示さない。

Asteriadeはproducerの実際のschemaとconsumerの期待を版管理し、元bytes checksum検証後に許可されたfieldだけdecodeする。テスト用の単純dictしか通さない設計を避け、nested envelope、欠落、未知版、値アクセス前の失敗、禁止結果field混入を試験する。

## 共通の防止原則と未検証

- 凍結前に科学的な意味と期待値を定める。hash一致だけで意味の正しさを証明しない。
- receiptを値アクセス経路の手前へ永続化し、不確実な失敗・復元欠落を未使用へ戻さない。単一directoryのfilelockは全Project横断台帳やOS隔離を代替しない。
- infra再試行、科学計算修正、結果後の戦略改変を別identity/別分類で保存する。corrected evidenceを独立確認へ昇格させない。
- 会計の手計算oracleではcashとequity、grossとnet、実現と未実現、費用の通貨と控除時点を区別する。[正解試験案](STAGE0B_REFERENCE_TESTS.md)の数値は人工例のみ。
- 過去文書の大量PASSやartifact保存は今回の実測ではない。raw source/arraysが失効・未保全なら完全再現を主張しない。Windows・現行lockでの数値/配布・OS安全性は未検証。

本調査は不具合をAsteriadeコードで修正したものではなく、再利用判断と次の試験契約を文書化したものである。
