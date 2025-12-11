# HTTP Server Escape Patterns

This example demonstrates common escape patterns in HTTP handlers and web applications.

## Run Analysis

```bash
heapcheck ./...
heapcheck -v ./...
heapcheck --format=html ./... > report.html
```

## Common Issues in Web Applications

### 1. Interface Boxing in JSON Responses

```go
// BAD - interface{} causes boxing
type Response struct {
    Data interface{} `json:"data"`
}

// GOOD - typed responses
type UserResponse struct {
    Data User `json:"data"`
}
```

### 2. String Formatting

```go
// BAD - fmt.Sprintf causes boxing
msg := fmt.Sprintf("Error: %d", code)

// GOOD - strconv avoids boxing
msg := "Error: " + strconv.Itoa(code)
```

### 3. Middleware Closures

```go
// BAD - closure captures logger
return func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w, r) {
        logger.Log(r.URL.Path)  // captured!
    })
}

// GOOD - struct with fields
type middleware struct {
    logger *Logger
    next   http.Handler
}
```

### 4. Request Object Pooling

For high-throughput servers, use `sync.Pool` to reuse request/response objects.

## When to Optimize

- **High QPS endpoints** (>1000 req/s): Optimize aggressively
- **Low QPS endpoints**: Don't worry about small escapes
- **Always profile first** with `pprof` before optimizing
