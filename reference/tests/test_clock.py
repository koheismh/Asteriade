import unittest
from datetime import datetime, timedelta, timezone

from asteriade_synthetic.clock import (
    AmbiguousBar,
    BarClock,
    authorize_signal,
    barrier_touch,
    validate_fill_time,
)


UTC = timezone.utc


def at(hour: int, minute: int = 0, second: int = 0) -> datetime:
    return datetime(2025, 1, 2, hour, minute, second, tzinfo=UTC)


class BarClockTests(unittest.TestCase):
    def setUp(self) -> None:
        self.bar = BarClock(at(10), at(11), at(11, 0, 2), at(11, 0, 5))

    def test_normalizes_aware_timestamps_to_utc(self) -> None:
        offset = timezone(timedelta(hours=2))
        bar = BarClock(
            datetime(2025, 1, 2, 12, tzinfo=offset),
            datetime(2025, 1, 2, 13, tzinfo=offset),
            datetime(2025, 1, 2, 13, 0, 2, tzinfo=offset),
            datetime(2025, 1, 2, 13, 0, 5, tzinfo=offset),
        )
        self.assertEqual(bar.start, at(10))
        self.assertEqual(bar.received_at, at(11, 0, 5))

    def test_rejects_naive_bar_timestamp(self) -> None:
        with self.assertRaises(ValueError):
            BarClock(datetime(2025, 1, 2, 10), at(11), at(11, 0, 2), at(11, 0, 5))

    def test_rejects_reversed_or_inconsistent_timestamps(self) -> None:
        with self.assertRaises(ValueError):
            BarClock(at(11), at(10), at(11, 0, 2), at(11, 0, 5))
        with self.assertRaises(ValueError):
            BarClock(at(10), at(11), at(11, 0, 5), at(11, 0, 2))
        with self.assertRaises(ValueError):
            BarClock(at(11), at(11), at(11, 0, 2), at(11, 0, 5))
        with self.assertRaises(ValueError):
            BarClock(at(10), at(11), at(10, 59), at(11, 0, 5))

    def test_query_times_require_aware_datetimes(self) -> None:
        naive = datetime(2025, 1, 2, 11, 0, 6)
        for value in (naive, None, "2025-01-02T11:00:06Z"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                authorize_signal(self.bar, value)
            for index in range(3):
                times = [at(11, 0, 5), at(11, 0, 6), at(11, 0, 6)]
                times[index] = value
                with self.subTest(index=index), self.assertRaises(ValueError):
                    validate_fill_time(self.bar, *times)

    def test_signal_and_fill_use_instants_across_timezones(self) -> None:
        offset = timezone(timedelta(hours=9))
        signal = at(11, 0, 5).astimezone(offset)
        fill = at(11, 0, 6).astimezone(offset)
        authorize_signal(self.bar, signal)
        validate_fill_time(self.bar, signal, fill, at(11, 0, 6))

    def test_signal_requires_local_receipt_inclusive_boundary(self) -> None:
        for unavailable in (at(11), at(11, 0, 2)):
            with self.subTest(at=unavailable), self.assertRaises(ValueError):
                authorize_signal(self.bar, unavailable)
        authorize_signal(self.bar, at(11, 0, 5))

    def test_fill_must_follow_eligible_time_and_signal(self) -> None:
        validate_fill_time(self.bar, at(11, 0, 5), at(11, 0, 6), at(11, 0, 6))
        with self.assertRaises(ValueError):
            validate_fill_time(self.bar, at(11, 0, 5), at(11, 0, 5), at(11, 0, 6))
        with self.assertRaises(ValueError):
            validate_fill_time(self.bar, at(11, 0, 5), at(11, 0, 6), at(11, 0, 5))
        with self.assertRaises(ValueError):
            validate_fill_time(self.bar, at(11, 0, 2), at(11, 0, 6), at(11, 0, 6))


class BarrierTouchTests(unittest.TestCase):
    def test_single_touch_or_no_touch(self) -> None:
        self.assertEqual(barrier_touch("105", "94", "95", "110"), "stop")
        self.assertEqual(barrier_touch("111", "99", "95", "110"), "target")
        self.assertEqual(barrier_touch("109.99", "95.01", "95", "110"), "none")

    def test_rejects_ambiguous_bar_touching_both_sides(self) -> None:
        with self.assertRaises(AmbiguousBar):
            barrier_touch("111", "94", "95", "110")

    def test_equal_barriers_are_inclusive_and_both_still_ambiguous(self) -> None:
        self.assertEqual(barrier_touch("109", "95", "95", "110"), "stop")
        self.assertEqual(barrier_touch("110", "96", "95", "110"), "target")
        with self.assertRaises(AmbiguousBar):
            barrier_touch("110", "95", "95", "110")

    def test_price_contract_in_every_parameter(self) -> None:
        invalid = ("1e2", "+1", " 1", "1_0", ".1", "1.", "01", None, 100)
        for value in invalid:
            for index in range(4):
                values = ["109", "96", "95", "110"]
                values[index] = value
                with self.subTest(value=value, index=index), self.assertRaises(ValueError):
                    barrier_touch(*values)

    def test_rejects_invalid_price_and_ohlc_inputs(self) -> None:
        for value in ("NaN", "Infinity", "-Infinity", "0", "-1", "oops", "1" * 65):
            with self.subTest(value=value), self.assertRaises(ValueError):
                barrier_touch(value, "96", "95", "110")
        with self.assertRaises(ValueError):
            barrier_touch("95", "96", "95", "110")
        with self.assertRaises(ValueError):
            barrier_touch("111", "94", "110", "110")


if __name__ == "__main__":
    unittest.main()
