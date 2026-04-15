# github.com/branched-services/go-mcf — Go Port of LEMON Network Simplex
## Implementation Specification

---

## Overview

This is a standalone, open-source Go port of the primal Network Simplex algorithm
for minimum-cost flow, based on LEMON 1.3.1's `network_simplex.h`.

**Source reference:** LEMON 1.3.1 `network_simplex.h`
Primary GitHub mirror: https://github.com/networkx/networkx-lemon/blob/main/src/lemon/network_simplex.h
Secondary mirror: https://github.com/bekaus/lemon-1.2.1/blob/master/lemon/network_simplex.h

**License:** LEMON is distributed under the Boost Software License 1.0, which is
permissive and allows use in proprietary software. The port must include the required
attribution in its LICENSE file.

---

## Repository Structure

This is a standalone public Go module — not embedded in any private codebase.
Downstream consumers import it as a dependency.

**Module path:** `github.com/branched-services/go-mcf`
**Repository:** https://github.com/branched-services/go-mcf

```
go-mcf/
  mcf.go               // public API: types and Solve function
  solver.go            // Network Simplex implementation
  solver_test.go       // all tests
  go.mod               // module github.com/branched-services/go-mcf
  go.sum
  README.md
  LICENSE              // Boost Software License 1.0 + attribution to LEMON
```

---

## Background: Why Network Simplex

The algorithm this port replaces (SSP with SPFA) requires patched negative-cycle
handling that permanently mutates the graph and is difficult to prove correct. The
root cause is that cross-pool arbitrage cycles produce negative-cost cycles in the
original graph, which SSP is not designed to handle.

Network Simplex is immune to this problem. Negative costs are handled natively
through the pivot mechanics — no special-casing, no graph mutation, no cycle-breaking
machinery required.

---

## Public API

The package exposes a minimal, self-contained interface. It has no knowledge of any
downstream graph representation. Callers translate their internal types into this
structure before calling Solve.

**Arc** represents a directed arc with an integer cost and 256-bit capacity. Flow
values are updated in-place on the arc slice after a successful solve.

**Result** carries the total flow pushed from source to sink, the total cost
(best-effort for large flows), and whether a feasible solution was found.

**Solve** takes a context, a slice of arcs, the number of nodes, and source/sink
node indices. It returns a Result and an error. Possible errors are ErrInfeasible
(supply cannot reach demand) and a wrapped context error if the context is cancelled
mid-solve.

The flow type is `*uint256.Int` from `github.com/holiman/uint256`. Costs are `int64`.
This split is intentional and must be maintained throughout the implementation —
token amounts in DeFi routing require full 256-bit precision; costs derived from
log-rates and fees fit safely in int64.

---

## Algorithm Overview

The agent should read LEMON's `network_simplex.h` directly and use it as the
authoritative implementation reference. The following is orientation, not prescription.

### Initialisation

An artificial root node is added. Each real node gets an artificial arc connecting
it to root, carrying the node's supply or demand as initial flow. These artificial
arcs form the initial spanning tree, which is always feasible. Real arcs start as
non-tree arcs at their lower bound (flow = 0).

### Pivot Loop

Each iteration consists of four steps:

1. **Pricing** — scan non-tree arcs to find one with a negative reduced cost
   (the entering arc). Use block search: partition arcs into blocks of
   approximately sqrt(N) arcs, scan one block per iteration, rotate the starting
   block each call. This is LEMON's default and gives the best empirical performance.
   Stop when no entering arc exists — the solution is optimal.

2. **Join node** — adding the entering arc to the spanning tree creates exactly one
   cycle. Find the cycle's apex (LCA of the arc's endpoints in the tree) by walking
   up from both endpoints using parent pointers.

3. **Leaving arc** — walk the cycle from both endpoints to the join node, tracking
   the arc with minimum residual capacity. This bottleneck arc leaves the tree.
   Compute the bottleneck using uint256 comparisons.

4. **Pivot** — push the bottleneck flow around the cycle (uint256 arithmetic on arc
   flows). Swap the entering arc into the tree and the leaving arc out. Update node
   potentials for the affected subtree only — this is O(subtree size), not O(n), and
   is the key to the algorithm's practical performance. Potential updates are pure
   int64 arithmetic.

### Feasibility Check

After the pivot loop terminates, check whether any artificial arc still carries
nonzero flow. If so, supply cannot meet demand — return ErrInfeasible.

### Degeneracy

Degenerate pivots (bottleneck = 0) are normal and do not indicate a bug. When the
entering arc equals the leaving arc, flip the arc's state (lower ↔ upper) without
modifying the tree. This is sufficient to prevent cycling in practice.

---

## Type Constraints

- **Flow / capacity:** `*uint256.Int` — all arc capacity and flow fields. Arithmetic
  in the augmentation step must use uint256 in-place operations. Avoid allocating
  new uint256 values inside the pivot hot path.
- **Costs / potentials:** `int64` — reduced cost computation and potential updates
  must guard against int64 overflow. Sentinel values near MaxInt64 / MinInt64 should
  be treated as unreachable arcs and skipped during pricing.
- **Total cost tracking:** best-effort. When the bottleneck flow fits in a uint64
  that is also within int64 range, accumulate cost normally. Otherwise skip.
- **Tree structure:** plain int slices — parent, predecessor arc, thread order,
  reverse thread, successor count, last successor, arc direction, arc state. No
  uint256 involved.

