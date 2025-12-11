// Package patterns demonstrates common escape analysis patterns in Go.
// Run: heapcheck ./... to see detailed escape analysis
package patterns

import (
	"fmt"
	"strconv"
	"sync"
)

// =============================================================================
// Pattern 1: Return Pointer (ESCAPES)
// =============================================================================

// User represents a simple user struct
type User struct {
	ID   int
	Name string
	Age  int
}

// NewUserBad returns a pointer to a local variable - ESCAPES to heap
func NewUserBad(name string) *User {
	u := User{Name: name} // moved to heap!
	return &u
}

// NewUserGood returns by value - stays on stack
func NewUserGood(name string) User {
	return User{Name: name} // no escape
}

// NewUserWithStorage lets caller control allocation
func NewUserWithStorage(u *User, name string) {
	u.Name = name // no escape, caller provides storage
}

// =============================================================================
// Pattern 2: Interface Boxing (ESCAPES)
// =============================================================================

// LogBad uses interface{} - causes boxing and escape
func LogBad(msg interface{}) {
	fmt.Println(msg) // msg escapes via interface
}

// LogGood uses concrete type - no boxing
func LogGood(msg string) {
	fmt.Println(msg)
}

// LogGeneric uses generics (Go 1.18+) - no boxing for value types
func LogGeneric[T any](msg T) {
	_ = msg // no escape if T is concrete
}

// =============================================================================
// Pattern 3: Closure Capture (ESCAPES)
// =============================================================================

// ProcessBad captures variable in closure - ESCAPES
func ProcessBad(items []string) {
	for _, item := range items {
		go func() {
			_ = item // item escapes! captured by closure
		}()
	}
}

// ProcessGood passes variable as parameter - no capture escape
func ProcessGood(items []string) {
	for _, item := range items {
		go func(s string) {
			_ = s // no escape, passed as parameter
		}(item)
	}
}

// =============================================================================
// Pattern 4: Slice Growth (MAY ESCAPE)
// =============================================================================

// CollectBad doesn't pre-allocate - slice may escape
func CollectBad(n int) []int {
	var result []int // may escape due to growth
	for i := 0; i < n; i++ {
		result = append(result, i)
	}
	return result
}

// CollectGood pre-allocates capacity
func CollectGood(n int) []int {
	result := make([]int, 0, n) // known capacity
	for i := 0; i < n; i++ {
		result = append(result, i)
	}
	return result
}

// =============================================================================
// Pattern 5: fmt vs strconv (ESCAPES via interface)
// =============================================================================

// FormatIDBad uses fmt - causes interface boxing
func FormatIDBad(id int) string {
	return fmt.Sprintf("%d", id) // id boxed to interface{}
}

// FormatIDGood uses strconv - no boxing
func FormatIDGood(id int) string {
	return strconv.Itoa(id) // no interface, no escape
}

// =============================================================================
// Pattern 6: Map Allocation (ALWAYS ESCAPES)
// =============================================================================

// CreateMapBad - maps always escape to heap
func CreateMapBad() map[string]int {
	m := make(map[string]int) // always escapes
	m["key"] = 1
	return m
}

// Alternative: Use sync.Pool for frequently created maps
var mapPool = sync.Pool{
	New: func() interface{} {
		return make(map[string]int)
	},
}

// CreateMapPooled reuses maps from pool
func CreateMapPooled() map[string]int {
	m := mapPool.Get().(map[string]int)
	// Clear existing entries
	for k := range m {
		delete(m, k)
	}
	return m
}

// ReturnMapToPool returns map to pool when done
func ReturnMapToPool(m map[string]int) {
	mapPool.Put(m)
}

// =============================================================================
// Pattern 7: Channel Send (MAY ESCAPE)
// =============================================================================

// SendBad sends pointer on channel - escapes
func SendBad(ch chan *User) {
	u := &User{Name: "test"} // escapes
	ch <- u
}

// SendGood sends value on channel (for small structs)
func SendGood(ch chan User) {
	u := User{Name: "test"} // may still escape but smaller
	ch <- u
}

// =============================================================================
// Pattern 8: Large Structs (ESCAPES due to size)
// =============================================================================

// LargeStruct is too large for stack (>64KB typically)
type LargeStruct struct {
	Data [1024 * 100]byte // 100KB - will escape
}

// CreateLarge - large structs escape to heap
func CreateLarge() LargeStruct {
	var l LargeStruct // too large, escapes
	return l
}

// SmallStruct fits on stack
type SmallStruct struct {
	Data [64]byte // 64 bytes - stays on stack
}

// CreateSmall - small structs stay on stack
func CreateSmall() SmallStruct {
	var s SmallStruct // stays on stack
	return s
}
