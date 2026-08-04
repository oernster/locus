# Locus: Technical Debt

A standing reference to the project's outstanding technical debt. It records what is still open, weighs whether each item is worth doing and gives the rationale. Every item is a behaviour-preserving internal concern: nothing here proposes reverting a feature or changing any UI or UX behaviour. Scope is the whole repository (the Go backend under `internal/`, the Wails shell, the React front end under `frontend/`, the install scripts and the GitHub Pages site) read against `ARCHITECTURE.md`, `TESTING.md` and `tests/structural/boundary_test.go`.

The Go side of this repository is in good order: 62 files, none over 350 lines, layer boundaries held by an AST scan, `VERSION` as the single source of truth and no hardcoded version anywhere. Everything below concerns the front end, the tooling around both, and one stray file.

---

## 1. The front end has no tests at all

`frontend/package.json` declares four scripts: `dev`, `build`, `lint`, `preview`. There is no `test` script, no Vitest, no jsdom and no testing library in `devDependencies`. `git ls-files frontend` matches nothing named `test` or `spec`.

So nine `.tsx` components and three `.ts` modules, including the board that *is* the product, are exercised by nothing. `make test` runs `go test ./...` and reports success while the entire user-facing half of the application is unverified.

PigeonPost, the other Wails project in the portfolio, tests its front end with Vitest and jsdom against the same stack. That configuration transfers almost unchanged. The first tests worth writing are the ones covering how a tool-call event becomes a card and how the board transitions between states, because that is the logic a Go test cannot reach and a type-check cannot catch.

This is item one because it is the largest untested surface in the repository by a wide margin.

## 2. `Board.tsx` is 1308 lines

`frontend/src/features/commands/Board.tsx` is over three times the 400-line module cap and is the largest file in the repository by a factor of three. Every other file in the tree, Go and TypeScript alike, is under 450.

`tests/structural/boundary_test.go` enforces layer imports and nothing about size, and it walks Go packages, so nothing measures TypeScript at all.

React gives the obvious decomposition for free: the board's rows, the card, the column headers and the state derivation are all extractable as sibling components and hooks without inventing any new concepts. Doing that also makes item 1 tractable, because a 1308-line component is difficult to test at any granularity below the whole thing.

## 3. A Go coverage profile is tracked at repository root

`service` is a tracked, 18.5 KB text file whose first line is `mode: set`, followed by per-line coverage records for `internal/application/service/*.go`. It is the output of `go test -coverprofile=service`, committed by accident because the filename has no extension and does not look like an artefact.

`service.out` and `persist.out` beside it are the same thing and are correctly untracked.

Delete `service` and add `*.out` and coverage profiles to `.gitignore` so the next one cannot be committed either.

## 4. `make test` runs less than the project's own standard

The `test` target is:

```
test:
	go test ./...
```

No `-cover`, no `gofmt`, no `go vet`, no `staticcheck` and no `npm run lint`. `TESTING.md` documents `go test ./internal/... -cover` as the coverage command, so the documented workflow and the `make` target already disagree, and the documented one is the more complete.

The portfolio's Go standard is gofmt, `go vet` and `staticcheck` on every change, with `staticcheck` invoked without a persistent install:

```
go run honnef.co/go/tools/cmd/staticcheck@latest ./...
```

Fold all of it into the `test` target (or a `check` target that `test` depends on) so one command is the gate. The front-end lint script already exists and should be part of the same target once item 1 adds a `test` script beside it.

## 5. Nothing enforces the Go coverage level

Go has no `--cov-fail-under`, so `TESTING.md` records the position in prose and documents which branches are uncoverable. That is honest and it is the right shape for Go, but it means the number can drift downward without anything noticing.

The usual answer is a threshold check in the same target as item 4: run `go test ./internal/... -coverprofile`, parse the total with `go tool cover -func` and fail below a stated figure. It takes a few lines and converts a documented intention into a gate. Pick the figure from what the suite actually achieves today rather than from an aspiration.

## 6. `install.ps1` sets a Run key and there is no test for the uninstall path

`install.ps1` deploys to `%LOCALAPPDATA%\locus\`, writes an autostart Run key and launches the app; `uninstall.ps1` stops the process, removes the key and removes the directory.

These two scripts are the only things in the repository that change the user's machine outside the application's own data, and neither is tested or exercised by CI. An uninstall that leaves the Run key behind means a deleted application that Windows tries to start at every login.

A PowerShell test is awkward and probably not worth it. What is worth it is making the pair symmetric by construction: have `install.ps1` write the list of everything it created (key path, install directory, shortcut paths) to a manifest file in the install directory, and have `uninstall.ps1` read that manifest rather than repeating the paths. Then the two cannot drift, which is the actual failure mode.

---

## Looks like debt, not worth touching

- `internal/application/service/snapshot_service_test.go` at 449 lines. Over the cap and it is a test file, so it counts, but the Go side has no size rule to enforce and this is the only offender. Worth splitting when next touched.
- The `hooks/` directory at root holding the Claude Code hook scripts. It is the integration surface, not application code.
- `locus.exe` and `build/` in the working tree. Build output, correctly untracked.
- The six tracked PNGs plus the `.ico`. Icon set from a single master, consumed by the Wails build and the site.
- `app.go` and `icon_windows.go` at root beside `main.go`. The Wails convention puts the bound struct at root; this is the framework's shape, not a layering violation.

## Not debt (do not "fix" these)

These look like candidates but are correct as they stand; changing them would regress or add cost for nothing.

- **`tests/structural/boundary_test.go`.** Domain forbidden from importing application and infrastructure, application forbidden from importing infrastructure, held by import scan. The same invariant the Python projects enforce, in Go, with no framework. Item 2 asks for a size rule beside it, not a replacement.
- **`VERSION` at root with nothing hardcoding a version string.** Verified clean across the whole tree, Go and TypeScript. One of the tidiest version stories in the portfolio.
- **The `internal/{domain,application,infrastructure}` layout with a root composition root.** The portfolio's Go structure, correctly applied.
- **`TESTING.md` documenting uncoverable branches explicitly** rather than hiding them. Naming what cannot be covered, and why, is what makes a coverage number mean anything in Go.
- **The absence of CGO.** Pure Go, single binary, direct Win32. Deliberate and load-bearing.
- **`install.ps1` and `uninstall.ps1` as PowerShell rather than a bespoke installer.** Locus is a background tool with no UI to install, so a per-user script pair is proportionate. Item 6 concerns their symmetry, not their existence.
