## Feature: Network Simplex Min-Cost Flow Solver (LEMON Port)

## Summary

A standalone, open-source Go port of the primal Network Simplex algorithm for
minimum-cost flow, based on LEMON 1.3.1's `network_simplex.h`. Solves min-cost
flow with fixed per-node supply/demand: routes exactly `demand` units from a
source to a sink at minimum cost, or returns `ErrInfeasible`. Strict-fill only
— no partial fills, no max-flow mode. DeFi-oriented: flow and capacity are
`*uint256.Int`, costs are `int64`.

## Background

The downstream consumer previously used a Successive-Shortest-Path solver with
SPFA that required patched negative-cycle handling — permanently mutating the
graph and difficult to prove correct. The root cause was cross-pool arbitrage
cycles producing negative-cost cycles in the original graph. Network Simplex
handles negative costs natively through pivot mechanics — no special-casing,
no graph mutation, no cycle-breaking machinery — which is why this port exists.

A prior partial implementation of this solver was merged into `main` through an
earlier (`.flywheel`) task system and is now discarded. This feature rebuilds
from zero under the current `.workflow/` task system using the authoritative
`docs/spec.md` as the reference and LEMON 1.3.1 `network_simplex.h` as the
algorithmic source of truth.

**Authoritative references:**
- `docs/spec.md` — project-level implementation specification (this feature
  derives from it and must stay consistent with it).
- LEMON 1.3.1 `lemon/network_simplex.h` — algorithmic source of truth.
  Canonical tarball: https://lemon.cs.elte.hu/pub/sources/lemon-1.3.1.tar.gz.
- LEMON `test/min_cost_flow_test.cc` — canonical correctness tests to port.

## Scope

### In Scope

- Public Go API: `Arc`, `Result`, `Solve`, `ErrInfeasible`, plus a single
  validation sentinel `ErrInvalidInput` (see FR-9).
- Primal Network Simplex algorithm with block-search pricing, strongly-
  feasible-tree invariant, and subtree-only potential updates.
- Big-M artificial-arc initialization and feasibility check on termination.
- `*uint256.Int` in-place arithmetic on flow/capacity; `int64` costs and
  potentials; scratch buffers preallocated on the solver struct (zero
  allocations in the pivot hot path).
- Validation of all numerical preconditions listed in `docs/spec.md`
  ("Numerical Preconditions") at `Solve` entry, before any pivot work.
- Context cancellation support: `(Result{}, ctx.Err())` on cancel, no partial
  results.
- Test Layer 1 — port of LEMON's `test/min_cost_flow_test.cc` cases.
- Test Layer 2 — `checkSolution` invariant helper asserted after every test.
- Test Layer 3 — structural pattern tests (negative-cost cycles, parallel arcs
  with strictly increasing costs, sparse multi-hop graphs, single-unit flow,
  demand = max-flow, demand = max-flow + 1 = infeasible, large uint256
  capacities).
- Legal/attribution artifacts: `LICENSE` (Boost Software License 1.0 verbatim),
  `NOTICE` (LEMON attribution), and `// SPDX-License-Identifier: BSL-1.0`
  header on every ported Go source file.
- `README.md` declaring the DeFi/uint256 scope up front plus a minimal public
  API summary and license reference.
- `go.mod` at module path `github.com/branched-services/go-mcf` targeting
  Go 1.22.

### Out of Scope

- **DIMACS parser and benchmark harness** (Layer 4 of the testing strategy).
  No `testdata/dimacs/`, no `go test -bench` instances, no LEMON-generated
  `expected.json` fixtures in this feature. Documented as future work.
- **Struct-of-Arrays memory layout.** Implement AoS `Arc` only. AoS→SoA
  conversion at `Solve` entry is a documented future optimization gated on a
  benchmark; not built now.
- **`SolveMaxFlow` wrapper / max-flow mode / partial-fill semantics.** Strict-
  fill only. Callers who need max-flow must compute their own max-flow upper
  bound and pass it as `demand`, or use a different library.
- **General lower-bound arcs** (`lower <= flow <= upper`). Public API exposes
  only `0 <= flow <= capacity`. Callers transform lower-bounded arcs
  themselves.
- **Dantzig / first-eligible / partial-augment pivot rules.** Block search is
  the sole pricing rule.
- **Maximum-iteration safety limit.** Termination proof comes from the
  strongly-feasible-tree invariant; no iteration cap.
- **CI/build automation.** No GitHub Actions workflow, no `Makefile` in MVP.
  Users run `go test`, `go vet`, `gofmt` locally.
