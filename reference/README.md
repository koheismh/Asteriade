# Stage 0C synthetic reference

開発用の小さな正解確認であり、汎用バックテストエンジン・製品Runnerではありません。Python 3.12の標準ライブラリだけで動き、Goアプリの起動依存には追加しません。旧quant-btc-lab/OSSのコードは移植していません。

リポジトリルートで実行:

```bash
PYTHONPATH=reference PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s reference/tests -v
```

Windows PowerShell（Python 3.12が別途存在する場合のみ）:

```powershell
$env:PYTHONPATH = "reference"
$env:PYTHONDONTWRITEBYTECODE = "1"
py -3.12 -m unittest discover -s reference/tests -v
```

Windowsでは未実行です。このコマンドの成功もOSのサンドボックスを証明しません。製品導入のためのPythonインストールは本作業に含めません。

## 時刻の契約

`BarClock`はtimezone-aware時刻だけを受け付け、UTCへ正規化します。確定した足のみを対象とし、`start < end <= published_at <= received_at`を要求します。`authorize_signal`は`signal_at >= received_at`、`validate_fill_time`はさらに`eligible_at > signal_at`かつ`fill_at >= eligible_at`を要求します。受領と同時の判断を許すのは計算遅延ゼロのtoy仮定で、実市場の遅延profileではありません。fixtureは10:00開始、11:00終了、11:00:02公開、11:00:05受領、11:00:06約定可能です。

`barrier_touch`はlong側のstop/target到達候補を値域で調べるだけです。両方に達する足は`AmbiguousBar`で拒否し、OHLCから順序を生成しません。単一候補でもバリア価格で約定したと判断するAPIではありません。ギャップ・注文時刻・板・O/C・約定価格の検証は別途必要です。

## 会計の契約

- 仮想quote通貨のlong-only現物。数量・価格は正、手数料は非負。有限長の通常10進文字列だけを受け付け、指数、符号、空白、NaN/Infinity、binary floatは拒否します。価格上限64字（時計検査）、会計入力上限128字。
- 会計は`Fraction`で正確に計算。手数料はすべてquote通貨の明示額です。買い手数料を取得原価に含め、平均原価で部分売却へ按分します。
- `realized_gross`は価格差、`realized_net`は売却数量に対応する買い手数料と売却手数料を控除。`unrealized_net`は残存取得原価に含まれる買い手数料だけを控除し、未発生の将来売却手数料は含みません。
- `equity = cash + quantity * mark`。各fillはそのfill価格で評価し、明示的な`mark`も観測点です。DD金額は各点の`peak-equity`の最大、DD率は各点の`(peak-equity)/peak`の最大を別々に保持します。DD率は割合で、百分率は100倍です。初期資産ゼロで保有もないときはDD率ゼロ。
- 二重fill、現金/在庫不足、不正値は拒否し、拒否後は会計・ID・評価履歴を更新しません。`Snapshot`は変更不可です。

正解例は[Stage 0B参照試験](../docs/research/STAGE0B_REFERENCE_TESTS.md)に基づきます。初期1,000、2単位を10・手数料0.02で買い、1単位を12・手数料0.012で売り、残りを11で評価すると、現金991.968、数量1、実現net損益1.978、評価net損益0.99、資産1,002.968です。四つの観測点でpeak1,003.968、最大DD金額1、最大DD率`1 / 1003.968`です。

## 検証と限界

固定リテラルの正解と、別の`Decimal`計算で照合します。複数買い・部分/全売却、損失、現金/在庫不足、二重約定、DD率の履歴維持、未来時刻、同時touch拒否と、検査後だけ会計に進む合成フローを確認します。

これらは独立した参照プリミティブです。会計クラス単体は時刻を検査せず、フローtestが明示的に先行検査を合成します。StrategySpec実行、将来値を物理的に隠すデータAPI、派生featureのavailability伝播、注文受付/板/約定生成、OS隔離、DB/UI、Provider、Holdout、統計、venueのtick/lot/丸め、税、外貨手数料、short/leverageは実装していません。fill回数の資源上限はtoyの呼出元管理で、敵対的なコードを安全に実行する機構ではありません。

[実測結果と次の条件](../docs/research/STAGE0C_SYNTHETIC_VALIDATION_2026-10-21.md)を参照してください。
