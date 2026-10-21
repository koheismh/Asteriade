"""Small, offline spot accounting reference for synthetic fixtures only."""

from dataclasses import dataclass
from fractions import Fraction
import re


_DECIMAL = re.compile(r"(?:0|[1-9][0-9]*)(?:\.[0-9]+)?\Z")
_MAX_DECIMAL_LENGTH = 128


def _amount(value: str, *, positive: bool, name: str) -> Fraction:
    if not isinstance(value, str) or len(value) > _MAX_DECIMAL_LENGTH or not _DECIMAL.fullmatch(value):
        raise ValueError(f"{name} must be a bounded nonnegative decimal string")
    whole, dot, fractional = value.partition(".")
    result = Fraction(int(whole + fractional), 10 ** len(fractional))
    if positive and result <= 0:
        raise ValueError(f"{name} must be positive")
    return result


@dataclass(frozen=True)
class Snapshot:
    cash: Fraction
    quantity: Fraction
    remaining_cost_basis: Fraction
    realized_gross: Fraction
    realized_net: Fraction
    total_fees: Fraction
    unrealized_gross: Fraction
    unrealized_net: Fraction
    equity: Fraction
    peak_equity: Fraction
    max_drawdown_amount: Fraction
    max_drawdown_fraction: Fraction


class SpotLedger:
    """Exact fractional ledger for long-only synthetic spot fills."""

    def __init__(self, initial_cash: str) -> None:
        self._cash = _amount(initial_cash, positive=False, name="initial_cash")
        self._quantity = Fraction(0)
        self._basis = Fraction(0)
        self._remaining_entry_fees = Fraction(0)
        self._realized_gross = Fraction(0)
        self._realized_net = Fraction(0)
        self._total_fees = Fraction(0)
        self._fill_ids: set[str] = set()
        self._mark = Fraction(0)
        self._equity = self._cash
        self._peak = self._equity
        self._max_drawdown = Fraction(0)
        self._max_drawdown_fraction = Fraction(0)

    def _observe(self, price: Fraction) -> None:
        equity = self._cash + self._quantity * price
        peak = max(self._peak, equity)
        drawdown = peak - equity
        self._mark = price
        self._equity = equity
        self._peak = peak
        self._max_drawdown = max(self._max_drawdown, drawdown)
        if peak > 0:
            self._max_drawdown_fraction = max(self._max_drawdown_fraction, drawdown / peak)

    def apply_fill(self, fill_id: str, side: str, quantity: str, price: str, fee: str) -> None:
        if not isinstance(fill_id, str) or not fill_id or len(fill_id) > 256:
            raise ValueError("fill_id must be a nonempty bounded string")
        if fill_id in self._fill_ids:
            raise ValueError("duplicate fill_id")
        if side not in ("buy", "sell"):
            raise ValueError("side must be 'buy' or 'sell'")
        qty = _amount(quantity, positive=True, name="quantity")
        px = _amount(price, positive=True, name="price")
        fee_value = _amount(fee, positive=False, name="fee")
        notional = qty * px

        # Compute every proposed value before touching the ledger, so rejects are atomic.
        if side == "buy":
            new_cash = self._cash - notional - fee_value
            if new_cash < 0:
                raise ValueError("insufficient cash")
            new_quantity = self._quantity + qty
            new_basis = self._basis + notional + fee_value
            new_entry_fees = self._remaining_entry_fees + fee_value
            new_realized_gross = self._realized_gross
            new_realized_net = self._realized_net
        else:
            if qty > self._quantity:
                raise ValueError("insufficient inventory")
            allocated_basis = self._basis * qty / self._quantity
            allocated_entry_fees = self._remaining_entry_fees * qty / self._quantity
            new_cash = self._cash + notional - fee_value
            if new_cash < 0:
                raise ValueError("insufficient cash for sell fee")
            new_quantity = self._quantity - qty
            new_basis = self._basis - allocated_basis
            new_entry_fees = self._remaining_entry_fees - allocated_entry_fees
            gross = notional - (allocated_basis - allocated_entry_fees)
            new_realized_gross = self._realized_gross + gross
            new_realized_net = self._realized_net + gross - allocated_entry_fees - fee_value
            if new_quantity == 0:
                new_basis = Fraction(0)
                new_entry_fees = Fraction(0)

        new_total_fees = self._total_fees + fee_value
        new_peak = max(self._peak, new_cash + new_quantity * px)
        new_equity = new_cash + new_quantity * px
        new_drawdown = new_peak - new_equity
        new_drawdown_fraction = new_drawdown / new_peak if new_peak > 0 else Fraction(0)

        self._cash = new_cash
        self._quantity = new_quantity
        self._basis = new_basis
        self._remaining_entry_fees = new_entry_fees
        self._realized_gross = new_realized_gross
        self._realized_net = new_realized_net
        self._total_fees = new_total_fees
        self._fill_ids.add(fill_id)
        self._mark = px
        self._equity = new_equity
        self._peak = new_peak
        self._max_drawdown = max(self._max_drawdown, new_drawdown)
        self._max_drawdown_fraction = max(self._max_drawdown_fraction, new_drawdown_fraction)

    def mark(self, price: str) -> Snapshot:
        px = _amount(price, positive=True, name="price")
        self._observe(px)
        return self.snapshot()

    def snapshot(self) -> Snapshot:
        unrealized_gross = self._quantity * self._mark - (self._basis - self._remaining_entry_fees)
        unrealized_net = self._quantity * self._mark - self._basis
        return Snapshot(
            cash=self._cash,
            quantity=self._quantity,
            remaining_cost_basis=self._basis,
            realized_gross=self._realized_gross,
            realized_net=self._realized_net,
            total_fees=self._total_fees,
            unrealized_gross=unrealized_gross,
            unrealized_net=unrealized_net,
            equity=self._equity,
            peak_equity=self._peak,
            max_drawdown_amount=self._max_drawdown,
            max_drawdown_fraction=self._max_drawdown_fraction,
        )