- **int64-flow variant for general MCF users.** Type split is fixed:
  `*uint256.Int` flow/capacity, `int64` cost.

## Requirements

### Functional Requirements

1. **FR-1 — Public API surface matches `docs/spec.md` exactly.** Exported
   symbols: type `Arc{From, To int; Cost int64; Capacity, Flow *uint256.Int}`,
   type `Result{TotalFlow *uint256.Int; TotalCost int64}`, function
   `Solve(ctx context.Context, arcs []Arc, n, source, sink int, demand *uint256.Int) (Result, error)`,
   sentinels `ErrInfeasible` and `ErrInvalidInput`.
   - Acceptance: `go doc github.com/branched-services/go-mcf` lists exactly
     these symbols; no others exported from the package root.

2. **FR-2 — `Solve` routes exactly `demand` units from `source` to `sink` at
   minimum cost and writes per-arc flows in place into `arcs[i].Flow`.** On
   success, `Result.TotalFlow.Eq(demand)` is true (verified as a
   post-condition inside `Solve`).
   - Acceptance: For every ported LEMON test case and structural test,
     `TotalFlow == demand` holds and per-arc `Flow` values satisfy the
     invariants in FR-7.

3. **FR-3 — Input precondition validation runs before any pivot work.**
   Validated at entry: `n >= 2`; `0 <= source, sink < n`; `source != sink`;
   `demand` non-nil and non-zero; every arc `From, To` in `[0, n)`; no
   self-loops; every `Capacity` non-nil; every `|Cost| * (n + 1) <
   MaxInt64 / 8`. First violation returns `(Result{}, err)` where
   `errors.Is(err, ErrInvalidInput)` is true and the wrapped message
   identifies the specific failure.
   - Acceptance: Unit tests cover each individual precondition failure and
     confirm `errors.Is(err, ErrInvalidInput)`.

4. **FR-4 — Big-M constant and artificial spanning tree.** `M = MaxInt64 /
   (8 * (n + 1))`. Initialization adds an artificial root, one artificial arc
   per real node connecting it to root (carrying `+demand` at source,
   `-demand` at sink, `0` elsewhere), at cost `M`. Artificial arcs form the
   initial spanning tree; real arcs start as non-tree at lower bound
   (`Flow = 0`). Artificial arcs are internal and never surface through the
   public API.
   - Acceptance: Reflection-free test inspects solver state after
     initialization and confirms the tree is spanning and feasible before any
     pivot runs.

5. **FR-5 — Pivot loop: block-search pricing, join-node via LCA, leaving-arc
   with strongly-feasible-tree tie-breaking, in-place uint256 flow push, and
   subtree-only potential update.** Behavior must match `docs/spec.md`
   "Algorithm Overview" precisely, including the sign-flipped violation
   expression for `STATE_UPPER` arcs and the tie-break rule that picks the
   leaving arc closest to the join node on the side opposite the entering
   arc's orientation. Degenerate pivots flip arc state (`STATE_LOWER` ↔
   `STATE_UPPER`) when entering and leaving arcs coincide.
   - Acceptance: Structural tests pass for negative-cost cycles (a case SSP
     cannot handle) and for demand-equals-max-flow-capacity (exact
     saturation).

6. **FR-6 — Feasibility determination on termination.** When the pivot loop
   reaches optimality (no eligible arc in any block), inspect artificial arcs.
   If any artificial arc carries nonzero flow, return
   `(Result{}, ErrInfeasible)`. Otherwise return
   `(Result{TotalFlow, TotalCost}, nil)`.
   - Acceptance: Tests cover both feasible (all artificial arcs at zero) and
     infeasible (at least one artificial arc carrying flow) outcomes;
     `errors.Is(err, ErrInfeasible)` holds for the latter.

7. **FR-7 — Correctness invariants via reusable `checkSolution` helper.**
   After every test solve, assert: flow conservation at every non-source,
   non-sink node (inflow == outflow); `0 <= Flow <= Capacity` per arc;
   optimality certificate (non-tree arcs at lower bound have reduced cost
   ≥ 0; non-tree arcs at upper bound have reduced cost ≤ 0);
   `TotalFlow.Eq(demand)`; and cost consistency
   (`Σ arc.Cost × arc.Flow == TotalCost` within the best-effort truncation
   window defined in FR-10).
   - Acceptance: A single `checkSolution` function is invoked by every
     Layer 1–3 test; individual invariant violations produce a failure that
     names the violated invariant.

