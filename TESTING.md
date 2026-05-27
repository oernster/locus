# Locus Testing Guide

## Running Tests

```powershell
# All tests
go test ./...

# With coverage summary
go test ./internal/... -cover

# Specific package with verbose output
go test ./internal/application/service/... -v
```

## Coverage Summary

| Package | Coverage | Notes |
|---|---|---|
| `domain/entity` | ~100% | Constant and type definitions |
| `application/service` | ~99% | All 6 services with mock repos |
| `infrastructure/persistence` | ~93% | All repos with in-memory SQLite |
| `infrastructure/focusreader` | ~89% | In-memory SQLite + stub appInfoFn |
| `infrastructure/focustracker` | ~53% | Win32 `foregroundExe()` not callable in tests |
| `tests/structural` | 100% | Layer boundary enforcement |

Uncoverable branches (documented, not bugs):
- `sql.Open` error (SQLite uses lazy connection, never fails at open time)
- `res.LastInsertId()` error (SQLite autoincrement never fails post-insert)
- `json.Marshal` error in `SnapshotService.serialise` (basic Go types never fail marshal)
- `foregroundExe()` Win32 calls in `focustracker` (require live Windows session)

## Test Categories

### Structural Tests

`tests/structural/boundary_test.go` enforces Clean Architecture layer boundaries:

- Domain must NOT import Application or Infrastructure.
- Application must NOT import Infrastructure.

### Unit Tests: Services

Service tests live in `internal/application/service/`. All repository dependencies are
injected via interface mocks defined in `mocks_test.go` (same package, white-box).

Each mock supports error injection via fields (`listErr`, `createErr`, etc.) for
testing error propagation paths:

```go
repo := &mockCommandRepo{getErr: errors.New("not found")}
svc := NewCommandService(repo)
_, err := svc.Get(ctx, 1)
// err wraps the injected error
```

### Unit Tests: Domain Entity

`internal/domain/entity/stage_test.go` verifies all `StageId` and `Status` constants
and that `Stages` contains the four IDs in the canonical display order.

### Integration Tests: Persistence

Persistence tests open an in-memory SQLite database using the production schema:

```go
db := newTestDB(t)  // defined in testhelper_test.go
repo := NewSQLiteCommandRepository(db)
```

Error-path tests use a closed DB (`closedDB(t)`) to trigger query failures
and verify that errors are propagated correctly.

### Integration Tests: Focus Reader

`focusreader` tests (`//go:build windows`) inject a stub `appInfoFn` to avoid
real PE version-info lookups and insert rows directly into an in-memory
`focus_sessions` table. Cases covered:

- Empty windows slice
- Single app accumulation
- System-process filtering (`C:\Windows\` prefix)
- Idle gap detection (> 5 min gap between sessions)
- Idle tail gap (window ends long after last session)
- Clamping (focus session extends beyond window boundaries)
- DeepWorkSeconds floor at zero
- Multiple windows merged
- `maxAppsInReport` cap (10 apps)
- Active session (NULL `ended_at`)

### Integration Tests: Focus Tracker

`focustracker` tests (`//go:build windows`) inject a `foregroundExeFn` stub to
avoid real Win32 API calls. Cases covered:

- `New` sets defaults
- `closeStale` ends open sessions from prior crashes
- `startSession` / `endSession` round-trip
- `run` loop switches sessions on exe change
- `Stop` ends the current session
- `Start` closes stale sessions then launches the loop

The real `foregroundExe()` Win32 function is NOT tested (requires a live desktop
session with foreground windows — not feasible in CI).

## Mocking Repositories

All repository interfaces are in `internal/domain/repository/`. The canonical mocks
live in `internal/application/service/mocks_test.go` and support error injection:

```go
type mockCommandRepo struct {
    cmds      []entity.Command
    nextID    int64
    listErr   error
    getErr    error
    createErr error
    updateErr error
    deleteErr error
    reorderErr error
}
```

## Injectable Dependencies for Testing

Two production dependencies have been made injectable (function fields) to enable
unit testing without Windows system APIs:

| Type | Field | Default | Purpose |
|---|---|---|---|
| `SQLiteFocusReader` | `appInfoFn` | `wininfo.GetAppInfo` | PE version-info lookup |
| `focustracker.Tracker` | `foregroundExeFn` | `foregroundExe` | Win32 foreground window query |

Both defaults are set by `New*` constructors. Tests override them with stubs.

## Go Idiom Notes

- All repository interfaces accept `context.Context` as the first argument.
- All DB operations use `QueryContext`/`ExecContext` (not bare `Query`/`Exec`).
- `defer rows.Close()` is used except inside loops where per-iteration close is required.
- `rows.Err()` is checked after every `rows.Next()` loop.
- Errors are wrapped with `%w` where the caller may need to use `errors.Is`.
- No `init()` functions. No global mutable state outside of the `wininfo` cache (which is safe via `sync.RWMutex`).
