package runtime_test

import (
	"testing"
	"time"

	"github.com/harshakonda/heapcheck/runtime"
)

func TestTakeSnapshot(t *testing.T) {
	snapshot := runtime.TakeSnapshot()

	if snapshot.Goroutines <= 0 {
		t.Error("expected positive goroutine count")
	}

	if snapshot.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestSnapshot_Compare_NoChange(t *testing.T) {
	snapshot := runtime.TakeSnapshot()
	time.Sleep(10 * time.Millisecond)

	diff := snapshot.Compare()

	// Should have minimal changes
	if diff.GoroutineGrowth > 5 {
		t.Errorf("unexpected goroutine growth: %d", diff.GoroutineGrowth)
	}
}

func TestSnapshot_Compare_GoroutineLeak(t *testing.T) {
	snapshot := runtime.TakeSnapshot()

	// Start goroutines that block forever
	leakChan := make(chan struct{})
	for i := 0; i < 3; i++ {
		go func() {
			<-leakChan // Will never receive
		}()
	}

	time.Sleep(50 * time.Millisecond)
	diff := snapshot.Compare()

	if diff.GoroutineGrowth < 3 {
		t.Errorf("expected at least 3 goroutine growth, got %d", diff.GoroutineGrowth)
	}

	// Cleanup
	close(leakChan)
}

func TestAnalyze(t *testing.T) {
	result := runtime.Analyze(func() {
		// Simple function
		data := make([]byte, 1024)
		_ = data
		time.Sleep(10 * time.Millisecond)
	})

	if result.Duration < 10*time.Millisecond {
		t.Errorf("expected duration >= 10ms, got %v", result.Duration)
	}

	if result.GoroutineStart <= 0 {
		t.Error("expected positive goroutine start count")
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := runtime.DefaultOptions()

	if opts.MaxGoroutineGrowth != 0 {
		t.Errorf("expected MaxGoroutineGrowth=0, got %d", opts.MaxGoroutineGrowth)
	}

	if opts.SettleTime != 100*time.Millisecond {
		t.Errorf("expected SettleTime=100ms, got %v", opts.SettleTime)
	}

	if opts.RetryCount != 3 {
		t.Errorf("expected RetryCount=3, got %d", opts.RetryCount)
	}
}

// MockT implements TestingT for testing
type MockT struct {
	errors []string
	logs   []string
}

func (m *MockT) Errorf(format string, args ...interface{}) {
	m.errors = append(m.errors, format)
}

func (m *MockT) Logf(format string, args ...interface{}) {
	m.logs = append(m.logs, format)
}

func (m *MockT) Helper() {}

func TestSnapshot_AssertNoLeak_Pass(t *testing.T) {
	mockT := &MockT{}
	snapshot := runtime.TakeSnapshot()

	// Do nothing that would leak
	time.Sleep(10 * time.Millisecond)

	snapshot.AssertNoLeak(mockT)

	if len(mockT.errors) > 0 {
		t.Errorf("expected no errors, got: %v", mockT.errors)
	}
}

// ExampleTakeSnapshot demonstrates basic snapshot usage
func ExampleTakeSnapshot() {
	// Take a snapshot at the start
	snapshot := runtime.TakeSnapshot()

	// ... run your code ...

	// Compare at the end
	diff := snapshot.Compare()

	if diff.GoroutineGrowth > 0 {
		// Handle potential leak
		_ = diff.LeakedGoroutines
	}
}

// ExampleAnalyze demonstrates the Analyze function
func ExampleAnalyze() {
	result := runtime.Analyze(func() {
		// Your code to analyze
	})

	if result.GoroutineLeak {
		// Handle goroutine leak
	}

	if result.HeapLeak {
		// Handle heap leak
	}
}
