// Package guard provides simple test integration for memory leak detection.
//
// IMPORTANT: This package is for CLIENT APPLICATION TESTS only.
//
// Use this package in YOUR application's test files to:
//   - Detect goroutine leaks after each test
//   - Monitor heap growth during test execution
//   - Fail tests when memory thresholds are exceeded
//   - Integrate memory checks into CI/CD pipelines
//
// This is NOT a production monitoring tool. For production, use
// dedicated observability tools (Prometheus, Datadog, Pyroscope).
//
// Basic Usage (goleak-compatible API):
//
//	import "github.com/harshakonda/heapcheck/guard"
//
//	func TestMyFunction(t *testing.T) {
//	    defer guard.VerifyNone(t)
//	    
//	    // Your test code here
//	    myFunction()
//	}
//
// With Custom Thresholds:
//
//	func TestWithThresholds(t *testing.T) {
//	    defer guard.VerifyNone(t,
//	        guard.MaxGoroutines(10),
//	        guard.MaxHeapMB(50),
//	    )
//	    
//	    // Your test code here
//	}
//
// Ignoring Specific Goroutines:
//
//	func TestWithIgnore(t *testing.T) {
//	    defer guard.VerifyNone(t,
//	        guard.IgnoreTopFunction("github.com/some/pkg.backgroundWorker"),
//	    )
//	    
//	    // Your test code here
//	}
//
// Package-Level Check (in TestMain):
//
//	func TestMain(m *testing.M) {
//	    guard.VerifyTestMain(m)
//	}
package guard

import (
	"os"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/harshakonda/heapcheck/runtime"
)

// TestingT is the interface for *testing.T and *testing.B
type TestingT interface {
	Errorf(format string, args ...interface{})
	Logf(format string, args ...interface{})
	Helper()
	Cleanup(func())
}

// TestingM is the interface for *testing.M
type TestingM interface {
	Run() int
}

// Option configures leak detection behavior
type Option func(*config)

type config struct {
	maxGoroutines     int
	maxHeapMB         int
	settleTime        time.Duration
	retryCount        int
	ignoreFuncs       []string
	ignoreContains    []string
}

func defaultConfig() *config {
	return &config{
		maxGoroutines: 0,  // Any growth is a leak
		maxHeapMB:     0,  // Unlimited
		settleTime:    100 * time.Millisecond,
		retryCount:    3,
	}
}

// MaxGoroutines sets the maximum allowed goroutine growth.
// Default is 0 (any growth is considered a leak).
func MaxGoroutines(n int) Option {
	return func(c *config) {
		c.maxGoroutines = n
	}
}

// MaxHeapMB sets the maximum allowed heap growth in megabytes.
// Default is 0 (unlimited).
func MaxHeapMB(mb int) Option {
	return func(c *config) {
		c.maxHeapMB = mb
	}
}

// SettleTime sets how long to wait for goroutines to settle.
// Default is 100ms.
func SettleTime(d time.Duration) Option {
	return func(c *config) {
		c.settleTime = d
	}
}

// RetryCount sets how many times to retry before reporting a leak.
// Default is 3.
func RetryCount(n int) Option {
	return func(c *config) {
		c.retryCount = n
	}
}

// IgnoreTopFunction ignores goroutines where the top function matches.
// Use this for known background goroutines that are expected.
//
//	guard.IgnoreTopFunction("github.com/some/pkg.backgroundWorker")
func IgnoreTopFunction(fn string) Option {
	return func(c *config) {
		c.ignoreFuncs = append(c.ignoreFuncs, fn)
	}
}

// IgnoreContains ignores goroutines whose stack contains the given string.
//
//	guard.IgnoreContains("database/sql.(*DB).connectionOpener")
func IgnoreContains(s string) Option {
	return func(c *config) {
		c.ignoreContains = append(c.ignoreContains, s)
	}
}

// VerifyNone verifies that no goroutines are leaked when the test completes.
// This is the primary API, designed to be compatible with goleak.
//
// Usage:
//
//	func TestSomething(t *testing.T) {
//	    defer guard.VerifyNone(t)
//	    // test code...
//	}
//
// With options:
//
//	func TestSomething(t *testing.T) {
//	    defer guard.VerifyNone(t,
//	        guard.MaxGoroutines(5),
//	        guard.MaxHeapMB(100),
//	    )
//	    // test code...
//	}
func VerifyNone(t TestingT, opts ...Option) {
	t.Helper()

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	snapshot := runtime.TakeSnapshot()

	// Register cleanup to run at end of test
	t.Cleanup(func() {
		verifyWithConfig(t, snapshot, cfg)
	})
}

