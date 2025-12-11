package categorizer

import (
	"testing"

	"github.com/harshakonda/heapcheck/internal/parser"
)

func TestCategorize(t *testing.T) {
	tests := []struct {
		name     string
		escape   parser.EscapeInfo
		expected Category
	}{
		{
			name: "return pointer",
			escape: parser.EscapeInfo{
				EscapeType: parser.MovedToHeap,
				Variable:   "u",
				Reason:     "moved to heap: u",
				FlowInfo:   []string{"from return &u (address-of)", "from &u (return)"},
			},
			expected: CategoryReturnPointer,
		},
		{
			name: "interface boxing",
			escape: parser.EscapeInfo{
				EscapeType: parser.EscapesToHeap,
				Variable:   "msg",
				Reason:     "msg escapes to heap",
				FlowInfo:   []string{"flow: interface-converted"},
			},
			expected: CategoryInterfaceBoxing,
		},
		{
			name: "closure capture",
			escape: parser.EscapeInfo{
				EscapeType: parser.EscapesToHeap,
				Variable:   "item",
				Reason:     "item escapes to heap",
				FlowInfo:   []string{"captured by closure"},
			},
			expected: CategoryClosureCapture,
		},
		{
			name: "goroutine escape",
			escape: parser.EscapeInfo{
				EscapeType: parser.EscapesToHeap,
				Variable:   "data",
				Reason:     "data escapes to heap",
				FlowInfo:   []string{"go func literal"},
			},
			expected: CategoryGoroutineEscape,
		},
		{
			name: "channel send",
			escape: parser.EscapeInfo{
				EscapeType: parser.EscapesToHeap,
				Variable:   "msg",
				Reason:     "msg escapes to heap",
				FlowInfo:   []string{"sent to channel"},
			},
			expected: CategoryChannelSend,
		},
		{
			name: "slice grow - append",
			escape: parser.EscapeInfo{
				EscapeType: parser.EscapesToHeap,
				Variable:   "result",
				Reason:     "result escapes to heap",
				FlowInfo:   []string{"appended to slice"},
			},
			expected: CategorySliceGrow,
		},
		{
			name: "unknown size",
			escape: parser.EscapeInfo{
				EscapeType: parser.EscapesToHeap,
				Variable:   "buf",
				Reason:     "make([]byte, n) escapes to heap",
				FlowInfo:   []string{"non-constant size"},
			},
			expected: CategoryUnknownSize,
		},
		{
			name: "too large",
			escape: parser.EscapeInfo{
				EscapeType: parser.MovedToHeap,
				Variable:   "big",
				Reason:     "moved to heap: big",
				FlowInfo:   []string{"too large for stack"},
			},
			expected: CategoryTooLarge,
		},
		{
			name: "fmt call",
			escape: parser.EscapeInfo{
				EscapeType: parser.EscapesToHeap,
				Variable:   "x",
				Reason:     "x escapes to heap",
				FlowInfo:   []string{"fmt.Println(x)"},
			},
			expected: CategoryFmtCall,
		},
		{
			name: "reflection",
			escape: parser.EscapeInfo{
				EscapeType: parser.EscapesToHeap,
				Variable:   "v",
				Reason:     "v escapes to heap",
				FlowInfo:   []string{"reflect.ValueOf"},
			},
			expected: CategoryReflection,
		},
		{
			name: "leaking param to result",
			escape: parser.EscapeInfo{
				EscapeType: parser.LeakingParam,
				Variable:   "s",
				Reason:     "leaking param: s to result",
				FlowInfo:   []string{},
			},
			expected: CategoryReturnPointer,
		},
		{
			name: "leaking param content",
			escape: parser.EscapeInfo{
				EscapeType: parser.LeakingParam,
				Variable:   "data",
				Reason:     "leaking param content: data",
				FlowInfo:   []string{},
			},
			expected: CategoryInterfaceBoxing,
		},
		{
			name: "leaking param generic",
			escape: parser.EscapeInfo{
				EscapeType: parser.LeakingParam,
				Variable:   "x",
				Reason:     "leaking param: x",
				FlowInfo:   []string{},
			},
			expected: CategoryLeakingParam,
		},
		{
			name: "string conversion",
			escape: parser.EscapeInfo{
				EscapeType: parser.EscapesToHeap,
				Variable:   "string(bytes)",
				Reason:     "string(bytes) escapes to heap",
				FlowInfo:   []string{},
			},
			expected: CategoryStringConversion,
		},
		{
			name: "spill",
			escape: parser.EscapeInfo{
				EscapeType: parser.EscapesToHeap,
				Variable:   "x",
				Reason:     "x escapes to heap",
				FlowInfo:   []string{"spill"},
			},
			expected: CategorySpill,
		},
		{
			name: "map allocation",
			escape: parser.EscapeInfo{
				EscapeType: parser.EscapesToHeap,
				Variable:   "make(map[string]int)",
				Reason:     "make(map[string]int) escapes to heap",
				FlowInfo:   []string{},
			},
			expected: CategoryMapAllocation,
		},
		{
			name: "slice make",
			escape: parser.EscapeInfo{
				EscapeType: parser.EscapesToHeap,
				Variable:   "make([]int, 10)",
				Reason:     "make([]int, 10) escapes to heap",
				FlowInfo:   []string{},
			},
			expected: CategorySliceGrow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := Categorize([]parser.EscapeInfo{tt.escape})
			if len(results.Escapes) != 1 {
				t.Fatalf("expected 1 escape result, got %d", len(results.Escapes))
			}
			if results.Escapes[0].Category != tt.expected {
				t.Errorf("expected category %s, got %s", tt.expected, results.Escapes[0].Category)
			}
		})
	}
}

