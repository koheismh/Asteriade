"""Synthetic reference checks for bar availability and barrier touches."""

from dataclasses import dataclass
from datetime import datetime, timezone
from decimal import Decimal, InvalidOperation
import re


_DECIMAL = re.compile(r"(?:0|[1-9][0-9]*)(?:\.[0-9]+)?\Z")


class AmbiguousBar(ValueError):
    """A bar touched both barriers, so its intrabar path is unknown."""


def _utc(value: datetime) -> datetime:
    if not isinstance(value, datetime) or value.tzinfo is None:
        raise ValueError("timestamp must be timezone-aware")
    try:
        offset = value.utcoffset()
    except Exception:
        raise ValueError("timestamp must be timezone-aware") from None
    if offset is None:
        raise ValueError("timestamp must be timezone-aware")
    return value.astimezone(timezone.utc)


@dataclass(frozen=True)
class BarClock:
    start: datetime
    end: datetime
    published_at: datetime
    received_at: datetime

    def __post_init__(self) -> None:
        for name in ("start", "end", "published_at", "received_at"):
            object.__setattr__(self, name, _utc(getattr(self, name)))
        if not self.start < self.end <= self.published_at <= self.received_at:
            raise ValueError("bar timestamps are out of order")


def authorize_signal(bar: BarClock, at: datetime) -> None:
    if _utc(at) < bar.received_at:
        raise ValueError("bar was not yet received")


def validate_fill_time(
    bar: BarClock,
    signal_at: datetime,
    eligible_at: datetime,
    fill_at: datetime,
) -> None:
    authorize_signal(bar, signal_at)
    signal_at_utc = _utc(signal_at)
    eligible_at_utc = _utc(eligible_at)
    fill_at_utc = _utc(fill_at)
    if eligible_at_utc <= signal_at_utc:
        raise ValueError("fill eligibility must follow the signal")
    if fill_at_utc < eligible_at_utc:
        raise ValueError("fill precedes eligibility")


def _price(value: str) -> Decimal:
    if not isinstance(value, str) or len(value) > 64 or not _DECIMAL.fullmatch(value):
        raise ValueError("price must be a short positive decimal string")
    try:
        result = Decimal(value)
    except (InvalidOperation, ValueError):
        raise ValueError("price must be a finite positive decimal") from None
    if not result.is_finite() or result <= 0:
        raise ValueError("price must be a finite positive decimal")
    return result


def barrier_touch(high: str, low: str, stop: str, target: str) -> str:
    """Return long-side stop/target/none without guessing intrabar order."""
    high_value, low_value = _price(high), _price(low)
    stop_value, target_value = _price(stop), _price(target)
    if high_value < low_value or stop_value >= target_value:
        raise ValueError("invalid OHLC bounds or barrier order")
    touched_stop = low_value <= stop_value
    touched_target = high_value >= target_value
    if touched_stop and touched_target:
        raise AmbiguousBar("bar touched both barriers")
    if touched_stop:
        return "stop"
    if touched_target:
        return "target"
    return "none"
