# heapcheck

[![CI](https://github.com/harshakonda/heapcheck/actions/workflows/ci.yml/badge.svg)](https://github.com/harshakonda/heapcheck/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/harshakonda/heapcheck.svg)](https://pkg.go.dev/github.com/harshakonda/heapcheck)
[![Go Report Card](https://goreportcard.com/badge/github.com/harshakonda/heapcheck)](https://goreportcard.com/report/github.com/harshakonda/heapcheck)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**heapcheck** is a developer-friendly CLI tool that makes Go's escape analysis output human-readable with actionable optimization suggestions.

## The Problem

Go's compiler escape analysis is powerful but cryptic:

```bash
$ go build -gcflags="-m" ./...
./main.go:15:6: can inline square
./main.go:12:2: moved to heap: z
./main.go:8:14: *y escapes to heap
./main.go:11:13: x does not escape
```

**What does this mean? Why did `z` move to heap? How do I fix it?**

## The Solution

```bash
$ heapcheck ./...

📊 heapcheck - Escape Analysis Report
──────────────────────────────────────────────────

Summary:
  Total variables analyzed: 847
  Stack allocated:          792 (93.5%)
  Heap allocated:            55 (6.5%) ⚠️

Escape Causes:
  1. interface-boxing      23 (41.8%)
  2. return-pointer        15 (27.3%)
  3. closure-capture        9 (16.4%)
  4. goroutine-escape       5 (9.1%)
  5. unknown-size           3 (5.5%)

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

Or with Homebrew (coming soon):

```bash
brew install heapcheck
```

## Usage

### Basic Usage

```bash
# Analyze all packages in current module
heapcheck ./...

# Analyze specific package
heapcheck ./pkg/server

# Verbose output with all details
heapcheck -v ./...
```

### Output Formats

```bash
# Human-readable text (default)
heapcheck ./...

# JSON for CI/CD integration
heapcheck --format=json ./...

# HTML visual report
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

## Escape Categories

heapcheck categorizes escapes by their cause and provides optimization suggestions:

| Category | Description | Suggestion |
|----------|-------------|------------|
| `return-pointer` | Returns pointer to local variable | Return by value if struct ≤ 64 bytes |
| `interface-boxing` | Assigned to `interface{}` | Use concrete types or generics |
| `closure-capture` | Captured by closure | Pass as parameter instead |
| `goroutine-escape` | Passed to goroutine | Use worker pools |
| `channel-send` | Sent over channel | Consider sync.Pool |
| `slice-grow` | Slice may grow | Pre-allocate capacity |
| `unknown-size` | Size unknown at compile time | Use fixed-size arrays |
| `fmt-call` | Passed to fmt functions | Use strconv in hot paths |

## CI/CD Integration

### GitHub Actions

```yaml
name: Heap Check
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
      
      - name: Run heapcheck
        run: heapcheck --format=sarif ./... > results.sarif
      
      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v3
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

## Understanding Escape Analysis

### Why Does It Matter?

- **Stack allocations are fast**: Just moving a pointer, automatically freed
- **Heap allocations are slow**: GC overhead, memory fragmentation
- **GC pressure**: More heap allocations = more GC pauses = higher latency

### Common Escape Patterns

#### 1. Returning Pointers

```go
// ❌ Escapes - pointer to local
func newUser() *User {
    u := User{Name: "test"}
    return &u  // escapes!
}

// ✅ No escape - return by value
func newUser() User {
    return User{Name: "test"}
}
```

#### 2. Interface Boxing

```go
// ❌ Escapes - interface boxing
func log(msg interface{}) { ... }
log(myStruct)  // escapes!

// ✅ No escape - concrete type or generics
func log[T any](msg T) { ... }
log(myStruct)
```

#### 3. Closure Capture

```go
// ❌ Escapes - captured by closure
func process(data []byte) {
    go func() {
        use(data)  // data escapes!
    }()
}

// ✅ No escape - pass as parameter
func process(data []byte) {
    go func(d []byte) {
        use(d)
    }(data)
}
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development

```bash
# Clone the repository
git clone https://github.com/harshakonda/heapcheck
cd heapcheck

# Run tests
go test ./...

# Build
go build ./cmd/heapcheck

# Test on a sample project
./heapcheck ./testdata/...
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Related Tools

- [staticcheck](https://staticcheck.io/) - Go static analysis
- [pprof](https://go.dev/blog/pprof) - Profiling Go programs
- [go-torch](https://github.com/uber-archive/go-torch) - Flame graphs

## Acknowledgments

- Go team for the excellent `-gcflags -m` escape analysis
- [ICSE 2020 paper](https://dl.acm.org/doi/10.1145/3377813.3381368) on Go escape analysis optimization