func TestGetSuggestion(t *testing.T) {
	categories := []Category{
		CategoryReturnPointer,
		CategoryInterfaceBoxing,
		CategoryClosureCapture,
		CategoryGoroutineEscape,
		CategoryChannelSend,
		CategorySliceGrow,
		CategoryUnknownSize,
		CategoryTooLarge,
		CategoryFmtCall,
		CategoryReflection,
		CategoryLeakingParam,
		CategoryStringConversion,
		CategorySpill,
		CategoryMapAllocation,
		CategoryUncategorized,
	}

	for _, cat := range categories {
		t.Run(string(cat), func(t *testing.T) {
			suggestion := GetSuggestion(cat)
			if suggestion.Short == "" {
				t.Errorf("category %s has empty short suggestion", cat)
			}
			if suggestion.Details == "" {
				t.Errorf("category %s has empty details", cat)
			}
		})
	}
}

func TestCategorizeEmpty(t *testing.T) {
	results := Categorize([]parser.EscapeInfo{})
	if len(results.Escapes) != 0 {
		t.Errorf("expected empty escapes for empty input, got %d", len(results.Escapes))
	}
	if results.Summary.TotalVariables != 0 {
		t.Errorf("expected 0 total variables, got %d", results.Summary.TotalVariables)
	}
}

func TestCategorizeMultiple(t *testing.T) {
	escapes := []parser.EscapeInfo{
		{
			EscapeType: parser.MovedToHeap,
			Variable:   "x",
			Reason:     "moved to heap: x",
			FlowInfo:   []string{"from return &x"},
		},
		{
			EscapeType: parser.EscapesToHeap,
			Variable:   "y",
			Reason:     "y escapes to heap",
			FlowInfo:   []string{"interface-converted"},
		},
	}

	results := Categorize(escapes)
	if len(results.Escapes) != 2 {
		t.Fatalf("expected 2 escape results, got %d", len(results.Escapes))
	}

	if results.Escapes[0].Category != CategoryReturnPointer {
		t.Errorf("first escape: expected %s, got %s", CategoryReturnPointer, results.Escapes[0].Category)
	}
	if results.Escapes[1].Category != CategoryInterfaceBoxing {
		t.Errorf("second escape: expected %s, got %s", CategoryInterfaceBoxing, results.Escapes[1].Category)
	}
}

func TestCategorizeCountsCorrectly(t *testing.T) {
	escapes := []parser.EscapeInfo{
		{EscapeType: parser.DoesNotEscape, Variable: "a"},
		{EscapeType: parser.DoesNotEscape, Variable: "b"},
		{EscapeType: parser.MovedToHeap, Variable: "c", Reason: "moved to heap: c", FlowInfo: []string{"return &c"}},
		{EscapeType: parser.EscapesToHeap, Variable: "d", Reason: "d escapes", FlowInfo: []string{"interface"}},
		{EscapeType: parser.CanInline, Variable: "func"},
		{EscapeType: parser.InliningCall, Variable: "call"},
	}

	results := Categorize(escapes)

	if results.Summary.TotalVariables != 6 {
		t.Errorf("expected 6 total, got %d", results.Summary.TotalVariables)
	}
	if results.Summary.StackAllocated != 2 {
		t.Errorf("expected 2 stack, got %d", results.Summary.StackAllocated)
	}
	if results.Summary.HeapAllocated != 2 {
		t.Errorf("expected 2 heap, got %d", results.Summary.HeapAllocated)
	}
	if results.Summary.Inlined != 2 {
		t.Errorf("expected 2 inlined, got %d", results.Summary.Inlined)
	}
}
