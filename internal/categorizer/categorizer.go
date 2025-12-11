// Package categorizer analyzes escape information and categorizes it
// by the cause of escape, adding actionable suggestions.
package categorizer

import (
	"strings"

	"github.com/harshakonda/heapcheck/internal/parser"
)

// Category represents why a variable escaped to the heap
type Category string

const (
	CategoryReturnPointer     Category = "return-pointer"
	CategoryInterfaceBoxing   Category = "interface-boxing"
	CategoryClosureCapture    Category = "closure-capture"
	CategoryGoroutineEscape   Category = "goroutine-escape"
	CategoryChannelSend       Category = "channel-send"
	CategorySliceGrow         Category = "slice-grow"
	CategoryUnknownSize       Category = "unknown-size"
	CategoryTooLarge          Category = "too-large"
	CategoryFmtCall           Category = "fmt-call"
	CategoryReflection        Category = "reflection"
	CategoryLeakingParam      Category = "leaking-param"
	CategoryStringConversion  Category = "string-conversion"
	CategorySpill             Category = "spill"
	CategoryAssignment        Category = "assignment"
	CategoryCallParameter     Category = "call-parameter"
	CategoryMapAllocation     Category = "map-allocation"
	CategoryNewAllocation     Category = "new-allocation"
	CategoryCompositeLiteral  Category = "composite-literal"
	CategoryUncategorized     Category = "uncategorized"
)

// Suggestion provides optimization advice for a category
type Suggestion struct {
	Short   string `json:"short"`
	Details string `json:"details"`
	DocLink string `json:"docLink,omitempty"`
}

// CategorizedEscape combines escape info with category and suggestion
type CategorizedEscape struct {
	Info       parser.EscapeInfo `json:"info"`
	Category   Category          `json:"category"`
	Suggestion Suggestion        `json:"suggestion"`
}

// Summary holds aggregate statistics
type Summary struct {
	TotalVariables int            `json:"totalVariables"`
	StackAllocated int            `json:"stackAllocated"`
	HeapAllocated  int            `json:"heapAllocated"`
	Inlined        int            `json:"inlined"`
	ByFile         map[string]int `json:"byFile"`
}

// Results holds the complete categorization results
type Results struct {
	Summary    Summary                  `json:"summary"`
	ByCategory map[Category]int         `json:"byCategory"`
	Escapes    []CategorizedEscape      `json:"escapes"`
}

