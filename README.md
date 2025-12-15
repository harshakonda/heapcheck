# heapcheck

[![CI](https://github.com/harshakonda/heapcheck/actions/workflows/ci.yml/badge.svg)](https://github.com/harshakonda/heapcheck/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/harshakonda/heapcheck.svg)](https://pkg.go.dev/github.com/harshakonda/heapcheck)
[![Go Report Card](https://goreportcard.com/badge/github.com/harshakonda/heapcheck)](https://goreportcard.com/report/github.com/harshakonda/heapcheck)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![DOI](https://zenodo.org/badge/1114227633.svg)](https://doi.org/10.5281/zenodo.17895742)

**heapcheck** is a unified memory analysis tool for Go that combines static escape analysis with test-time leak detection.

## Features

- **Static escape analysis** - transforms cryptic compiler output into human-readable reports with actionable suggestions
- **Runtime leak detection** - detects goroutine and heap leaks during test execution
- **Test integration** - simple API to add memory checks to your tests (goleak-compatible)
- **Multiple output formats** - text, JSON, HTML, and SARIF for CI/CD integration

## The Problem

Go's compiler escape analysis is powerful but cryptic:

```
$ go build -gcflags="-m" ./...
./main.go:15:6: can inline square
./main.go:12:2: moved to heap: z
./main.go:8:14: *y escapes to heap
./main.go:11:13: x does not escape
```

What does this mean? Why did `z` move to heap? How do I fix it?

## The Solution

```
$ heapcheck ./...

heapcheck - Escape Analysis Report
--------------------------------------------------

Summary:
  Total variables analyzed: 847
  Stack allocated:          792 (93.5%)
  Heap allocated:            55 (6.5%)

Escape Causes:
  1. interface-boxing      23 (41.8%)  -> Use concrete types or generics
  2. return-pointer        15 (27.3%)  -> Return by value if struct <= 64 bytes
  3. closure-capture        9 (16.4%)  -> Pass as parameter instead
  4. goroutine-escape       5 (9.1%)   -> Use worker pools
  5. unknown-size           3 (5.5%)   -> Pre-allocate capacity

Hotspots (files with most escapes):
  pkg/server/handler.go                      12 escapes
  pkg/cache/store.go                          8 escapes
  internal/util/strings.go                    6 escapes

Run with -v for detailed breakdown of all 55 escapes.
```

## Installation

```bash
go install github.com/harshakonda/heapcheck/cmd/heapcheck@latest
```

## CLI Usage

### Basic Analysis

```bash
# Analyze all packages in current module
heapcheck ./...

# Analyze specific package
heapcheck ./pkg/server

# Verbose output with all escape details
heapcheck -v ./...
```

### Output Formats

```bash
# Human-readable text (default)
heapcheck ./...

# JSON for CI/CD integration
heapcheck --format=json ./...

# HTML visual report with charts
heapcheck --format=html ./... > report.html

# SARIF for GitHub Code Scanning
heapcheck --format=sarif ./... > results.sarif
```

### Filtering

```bash
# Show only heap escapes (hide "does not escape")
heapcheck --escapes-only ./...

# Filter by package path
heapcheck --filter=pkg/server ./...
```

## Test Integration (guard package)

Add leak detection to your tests with the `guard` package. The API is compatible with [goleak](https://github.com/uber-go/goleak).

**Note:** The guard package is for test-time only (unit tests, integration tests, CI/CD). For production monitoring, use dedicated tools like Prometheus, Datadog, or Pyroscope.

### Basic Usage

```go
import "github.com/harshakonda/heapcheck/guard"

func TestMyFunction(t *testing.T) {
    defer guard.VerifyNone(t)
    
    // Your test code here
    result := myHandler.Process(input)
    assert.Equal(t, expected, result)
}
```

### With Thresholds

```go
func TestWithThresholds(t *testing.T) {
    defer guard.VerifyNone(t,
        guard.MaxGoroutines(5),   // Allow up to 5 transient goroutines
        guard.MaxHeapMB(50),      // Allow up to 50MB heap growth
    )
    
    // Your test code here
    processLargeDataset()
}
```

### Ignoring Known Goroutines

```go
func TestWithBackgroundWorker(t *testing.T) {
    defer guard.VerifyNone(t,
        guard.IgnoreTopFunction("github.com/myapp/pkg.backgroundWorker"),
        guard.IgnoreContains("database/sql.(*DB).connectionOpener"),
    )
    
    // Your test code here
}
```

### Package-Level Check

```go
func TestMain(m *testing.M) {
    guard.VerifyTestMain(m)
}
```

### Manual Control with Checkpoints

```go
func TestComplex(t *testing.T) {
    g := guard.Check(t)
    
    // Phase 1
    setupDatabase()
    g.Checkpoint("after setup")
    
    // Phase 2
    runMigrations()
    g.Checkpoint("after migrations")
    
    // Final verification
    g.Verify()
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output (shows checkpoints and leak details)
go test -v ./...

# Run example tests
go test ./examples/...
```

### What Failure Looks Like

When a leak is detected, the test fails with details:

```
--- FAIL: TestWorkerPool (0.31s)
    guard.go:142: heapcheck: goroutine leak detected
      Leaked: 2 (max allowed: 0)
      
      goroutine 25 [running]:
        github.com/myapp/worker.(*Pool).worker(...)
            /app/worker/pool.go:45
        ...
```

## Runtime Analysis

For custom analysis scenarios, use the `runtime` package directly:

```go
import "github.com/harshakonda/heapcheck/runtime"

func TestDetailed(t *testing.T) {
    snapshot := runtime.TakeSnapshot()
    
    // Run your code
    myFunction()
    
    // Get detailed diff
    diff := snapshot.Compare()
    
    t.Logf("Goroutine growth: %d", diff.GoroutineGrowth)
    t.Logf("Heap growth: %.2f MB", float64(diff.HeapGrowthBytes)/1024/1024)
    
    for _, g := range diff.LeakedGoroutines {
        t.Logf("Leaked: goroutine %d [%s]", g.ID, g.State)
    }
}
```

## Escape Categories

heapcheck categorizes escapes by their cause and provides optimization suggestions:

| Category | Description | Suggestion |
|----------|-------------|------------|
| `return-pointer` | Returns pointer to local variable | Return by value if struct <= 64 bytes |
| `interface-boxing` | Assigned to `interface{}` | Use concrete types or generics |
| `closure-capture` | Captured by closure | Pass as parameter instead |
| `goroutine-escape` | Passed to goroutine | Use worker pools |
| `channel-send` | Sent over channel | Consider sync.Pool |
| `slice-grow` | Slice may grow | Pre-allocate capacity |
| `unknown-size` | Size unknown at compile time | Use fixed-size arrays |
| `fmt-call` | Passed to fmt functions | Use strconv in hot paths |
| `reflection` | Uses reflect package | Avoid in hot paths |
| `leaking-param` | Parameter escapes function | Review function signature |
| `map-allocation` | make(map[K]V) | Expected behavior |
| `new-allocation` | new(T) | Expected behavior |
| `too-large` | Struct too large for stack | Expected behavior |

## CI/CD Integration

### GitHub Actions

```yaml
name: Memory Analysis
on: [push, pull_request]

jobs:
  heapcheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Install heapcheck
        run: go install github.com/harshakonda/heapcheck/cmd/heapcheck@latest
      
      - name: Run Static Analysis
        run: heapcheck ./...
      
      - name: Run Tests with Leak Detection
        run: go test -v ./...
      
      - name: Upload SARIF
        run: heapcheck --format=sarif ./... > results.sarif
      
      - uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: results.sarif
```

### GitLab CI

```yaml
heapcheck:
  stage: lint
  script:
    - go install github.com/harshakonda/heapcheck/cmd/heapcheck@latest
    - heapcheck --format=json ./... > heapcheck.json
  artifacts:
    reports:
      codequality: heapcheck.json
```

### Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

heapcheck ./...
if [ $? -ne 0 ]; then
    echo "heapcheck found issues"
    exit 1
fi
```

## Understanding Escape Analysis

### Why Does It Matter?

- **Stack allocations are fast**: Just moving a pointer, automatically freed when function returns
- **Heap allocations are slow**: Requires GC overhead, causes memory fragmentation
- **GC pressure**: More heap allocations = more GC pauses = higher latency

### Common Escape Patterns

**1. Returning Pointers**

```go
// Escapes - pointer to local variable
func newUser() *User {
    u := User{Name: "test"}
    return &u  // escapes!
}

// No escape - return by value
func newUser() User {
    return User{Name: "test"}
}
```

**2. Interface Boxing**

```go
// Escapes - interface boxing
func log(msg interface{}) { ... }
log(myStruct)  // escapes!

// No escape - concrete type or generics
func log[T any](msg T) { ... }
log(myStruct)
```

**3. Closure Capture**

```go
// Escapes - captured by closure
func process(data []byte) {
    go func() {
        use(data)  // data escapes!
    }()
}

// No escape - pass as parameter
func process(data []byte) {
    go func(d []byte) {
        use(d)
    }(data)
}
```

## For Researchers

heapcheck is designed to be citable in academic work.

### Citation

Konda, S. H. (2025). *heapcheck: Unified Memory Analysis for Go*. Zenodo. https://doi.org/10.5281/zenodo.17895742

### BibTeX

```bibtex
@software{konda2025heapcheck,
  author       = {Konda, Sri Harsha},
  title        = {heapcheck: Unified Memory Analysis for Go},
  year         = {2025},
  publisher    = {Zenodo},
  version      = {0.2.0},
  doi          = {10.5281/zenodo.17895742},
  url          = {https://doi.org/10.5281/zenodo.17895742}
}
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development

```bash
git clone https://github.com/harshakonda/heapcheck
cd heapcheck
go test ./...
go build ./cmd/heapcheck
./heapcheck ./examples/...
```

## License

MIT License - see [LICENSE](LICENSE) for details.
