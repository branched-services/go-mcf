# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- OSS scaffolding: CI, community health files, issue/PR templates,
  godoc examples, contributor documentation.

## [0.1.0] - 2026-04-17

### Added
- Initial Go port of LEMON 1.3.1 primal Network Simplex min-cost-flow solver.
- DeFi-oriented numeric domain: uint256 flow/capacity, int64 costs.
- Layer 1 (algorithmic) and Layer 3 (property-based) test suites.
- `checkSolution` invariant verification.
- Context cancellation support in `Solve`.
- Strict-fill semantics with configurable tolerance for negative-cost cycles.