// suggestions maps categories to their suggestions
var suggestions = map[Category]Suggestion{
	CategoryReturnPointer: {
		Short:   "Return by value if struct size ≤ 64 bytes",
		Details: "Returning a pointer to a local variable forces heap allocation. If the struct is small, return by value instead. For larger structs in hot paths, consider sync.Pool.",
		DocLink: "https://go.dev/doc/faq#stack_or_heap",
	},
	CategoryInterfaceBoxing: {
		Short:   "Use concrete types in hot paths",
		Details: "Assigning to interface{} or any causes heap allocation for the type metadata. Use generics (Go 1.18+) or concrete types in performance-critical code.",
		DocLink: "https://go.dev/blog/intro-generics",
	},
	CategoryClosureCapture: {
		Short:   "Pass variables as parameters instead of capturing",
		Details: "Variables captured by closures often escape. Pass them as function parameters instead, especially for goroutines.",
	},
	CategoryGoroutineEscape: {
		Short:   "Consider worker pools for high-frequency goroutines",
		Details: "Variables passed to goroutines must outlive the creating function and thus escape. For high-throughput scenarios, use worker pools with pre-allocated buffers.",
	},
	CategoryChannelSend: {
		Short:   "Buffer channels or use sync.Pool for sent values",
		Details: "Values sent on channels may escape. For frequently sent large objects, consider using sync.Pool.",
	},
	CategorySliceGrow: {
		Short:   "Pre-allocate slice capacity",
		Details: "Slices that may grow via append can escape. Pre-allocate with make([]T, 0, expectedCap) when the final size is predictable.",
	},
	CategoryUnknownSize: {
		Short:   "Use fixed-size arrays when length is known",
		Details: "make([]T, n) with non-constant n causes heap allocation. If size is known at compile time, use arrays [N]T or pre-allocate.",
	},
	CategoryTooLarge: {
		Short:   "Large allocations go to heap by design",
		Details: "Very large structs or arrays are placed on heap regardless of escape. Consider if the full size is necessary or if you can use pointers to smaller chunks.",
	},
	CategoryFmtCall: {
		Short:   "Use strconv in hot paths",
		Details: "fmt.Sprintf and similar cause interface boxing. Use strconv.Itoa, strconv.FormatFloat, etc. for simple conversions in hot paths.",
	},
	CategoryReflection: {
		Short:   "Avoid reflect in hot paths",
		Details: "Reflection defeats escape analysis. Avoid reflect package in performance-critical code; use code generation or generics instead.",
	},
	CategoryLeakingParam: {
		Short:   "Parameter escapes function scope",
		Details: "This parameter is stored or returned, causing it to escape. Consider if the storage is necessary or if you can restructure to avoid it.",
	},
	CategoryStringConversion: {
		Short:   "String conversion allocates",
		Details: "Converting []byte to string (or vice versa) allocates. In hot paths, consider using unsafe conversion or reusing buffers.",
	},
	CategorySpill: {
		Short:   "Compiler spilled value to heap",
		Details: "The compiler determined this value may outlive the stack frame. Check if the value is stored in a long-lived data structure.",
	},
	CategoryAssignment: {
		Short:   "Value assigned to escaping location",
		Details: "This value is assigned to a variable that escapes (field, global, etc.). Consider if the assignment is necessary.",
	},
	CategoryCallParameter: {
		Short:   "Value escapes via function call",
		Details: "This value is passed to a function that causes it to escape. Check if the called function stores the parameter.",
	},
	CategoryMapAllocation: {
		Short:   "Maps always allocate on heap",
		Details: "Maps in Go always escape to heap. Consider using arrays for small fixed-size lookups, or sync.Pool for frequently created maps.",
	},
	CategoryNewAllocation: {
		Short:   "new() always allocates on heap",
		Details: "The new() builtin allocates on heap. For small structs, consider stack allocation with var x T followed by &x if needed.",
	},
	CategoryCompositeLiteral: {
		Short:   "Composite literal escapes",
		Details: "Struct/slice/map literals that escape the function are heap allocated. For hot paths, consider reusing allocations.",
	},
	CategoryUncategorized: {
		Short:   "Review escape flow details",
		Details: "This escape couldn't be automatically categorized. Check the flow information for details on why the variable escapes.",
	},
}

// Categorize processes escape info and adds categories and suggestions
func Categorize(escapes []parser.EscapeInfo) *Results {
	results := &Results{
		Summary: Summary{
			ByFile: make(map[string]int),
		},
		ByCategory: make(map[Category]int),
		Escapes:    make([]CategorizedEscape, 0, len(escapes)),
	}

	for _, e := range escapes {
		results.Summary.TotalVariables++

		switch e.EscapeType {
		case parser.DoesNotEscape:
			results.Summary.StackAllocated++
		case parser.MovedToHeap, parser.EscapesToHeap, parser.LeakingParam:
			results.Summary.HeapAllocated++
			results.Summary.ByFile[e.File]++

			cat := categorize(e)
			results.ByCategory[cat]++

			results.Escapes = append(results.Escapes, CategorizedEscape{
				Info:       e,
				Category:   cat,
				Suggestion: suggestions[cat],
			})
		case parser.CanInline, parser.InliningCall:
			results.Summary.Inlined++
		}
	}

	return results
}

