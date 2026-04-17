// SPDX-License-Identifier: BSL-1.0

// Command basic is the runnable version of the README quickstart:
// a three-node network with two parallel source-to-sink paths. It
// prints the optimal TotalFlow and TotalCost reported by Solve.
package main

import (
	"context"
	"fmt"
	"log"

	mcf "github.com/branched-services/go-mcf"
	"github.com/holiman/uint256"
)

func main() {
	arcs := []mcf.Arc{
		{From: 0, To: 1, Cost: 10, Capacity: uint256.NewInt(100), Flow: new(uint256.Int)},
		{From: 1, To: 2, Cost: 20, Capacity: uint256.NewInt(100), Flow: new(uint256.Int)},
		{From: 0, To: 2, Cost: 50, Capacity: uint256.NewInt(100), Flow: new(uint256.Int)},
	}

	result, err := mcf.Solve(context.Background(), arcs, 3, 0, 2, uint256.NewInt(50))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("TotalFlow:", result.TotalFlow)
	fmt.Println("TotalCost:", result.TotalCost)
}
