// SPDX-License-Identifier: BSL-1.0

package mcf_test

import (
	"context"
	"errors"
	"fmt"

	mcf "github.com/branched-services/go-mcf"
	"github.com/holiman/uint256"
)

// Example demonstrates routing 50 units of flow across a three-node network
// with two parallel paths. The cheaper path (0->1->2, cost 30/unit) is
// saturated in preference to the direct arc (0->2, cost 50/unit).
func Example() {
	arcs := []mcf.Arc{
		{From: 0, To: 1, Cost: 10, Capacity: uint256.NewInt(100), Flow: new(uint256.Int)},
		{From: 1, To: 2, Cost: 20, Capacity: uint256.NewInt(100), Flow: new(uint256.Int)},
		{From: 0, To: 2, Cost: 50, Capacity: uint256.NewInt(100), Flow: new(uint256.Int)},
	}

	result, err := mcf.Solve(context.Background(), arcs, 3, 0, 2, uint256.NewInt(50))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("TotalFlow:", result.TotalFlow)
	fmt.Println("TotalCost:", result.TotalCost)
	// Output:
	// TotalFlow: 50
	// TotalCost: 1500
}

// ExampleSolve_infeasible shows how Solve reports an infeasible instance:
// the requested demand exceeds every source-to-sink cut in the network.
func ExampleSolve_infeasible() {
	arcs := []mcf.Arc{
		{From: 0, To: 1, Cost: 1, Capacity: uint256.NewInt(10), Flow: new(uint256.Int)},
	}

	_, err := mcf.Solve(context.Background(), arcs, 2, 0, 1, uint256.NewInt(20))
	if errors.Is(err, mcf.ErrInfeasible) {
		fmt.Println("infeasible")
	}
	// Output: infeasible
}
