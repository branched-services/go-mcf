# go-mcf

A Go port of [LEMON 1.3.1](https://lemon.cs.elte.hu)'s primal Network Simplex min-cost-flow solver.

Scope is DeFi-oriented: flows and capacities use `*uint256.Int` (256-bit unsigned integers suitable for EVM token amounts) while costs are `int64`. General-purpose `int64`-flow users should use a different library.

## License

Boost Software License 1.0 -- see [LICENSE](LICENSE).

This project includes code derived from LEMON -- see [NOTICE](NOTICE) for attribution.
