// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"errors"
	"fmt"

	"github.com/holiman/uint256"
)

// validateSolveInputs checks all preconditions on the inputs to Solve.
// It returns a descriptive error for the first violation found, or nil.
func validateSolveInputs(arcs []Arc, n, source, sink int, demand *uint256.Int) error {
	if n < 2 {
		return fmt.Errorf("node count n=%d: %w", n, errTooFewNodes)
	}
	if source < 0 || source >= n {
		return fmt.Errorf("source index %d out of range [0, %d)", source, n)
	}
	if sink < 0 || sink >= n {
		return fmt.Errorf("sink index %d out of range [0, %d)", sink, n)
	}
	if source == sink {
		return fmt.Errorf("source and sink are the same node %d", source)
	}
	if demand == nil {
		return errors.New("demand is nil")
	}
	if demand.IsZero() {
		return errors.New("demand is zero")
	}

	for i := range arcs {
		a := &arcs[i]
		if a.From < 0 || a.From >= n {
			return fmt.Errorf("arc %d: From index %d out of range [0, %d)", i, a.From, n)
		}
		if a.To < 0 || a.To >= n {
			return fmt.Errorf("arc %d: To index %d out of range [0, %d)", i, a.To, n)
		}
		if a.From == a.To {
			return fmt.Errorf("arc %d: self-loop on node %d", i, a.From)
		}
		if a.Capacity == nil {
			return fmt.Errorf("arc %d: capacity is nil", i)
		}
		if !arcCostWithinBound(a.Cost, n) {
			return fmt.Errorf("arc %d: cost %d exceeds safe bound for n=%d", i, a.Cost, n)
		}
	}

	return nil
}

var errTooFewNodes = errors.New("need at least 2 nodes")
