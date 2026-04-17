# Contributing to go-mcf

Thanks for your interest. This project ports a specific algorithm (LEMON's
primal Network Simplex) and intentionally keeps a narrow scope: min-cost flow
with `*uint256.Int` flows/capacities and `int64` costs. Contributions that
preserve that scope are welcome.

## Ground rules

- **Scope.** Bug fixes, correctness patches, and performance work on the
  existing algorithm are in scope. New algorithms, alternate numeric types,
  or max-flow/partial-fill modes are out of scope — open an issue first.
- **License.** Every new Go file must begin with `// SPDX-License-Identifier:
  BSL-1.0`. Code derived from LEMON must be noted in `NOTICE`.
- **No mocks or stubs.** Tests exercise the real solver. Follow the existing
  `layer1_test.go` / `layer3_test.go` / invariant-helper patterns.

## Workflow

1. Fork, branch from `main` (e.g. `feat/short-name` or `fix/short-name`).
2. Make your change. Run the full gate:
   ```
   make ci
   ```
   This runs `gofmt -l`, `go vet`, `go test -race`, and `govulncheck`.
3. Update `CHANGELOG.md` under `## [Unreleased]`.
4. Open a PR. Fill in the template. Link any related issue.

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(pivot): switch to block-search entering-arc rule
fix(validate): reject negative capacities
test(layer1): add LEMON parity cases for negative-cost cycles
chore(repo): bump go directive to 1.23
```

## Dev tools

Install once:

```
go install golang.org/x/vuln/cmd/govulncheck@latest
go install golang.org/x/tools/cmd/goimports@latest
```

`make help` lists all available targets.

## Running the examples

```
go test ./... -run Example   # runs godoc examples and checks their Output
go run ./examples/basic      # runs the standalone example binary
```

## Reporting bugs

Open an issue with a minimal reproducer: the `[]Arc` slice, `n`, `source`,
`sink`, `demand`, and the observed vs. expected result. If the bug is a
security issue, follow [SECURITY.md](./SECURITY.md) instead.

## Community

By participating, you agree to the [Code of Conduct](./CODE_OF_CONDUCT.md).
