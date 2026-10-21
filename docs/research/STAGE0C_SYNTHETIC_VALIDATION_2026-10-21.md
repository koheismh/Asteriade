# Stage 0C — 小規模合成参照検証（2026-10-21）

状態: **今回許可された合成増分を完了。Stage 0全体は未合格、Stage 1A/1Bは未着手。** 本報告は実市場の妥当性、Windowsの実効境界、収益性を主張しない。

## 基準と変更理由

開始時に開発ブランチをfetchし、`codex/investment-research-stage0`は前回と同じ `f721104b6adc6880e6d4bd8671cf78bc4aa5ff6c` と確認した。その最新状態から専用ブランチ `codex/lean-mvp-synthetic-stage0c` を作った。PR #2はGitのPR head `58ba796c8bdfdf4b048ed76db70d3855a4ba6fea` にある提案本文を確認した。API接続が環境ポリシーのCONNECT 403で拒否され、追加コメント・レビュー状態は未取得。PR #2をマージ・クローズしていない。

Stage 0Bの[比較報告](STAGE0B_REUSE_EVALUATION_2026-10-21.md)、[参照試験](STAGE0B_REFERENCE_TESTS.md)、[引き継ぎ](STAGE0B_HANDOFF.md)、[未決事項](STAGE0C_DECISIONS.md)、Stage 0Aの[凍結記録](STAGE0A_IDENTITY_2026-10-21.md)を基準とした。今回のユーザー指示に従い、次の合理的な順序変更だけを採用する。

- Stage 1A: 現物1市場・単一Manager＋機械ゲートで縦断完走、最小Web画面、Markdown/JSON・Market Knowledge。
- Stage 1B: 独立Reviewerと最大2ラウンドの既存5-step方式を追加し、1Aと品質・費用・時間を比較する。
- 全42製品AT、安全/科学ゲート、凍結ID、月3,000円のアプリ内起動拒否上限を維持。新たなUIや汎用エンジンを先行実装しない。

採否、影響テスト、先送りと理由は[MVPスコープ改訂案](MVP_SCOPE_REVISION_2026-10-21.md)。要件・実装計画・基本設計・研究/引き継ぎ/安全仕様の該当箇所も整合更新した。受領原本と過去のStage 0記録は保全する。

## 実装した小さな単位

| 単位 | 内容 | 検証できた範囲 |
|---|---|---|
| `reference/asteriade_synthetic/clock.py` | 足開始/終了、公開、受領、signal、eligible/fill時刻。long stop/target曖昧性拒否 | UTC変換、受領前拒否、同時刻境界、早過ぎるfill拒否、両barrier到達拒否、不正時刻/価格拒否 |
| `reference/asteriade_synthetic/spot.py` | 正確な有理数の現物会計。quote手数料・平均原価・部分売却・評価損益・equity/DD | 固定golden、複数買い/部分・全売却、損失、保存則、残高/在庫/ID・失敗時の原子性 |
| `reference/tests` | 固定期待値、別のDecimal oracle、検査→会計の合成フロー | 未来時刻/曖昧性による拒否後に会計が変わらず、許可したtoy fillだけ反映 |
| `docs/MARKET_KNOWLEDGE.md` とMD/JSON例 | 現象/機構/反証/否定・未検証条件/次仮説、根拠範囲、状態、可視性、研究参照 | 最小記録仕様と構文確認。実研究カード自動生成や科学的な主張の検証は未実装 |

StrategySpec/freeze、Goコード、DB schema、既存UI、Runner capability/start拒否は変更していない。参照例はアプリ未接続で、任意StrategySpec/コードを実行しない。会計と時刻検査は独立プリミティブであり、会計単体へ時刻を渡して強制する実行APIではない。将来のRunner統合では先行検査とデータ可視境界を回避できない契約を別途作る。

### 固定正解値

| 観測 | 現金 | 数量 | 評価資産 |
|---|---:|---:|---:|
| 初期 | 1000 | 0 | 1000 |
| 2を10で買い、fee 0.02 | 979.98 | 2 | 999.98 |
| 1を12で売り、fee 0.012 | 991.968 | 1 | 1003.968 |
| 残りを11でmark | 991.968 | 1 | 1002.968 |

部分売却の実現gross/netは2 / 1.978、残存原価10.01、評価gross/netは1 / 0.99、総手数料0.032。将来の売却feeは未発生なので含めない。四つの観測点でpeak1003.968、最大DD金額1、最大DD率 `1 / 1003.968`（約0.099604768%）。この標本外の価格経路・連続時間DDは不明。

## 実測したチェック

Linux x86_64、Python **3.12.14**、固定Go **1.27.2**。追加依存なし。

| コマンド/検査 | 結果 | 意味・限界 |
|---|---|---|
| `PYTHONPATH=reference PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s reference/tests -v` | **24テスト成功、失敗0、skip 0** | 時刻/曖昧性12、現物会計6、独立oracle/回帰4、合成フロー2。製品ATの件数ではない |
| `source scripts/env.sh` 後 `go test -race -count=1 -json ./...` | **40トップレベル成功、サブテスト込み105成功、失敗0、skip 0** | 既存Go基盤・凍結/改変検出のLinux回帰。105と40を加算しない |
| `go vet ./...` / `go mod verify` | 成功 | 既存module検証・静的検査。依存追加/変更なし |
| `CGO_ENABLED=0 go build -trimpath ... ./cmd/asteriade` | Linuxビルド成功 | 出力は作業用、Gitへ入れない |
| `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath ... ./cmd/asteriade` | Windows x64クロスビルド成功 | Windows実行・OS境界合格を意味しない |
| JSON例の標準`json`構文確認 / `git diff --check` | 成功 | 形式/差分確認のみ。Market Knowledgeの内容・製品validator合格ではない |

