# github.com/branched-services/go-mcf — Go Port of LEMON Network Simplex
## Implementation Specification

---

## Overview

A standalone, open-source Go port of the primal Network Simplex algorithm for
minimum-cost flow, based on LEMON 1.3.1's `network_simplex.h`. Solves min-cost
flow with fixed per-node supply/demand: route exactly `demand` units from source
to sink at minimum cost, or return `ErrInfeasible`. Strict-fill only — no partial
fills, no max-flow mode.

The library is DeFi-oriented: flow and capacity are `*uint256.Int`, costs are
`int64`. General min-cost-flow users with int64 flow should use a different
library.

**Source reference:** LEMON 1.3.1, file `lemon/network_simplex.h`.
Canonical source: https://lemon.cs.elte.hu/pub/sources/lemon-1.3.1.tar.gz
GitHub mirrors exist but are unofficial; treat them as browsing convenience
only, not as the source of truth.

**License:** LEMON is distributed under the Boost Software License 1.0. Every
ported Go file carries the SPDX header `// SPDX-License-Identifier: BSL-1.0`.
The full Boost license text and LEMON attribution appear in `LICENSE` and
`NOTICE`.

---

## Repository Structure

Standalone public Go module. Downstream consumers import it as a dependency.

**Module path:** `github.com/branched-services/go-mcf`
**Repository:** https://github.com/branched-services/go-mcf

```
go-mcf/
  mcf.go               // public API: types and Solve function
  solver.go            // Network Simplex implementation
  solver_test.go       // all tests
  go.mod
  go.sum
  README.md            // declares DeFi/uint256 scope up front
  LICENSE              // Boost Software License 1.0 verbatim
  NOTICE               // LEMON attribution
```

---

## Background: Why Network Simplex

The algorithm this port replaces (SSP with SPFA) requires patched negative-cycle
handling that permanently mutates the graph and is difficult to prove correct.
The root cause is that cross-pool arbitrage cycles produce negative-cost cycles
in the original graph, which SSP is not designed to handle.

Network Simplex is immune. Negative costs are handled natively through the pivot
mechanics — no special-casing, no graph mutation, no cycle-breaking machinery.

---

## Public API

```go
type Arc struct {
    From, To int
    Cost     int64
    Capacity *uint256.Int
    Flow     *uint256.Int // written in-place on success
}

type Result struct {
    TotalFlow *uint256.Int // always equals demand on success; invariant sanity check
    TotalCost int64        // best-effort; see Type Constraints
}

func Solve(
    ctx context.Context,
    arcs []Arc,
    n, source, sink int,
    demand *uint256.Int,
) (Result, error)

var (
    ErrInfeasible = errors.New("mcf: demand cannot be routed from source to sink")
    // Context cancellation returns the wrapped ctx.Err() with zero Result.
)
```

Semantics: `Solve` routes exactly `demand` units from `source` to `sink` at
minimum cost. On success, `Result.TotalFlow == demand` (verified as a
post-condition). If the graph cannot carry `demand` units from source to sink,
returns `(Result{}, ErrInfeasible)`. On context cancellation, returns
`(Result{}, ctx.Err())` — no partial results.

There is no `SolveMaxFlow` wrapper and none will be added. Callers who need
max-flow semantics must compute their own max-flow upper bound and pass it as
demand, or use a different library.

**Type split rationale.** `*uint256.Int` flow/capacity is required for DeFi
token-amount precision. `int64` costs are derived from log-rates and fees that
fit safely. This split must be preserved throughout the implementation.

---

## Numerical Preconditions

Validated at `Solve` entry; violations return an error before any pivot work.

- **Node count:** `n >= 2`, `0 <= source, sink < n`, `source != sink`.
- **Demand:** non-nil, non-zero.
- **Arcs:** `From, To` in `[0, n)`. Self-loops (`From == To`) are rejected.
- **Arc costs:** `|cost| * (n + 1) < MaxInt64 / 8`. This bound ensures reduced-
  cost arithmetic and the Big-M comparison cannot overflow int64.
- **Arc capacities:** non-nil. Capacities must be finite and reasonable;
  passing extremely large (e.g. near-MaxUint256) capacities on a cycle with
  negative reduced cost produces unbounded pivoting. Callers are responsible
  for supplying realistic bounds. Zero capacity is allowed (inert arc).
- **Lower bounds:** arcs are implicitly `0 <= flow <= capacity`. LEMON's
  general `lower <= flow <= upper` is not exposed; callers transform
  lower-bounded arcs themselves.
- **Big-M constant:** `M = MaxInt64 / (8 * (n + 1))`. Used as the cost on
  artificial arcs connecting each node to the internal root during
  initialization. The `* (n + 1)` factor bounds potential accumulation along
  the longest possible tree path; the `/ 8` factor leaves headroom for
  reduced-cost subtraction. The arc-cost validation above guarantees
  `real_cost * (n + 1) < M`, so any artificial arc's reduced cost strictly
  dominates any real arc's reduced cost — artificial arcs leave the basis
  before optimality unless the problem is infeasible.

