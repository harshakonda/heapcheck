// Package runtime provides test-time runtime leak detection for Go applications.
//
// IMPORTANT: This package is designed for TEST-TIME ONLY, not production.
//
// It monitors goroutine counts and heap usage during test execution to detect:
//   - Goroutine leaks: goroutines that don't terminate after test completes
//   - Heap leaks: excessive heap growth that may indicate memory leaks
//
// Use Cases:
//   - Unit tests: Detect goroutine leaks after test execution
//   - Integration tests: Monitor heap growth during test scenarios
//   - CI/CD pipelines: Fail builds if memory thresholds exceeded
//
// NOT For:
//   - Production monitoring (use Prometheus, Datadog, Pyroscope instead)
//   - Long-running background analysis
//   - Real-time alerting in live systems
//
// Example usage in your application's tests:
//
//	func TestNoGoroutineLeak(t *testing.T) {
//	    snapshot := runtime.Snapshot()
//	    defer snapshot.AssertNoLeak(t)
//	
//	    // ... your test code that spawns goroutines ...
//	}
package runtime

import (
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Snapshot captures the current runtime state for later comparison.
// Use this at the beginning of a test to establish a baseline.
type Snapshot struct {
	Goroutines    int
	HeapAllocated uint64
	HeapObjects   uint64
	Timestamp     time.Time
	GoroutineIDs  map[int]bool
}

// TakeSnapshot captures current runtime state.
// Call this at the start of your test to establish a baseline.
func TakeSnapshot() *Snapshot {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &Snapshot{
		Goroutines:    runtime.NumGoroutine(),
		HeapAllocated: memStats.HeapAlloc,
		HeapObjects:   memStats.HeapObjects,
		Timestamp:     time.Now(),
		GoroutineIDs:  captureGoroutineIDs(),
	}
}

// Diff represents the difference between two snapshots
type Diff struct {
	GoroutineGrowth   int
	HeapGrowthBytes   int64
	HeapGrowthObjects int64
	Duration          time.Duration
	LeakedGoroutines  []GoroutineInfo
}

// GoroutineInfo contains information about a goroutine
type GoroutineInfo struct {
	ID    int
	State string
	Stack string
}

// Compare compares current state against the snapshot.
// Call this at the end of your test to detect leaks.
func (s *Snapshot) Compare() *Diff {
	// Force GC to get accurate heap stats
	runtime.GC()
	time.Sleep(10 * time.Millisecond)

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	currentIDs := captureGoroutineIDs()
	leakedGoroutines := findLeakedGoroutines(s.GoroutineIDs, currentIDs)

	return &Diff{
		GoroutineGrowth:   runtime.NumGoroutine() - s.Goroutines,
		HeapGrowthBytes:   int64(memStats.HeapAlloc) - int64(s.HeapAllocated),
		HeapGrowthObjects: int64(memStats.HeapObjects) - int64(s.HeapObjects),
		Duration:          time.Since(s.Timestamp),
		LeakedGoroutines:  leakedGoroutines,
	}
}

// TestingT is the interface for *testing.T
type TestingT interface {
	Errorf(format string, args ...interface{})
	Logf(format string, args ...interface{})
	Helper()
}

// AssertNoLeak checks that no goroutines were leaked since the snapshot.
// Use with defer at the start of your test:
//
//	func TestSomething(t *testing.T) {
//	    snapshot := runtime.TakeSnapshot()
//	    defer snapshot.AssertNoLeak(t)
//	    // test code...
//	}
func (s *Snapshot) AssertNoLeak(t TestingT) {
	t.Helper()
	s.AssertNoLeakWithOptions(t, DefaultOptions())
}

// Options configures leak detection behavior
type Options struct {
	MaxGoroutineGrowth int           // Maximum allowed goroutine growth (default: 0)
	MaxHeapGrowthMB    int           // Maximum allowed heap growth in MB (default: 0 = unlimited)
	SettleTime         time.Duration // Time to wait for goroutines to settle (default: 100ms)
	RetryCount         int           // Number of retries before failing (default: 3)
}

// DefaultOptions returns sensible defaults
func DefaultOptions() Options {
	return Options{
		MaxGoroutineGrowth: 0,
		MaxHeapGrowthMB:    0, // Unlimited by default
		SettleTime:         100 * time.Millisecond,
		RetryCount:         3,
	}
}

// AssertNoLeakWithOptions checks for leaks with custom options
func (s *Snapshot) AssertNoLeakWithOptions(t TestingT, opts Options) {
	t.Helper()

	var diff *Diff

	// Retry loop to allow goroutines to settle
	for i := 0; i < opts.RetryCount; i++ {
		runtime.GC()
		time.Sleep(opts.SettleTime)

		diff = s.Compare()

		// Check if within thresholds
		if diff.GoroutineGrowth <= opts.MaxGoroutineGrowth {
			if opts.MaxHeapGrowthMB == 0 || diff.HeapGrowthBytes <= int64(opts.MaxHeapGrowthMB)*1024*1024 {
				return // No leak detected
			}
		}
	}

	// Still have leaks after retries
	if diff.GoroutineGrowth > opts.MaxGoroutineGrowth {
		t.Errorf("goroutine leak detected: grew by %d (max allowed: %d)\n%s",
			diff.GoroutineGrowth, opts.MaxGoroutineGrowth, formatLeakedGoroutines(diff.LeakedGoroutines))
	}

	if opts.MaxHeapGrowthMB > 0 && diff.HeapGrowthBytes > int64(opts.MaxHeapGrowthMB)*1024*1024 {
		t.Errorf("heap leak detected: grew by %.2f MB (max allowed: %d MB)",
			float64(diff.HeapGrowthBytes)/1024/1024, opts.MaxHeapGrowthMB)
	}
}

// captureGoroutineIDs returns a set of current goroutine IDs
func captureGoroutineIDs() map[int]bool {
	ids := make(map[int]bool)

	buf := make([]byte, 1<<20) // 1MB buffer
	n := runtime.Stack(buf, true)
	stackDump := string(buf[:n])

	// Parse goroutine IDs from stack dump
	// Format: "goroutine 1 [running]:"
	pattern := regexp.MustCompile(`goroutine\s+(\d+)\s+\[([^\]]+)\]`)
	matches := pattern.FindAllStringSubmatch(stackDump, -1)

	for _, match := range matches {
		if id, err := strconv.Atoi(match[1]); err == nil {
			ids[id] = true
		}
	}

	return ids
}

// findLeakedGoroutines identifies goroutines that exist now but didn't before
func findLeakedGoroutines(before, after map[int]bool) []GoroutineInfo {
	var leaked []GoroutineInfo

	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	stackDump := string(buf[:n])

	// Split into individual goroutine stacks
	stacks := splitGoroutineStacks(stackDump)

	for id := range after {
		if !before[id] {
			// This is a new goroutine - potential leak
			if info := findGoroutineInfo(stacks, id); info != nil {
				// Filter out expected goroutines
				if !isExpectedGoroutine(info.Stack) {
					leaked = append(leaked, *info)
				}
			}
		}
	}

	// Sort by ID for consistent output
	sort.Slice(leaked, func(i, j int) bool {
		return leaked[i].ID < leaked[j].ID
	})

	return leaked
}

// splitGoroutineStacks splits a stack dump into individual goroutine stacks
func splitGoroutineStacks(dump string) map[int]string {
	stacks := make(map[int]string)
	pattern := regexp.MustCompile(`goroutine\s+(\d+)\s+\[([^\]]+)\]`)

	// Find all goroutine headers
	indices := pattern.FindAllStringSubmatchIndex(dump, -1)

	for i, match := range indices {
		idStr := dump[match[2]:match[3]]
		id, _ := strconv.Atoi(idStr)

		// Get stack content until next goroutine or end
		start := match[0]
		end := len(dump)
		if i+1 < len(indices) {
			end = indices[i+1][0]
		}

		stacks[id] = dump[start:end]
	}

	return stacks
}

// findGoroutineInfo extracts info for a specific goroutine ID
func findGoroutineInfo(stacks map[int]string, id int) *GoroutineInfo {
	stack, ok := stacks[id]
	if !ok {
		return nil
	}

	// Extract state from header
	pattern := regexp.MustCompile(`goroutine\s+\d+\s+\[([^\]]+)\]`)
	match := pattern.FindStringSubmatch(stack)

	state := "unknown"
	if match != nil {
		state = match[1]
	}

	return &GoroutineInfo{
		ID:    id,
		State: state,
		Stack: stack,
	}
}

// isExpectedGoroutine checks if a goroutine is expected (runtime, testing, etc.)
func isExpectedGoroutine(stack string) bool {
	expectedPatterns := []string{
		"runtime.gopark",
		"runtime.chanrecv",
		"runtime.chansend",
		"testing.(*T).Run",
		"testing.tRunner",
		"runtime.main",
		"runtime.gcBgMarkWorker",
		"runtime.bgsweep",
		"runtime.bgscavenge",
		"runtime.forcegchelper",
		"runtime.timerproc",
		"signal.signal_recv",
		"os/signal.loop",
		"runtime.runfinq",
		"runtime.goexit",
	}

	stackLower := strings.ToLower(stack)
	for _, pattern := range expectedPatterns {
		if strings.Contains(stackLower, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// formatLeakedGoroutines formats leaked goroutines for error output
func formatLeakedGoroutines(leaked []GoroutineInfo) string {
	if len(leaked) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\nLeaked goroutines (%d):\n", len(leaked)))

	for _, g := range leaked {
		sb.WriteString(fmt.Sprintf("\n--- Goroutine %d [%s] ---\n", g.ID, g.State))
		// Truncate stack to first 10 lines for readability
		lines := strings.Split(g.Stack, "\n")
		if len(lines) > 12 {
			lines = append(lines[:12], "    ...")
		}
		sb.WriteString(strings.Join(lines, "\n"))
		sb.WriteString("\n")
	}

	return sb.String()
}

// Result holds the complete runtime analysis result
type Result struct {
	GoroutineStart int           `json:"goroutineStart"`
	GoroutineEnd   int           `json:"goroutineEnd"`
	GoroutineGrowth int          `json:"goroutineGrowth"`
	GoroutineLeak  bool          `json:"goroutineLeak"`
	HeapStartBytes uint64        `json:"heapStartBytes"`
	HeapEndBytes   uint64        `json:"heapEndBytes"`
	HeapGrowthBytes int64        `json:"heapGrowthBytes"`
	HeapLeak       bool          `json:"heapLeak"`
	Duration       time.Duration `json:"duration"`
	LeakedCount    int           `json:"leakedCount"`
}

// Analyze runs a function and returns runtime analysis
func Analyze(fn func()) *Result {
	snapshot := TakeSnapshot()

	fn()

	diff := snapshot.Compare()

	return &Result{
		GoroutineStart:  snapshot.Goroutines,
		GoroutineEnd:    snapshot.Goroutines + diff.GoroutineGrowth,
		GoroutineGrowth: diff.GoroutineGrowth,
		GoroutineLeak:   diff.GoroutineGrowth > 0 && len(diff.LeakedGoroutines) > 0,
		HeapStartBytes:  snapshot.HeapAllocated,
		HeapEndBytes:    uint64(int64(snapshot.HeapAllocated) + diff.HeapGrowthBytes),
		HeapGrowthBytes: diff.HeapGrowthBytes,
		HeapLeak:        diff.HeapGrowthBytes > 10*1024*1024, // >10MB growth
		Duration:        diff.Duration,
		LeakedCount:     len(diff.LeakedGoroutines),
	}
}
