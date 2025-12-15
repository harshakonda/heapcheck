package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/harshakonda/heapcheck/guard"
)

func TestProcessTasksGood(t *testing.T) {
	defer guard.VerifyNone(t,
		guard.MaxGoroutines(5),
		guard.SettleTime(200*time.Millisecond),
	)

	tasks := []Task{
		{ID: 1, Payload: []byte("task1")},
		{ID: 2, Payload: []byte("task2")},
		{ID: 3, Payload: []byte("task3")},
	}

	results := ProcessTasksGood(tasks)
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestWorkerPool(t *testing.T) {
	defer guard.VerifyNone(t,
		guard.MaxGoroutines(5),
		guard.SettleTime(200*time.Millisecond),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool := NewWorkerPool(2, 10)
	pool.Start(ctx)

	// Submit tasks
	pool.Submit(Task{ID: 1, Payload: []byte("task1")})
	pool.Submit(Task{ID: 2, Payload: []byte("task2")})

	// Collect results with proper synchronization
	var results []Result
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for r := range pool.Results() {
			mu.Lock()
			results = append(results, r)
			mu.Unlock()
		}
	}()

	// Close and wait for collection to finish
	pool.Close()
	wg.Wait()

	mu.Lock()
	resultCount := len(results)
	mu.Unlock()

	if resultCount != 2 {
		t.Errorf("expected 2 results, got %d", resultCount)
	}
}

func TestChannelPatterns(t *testing.T) {
	defer guard.VerifyNone(t)

	// Test buffered channel
	ch := BufferedGood(10)
	ch <- Task{ID: 1}
	task := <-ch
	if task.ID != 1 {
		t.Errorf("expected ID 1, got %d", task.ID)
	}
}

func TestSyncPool(t *testing.T) {
	defer guard.VerifyNone(t)

	// Get from pool
	task := GetTask()
	task.ID = 42
	task.Payload = append(task.Payload, []byte("test")...)

	if task.ID != 42 {
		t.Errorf("expected ID 42, got %d", task.ID)
	}

	// Return to pool
	PutTask(task)

	// Get result from pool
	result := GetResult()
	result.TaskID = 1
	result.Output = append(result.Output, []byte("output")...)

	if result.TaskID != 1 {
		t.Errorf("expected TaskID 1, got %d", result.TaskID)
	}

	// Return to pool
	PutResult(result)
}

func TestSendValueGood(t *testing.T) {
	defer guard.VerifyNone(t)

	ch := make(chan Result, 1)
	SendValueGood(ch)

	result := <-ch
	if result.TaskID != 1 {
		t.Errorf("expected TaskID 1, got %d", result.TaskID)
	}
}