---

## Algorithm Overview

The agent should read LEMON's `network_simplex.h` directly and use it as the
authoritative implementation reference. The following is orientation.

### Initialisation

An artificial root node is added. Each real node gets an artificial arc
connecting it to root, carrying that node's supply (`+demand` at source,
`-demand` at sink, `0` elsewhere). Artificial arcs form the initial spanning
tree, which is trivially feasible. Real arcs start as non-tree arcs at lower
bound (flow = 0). Artificial arcs carry cost `M` (see Numerical
Preconditions) and are an internal initialization detail not exposed through
the API.

### Pivot Loop

1. **Pricing (block search).** Partition non-tree arcs into blocks of ~sqrt(E).
   Scan one block per iteration, rotating the starting block each call. Within
   the block, select the arc with the largest **state-dependent violation**:

   ```
   violation = (state == STATE_LOWER) ? -reduced_cost
             : (state == STATE_UPPER) ? +reduced_cost
             : 0
   ```

   An arc is eligible only if `violation > 0`. Lower-bounded arcs enter when
   reduced cost is negative (pushing flow reduces cost); upper-bounded arcs
   enter when reduced cost is positive (reducing flow reduces cost). Without
   this sign flip, saturated arcs can never re-enter the basis. Stop when no
   block contains an eligible arc — optimal.

2. **Join node.** Adding the entering arc creates one cycle in the tree. Find
   the cycle apex (LCA of the arc's endpoints) by walking up from both
   endpoints via parent pointers, using `succ_num` to equalize depth.

3. **Leaving arc.** Walk the cycle from both endpoints to the join node,
   tracking the arc with minimum residual capacity (bottleneck). Compute the
   bottleneck using uint256 comparisons against a pre-allocated scratch
   buffer on the solver struct.

   **Tie-breaking is not optional.** Among candidates tied at the minimum
   bottleneck, LEMON picks the one that preserves the *strongly feasible
   spanning tree* invariant — specifically, the arc closest to the join node
   on the side opposite the entering arc's orientation. This is the real
   anti-cycling mechanism; Bland's rule alone is insufficient here. The
   invariant must be stated explicitly in code comments so future readers do
   not delete the tie-breaking as "redundant."

4. **Pivot.** Push bottleneck flow around the cycle using uint256 in-place
   arithmetic on arc flow pointers. Swap the entering arc into the tree and
   the leaving arc out. Update node potentials for the affected subtree only
   — O(subtree size), int64 arithmetic. This selective update is the key to
   practical performance.

### Degeneracy

Degenerate pivots (bottleneck = 0) are expected. When the entering arc equals
the leaving arc, flip the arc's state (`STATE_LOWER` ↔ `STATE_UPPER`) without
modifying the tree. The strongly-feasible-tree invariant (maintained by
leaving-arc tie-breaking above) guarantees the algorithm cannot cycle — this
is a termination proof, not a heuristic.

### Feasibility Check

After the pivot loop terminates, if any artificial arc still carries nonzero
flow, the problem is infeasible → return `ErrInfeasible`.

### Termination

No maximum iteration limit. Network Simplex with the strongly-feasible tree
invariant and block-search pricing has polynomial termination. Context
cancellation is the only early-exit mechanism and returns `(Result{}, ctx.Err())`.

---

## Type Constraints

- **Flow / capacity:** `*uint256.Int`. All arithmetic is in-place (`Add`, `Sub`,
  `Set`). No `+`/`-`. Scratch buffers for bottleneck computation and flow
  updates are preallocated on the solver struct — zero allocations in the
  pivot hot path.
- **Costs / potentials:** `int64`. Overflow is prevented by the Numerical
  Preconditions. Sentinel values near MaxInt64/MinInt64 are treated as
  unreachable and skipped in pricing.
- **Total cost tracking:** best-effort. When bottleneck × cost fits safely in
  int64 (bottleneck ≤ MaxInt64 and product does not overflow), accumulate.
  Otherwise skip. Document that `TotalCost` is advisory for very large flows.
- **Tree structure:** plain `[]int` slices — `parent`, `pred_arc`, `thread`,
  `rev_thread`, `succ_num`, `last_succ`, `direction`, `state`. No uint256.
  This layout matches LEMON exactly and must not deviate.

---

## Memory Layout

Public `Arc` struct is AoS for caller ergonomics. Internally, `Solve` may
convert to SoA at entry: the pricing loop reads `(cost, state, pi[from], pi[to])`
for each non-tree arc and benefits from cache-dense int64/int32 columns
separated from the uint256 pointer columns. This conversion is an internal
optimization knob — implement AoS first; SoA is a documented future
optimization with a benchmark gate.

---

## What to Port from LEMON

Port block search pivot rule and the core spanning tree pivot (join, leaving
arc with tie-breaking, pivot, subtree potential update). Do not port:

- Dantzig or first-eligible pivot rules
- `PARTIAL_AUGMENT` supply type
- C++ template machinery or graph abstraction layers
- `supplyMap` / `stSupply` setup variants — the public `demand` argument
  handles supply setup directly (`+demand` at source, `-demand` at sink)
- General lower-bound support (`lower <= flow <= upper`)

Core implementation target: ~400–500 lines of Go excluding tests.

---

## What Not to Do

- Do not use Bonneel's `network_simplex_simple.h`. Specialized EMD solver;
  cannot represent sparse multi-hop graphs.
- Do not implement SPFA, Dijkstra, or Bellman-Ford. Network Simplex does not
  use shortest paths.
- Do not add a maximum iteration limit.
- Do not add a max-flow mode or `SolveMaxFlow` wrapper. Strict-fill only.
- Do not return partial results on cancellation.

---

## Testing Strategy

Tests live in `solver_test.go`. Four layers.

### Layer 1: LEMON's Own Test Suite

LEMON's `test/min_cost_flow_test.cc` (in the official source tree) is the
canonical correctness reference. Port its test cases directly:

- Small hand-constructed graphs with known optimal cost and flow
- Feasibility and infeasibility detection
- Zero-supply / zero-demand edge cases
- Parallel arcs
- Negative-cost arcs
- The `checkMcf` helper pattern — after each solve, verify flow conservation
  at every node, flows within capacity, and total cost matching expected

Porting these is non-negotiable.

### Layer 2: Correctness Invariants

Reusable `checkSolution` helper called after every test case:

- **Flow conservation:** every non-source, non-sink node has inflow == outflow
- **Capacity feasibility:** `0 <= flow <= capacity` per arc
- **Optimality certificate:** non-tree arcs at lower bound have reduced cost
  ≥ 0; at upper bound have reduced cost ≤ 0
- **Demand satisfied:** `TotalFlow == demand` exactly
- **Cost consistency:** `Σ arc.cost × arc.flow == TotalCost` (within
  best-effort truncation)

### Layer 3: Structural Patterns

- **Negative-cost cycles in the original graph** — solver must produce
  correct optimum with no special handling. Primary correctness gap closed
  by this port.
- **Parallel arcs with strictly increasing costs** — lower-cost arcs must
  saturate before higher-cost ones (supports piecewise-linearized concave
  costs downstream).
- **Sparse multi-hop graphs** — many intermediate nodes, few direct arcs.
- **Single-unit flow** — `demand = 1`, optimal routes through one path.
- **Demand = graph max-flow capacity** — solver saturates every path to
  sink, succeeds with `TotalFlow == demand`.
- **Demand = graph max-flow capacity + 1** — returns `ErrInfeasible`. This
  is the strict-fill correctness guarantee.
- **Large uint256 capacities** — values exceeding uint64 range to exercise
  the uint256 path.

### Layer 4: DIMACS Benchmark Instances

Dataset: http://lemon.cs.elte.hu/trac/lemon/wiki/MinCostFlowData (NETGEN,
GRIDGEN, GOTO, GRIDGRAPH, ROAD, VISION families, DIMACS format, integer data).

