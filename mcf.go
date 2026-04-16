// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"errors"

	"github.com/holiman/uint256"
)

// Arc represents a directed edge in the min-cost flow network.
// Capacity must be non-nil. Flow is written in-place by the solver on success.
type Arc struct {
	From, To int
	Cost     int64
	Capacity *uint256.Int
	Flow     *uint256.Int
}

// Result holds the output of a successful min-cost flow computation.
// TotalFlow equals the requested demand on success. TotalCost is best-effort
// and may be truncated when bottleneck*cost overflows int64.
type Result struct {
	TotalFlow *uint256.Int
	TotalCost int64
}

// ErrInfeasible is returned when the requested demand cannot be routed from
// source to sink within the given network capacity.
var ErrInfeasible = errors.New("mcf: demand cannot be routed from source to sink")