8. **FR-8 — Context cancellation yields no partial result.** `Solve` checks
   `ctx.Err()` at pivot-loop boundaries; on cancellation, returns
   `(Result{}, ctx.Err())`. No `Arc.Flow` values are mutated beyond whatever
   pivot was already in progress at the check point (callers must not rely
   on `Flow` values after a cancelled solve).
   - Acceptance: A test cancels the context mid-solve and asserts
     `errors.Is(err, context.Canceled)` and `Result{} == (Result{})`.

9. **FR-9 — Validation errors use a single sentinel.** All precondition
   violations in FR-3 return an error satisfying
   `errors.Is(err, ErrInvalidInput)` with a descriptive wrapped message
   identifying the failed condition. `ErrInfeasible` and `context` errors
   are the only other error classes `Solve` returns.
   - Acceptance: `errors.Is(err, ErrInvalidInput)` is true for every
     precondition failure and false for `ErrInfeasible` and `ctx.Err()`
     cases.

10. **FR-10 — `TotalCost` is best-effort.** When `bottleneck.IsUint64()` is
    true and the product of `int64(bottleneck.Uint64()) × cost` does not
    overflow int64, accumulate into `Result.TotalCost`. Otherwise skip
    accumulation for that pivot. Godoc on `Result.TotalCost` must note that
    the value is advisory for very large flows and may under-report.
    - Acceptance: A test uses a uint256 flow magnitude that forces
      skip-accumulation; `TotalFlow` is correct, `TotalCost` is documented as
      advisory.

11. **FR-11 — Legal and attribution artifacts.** `LICENSE` contains the Boost
    Software License 1.0 text verbatim. `NOTICE` contains LEMON attribution
    (© 2003-2018 Egerváry Research Group on Combinatorial Optimization,
    distributed under BSL-1.0). Every `.go` file in the module begins with
    `// SPDX-License-Identifier: BSL-1.0`.
    - Acceptance: A linter (or a test) walks `*.go` files and asserts the
      SPDX header on line 1; `LICENSE` and `NOTICE` are present at module
      root.

12. **FR-12 — `README.md` declares DeFi/uint256 scope up front.** README must
    state within the first paragraph that this library is DeFi-oriented
    (`*uint256.Int` flow/capacity, `int64` cost) and that general MCF users
    with int64 flow should use a different library. Include a minimal
    public-API code snippet and a license reference.
    - Acceptance: README begins with the type-split declaration; the first
      code block is a working `Solve` example; last section links to
      `LICENSE` and `NOTICE`.

### Non-Functional Requirements

- **Correctness (primary).** Every Layer 1–3 test passes. `checkSolution`
  invariants hold on every solved instance. No special casing for negative-
  cost cycles in the input graph.
- **Performance bar.** Polynomial termination guaranteed by the strongly-
  feasible-tree invariant and block-search pricing. Zero allocations in the
  pivot hot path (scratch buffers preallocated on the solver struct). No
  specific wall-clock target in MVP; DIMACS regression tracking is deferred
  to a future feature.
- **Memory.** AoS public struct; internal SoA conversion deferred. Solver
  state is allocated per `Solve` call (no global pool in MVP).
- **Concurrency.** `Solve` is safe to call concurrently from multiple
  goroutines **on independent inputs**. Each call owns its own solver state.
  Callers must not share the same `[]Arc` slice across concurrent `Solve`
  calls because `Arc.Flow` pointers are written in place. This contract is
  stated in godoc on `Solve`.
- **Determinism.** Given identical inputs (including the rotating block-
  search starting index, which is seeded deterministically from problem
  size), `Solve` produces identical `Arc.Flow` values and `Result`. No RNG.
- **Go version.** `go.mod` declares `go 1.22`.
- **Coverage.** `go test -race -cover ./...` reports ≥ 85% statement
  coverage on non-test files.
- **Style.** `gofmt -d .` produces no diff; `go vet ./...` clean.

## Behavior Specification

### Happy Path

1. Caller builds `[]Arc` with `Capacity` set (non-nil) and `Flow` set to a
   zero-valued `*uint256.Int` (caller-allocated) per arc, plus `n`, `source`,
   `sink`, and a non-nil non-zero `demand`.
2. Caller invokes `Solve(ctx, arcs, n, source, sink, demand)`.
3. `Solve` validates preconditions; on success proceeds to initialization.
4. Initialization builds the Big-M artificial spanning tree and initial
   potentials.
