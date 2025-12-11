# Basic Escape Patterns

This example demonstrates all common escape analysis patterns in Go.

## Run Analysis

```bash
# Basic analysis
heapcheck ./...

# Verbose - see all escapes
heapcheck -v ./...

# Generate HTML report
heapcheck --format=html ./... > report.html
open report.html
```

## Expected Output

```
📊 heapcheck - Escape Analysis Report
──────────────────────────────────────────────────

Summary:
  Total variables analyzed: 45
  Stack allocated:          12 (26.7%)
  Heap allocated:           28 (62.2%) ⚠️
  Inlined calls:             5

Escape Causes:
  1. return-pointer        8 (28.6%)
  2. interface-boxing      7 (25.0%)
  3. closure-capture       4 (14.3%)
  4. slice-grow            3 (10.7%)
  5. map-allocation        2 (7.1%)
  ...
```

## Patterns Demonstrated

| Pattern | Bad Function | Good Function | Why |
|---------|--------------|---------------|-----|
| Return Pointer | `NewUserBad()` | `NewUserGood()` | Return value vs pointer |
| Interface Boxing | `LogBad()` | `LogGood()` | interface{} vs concrete type |
| Closure Capture | `ProcessBad()` | `ProcessGood()` | Capture vs parameter |
| Slice Growth | `CollectBad()` | `CollectGood()` | Dynamic vs pre-allocated |
| fmt vs strconv | `FormatIDBad()` | `FormatIDGood()` | Interface boxing |
| Map Allocation | `CreateMapBad()` | `CreateMapPooled()` | Maps always escape |
| Large Structs | `CreateLarge()` | `CreateSmall()` | Size matters |

## Learning Exercise

1. Run `heapcheck -v ./...` and identify each escape
2. Read the suggestion for each escape
3. Compare "Bad" vs "Good" functions
4. Understand why one escapes and the other doesn't
