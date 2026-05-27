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

Unit tests live alongside the code they test (e.g., `service_test.go` next to
`service.go`). Write table-driven tests for service logic, passing mock
repository implementations via the repository interfaces.

Run with:
```powershell
go test ./internal/...
```

### Integration Tests

Integration tests use a real in-memory SQLite database. Create a temp DB,
instantiate the real repository implementations, and verify service behaviour
end-to-end.

Example:
```go
db, _ := persistence.Open(":memory:")
defer db.Close()
repo := persistence.NewSQLiteCommandRepository(db)
svc := service.NewCommandService(repo)
// ... test svc methods
```

## Mocking Repositories

All repository interfaces are in `internal/domain/repository/`. To mock them
in tests, implement the interface directly in the test file:

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

The `FocusReader` interface in `service/focus_service.go` accepts any
implementation. In tests, provide a stub that returns predictable data without
requiring the focus-reader database to be present.
