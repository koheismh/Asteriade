"""Compose independent checks explicitly; this is not a product execution API."""

from datetime import datetime, timezone
from fractions import Fraction
import unittest

from asteriade_synthetic.clock import AmbiguousBar, BarClock, barrier_touch, validate_fill_time
from asteriade_synthetic.spot import SpotLedger


def at(second: int) -> datetime:
    return datetime(2025, 1, 2, 11, 0, second, tzinfo=timezone.utc)


class GuardedScenarioTests(unittest.TestCase):
    def test_unreceived_data_never_reaches_accounting_in_guarded_flow(self) -> None:
        ledger = SpotLedger("1000")
        bar = BarClock(datetime(2025, 1, 2, 10, tzinfo=timezone.utc), at(0), at(2), at(5))
        before = ledger.snapshot()
        with self.assertRaisesRegex(ValueError, "received"):
            validate_fill_time(bar, at(2), at(6), at(6))
            ledger.apply_fill("b", "buy", "2", "10", "0.02")
        self.assertEqual(ledger.snapshot(), before)
        validate_fill_time(bar, at(5), at(6), at(6))
        ledger.apply_fill("b", "buy", "2", "10", "0.02")
        self.assertEqual(ledger.snapshot().cash, Fraction("979.98"))

    def test_ambiguous_path_never_reaches_accounting_in_guarded_flow(self) -> None:
        ledger = SpotLedger("100")
        ledger.apply_fill("b", "buy", "1", "100", "0")
        before = ledger.snapshot()
        with self.assertRaises(AmbiguousBar):
            barrier_touch("111", "94", "95", "110")
            ledger.apply_fill("exit", "sell", "1", "110", "0")
        self.assertEqual(ledger.snapshot(), before)
