# go-mcf

A Go port of [LEMON 1.3.1](https://lemon.cs.elte.hu)'s primal Network Simplex
min-cost-flow solver, specialised for DeFi.

[![Go Reference](https://pkg.go.dev/badge/github.com/branched-services/go-mcf.svg)](https://pkg.go.dev/github.com/branched-services/go-mcf)
[![Go Report Card](https://goreportcard.com/badge/github.com/branched-services/go-mcf)](https://goreportcard.com/report/github.com/branched-services/go-mcf)
[![CI](https://github.com/branched-services/go-mcf/actions/workflows/ci.yml/badge.svg)](https://github.com/branched-services/go-mcf/actions/workflows/ci.yml)
[![CodeQL](https://github.com/branched-services/go-mcf/actions/workflows/codeql.yml/badge.svg)](https://github.com/branched-services/go-mcf/actions/workflows/codeql.yml)
[![License: BSL-1.0](https://img.shields.io/badge/license-BSL--1.0-blue.svg)](LICENSE)
[![Go 1.22+](https://img.shields.io/badge/go-1.22%2B-00ADD8.svg)](go.mod)

## Scope

go-mcf targets DeFi routing on EVM-compatible networks: flows and capacities
are `*uint256.Int` to match 256-bit token amounts, while costs are `int64`.
If your workload fits in `int64` flows, a general-purpose min-cost-flow
library will serve you better.

## Features

- Primal Network Simplex ported from LEMON 1.3.1.
- `*uint256.Int` flows and capacities; `int64` costs.
- Strict-fill semantics with configurable tolerance for negative-cost cycles.
- Context cancellation in `Solve`.
- In-place flow writes on caller-owned `Arc` slices.
- Invariant-checking test helpers (see `checksolution_test.go`).

## Install

```
go get github.com/branched-services/go-mcf
```

## Quickstart

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

Runnable version: [`examples/basic`](examples/basic/main.go).

## Documentation

- API reference: https://pkg.go.dev/github.com/branched-services/go-mcf
- Algorithmic notes and invariants: [`docs/spec.md`](docs/spec.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

See [SECURITY.md](SECURITY.md). Report vulnerabilities privately via GitHub
Security Advisories.

## Acknowledgements

This project is derived from [LEMON 1.3.1](https://lemon.cs.elte.hu). See
[NOTICE](NOTICE) for attribution.

## License

Boost Software License 1.0 — see [LICENSE](LICENSE).
