package parser

import (
	"testing"
)

func TestParseMovedToHeap(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantFile string
		wantLine int
		wantVar  string
	}{
		{
			name:     "basic moved to heap",
			input:    "./main.go:12:2: moved to heap: z",
			wantFile: "./main.go",
			wantLine: 12,
			wantVar:  "z",
		},
		{
			name:     "nested path",
			input:    "./pkg/server/handler.go:45:8: moved to heap: data",
			wantFile: "./pkg/server/handler.go",
			wantLine: 45,
			wantVar:  "data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("Parse() got %d results, want 1", len(results))
			}
			r := results[0]
			if r.File != tt.wantFile {
				t.Errorf("File = %v, want %v", r.File, tt.wantFile)
			}
			if r.Line != tt.wantLine {
				t.Errorf("Line = %v, want %v", r.Line, tt.wantLine)
			}
			if r.Variable != tt.wantVar {
				t.Errorf("Variable = %v, want %v", r.Variable, tt.wantVar)
			}
			if r.EscapeType != MovedToHeap {
				t.Errorf("EscapeType = %v, want MovedToHeap", r.EscapeType)
			}
		})
	}
}

func TestParseEscapesToHeap(t *testing.T) {
	input := "./main.go:8:14: *y escapes to heap"
	results, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Parse() got %d results, want 1", len(results))
	}
	r := results[0]
	if r.EscapeType != EscapesToHeap {
		t.Errorf("EscapeType = %v, want EscapesToHeap", r.EscapeType)
	}
	if r.Variable != "*y" {
		t.Errorf("Variable = %v, want *y", r.Variable)
	}
}

func TestParseDoesNotEscape(t *testing.T) {
	input := "./main.go:11:13: x does not escape"
	results, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Parse() got %d results, want 1", len(results))
	}
	r := results[0]
	if r.EscapeType != DoesNotEscape {
		t.Errorf("EscapeType = %v, want DoesNotEscape", r.EscapeType)
	}
}

func TestParseLeakingParam(t *testing.T) {
	input := "./main.go:20:6: leaking param: p to result ~r0 level=0"
	results, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Parse() got %d results, want 1", len(results))
	}
	r := results[0]
	if r.EscapeType != LeakingParam {
		t.Errorf("EscapeType = %v, want LeakingParam", r.EscapeType)
	}
}

func TestParseInlining(t *testing.T) {
	input := `./main.go:15:6: can inline square
./main.go:35:10: inlining call to foo`

	results, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Parse() got %d results, want 2", len(results))
	}
	if results[0].EscapeType != CanInline {
		t.Errorf("results[0].EscapeType = %v, want CanInline", results[0].EscapeType)
	}
	if results[1].EscapeType != InliningCall {
		t.Errorf("results[1].EscapeType = %v, want InliningCall", results[1].EscapeType)
	}
}

func TestParseMultipleLines(t *testing.T) {
	input := `./main.go:15:6: can inline square
./main.go:7:13: inlining call to square
./main.go:8:13: inlining call to fmt.Println
./main.go:12:2: moved to heap: z
./main.go:8:14: *y escapes to heap
./main.go:11:13: x does not escape`

	results, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(results) != 6 {
		t.Fatalf("Parse() got %d results, want 6", len(results))
	}

	// Check counts by type
	counts := make(map[EscapeType]int)
	for _, r := range results {
		counts[r.EscapeType]++
	}

	if counts[CanInline] != 1 {
		t.Errorf("CanInline count = %d, want 1", counts[CanInline])
	}
	if counts[InliningCall] != 2 {
		t.Errorf("InliningCall count = %d, want 2", counts[InliningCall])
	}
	if counts[MovedToHeap] != 1 {
		t.Errorf("MovedToHeap count = %d, want 1", counts[MovedToHeap])
	}
	if counts[EscapesToHeap] != 1 {
		t.Errorf("EscapesToHeap count = %d, want 1", counts[EscapesToHeap])
	}
	if counts[DoesNotEscape] != 1 {
		t.Errorf("DoesNotEscape count = %d, want 1", counts[DoesNotEscape])
	}
}

func TestParseWithFlowInfo(t *testing.T) {
	input := `./main.go:10:2: x escapes to heap:
./main.go:10:2:   flow: ~r0 = &x:
./main.go:10:2:     from &x (address-of) at ./main.go:10:9
./main.go:10:2:     from return &x (return) at ./main.go:10:2`

	results, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Parse() got %d results, want 1", len(results))
	}
	r := results[0]
	if len(r.FlowInfo) != 3 {
		t.Errorf("FlowInfo length = %d, want 3", len(r.FlowInfo))
	}
}

func TestEscapeTypeString(t *testing.T) {
	tests := []struct {
		et   EscapeType
		want string
	}{
		{MovedToHeap, "moved-to-heap"},
		{EscapesToHeap, "escapes-to-heap"},
		{DoesNotEscape, "does-not-escape"},
		{LeakingParam, "leaking-param"},
		{CanInline, "can-inline"},
		{InliningCall, "inlining-call"},
		{Unknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.et.String(); got != tt.want {
				t.Errorf("EscapeType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