5. Pivot loop runs: block-search pricing selects an entering arc; LCA finds
   the join node; cycle walk finds the leaving arc (with tie-break);
   bottleneck flow is pushed in place; subtree potentials update; state
   flags update. Loop repeats until no block contains an eligible arc.
6. Feasibility check: all artificial arcs carry zero flow.
7. `Solve` writes per-arc `Flow` values (already in place), computes
   `Result.TotalFlow` and best-effort `Result.TotalCost`, and returns
   `(Result, nil)`.

### Error Handling

| Error Condition                                   | Expected Behavior                                              |
| ------------------------------------------------- | -------------------------------------------------------------- |
| `n < 2`                                           | Return `(Result{}, err)` with `errors.Is(err, ErrInvalidInput)` |
| `source` or `sink` out of `[0, n)`                | Return `ErrInvalidInput`-wrapped                               |
| `source == sink`                                  | Return `ErrInvalidInput`-wrapped                               |
| `demand` nil or zero                              | Return `ErrInvalidInput`-wrapped                               |
| Arc `From` or `To` out of `[0, n)`                | Return `ErrInvalidInput`-wrapped                               |
| Arc `From == To` (self-loop)                      | Return `ErrInvalidInput`-wrapped                               |
| Arc `Capacity` is nil                             | Return `ErrInvalidInput`-wrapped                               |
| Arc `|Cost| * (n+1) >= MaxInt64 / 8`              | Return `ErrInvalidInput`-wrapped (overflow guard)              |
| Graph cannot route `demand` from source to sink   | Return `(Result{}, ErrInfeasible)`                             |
| `ctx` cancelled or deadline-exceeded mid-solve    | Return `(Result{}, ctx.Err())`; no partial result              |

### Edge Cases

| Case                                                                   | Expected Behavior                                                                                 |
| ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| Zero-capacity arc (`Capacity.IsZero()`)                                | Permitted; behaves as an inert arc (can never carry flow). Does not trigger `ErrInvalidInput`.    |
| Parallel arcs with strictly increasing costs                           | Lower-cost arc saturates before the higher-cost parallel arc.                                     |
| Negative-cost arc (real input, not artificial)                         | Handled natively; no special casing, no graph mutation.                                           |
| Negative-cost cycle entirely within the input graph                    | Correct optimum found; tested explicitly in Layer 3.                                              |
| `demand` exactly equals the source-to-sink max-flow capacity           | Solver saturates every path to sink and returns success with `TotalFlow.Eq(demand)`.              |
| `demand` exceeds max-flow capacity by one unit                         | Returns `(Result{}, ErrInfeasible)`.                                                              |
| `demand = 1`                                                           | Routes one unit along the min-cost path; returns success.                                         |
| Capacity values exceeding `MaxUint64` (true uint256 regime)            | Flow updates stay correct; `TotalCost` may fall back to "not accumulated" per FR-10.              |
| `len(arcs) == 0` and `demand > 0`                                      | Returns `ErrInfeasible` (no arcs to carry demand; artificial arcs retain nonzero flow).           |
| Caller shares `[]Arc` across concurrent `Solve` calls                  | Undefined. Godoc explicitly forbids this on `Solve`.                                              |
| Degenerate pivot (bottleneck = 0)                                      | Flip state (`STATE_LOWER` ↔ `STATE_UPPER`); tree unchanged; loop continues.                       |
| Sentinel-cost potential (`MaxInt64`/`MinInt64` used as "unreachable")  | Pricing skips arcs touching these nodes; they cannot become entering arcs.                        |

## Technical Context

### Affected Apps

- `go-mcf` (the module itself). Single-package public Go module at
  `github.com/branched-services/go-mcf`. No other apps in this repo.

### Integration Points

- **`github.com/holiman/uint256`** — required dependency for
  `*uint256.Int` flow and capacity. Used exclusively in in-place mode
  (`Add`, `Sub`, `Set`, `Cmp`); no operator-based arithmetic.
- **LEMON 1.3.1** — source reference, not a runtime dependency. The
  algorithmic structure (pivot rule, tree layout arrays, state flags,
  tie-breaking rules) must match LEMON's implementation. Borrow *design*,
  not code; every ported Go file carries `// SPDX-License-Identifier: BSL-1.0`.
- **Downstream consumer (private codebase, not in this repo).** Imports
  `go-mcf` as a dependency; adapts its internal graph representation to
  `[]Arc` through a thin translation layer. Behavioral contracts that the
  downstream relies on (strict-fill, no partial-result-on-cancel, demand
  rather than source-arc-capacity for trade size) are locked by this
  feature's public API.

