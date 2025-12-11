# Worker Pool Escape Patterns

This example demonstrates escape analysis in concurrent Go code with goroutines and channels.

## Run Analysis

```bash
heapcheck ./...
heapcheck -v ./...
heapcheck --format=html ./... > report.html
```

## Key Patterns

### 1. Closure Capture (Most Common Issue!)

```go
// BAD - 'task' captured by closure
for _, task := range tasks {
    go func() {
        process(task)  // ESCAPES - captured variable
    }()
}

// GOOD - pass as parameter
for _, task := range tasks {
    go func(t Task) {
        process(t)  // No capture escape
    }(task)
}
```

### 2. Worker Pools vs Ad-hoc Goroutines

- **Ad-hoc goroutines**: Each spawns allocations
- **Worker pool**: Fixed workers, reuse goroutines

### 3. Channel Best Practices

```go
// Prefer value types for small structs
ch := make(chan Result)  // value

// Use pointers only for large structs
ch := make(chan *LargeResult)  // pointer

// Buffer channels to reduce blocking
ch := make(chan Task, 100)  // buffered
```

### 4. sync.Pool for High Throughput

```go
var taskPool = sync.Pool{
    New: func() interface{} {
        return &Task{}
    },
}

// Get from pool instead of allocating
task := taskPool.Get().(*Task)
defer taskPool.Put(task)
```

## When to Use Worker Pools

| Scenario | Approach |
|----------|----------|
| < 100 tasks | Simple goroutines fine |
| 100-10K tasks | Consider worker pool |
| > 10K tasks | Definitely use worker pool |
| Continuous stream | Always use worker pool |
