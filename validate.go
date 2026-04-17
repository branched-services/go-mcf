// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"fmt"
	"math"

	"github.com/holiman/uint256"
)

func validate(arcs []Arc, n, source, sink int, demand *uint256.Int) error {
	if n < 2 {
		return fmt.Errorf("mcf: invalid input: n (%d) must be >= 2: %w", n, ErrInvalidInput)
	}
	if source < 0 || source >= n {
		return fmt.Errorf("mcf: invalid input: source (%d) out of range [0,%d): %w", source, n, ErrInvalidInput)
	}
	if sink < 0 || sink >= n {
		return fmt.Errorf("mcf: invalid input: sink (%d) out of range [0,%d): %w", sink, n, ErrInvalidInput)
	}
	if source == sink {
		return fmt.Errorf("mcf: invalid input: source (%d) == sink (%d): %w", source, sink, ErrInvalidInput)
	}
	if demand == nil {
		return fmt.Errorf("mcf: invalid input: demand is nil: %w", ErrInvalidInput)
	}
	if demand.IsZero() {
		return fmt.Errorf("mcf: invalid input: demand is zero: %w", ErrInvalidInput)
	}

	costBound := math.MaxInt64 / 8
	for i := range arcs {
		a := &arcs[i]
		if a.From < 0 || a.From >= n {
			return fmt.Errorf("mcf: invalid input: arcs[%d].From (%d) out of range [0,%d): %w", i, a.From, n, ErrInvalidInput)
		}
		if a.To < 0 || a.To >= n {
			return fmt.Errorf("mcf: invalid input: arcs[%d].To (%d) out of range [0,%d): %w", i, a.To, n, ErrInvalidInput)
		}
		if a.From == a.To {
			return fmt.Errorf("mcf: invalid input: arcs[%d] is a self-loop (%d -> %d): %w", i, a.From, a.To, ErrInvalidInput)
		}
		if a.Capacity == nil {
			return fmt.Errorf("mcf: invalid input: arcs[%d].Capacity is nil: %w", i, ErrInvalidInput)
		}
		// Check |Cost| * (n+1) < MaxInt64/8 without overflowing.
		// abs(Cost): handle math.MinInt64 specially since -MinInt64 overflows int64.
		if a.Cost == math.MinInt64 {
			return fmt.Errorf("mcf: invalid input: arcs[%d].Cost (%d) overflows guard |cost|*(n+1) < MaxInt64/8: %w", i, a.Cost, ErrInvalidInput)
		}
		absCost := a.Cost
		if absCost < 0 {
			absCost = -absCost
		}
		nPlus1 := int64(n + 1)
		// Check absCost * nPlus1 < costBound without overflow:
		// absCost < costBound / nPlus1 (integer division rounds down, so use >=)
		if nPlus1 > 0 && absCost >= int64(costBound)/nPlus1 {
			return fmt.Errorf("mcf: invalid input: arcs[%d].Cost (%d) overflows guard |cost|*(n+1) < MaxInt64/8: %w", i, a.Cost, ErrInvalidInput)
		}
	}
	return nil
}
