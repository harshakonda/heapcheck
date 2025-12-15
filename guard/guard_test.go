package guard_test

import (
	"testing"
	"time"

	"github.com/harshakonda/heapcheck/guard"
)

func TestVerifyNone_NoLeak(t *testing.T) {
	defer guard.VerifyNone(t)

	// This should not leak
	x := make([]int, 100)
	_ = x
}

func TestVerifyNone_WithOptions(t *testing.T) {
	defer guard.VerifyNone(t,
		guard.MaxGoroutines(5),
		guard.MaxHeapMB(100),
		guard.SettleTime(50*time.Millisecond),
	)

	// This should not leak
	done := make(chan struct{})
	go func() {
		time.Sleep(10 * time.Millisecond)
		close(done)
	}()
	<-done
}

func TestCheck_Checkpoint(t *testing.T) {
	g := guard.Check(t)

	// Phase 1
	data := make([]byte, 1024)
	_ = data
	g.Checkpoint("after allocation")

	// Phase 2
	data = nil
	g.Checkpoint("after nil")

	g.Verify()
}

func TestCheck_Reset(t *testing.T) {
	g := guard.Check(t)

	// Do something that might allocate
	data := make([]byte, 1024*1024)
	_ = data

	// Reset baseline
	g.Reset()

	// Now this is the new baseline
	g.Verify()
}

// Example of testing with ignored goroutines
func TestVerifyNone_WithIgnore(t *testing.T) {
	defer guard.VerifyNone(t,
		guard.IgnoreContains("testing.tRunner"),
	)

	// Test code
}

// ExampleVerifyNone demonstrates basic usage
func ExampleVerifyNone() {
	// In your test file:
	//
	// func TestMyFunction(t *testing.T) {
	//     defer guard.VerifyNone(t)
	//
	//     result := myFunction()
	//     if result != expected {
	//         t.Error("unexpected result")
	//     }
	// }
}

// ExampleVerifyNone_withOptions demonstrates usage with options
func ExampleVerifyNone_withOptions() {
	// In your test file:
	//
	// func TestMyFunction(t *testing.T) {
	//     defer guard.VerifyNone(t,
	//         guard.MaxGoroutines(5),    // Allow up to 5 transient goroutines
	//         guard.MaxHeapMB(50),       // Allow up to 50MB heap growth
	//     )
	//
	//     result := myFunction()
	//     if result != expected {
	//         t.Error("unexpected result")
	//     }
	// }
}

// ExampleVerifyTestMain demonstrates package-level leak detection
func ExampleVerifyTestMain() {
	// In your test file:
	//
	// func TestMain(m *testing.M) {
	//     guard.VerifyTestMain(m)
	// }
}
