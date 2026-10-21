"""Expected values use independent Decimal arithmetic, never ledger internals."""

from decimal import Decimal, localcontext
from fractions import Fraction
import unittest

from asteriade_synthetic.spot import SpotLedger


def decimal_value(value: Fraction) -> Decimal:
    return Decimal(value.numerator) / Decimal(value.denominator)


class IndependentAccountingTests(unittest.TestCase):
    def test_partial_sale_matches_decimal_oracle_and_conservation(self) -> None:
        # Independent cash-flow and per-unit acquisition calculation, not a port
        # of apply_fill. The numbers originate in STAGE0B_REFERENCE_TESTS.md.
        d = Decimal
        cash = d("1000") - d("2") * d("10") - d("0.02") + d("12") - d("0.012")
        realized = d("12") - d("0.012") - (d("10") + d("0.02") / d("2"))
        unrealized = d("11") - (d("10") + d("0.02") / d("2"))
        equity = cash + d("11")
        self.assertEqual((cash, realized, unrealized, equity),
                         (d("991.968"), d("1.978"), d("0.99"), d("1002.968")))
        ledger = SpotLedger("1000")
        ledger.apply_fill("buy", "buy", "2", "10", "0.02")
        ledger.apply_fill("partial", "sell", "1", "12", "0.012")
        snap = ledger.mark("11")
        self.assertEqual(tuple(decimal_value(v) for v in (
            snap.cash, snap.realized_net, snap.unrealized_net, snap.equity)),
            (cash, realized, unrealized, equity))
        self.assertEqual(snap.equity - 1000, snap.realized_net + snap.unrealized_net)
        with localcontext() as ctx:
            ctx.prec = 40
            oracle_dd = (d("1003.968") - equity) / d("1003.968")
            self.assertEqual(decimal_value(snap.max_drawdown_fraction), oracle_dd)

    def test_multiple_buys_and_sales_conserve_cash_and_net_pnl(self) -> None:
        ledger = SpotLedger("100")
        # Unit acquisition cost = (10 + .1 + 20 + .2) / 2 = 15.15.
        ledger.apply_fill("b1", "buy", "1", "10", "0.1")
        ledger.apply_fill("b2", "buy", "1", "20", "0.2")
        ledger.apply_fill("s1", "sell", "0.5", "18", "0.05")
        snap = ledger.mark("16")
        self.assertEqual(snap.cash, Fraction("78.65"))
        self.assertEqual(snap.remaining_cost_basis, Fraction("22.725"))
        self.assertEqual(snap.realized_net, Fraction("1.375"))
        self.assertEqual(snap.unrealized_net, Fraction("1.275"))
        self.assertEqual(snap.equity, Fraction("102.65"))
        self.assertEqual(snap.equity - 100, snap.realized_net + snap.unrealized_net)
        ledger.apply_fill("s2", "sell", "1.5", "16", "0.1")
        final = ledger.snapshot()
        self.assertEqual((final.cash, final.quantity, final.remaining_cost_basis),
                         (Fraction("102.55"), 0, 0))
        self.assertEqual(final.equity - 100, final.realized_net)

    def test_historical_maximum_drawdown_rate_survives_a_higher_peak(self) -> None:
        ledger = SpotLedger("100")
        ledger.apply_fill("b", "buy", "1", "100", "0")
        self.assertEqual(ledger.mark("80").max_drawdown_fraction, Fraction("0.2"))
        ledger.mark("200")
        self.assertEqual(ledger.mark("180").max_drawdown_fraction, Fraction("0.2"))
        # Maximum absolute drawdown and maximum relative drawdown may come from
        # different observations: 200 -> 170 loses 30, but only 15%.
        snap = ledger.mark("170")
        self.assertEqual(snap.max_drawdown_amount, Fraction("30"))
        self.assertEqual(snap.max_drawdown_fraction, Fraction("0.2"))

    def test_sell_fee_cannot_make_cash_negative_and_rejection_is_atomic(self) -> None:
        ledger = SpotLedger("10")
        ledger.apply_fill("b", "buy", "1", "10", "0")
        before = ledger.snapshot()
        with self.assertRaisesRegex(ValueError, "cash"):
            ledger.apply_fill("retry", "sell", "1", "1", "1.01")
        self.assertEqual(ledger.snapshot(), before)
        ledger.apply_fill("retry", "sell", "1", "1", "1")
        self.assertEqual(ledger.snapshot().cash, 0)
