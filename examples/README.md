# heapcheck Examples

This directory contains real-world examples demonstrating escape analysis patterns.

## Examples Overview

| Directory | Description | Key Learnings |
|-----------|-------------|---------------|
| [basic-patterns](basic-patterns/) | All escape categories | Fundamentals of escape analysis |
| [http-server](http-server/) | Web application patterns | Interface boxing, middleware closures |
| [worker-pool](worker-pool/) | Concurrent code | Closure captures, channel usage |
| [json-processor](json-processor/) | JSON handling | Reflection, buffer pooling |

## Quick Start

```bash
# Analyze any example
cd basic-patterns
heapcheck ./...

# Verbose output
heapcheck -v ./...

# Generate HTML report
heapcheck --format=html ./... > report.html
open report.html

# JSON output
heapcheck --format=json ./... | jq '.summary'
```

## Learning Path

1. **Start with `basic-patterns`** - Learn all escape categories
2. **Move to `http-server`** - Real-world web patterns
3. **Study `worker-pool`** - Concurrent programming
4. **Explore `json-processor`** - Performance optimization

## Running All Examples

```bash
# From heapcheck root
for dir in examples/*/; do
    echo "=== $dir ==="
    heapcheck "$dir..."
    echo
done
```

## Contributing Examples

We welcome new examples! Good candidates:
- Database access patterns
- gRPC server/client
- File I/O
- Caching implementations

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.
