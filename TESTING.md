# Locus Testing Guide

## Test Categories

### Structural Tests

`tests/structural/boundary_test.go` enforces Clean Architecture layer boundaries:

- Domain must NOT import Application or Infrastructure.
- Application must NOT import Infrastructure.

Run with:

```powershell
go test ./tests/structural/...
```

### Unit Tests

Unit tests live alongside the code they test (e.g., `service_test.go` next to `service.go`). Write table-driven tests for service logic, passing mock repository implementations via the repository interfaces.

Run with:

```powershell
go test ./internal/...
```

### Integration Tests

Integration tests use a real in-memory SQLite database. The schema in `persistence.Open` creates all tables including `focus_sessions`, so focus-related code can be tested end-to-end without a running tracker.

Example:

```go
db, _ := persistence.Open(":memory:")
defer db.Close()
repo := persistence.NewSQLiteCommandRepository(db)
svc := service.NewCommandService(repo)
// ... test svc methods
```

## Mocking Repositories

All repository interfaces are in `internal/domain/repository/`. To mock them in tests, implement the interface directly in the test file:

```go
type mockCommandRepo struct {
    commands []entity.Command
}
func (m *mockCommandRepo) List(ctx context.Context, stageId *entity.StageId) ([]entity.Command, error) {
    return m.commands, nil
}
// ... implement remaining methods
```

## Focus Reader Testing

The `FocusReader` interface in `service/focus_service.go` accepts any implementation. In tests, provide a stub that returns predictable data without requiring the focus tracker to be running or the database to contain real sessions:

```go
type stubFocusReader struct{}

func (s *stubFocusReader) GetFocusDataForSessions(
    sessions []service.FocusSessionWindow,
) dto.FocusDataDTO {
    return dto.FocusDataDTO{
        Available:    true,
        TotalSeconds: 3600,
        Apps: []dto.AppFocusDTO{
            {ExePath: `C:\tools\editor.exe`, FriendlyName: "Editor", TotalSeconds: 3600},
        },
    }
}
```

## Focus Tracker Testing

`focustracker.Tracker` takes a `*sql.DB`. In tests, open an in-memory database, apply the schema, and inject it:

```go
db, _ := persistence.Open(":memory:")
tracker := focustracker.New(db)
tracker.Start()
// ... do work
tracker.Stop()
// query focus_sessions to assert rows were written
```

The tracker is Windows-only (`//go:build windows`). Tests that instantiate it directly will only compile and run on Windows.