**Ground-truth provenance.** DIMACS files do not ship expected optimal costs.
Run LEMON's own `network_simplex` binary once against each instance in the
test subset and commit outputs to `testdata/dimacs/expected.json`, recording
the LEMON version used. This is the only defensible baseline — do not rely
on third-party "optimal = X" claims, which may use a different cost
normalization.

DIMACS integer capacities cast cleanly into uint256. Run these as `go test
-bench`, not unit tests. Assert cost matches fixture and record solve time
for regression tracking.

---

## Integration Notes for Downstream Consumer

The downstream private codebase adapts its internal graph representation to
the public API through a thin translation layer. These constraints are
enforced downstream, documented here for design intent:

- **Gas cost on first slice only.** When a pool splits into k parallel arcs
  (piecewise linearization of AMM price impact), gas cost appears only on
  the first slice. The solver's lower-cost-first saturation guarantee ensures
  gas is charged exactly once per pool. Graph-construction concern, not a
  solver concern.

- **Trade size is expressed via `demand`, not source-arc capacity.** Callers
  do not cap the source's outgoing capacity to bound the trade. They pass
  the intended trade size as the `demand` argument. The source-arc capacity
  cap pattern used with the old SSP solver must be removed.

- **Context cancellation yields no partial result.** Callers set deadlines
  and handle `ctx.Err()`. They do not receive partial flows.

---

## Attribution

Port of the Network Simplex implementation from the LEMON graph library
(https://lemon.cs.elte.hu), copyright © 2003-2018 by the Egerváry Research
Group on Combinatorial Optimization. LEMON is distributed under the Boost
Software License 1.0. `LICENSE` contains the full Boost license text verbatim;
`NOTICE` contains the LEMON attribution above. Every ported Go source file
begins with `// SPDX-License-Identifier: BSL-1.0`.
