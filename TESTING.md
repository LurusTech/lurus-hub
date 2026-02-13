# Testing Guide

## Quick Start

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage report
go test -cover ./...
```

---

## Test Commands Reference

### Basic Commands

```bash
# ✅ CORRECT - Run all tests
go test ./...

# ✅ CORRECT - Run tests for specific package
go test ./internal/adapter/handler/
go test ./internal/app/

# ✅ CORRECT - Run tests with verbose output
go test -v ./internal/adapter/handler/

# ❌ INCORRECT - Do NOT run individual test files
# This will fail with "undefined: repo", "undefined: common", etc.
go test ./internal/adapter/handler/alipay_test.go
```

**Why?** Individual test files lack the full package context (imports, dependencies). Always test at the package level or use `./...` for all packages.

---

## Test Types

### Unit Tests

```bash
# Run only fast unit tests (skip integration tests)
go test -short ./...

# Run unit tests with coverage
go test -short -cover ./...
```

### Integration Tests

```bash
# Run only integration tests
go test -run Integration ./...

# Integration tests require:
# 1. Database connection (PostgreSQL)
# 2. Redis connection
# 3. Meilisearch connection
```

**Note**: Integration tests are automatically skipped when running `go test -short` (via `testing.Short()` check).

---

## Coverage Analysis

### Generate Coverage Report

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

### Coverage Targets (from CLAUDE.md)

| Layer | Target Coverage |
|-------|----------------|
| `internal/app/` | ≥ 80% |
| `internal/adapter/repo/` | ≥ 60% |
| `internal/adapter/handler/` | ≥ 50% |

### Check Coverage by Package

```bash
# Coverage for specific packages
go test -cover ./internal/app/
go test -cover ./internal/adapter/handler/
go test -cover ./internal/adapter/repo/
```

---

## Race Detection

```bash
# Run tests with race detector (detects concurrent access bugs)
go test -race ./...

# Run race detector on specific package
go test -race ./internal/adapter/handler/
```

**Recommended**: Always run race detector before merging to main.

---

## Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./...

# Run benchmarks for specific package
go test -bench=. ./internal/adapter/handler/

# Run specific benchmark
go test -bench=BenchmarkSyncAllChannelModels ./internal/adapter/handler/

# Benchmark with memory stats
go test -bench=. -benchmem ./...
```

---

## Test by Feature

### Alipay OAuth Tests

```bash
# Run all Alipay tests
go test -v ./internal/adapter/handler/ -run Alipay

# Run specific scenario
go test -v ./internal/adapter/handler/ -run TestAlipayOAuth_MissingState
```

### Release Download Tests

```bash
# Run all Release tests
go test -v ./internal/adapter/handler/ -run Release
go test -v ./internal/app/ -run Release

# Version comparison tests
go test -v ./internal/app/ -run TestCompareVersions
```

### Model Sync Worker Tests

```bash
# Run all Model Sync tests
go test -v ./internal/adapter/handler/ -run ModelSync

# Test context cancellation
go test -v ./internal/adapter/handler/ -run TestAutoSyncChannelModelsWithContext_ContextCancellation
```

---

## Frontend Testing

### React/TypeScript Tests

```bash
cd web

# Run all frontend tests
bun run test

# Run with watch mode
bun run test:watch

# Type checking
bun run typecheck

# Linting
bun run lint

# Fix linting issues
bun run lint:fix
```

---

## Continuous Integration (CI)

### Recommended CI Test Commands

```bash
# Step 1: Unit tests with coverage
go test -short -cover ./...

# Step 2: Race detection
go test -race -short ./...

# Step 3: Integration tests (with database)
go test -run Integration ./...

# Step 4: Frontend tests
cd web && bun run test && bun run typecheck && bun run lint
```

### Example GitHub Actions

```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'

      # Unit tests
      - name: Unit Tests
        run: go test -short -cover ./...

      # Race detection
      - name: Race Detection
        run: go test -race -short ./...

      # Integration tests (with services)
      - name: Integration Tests
        run: go test -run Integration ./...
        env:
          SQL_DSN: ${{ secrets.TEST_DB_URL }}
```

---

## Test Structure

### Test File Naming

- Test files: `*_test.go`
- Integration tests: `*_integration_test.go`
- Benchmark tests: `*_benchmark_test.go`

### Test Function Naming

```go
// Format: Test<Subject>_<Method>_<Behavior>
func TestAlipayOAuth_MissingState(t *testing.T) { ... }
func TestCompareVersions_DoubleDigitMajor(t *testing.T) { ... }

// Table-driven tests
func TestAlipayOAuth_Scenarios(t *testing.T) {
    tests := []struct{
        name string
        // ...
    }{
        {name: "missing_state", ...},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) { ... })
    }
}
```

---

## Common Issues & Solutions

### Issue 1: Test fails with "undefined: repo"

**Cause**: Running individual test file instead of package.

**Solution**:
```bash
# ❌ Wrong
go test ./internal/adapter/handler/alipay_test.go

# ✅ Correct
go test ./internal/adapter/handler/
```

### Issue 2: Integration test fails with "database connection refused"

**Cause**: Database not running or wrong connection string.

**Solution**:
```bash
# Skip integration tests
go test -short ./...

# Or set up database
docker-compose up -d postgres
export SQL_DSN="postgres://user:pass@localhost:5432/testdb"
go test ./...
```

### Issue 3: "too many open files" error

**Cause**: macOS file descriptor limit.

**Solution**:
```bash
# Increase limit
ulimit -n 4096

# Or run fewer tests in parallel
go test -p 4 ./...
```

---

## Test Coverage Reports

### Current Coverage (2026-02-13)

| Package | Files | Tests | Coverage |
|---------|-------|-------|----------|
| `internal/adapter/handler/` | 14 test files | ~60 tests | ~45% |
| `internal/app/` | 2 test files | ~15 tests | ~30% |
| `internal/adapter/repo/` | - | - | TBD |

### Generate Coverage Badge

```bash
# Install gocov and gocov-html
go install github.com/axw/gocov/gocov@latest
go install github.com/matm/gocov-html/cmd/gocov-html@latest

# Generate coverage
gocov test ./... | gocov-html > coverage.html
```

---

## Best Practices

1. **Always run `go test ./...` before committing**
2. **Use table-driven tests for multiple scenarios**
3. **Skip slow tests with `testing.Short()`**
4. **Use `t.Parallel()` for independent tests**
5. **Mock external dependencies (database, HTTP clients)**
6. **Test error cases, not just happy paths**
7. **Run race detector regularly (`go test -race`)**

---

## Resources

- [Go Testing Documentation](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://go.dev/wiki/TableDrivenTests)
- [Testing Best Practices](https://go.dev/doc/effective_go#testing)
- [Project Testing Standards](./CLAUDE.md#tdd)