// categorize determines the category based on escape info and flow details
func categorize(e parser.EscapeInfo) Category {
	reason := strings.ToLower(e.Reason)
	flowInfo := strings.ToLower(strings.Join(e.FlowInfo, " "))
	combined := reason + " " + flowInfo
	variable := strings.ToLower(e.Variable)

	// === HIGH CONFIDENCE PATTERNS ===

	// Return pointer pattern: "from return &x" or "from &x (address-of)"
	if strings.Contains(flowInfo, "from return") && strings.Contains(flowInfo, "&") {
		return CategoryReturnPointer
	}
	if strings.Contains(flowInfo, "address-of") && strings.Contains(flowInfo, "return") {
		return CategoryReturnPointer
	}

	// Interface conversion: "interface-converted" in flow
	if strings.Contains(flowInfo, "interface-converted") {
		return CategoryInterfaceBoxing
	}
	if strings.Contains(combined, "interface") {
		return CategoryInterfaceBoxing
	}

	// Closure capture
	if strings.Contains(combined, "closure") || strings.Contains(combined, "captured") {
		return CategoryClosureCapture
	}

	// Goroutine escape
	if strings.Contains(combined, "go func") || strings.Contains(combined, "goroutine") {
		return CategoryGoroutineEscape
	}

	// Channel operations
	if strings.Contains(combined, "chan") || strings.Contains(combined, "channel") {
		return CategoryChannelSend
	}

	// Slice/append patterns
	if strings.Contains(combined, "append") {
		return CategorySliceGrow
	}
	if strings.Contains(flowInfo, "appended") {
		return CategorySliceGrow
	}

	// Unknown size at compile time
	if strings.Contains(combined, "non-constant") {
		return CategoryUnknownSize
	}

	// Too large for stack
	if strings.Contains(combined, "too large") {
		return CategoryTooLarge
	}

	// fmt package calls
	if strings.Contains(combined, "fmt.") {
		return CategoryFmtCall
	}

	// Reflection
	if strings.Contains(combined, "reflect") {
		return CategoryReflection
	}

	// === MEDIUM CONFIDENCE PATTERNS ===

	// Leaking param often means it's stored somewhere or returned
	if e.EscapeType == parser.LeakingParam {
		// Check if it's leaking to result (return value)
		if strings.Contains(reason, "to result") {
			return CategoryReturnPointer
		}
		// Leaking param content usually means interface boxing or slice
		if strings.Contains(reason, "content") {
			return CategoryInterfaceBoxing
		}
		return CategoryLeakingParam
	}

	// String conversion often escapes (string(bytes))
	if strings.Contains(variable, "string(") {
		return CategoryStringConversion
	}

	// Spill to heap (compiler decision)
	if strings.Contains(flowInfo, "spill") {
		return CategorySpill
	}

	// Moved to heap without clear reason - check flow
	if e.EscapeType == parser.MovedToHeap {
		// Check for assign patterns
		if strings.Contains(flowInfo, "assign") {
			// Assigned to field or external variable
			return CategoryAssignment
		}
		// Check for call parameter
		if strings.Contains(flowInfo, "call parameter") {
			return CategoryCallParameter
		}
	}

	// Variadic arguments (... interface{})
	if strings.Contains(variable, "...") || strings.Contains(reason, "... argument") {
		return CategoryInterfaceBoxing
	}

	// === LOWER CONFIDENCE PATTERNS ===

	// Map allocations
	if strings.Contains(variable, "make(map") || strings.Contains(reason, "make(map") {
		return CategoryMapAllocation
	}

	// Slice make (not append)
	if strings.Contains(variable, "make([]") || strings.Contains(reason, "make([]") {
		return CategorySliceGrow
	}

	// New allocations
	if strings.Contains(variable, "new(") || strings.Contains(reason, "new(") {
		return CategoryNewAllocation
	}

	// Composite literals (struct{}{}, []T{}, map[]{})
	if strings.Contains(variable, "literal") || strings.Contains(reason, "literal") {
		return CategoryCompositeLiteral
	}

	// &composite literal
	if strings.Contains(reason, "&") && !strings.Contains(flowInfo, "return") {
		return CategoryCompositeLiteral
	}

	// === FALLBACK ===
	return CategoryUncategorized
}

// GetSuggestion returns the suggestion for a category
func GetSuggestion(cat Category) Suggestion {
	if s, ok := suggestions[cat]; ok {
		return s
	}
	return suggestions[CategoryUncategorized]
}