再現コマンドと演算/時刻/DDの厳密なtoy契約は[reference/README.md](../../reference/README.md)。SHA-256は `clock.py: 55fa359cede4b1aa165af4b5d386a50ffed89cd03627ba0d39e4e8884407db00`、`spot.py: 41f504ae130ffe7650605c902564a0c65b67d70df541653e314c03ce23acb68a`。テストとコードは同じGit revisionから使用する。

### 見つけて修正した失敗

親の独立oracle追加後の18テスト実行で、16成功・**2失敗**を実際に再現した。失敗を消したり期待値を実装へ合わせたりせず、参照コードを修正した。

1. **最大DD率の履歴消失**: 過去最大DD金額を現在peakで割る実装だと、100→80（20%）の後で200→180となった際10%に下がる。各観測のDD率の最大を保持する修正により20%を維持。金額の最大と率の最大が別時点でも正しく保持する回帰を追加。
2. **売却feeで現金が負**: 残金ゼロ・売却代金1・fee1.01を拒否できなかった。手数料控除後のcash非負をcommit前に検査し、拒否時にfill IDも消費しないよう確認。

価格文字列についても符号・指数・空白等を受理しない契約へ揃え、全barrierパラメータの拒否テストを追加。最終24件で未解決の失敗はない。試験を通過した範囲だけを成功とする。

## 技術判断

| 現時点の扱い | 技術 | 理由 |
|---|---|---|
| 維持 | Go Control Plane、埋込Web、SQLite、既存凍結identity | 稼働済み基盤・境界を壊さず、小さな増分で確認できる |
| 今回のみ採用 | 開発用Python標準ライブラリ、`unittest`、内部Fraction＋独立Decimal | 新規フレームワークなしで既存Python資産に近い参照試験を作り、二進float誤差を避ける。全エンジンGo化を前提にしない |
| 条件付き候補 | Go CP＋独立Python Runner、quantの一部再利用 | ライセンス、版/依存固定、Windows配布、境界、同条件golden一致を満たすまで製品採用しない。コードは直接移植しない |
| 予備候補/初期非採用 | Backtrader独立比較器 / Freqtrade初期ランタイム | Stage 0Bの結論を維持し、今回候補を増やさない。実行適合未確認 |
| 後段 | 知識DB/検索UI、専門Reviewer群、並列Runner、GA、Prospective自動化 | 1A完走と追加価値測定を先に行う。基本の科学/安全ゲートは先送りしない |

## 未実施・未検証

WindowsでのPython参照試験・EXE実行・実効Sandbox、製品データ参照APIによる未来値隔離、派生Featureのavailability伝播、market profile/tick/lot/丸め/価格ギャップ/板/注文状態/fee別通貨/short/leverage、実engine差分・性能、Holdout registry/OS境界・復旧proof、統計/多重選抜/GA、Providerの実接続/課金、研究loop/Cockpit/MK自動生成は未実施。今回のAPIとfixtureだけでこれらを合格としない。

実市場データのダウンロード、有料API、取引所/口座、実注文、Runner自律実行、討論本実装、UI刷新、masterマージは実施しない。既存PR #2にも変更を加えない。

## 次の作業と開始条件

1. 本PR相当の差分をレビューし、段階化案・参照会計定義・未決事項を確認する。今回の報告で停止する。
2. 次の限定作業として、0Cのvenue/pair/feed/timeframe・データ利用/保管権・注文/費用/丸めprofileを資料から確定する。資料確認と実取得を区別し、実データ/支出には必要な許可を得る。選定profileの独立goldenと既存engine比較を次の検証単位にする。
3. 0Dではユーザー手元のWindowsでidentity/ACL/プロセス/通信/秘密分離/停止とHoldout否定試験を実測する。既存の基盤検証スクリプト成功だけでSandboxを合格にしない。
4. Stage 0全体の必要な安全・科学・データ・復旧証拠、Provider利用条件/料金、1A範囲と上限、明示的な開始許可が揃うまでStage 1Aを開始しない。

## 委任と保存

`gpt-6-luna`指定の3担当に、時計とfixture、現物会計とfixture、PR2採否/MK記録案を委任した。時計/会計は互いに読み取りレビューし、文書担当は親の要件/計画も確認。親が独立oracle・境界回帰・合成フローを追加し、欠陥を修正、重要判断・安全性の限界・全体整合・テスト実測を検証した。ツールはモデル指定を受理したが、実行基盤内部の独立したモデル証明は取得していない。有料の別APIへ切り替えていない。

今回の保存対象は公開可能なソース/設計/合成例だけ。PR作成APIは環境ネットワークで拒否されたため、必要な`api.github.com`許可を環境設定の下書きへ保存した（未反映・未公開）。[PR本文](STAGE0C_PR_DESCRIPTION.md)を用意し、専用ブランチをレビュー対象とする。Pushと実PRの結果は作業完了時に区別して報告する。