// verifyWithConfig performs the actual verification
func verifyWithConfig(t TestingT, snapshot *runtime.Snapshot, cfg *config) {
	t.Helper()

	var diff *runtime.Diff
	var leaked []runtime.GoroutineInfo

	// Retry loop to allow goroutines to settle
	for i := 0; i < cfg.retryCount; i++ {
		goruntime.GC()
		time.Sleep(cfg.settleTime)

		diff = snapshot.Compare()
		leaked = filterIgnored(diff.LeakedGoroutines, cfg)

		// Check if within thresholds
		goroutineOK := len(leaked) <= cfg.maxGoroutines
		heapOK := cfg.maxHeapMB == 0 || diff.HeapGrowthBytes <= int64(cfg.maxHeapMB)*1024*1024

		if goroutineOK && heapOK {
			return // No leak detected
		}
	}

	// Report failures
	if len(leaked) > cfg.maxGoroutines {
		t.Errorf("heapcheck: goroutine leak detected\n"+
			"  Leaked: %d (max allowed: %d)\n"+
			"  %s",
			len(leaked), cfg.maxGoroutines, formatLeaked(leaked))
	}

	if cfg.maxHeapMB > 0 && diff.HeapGrowthBytes > int64(cfg.maxHeapMB)*1024*1024 {
		t.Errorf("heapcheck: heap leak detected\n"+
			"  Growth: %.2f MB (max allowed: %d MB)",
			float64(diff.HeapGrowthBytes)/1024/1024, cfg.maxHeapMB)
	}
}

// filterIgnored removes goroutines that match ignore patterns
func filterIgnored(leaked []runtime.GoroutineInfo, cfg *config) []runtime.GoroutineInfo {
	if len(cfg.ignoreFuncs) == 0 && len(cfg.ignoreContains) == 0 {
		return leaked
	}

	var filtered []runtime.GoroutineInfo
	for _, g := range leaked {
		if shouldIgnore(g, cfg) {
			continue
		}
		filtered = append(filtered, g)
	}
	return filtered
}

// shouldIgnore checks if a goroutine should be ignored
func shouldIgnore(g runtime.GoroutineInfo, cfg *config) bool {
	for _, fn := range cfg.ignoreFuncs {
		if strings.Contains(g.Stack, fn) {
			return true
		}
	}
	for _, s := range cfg.ignoreContains {
		if strings.Contains(g.Stack, s) {
			return true
		}
	}
	return false
}

// formatLeaked formats leaked goroutines for error output
func formatLeaked(leaked []runtime.GoroutineInfo) string {
	if len(leaked) == 0 {
		return "  (no details available)"
	}

	var sb strings.Builder
	for i, g := range leaked {
		if i >= 3 {
			sb.WriteString("\n  ... and more")
			break
		}
		sb.WriteString("\n  ")
		sb.WriteString(truncateStack(g.Stack, 5))
	}
	return sb.String()
}

// truncateStack truncates a stack trace to n lines
func truncateStack(stack string, n int) string {
	lines := strings.Split(stack, "\n")
	if len(lines) <= n*2 {
		return strings.Join(lines, "\n  ")
	}
	truncated := lines[:n*2]
	return strings.Join(truncated, "\n  ") + "\n  ..."
}

// VerifyTestMain runs tests and checks for leaks at package level.
// Use in TestMain to check for leaks after all tests complete.
//
//	func TestMain(m *testing.M) {
//	    guard.VerifyTestMain(m)
//	}
//
// With options:
//
//	func TestMain(m *testing.M) {
//	    guard.VerifyTestMain(m,
//	        guard.IgnoreTopFunction("database/sql.connectionOpener"),
//	    )
//	}
func VerifyTestMain(m TestingM, opts ...Option) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	snapshot := runtime.TakeSnapshot()

	// Run tests
	exitCode := m.Run()

	// Check for leaks
	goruntime.GC()
	time.Sleep(cfg.settleTime)

	diff := snapshot.Compare()
	leaked := filterIgnored(diff.LeakedGoroutines, cfg)

	if len(leaked) > cfg.maxGoroutines {
		os.Stderr.WriteString("\nheapcheck: goroutine leak detected after tests\n")
		for _, g := range leaked {
			os.Stderr.WriteString("\n" + g.Stack + "\n")
		}
		if exitCode == 0 {
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}

// Check creates a guard that can be manually verified.
// Use this for more complex test scenarios.
//
//	func TestComplex(t *testing.T) {
//	    g := guard.Check(t)
//	    
//	    // Phase 1
//	    doSomething()
//	    g.Checkpoint("after phase 1")
//	    
//	    // Phase 2
//	    doSomethingElse()
//	    g.Verify() // Final check
//	}
func Check(t TestingT, opts ...Option) *Guard {
	t.Helper()

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return &Guard{
		t:        t,
		cfg:      cfg,
		snapshot: runtime.TakeSnapshot(),
	}
}

// Guard provides manual control over leak checking
type Guard struct {
	t        TestingT
	cfg      *config
	snapshot *runtime.Snapshot
}

// Checkpoint logs current state without failing
func (g *Guard) Checkpoint(label string) {
	g.t.Helper()

	diff := g.snapshot.Compare()
	g.t.Logf("heapcheck checkpoint [%s]: goroutines=%+d, heap=%+.2f MB",
		label, diff.GoroutineGrowth, float64(diff.HeapGrowthBytes)/1024/1024)
}

// Verify checks for leaks and fails the test if found
func (g *Guard) Verify() {
	g.t.Helper()
	verifyWithConfig(g.t, g.snapshot, g.cfg)
}

// Reset takes a new snapshot, useful between test phases
func (g *Guard) Reset() {
	g.snapshot = runtime.TakeSnapshot()
}

// Result returns the current diff without failing
func (g *Guard) Result() *runtime.Diff {
	return g.snapshot.Compare()
}
