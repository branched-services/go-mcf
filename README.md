# go-mcf

A Go port of [LEMON 1.3.1](https://lemon.cs.elte.hu)'s primal Network Simplex min-cost-flow solver.

Scope is DeFi-oriented: flows and capacities use `*uint256.Int` (256-bit unsigned integers suitable for EVM token amounts) while costs are `int64`. General-purpose `int64`-flow users should use a different library.

## Usage

```go
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
```

## License

Boost Software License 1.0 -- see [LICENSE](LICENSE).

This project includes code derived from LEMON -- see [NOTICE](NOTICE) for attribution.
