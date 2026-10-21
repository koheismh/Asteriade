from dataclasses import FrozenInstanceError
from fractions import Fraction
import unittest

from asteriade_synthetic.spot import SpotLedger


def f(value: str) -> Fraction:
    """Independent expected values, written as decimal literals."""
    return Fraction(value)


class SpotLedgerTests(unittest.TestCase):
    def test_golden_buy_sell_mark_and_drawdown(self) -> None:
        ledger = SpotLedger("1000")
        self.assertEqual(ledger.snapshot().equity, f("1000"))

        ledger.apply_fill("buy-1", "buy", "2", "10", "0.02")
        bought = ledger.snapshot()
        self.assertEqual((bought.cash, bought.quantity, bought.remaining_cost_basis),
                         (f("979.98"), f("2"), f("20.02")))
        self.assertEqual(bought.equity, f("999.98"))
        self.assertEqual(bought.total_fees, f("0.02"))

        ledger.apply_fill("sell-1", "sell", "1", "12", "0.012")
        sold = ledger.snapshot()
        self.assertEqual((sold.cash, sold.quantity, sold.remaining_cost_basis),
                         (f("991.968"), f("1"), f("10.01")))
        self.assertEqual((sold.realized_gross, sold.realized_net),
                         (f("2"), f("1.978")))
        self.assertEqual(sold.total_fees, f("0.032"))
        self.assertEqual(sold.equity, f("1003.968"))

        marked = ledger.mark("11")
        self.assertEqual((marked.unrealized_gross, marked.unrealized_net),
                         (f("1"), f("0.99")))
        self.assertEqual(marked.equity, f("1002.968"))
        self.assertEqual(marked.peak_equity, f("1003.968"))
        self.assertEqual(marked.max_drawdown_amount, f("1"))
        self.assertEqual(marked.max_drawdown_fraction, Fraction(1, 1003968) * 1000)

    def test_close_position_clears_basis_and_can_realize_loss(self) -> None:
        ledger = SpotLedger("50")
        ledger.apply_fill("b", "buy", "2", "10", "0.2")
        ledger.apply_fill("s", "sell", "2", "9", "0.1")
        snap = ledger.snapshot()
        self.assertEqual((snap.quantity, snap.remaining_cost_basis), (f("0"), f("0")))
        self.assertEqual((snap.realized_gross, snap.realized_net),
                         (f("-2"), f("-2.3")))
        self.assertEqual(snap.cash, f("47.7"))
        self.assertEqual(snap.unrealized_gross, f("0"))
        self.assertEqual(snap.unrealized_net, f("0"))

    def test_exact_cash_and_one_unit_short(self) -> None:
        ledger = SpotLedger("10")
        ledger.apply_fill("exact", "buy", "1", "9", "1")
        self.assertEqual(ledger.snapshot().cash, f("0"))
        before = ledger.snapshot()
        with self.assertRaises(ValueError):
            ledger.apply_fill("short", "buy", "1", "0.01", "0")
        self.assertEqual(ledger.snapshot(), before)
        ledger.apply_fill("exact-2", "sell", "1", "10", "0")
        self.assertEqual(ledger.snapshot().quantity, f("0"))

    def test_rejected_operations_are_atomic_and_fill_id_can_be_reused(self) -> None:
        ledger = SpotLedger("10")
        before = ledger.snapshot()
        bad_fills = [
            ("retry", "buy", "1", "11", "0"),
            ("inventory", "sell", "1", "1", "0"),
            ("side", "hold", "1", "1", "0"),
            ("quantity", "buy", "0", "1", "0"),
            ("fee", "buy", "1", "1", "-0.1"),
        ]
        for fill in bad_fills:
            with self.subTest(fill=fill), self.assertRaises(ValueError):
                ledger.apply_fill(*fill)
            self.assertEqual(ledger.snapshot(), before)
        ledger.apply_fill("retry", "buy", "1", "10", "0")
        self.assertEqual(ledger.snapshot().quantity, f("1"))

    def test_duplicate_fill_and_bad_decimal_strings(self) -> None:
        ledger = SpotLedger("20")
        ledger.apply_fill("once", "buy", "1", "2", "0")
        before = ledger.snapshot()
        with self.assertRaises(ValueError):
            ledger.apply_fill("once", "sell", "1", "3", "0")
        self.assertEqual(ledger.snapshot(), before)

        invalid = ["-1", "+1", ".1", "1.", "1e2", "NaN", "Infinity", " 1", "01", "1/2"]
        for value in invalid:
            with self.subTest(value=value), self.assertRaises(ValueError):
                ledger.mark(value)
            self.assertEqual(ledger.snapshot(), before)

    def test_inventory_overrun_and_snapshot_is_frozen(self) -> None:
        ledger = SpotLedger("10")
        ledger.apply_fill("buy", "buy", "1", "2", "0")
        before = ledger.snapshot()
        with self.assertRaises(ValueError):
            ledger.apply_fill("overrun", "sell", "1.0001", "3", "0")
        self.assertEqual(ledger.snapshot(), before)
        with self.assertRaises(FrozenInstanceError):
            before.cash = f("0")  # type: ignore[misc]


if __name__ == "__main__":
    unittest.main()
