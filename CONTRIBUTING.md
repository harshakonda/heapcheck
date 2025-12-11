# Contributing to heapcheck

Thank you for your interest in contributing to heapcheck!

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/harshakonda/heapcheck.git`
3. Create a branch: `git checkout -b feature/my-feature`
4. Make your changes
5. Run tests: `go test ./...`
6. Commit: `git commit -m "Add my feature"`
7. Push: `git push origin feature/my-feature`
8. Open a Pull Request

## Development Setup

```bash
# Clone
git clone https://github.com/harshakonda/heapcheck.git
cd heapcheck

# Build
go build -o heapcheck ./cmd/heapcheck

# Test
go test -v ./...

# Run on itself
./heapcheck ./...
```

## Code Style

- Follow standard Go conventions
- Run `gofmt` before committing
- Add tests for new functionality
- Update documentation as needed

## Areas for Contribution

### High Impact
- **Improve categorization**: Reduce "uncategorized" escapes by adding more patterns
- **Add tests**: Increase test coverage for categorizer and reporter
- **Real-world testing**: Test on popular Go projects and report issues

### Medium Impact
- **Documentation**: Improve README, add examples
- **Output formats**: Add new reporter formats (CSV, Markdown)
- **Editor integration**: VS Code extension, vim plugin

### Ideas Welcome
- Performance optimizations
- New analysis features
- Better suggestions

## Reporting Issues

When reporting issues, please include:
- Go version (`go version`)
- OS and architecture
- heapcheck version (`heapcheck --version`)
- Minimal reproduction case
- Expected vs actual behavior

## Pull Request Guidelines

1. Keep PRs focused - one feature/fix per PR
2. Add tests for new code
3. Update README if adding features
4. Ensure CI passes
5. Respond to review feedback promptly

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