### Relevant Existing Code

- `docs/spec.md` — authoritative implementation specification. This feature
  spec inherits every decision in it; any deviation must be flagged here.
- Prior (discarded) implementation on `main` HEAD
  (`mcf.go`, `bigm.go`, `init.go`, `join.go`, `leaving.go`, `pivot.go`,
  `potentials.go`, `pricing.go`, `reduced_cost.go`, `state.go`, `types.go`,
  `validate.go`, plus per-file `_test.go`). Not to be consulted during the
  rebuild — full rewrite from zero is a locked decision.

## Decisions Log

| Decision                             | Choice                                                                                               | Rationale                                                                                           |
| ------------------------------------ | ---------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| Starting state                       | Full rewrite from zero                                                                               | Prior `.flywheel`-era files exist at HEAD but are deleted in the working tree; treat as discarded.  |
| File layout                          | Deferred to `/task`                                                                                  | Layout is an implementation concern; spec stays agnostic between "3 files per spec" vs. finer split. |
| Testing MVP                          | Layers 1, 2, 3 only (ported LEMON cases, `checkSolution` invariants, structural patterns)            | User goal: simple yet correct. Layer 4 (DIMACS) deferred.                                           |
| DIMACS parser / benchmark harness    | Out of scope                                                                                         | No external LEMON binary dependency; no large fixtures. Future feature.                             |
| Memory layout                        | AoS public; no internal SoA                                                                          | Spec: "implement AoS first; SoA is a documented future optimization with a benchmark gate."          |
| Validation error shape               | Single `ErrInvalidInput` sentinel wrapping a descriptive message                                     | Idiomatic Go; small public surface; `errors.Is` works; messages carry precise detail.               |
| Concurrency                          | Safe across goroutines on independent inputs; not safe for shared `[]Arc`                            | Per-call state allocation; `Arc.Flow` in-place writes forbid shared slices.                         |
| Validation errors distinct from infeasibility | `ErrInvalidInput` ≠ `ErrInfeasible` ≠ `ctx.Err()`                                           | Callers can distinguish "bad input" from "unroutable demand" from "cancelled."                      |
| Deliverables (non-code)              | `LICENSE` + `NOTICE` + SPDX headers + `README.md`                                                    | Boost license requires attribution; README required for public Go module.                           |
| Godoc on exported symbols            | Not a separately-tracked deliverable in MVP                                                          | User did not flag it as explicit scope; baseline Go doc comments expected but not coverage-audited. |
| Coverage floor                       | ≥ 85% statement coverage                                                                             | Strong but achievable on a single-package solver.                                                   |
| Done bar                             | Layer 1–3 tests green + `go vet` clean + `go test -race` clean + coverage ≥ 85%                      | Matches user-selected acceptance level.                                                             |
| Module path                          | `github.com/branched-services/go-mcf`                                                                | Matches `docs/spec.md`.                                                                             |
| Go version                           | `go 1.22`                                                                                            | Conservative, broadly deployed.                                                                     |
| CI/build automation                  | None in MVP                                                                                          | Deferred to follow-up; users run tests locally.                                                     |
| Iteration cap                        | None                                                                                                 | Strongly-feasible-tree invariant guarantees polynomial termination; cap would mask bugs.            |
| `SolveMaxFlow` / partial-fill        | Will never be added                                                                                  | Strict-fill is a load-bearing contract for downstream; locked in `docs/spec.md`.                    |
| General lower-bound arcs             | Out of scope                                                                                         | Public API exposes `0 <= flow <= capacity`; callers transform themselves.                           |
| Context cancellation                 | Returns `(Result{}, ctx.Err())`; no partial result                                                   | Matches `docs/spec.md` and downstream contract.                                                     |

## Open Questions

None blocking. Two forward-looking items logged here so `/task` does not
accidentally expand scope:

- DIMACS benchmark harness + LEMON-generated `testdata/dimacs/expected.json`
  will be a separate future feature once this one is green. The expected
  values must come from running LEMON's own `network_simplex` binary against
  each instance; no third-party "optimal = X" claims.
- Internal AoS→SoA conversion at `Solve` entry is a candidate future
  optimization; should only land behind a benchmark that demonstrates the
  speedup.

## Next Steps

Run `/task 00002-FEATURE-network-simplex-mcf` to generate the implementation
task breakdown from this spec.