---

## What to Port from LEMON

Port the block search pivot rule and the core spanning tree pivot. Do not port:

- Dantzig or first-eligible pivot rules
- The `PARTIAL_AUGMENT` supply type
- C++ template machinery or graph abstraction layers
- The `supplyMap` / `stSupply` setup variants (the public API handles supply setup directly)

The core implementation should be approximately 400–500 lines of Go excluding tests.

---

## What Not to Do

- Do not use Bonneel's `network_simplex_simple.h` as a reference. That implementation
  is hardcoded for dense bipartite (Earth Mover's Distance) problems and cannot
  represent multi-hop sparse graphs.
- Do not implement SPFA, Dijkstra, Bellman-Ford, or any shortest-path subroutine.
  Network Simplex does not use shortest paths.
- Do not add a maximum iteration limit. Network Simplex terminates by proof, not by
  a user-supplied bound. Context cancellation is the only early-exit mechanism.

---

## Testing Strategy

Tests live in `solver_test.go`. The strategy has four layers, ordered from most
foundational to most domain-specific.

### Layer 1: LEMON's Own Test Suite (Mirror)

LEMON's `min_cost_flow_test.cc` is the canonical correctness reference. Locate this
file in any LEMON mirror (e.g., `hcorrada/lemon-mirror` on GitHub under `test/`) and
port its test cases directly to Go. These cover:

- Small hand-constructed graphs with known optimal cost and flow
- Feasibility and infeasibility detection
- Zero-supply / zero-demand edge cases
- Graphs with parallel arcs
- Graphs with negative-cost arcs
- The `checkMcf` helper pattern — after each solve, verify flow conservation at
  every node, arc flows within capacity bounds, and total cost matching expected value

Porting these tests is non-negotiable. They represent the ground truth the original
implementation was validated against.

### Layer 2: Correctness Invariants

For any valid solution, these properties must hold regardless of input:

- Flow conservation: at every non-source, non-sink node, inflow equals outflow
- Capacity feasibility: every arc's flow is between 0 and its capacity (inclusive)
- Optimality certificate: all non-tree arcs at lower bound have non-negative reduced
  cost; all non-tree arcs at upper bound have non-positive reduced cost
- Cost consistency: sum of (arc.cost × arc.flow) over all real arcs equals the
  reported TotalCost (within best-effort truncation bounds)

These should be implemented as a reusable `checkSolution` helper called after every
test case, not as standalone tests.

### Layer 3: Structural Patterns

These cases reflect graph topologies the solver must handle correctly for the
intended downstream use:

- **Negative-cost cycles in the original graph** — a cycle of arcs with negative
  total cost. The solver must produce a correct optimal solution without any special
  handling. This is the primary correctness gap being closed by this port.
- **Parallel arcs with strictly increasing costs** — models piecewise-linearised
  concave cost functions. Verify the solver always saturates lower-cost arcs before
  higher-cost ones.
- **Sparse graphs with many intermediate nodes** — verify correctness on multi-hop
  paths where most node pairs have no direct arc.
- **Single-unit flow** — optimal flow is exactly 1 unit through a single path.
- **Maximum flow** — optimal solution saturates all paths to sink.
- **Large uint256 capacities** — capacities that exceed uint64 range to verify
  the uint256 path is exercised, not silently truncated.

### Layer 4: DIMACS Benchmark Instances

LEMON publishes a benchmark dataset used in the paper:

> Péter Kovács, "Minimum-cost flow algorithms: an experimental evaluation,"
> Optimization Methods and Software, 30:94–127, 2015.

The dataset is available at:
http://lemon.cs.elte.hu/trac/lemon/wiki/MinCostFlowData

Families include NETGEN, GRIDGEN, GOTO, GRIDGRAPH, ROAD, and VISION instances, all
in standard DIMACS format with integer data. Implement a DIMACS parser and run the
solver against a representative subset (at minimum the smaller NETGEN and GRIDGEN
instances). Run these as benchmarks via `go test -bench`, not as unit tests. Verify
that the solver's output cost matches the known optimal for each instance, and record
solve time per instance for regression tracking.

Note: DIMACS instances use integer capacities within int64 range. The uint256
capacity type handles these without issue. The DIMACS parser should produce uint256
values from the integer inputs.

---

## Integration Notes for Downstream Consumer

The downstream private codebase adapts its internal graph representation to the
public API through a thin translation layer. These constraints are enforced in the
downstream code, not in this package, but are documented here to clarify design intent:

- **Gas cost on first slice only.** When a pool is split into k parallel arcs
  (piecewise linearisation of AMM price impact), gas cost appears only on the first
  slice arc. The solver's guarantee that lower-cost arcs are saturated first ensures
  gas is charged exactly once per pool. This is a graph construction concern, not a
  solver concern.

- **Context cancellation returns partial result.** On cancellation, the solver
  returns the flow and cost accumulated so far alongside the context error, allowing
  the caller to decide whether partial results are usable.

---

## Attribution

This package is a port of the Network Simplex implementation from the LEMON graph
library (https://lemon.cs.elte.hu), copyright © 2003-2018 by the Egerváry Research
Group on Combinatorial Optimization. LEMON is distributed under the Boost Software
License 1.0. The LICENSE file in this repository must include the full Boost Software
License text and the above attribution.