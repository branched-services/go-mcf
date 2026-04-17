// SPDX-License-Identifier: BSL-1.0

package mcf

import (
	"context"
	"errors"

	"github.com/holiman/uint256"
)

// Arc represents a directed arc in the network. From and To are zero-based
// node indices, Cost is the per-unit cost of sending flow along the arc, and
// Capacity is the maximum flow the arc can carry.
//
// On a successful call to [Solve], Flow is written in place to the optimal
// flow value for this arc. Callers must not share the same []Arc slice across
// concurrent Solve calls.
type Arc struct {
	From     int
	To       int
	Cost     int64
	Capacity *uint256.Int
	Flow     *uint256.Int
}

// Result holds the output of a successful [Solve] call.
type Result struct {
	// TotalFlow is the total flow pushed from source to sink.
	TotalFlow *uint256.Int

	// TotalCost is the sum of arc costs weighted by their flows, computed as
	// int64. Because individual arc flows may exceed int64 range, TotalCost is
	// advisory and may under-report for very large flows where the true
	// weighted sum exceeds math.MaxInt64.
	TotalCost int64
}

// ErrInfeasible is returned when the requested demand cannot be routed from
// source to sink within the network's capacity constraints.
var ErrInfeasible = errors.New("mcf: demand cannot be routed from source to sink")

// ErrInvalidInput is returned when Solve receives input that violates
// preconditions (see [Solve] documentation for the full list).
var ErrInvalidInput = errors.New("mcf: invalid input")

// Solve computes a minimum-cost flow of the given demand from source to sink
// in a network with n nodes and the provided arcs. On success it returns a
// [Result] and writes each arc's optimal flow into Arc.Flow in place.
//
// Solve is safe to call concurrently from multiple goroutines on independent
// inputs. Callers must not share the same arcs slice across concurrent Solve
// calls because Arc.Flow is written in place.
//
// If ctx is cancelled or its deadline expires, Solve returns (Result{}, ctx.Err())
// with no partial results written to arcs.
func Solve(ctx context.Context, arcs []Arc, n, source, sink int, demand *uint256.Int) (Result, error) {
	return Result{}, errors.New("mcf: not implemented")
}
