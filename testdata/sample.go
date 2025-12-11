// Package testdata contains sample code for testing heapcheck
package testdata

import "fmt"

// ReturnPointer demonstrates return-pointer escape
func ReturnPointer() *int {
	x := 42
	return &x // x escapes to heap
}

// NoEscape demonstrates stack allocation
func NoEscape() int {
	x := 42
	return x // x does not escape
}

// InterfaceBoxing demonstrates interface boxing escape
func InterfaceBoxing() {
	x := 42
	fmt.Println(x) // x escapes via interface{}
}

// ClosureCapture demonstrates closure capture escape
func ClosureCapture() func() int {
	x := 42
	return func() int {
		return x // x escapes - captured by closure
	}
}

// GoroutineEscape demonstrates goroutine escape
func GoroutineEscape() {
	data := make([]byte, 1024)
	go func() {
		_ = data // data escapes to goroutine
	}()
}

// SliceGrow demonstrates slice growth escape
func SliceGrow(n int) []int {
	s := make([]int, 0) // may escape due to append
	for i := 0; i < n; i++ {
		s = append(s, i)
	}
	return s
}

// UnknownSize demonstrates unknown size escape
func UnknownSize(n int) []int {
	return make([]int, n) // escapes - size not known at compile time
}

// LargeStruct demonstrates large struct escape
type LargeStruct struct {
	data [1024 * 1024]byte // 1MB
}

func CreateLarge() LargeStruct {
	return LargeStruct{} // may escape due to size
}

// FmtCall demonstrates fmt escape
func FmtCall(x int) string {
	return fmt.Sprintf("value: %d", x) // x escapes via interface{}
}
