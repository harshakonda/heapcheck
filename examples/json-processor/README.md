# JSON Processor Escape Patterns

This example demonstrates escape analysis in JSON processing - one of the most allocation-heavy operations in Go.

## Run Analysis

```bash
heapcheck ./...
heapcheck -v ./...
heapcheck --format=html ./... > report.html
```

## Why JSON Causes Escapes

1. **Reflection**: `encoding/json` uses reflect package
2. **Interface boxing**: `interface{}` fields cause boxing  
3. **Buffer allocation**: Each Marshal/Unmarshal allocates
4. **Map creation**: JSON objects become maps

## Key Patterns

### 1. Buffer Pooling

```go
// BAD - allocates new buffer each time
data, _ := json.Marshal(event)

// GOOD - reuse buffers
buf := bufferPool.Get().(*bytes.Buffer)
defer bufferPool.Put(buf)
json.NewEncoder(buf).Encode(event)
```

### 2. Avoid interface{} Fields

```go
// BAD - interface causes boxing
type Message struct {
    Payload interface{} `json:"payload"`
}

// GOOD - use generics
type Message[T any] struct {
    Payload T `json:"payload"`
}
```

### 3. Lazy Map Allocation

```go
// BAD - always allocates
Fields: make(map[string]string)

// GOOD - nil until needed
Fields: nil  // allocate only when AddField called
```

### 4. Manual JSON for Hot Paths

For extremely hot paths, manual JSON building avoids reflection:

```go
buf = append(buf, `{"level":"`...)
buf = append(buf, level...)
buf = append(buf, `"}`...)
```

### 5. Streaming for Large Arrays

```go
// BAD - loads entire array
var events []Event
json.Unmarshal(data, &events)

// GOOD - stream one at a time
dec := json.NewDecoder(reader)
for dec.More() {
    var event Event
    dec.Decode(&event)
    // process and discard
}
```

## Benchmarking

Always benchmark before optimizing:

```bash
go test -bench=. -benchmem
```

| Method | ns/op | B/op | allocs/op |
|--------|-------|------|-----------|
| json.Marshal | 850 | 256 | 4 |
| Buffer Pool | 620 | 128 | 2 |
| Manual JSON | 180 | 64 | 1 |
