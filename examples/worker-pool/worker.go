// Package worker demonstrates escape analysis in concurrent Go code.
// Goroutines and channels often cause escapes - here's how to minimize them.
package worker

import (
	"context"
	"sync"
)

// Task represents a unit of work
type Task struct {
	ID      int
	Payload []byte
}

// Result represents the output of processing a task
type Result struct {
	TaskID int
	Output []byte
	Err    error
}

// =============================================================================
// Pattern: Closure Capture in Goroutines
// =============================================================================

// ProcessTasksBad - closure captures task variable
func ProcessTasksBad(tasks []Task) []Result {
	var results []Result
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, task := range tasks {
		wg.Add(1)
		go func() { // BAD: closure captures 'task'
			defer wg.Done()
			result := processOne(task) // task escapes!

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}()
	}

	wg.Wait()
	return results
}

// ProcessTasksGood - passes task as parameter
func ProcessTasksGood(tasks []Task) []Result {
	var results []Result
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, task := range tasks {
		wg.Add(1)
		go func(t Task) { // GOOD: task passed as parameter
			defer wg.Done()
			result := processOne(t) // no capture escape

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(task)
	}

	wg.Wait()
	return results
}

// =============================================================================
// Pattern: Worker Pool (Best for High Throughput)
// =============================================================================

// WorkerPool manages a fixed number of workers
type WorkerPool struct {
	tasks   chan Task
	results chan Result
	workers int
	wg      sync.WaitGroup
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers, bufferSize int) *WorkerPool {
	return &WorkerPool{
		tasks:   make(chan Task, bufferSize),
		results: make(chan Result, bufferSize),
		workers: workers,
	}
}

// Start starts the worker pool
func (p *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx)
	}
}

// worker is a single worker goroutine
func (p *WorkerPool) worker(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			result := processOne(task)
			p.results <- result
		}
	}
}

// Submit submits a task to the pool
func (p *WorkerPool) Submit(task Task) {
	p.tasks <- task
}

// Results returns the results channel
func (p *WorkerPool) Results() <-chan Result {
	return p.results
}

// Close closes the pool and waits for workers to finish
func (p *WorkerPool) Close() {
	close(p.tasks)
	p.wg.Wait()
	close(p.results)
}

// =============================================================================
// Pattern: Channel Send Escapes
// =============================================================================

// SendPointerBad sends pointer on channel - always escapes
func SendPointerBad(ch chan *Result) {
	result := &Result{TaskID: 1} // escapes to heap
	ch <- result
}

// SendValueGood sends value on channel - smaller overhead
func SendValueGood(ch chan Result) {
	result := Result{TaskID: 1} // may still escape but value copy
	ch <- result
}

// =============================================================================
// Pattern: Buffered vs Unbuffered Channels
// =============================================================================

// UnbufferedBad - unbuffered channel causes more synchronization
func UnbufferedBad() chan Task {
	return make(chan Task) // unbuffered - blocking
}

// BufferedGood - buffered channel reduces blocking
func BufferedGood(size int) chan Task {
	return make(chan Task, size) // buffered - less blocking
}

// =============================================================================
// Helper Functions
// =============================================================================

func processOne(task Task) Result {
	// Simulate processing
	return Result{
		TaskID: task.ID,
		Output: task.Payload,
	}
}

// =============================================================================
// Pattern: sync.Pool for Task/Result Reuse
// =============================================================================

var taskPool = sync.Pool{
	New: func() interface{} {
		return &Task{
			Payload: make([]byte, 0, 1024), // pre-allocate
		}
	},
}

var resultPool = sync.Pool{
	New: func() interface{} {
		return &Result{
			Output: make([]byte, 0, 1024),
		}
	},
}

// GetTask gets a task from pool
func GetTask() *Task {
	t := taskPool.Get().(*Task)
	t.ID = 0
	t.Payload = t.Payload[:0] // reset but keep capacity
	return t
}

// PutTask returns task to pool
func PutTask(t *Task) {
	taskPool.Put(t)
}

// GetResult gets a result from pool
func GetResult() *Result {
	r := resultPool.Get().(*Result)
	r.TaskID = 0
	r.Output = r.Output[:0]
	r.Err = nil
	return r
}

// PutResult returns result to pool
func PutResult(r *Result) {
	resultPool.Put(r)
}
