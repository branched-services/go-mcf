// SPDX-License-Identifier: BSL-1.0

// Package mcf provides a primal Network Simplex min-cost-flow solver ported
// from LEMON 1.3.1. It uses [github.com/holiman/uint256.Int] for flow and
// capacity values and int64 for costs, targeting DeFi applications that
// require 256-bit unsigned arithmetic on EVM-compatible networks.
package mcf
