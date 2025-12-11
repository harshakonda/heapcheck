package reporter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/harshakonda/heapcheck/internal/categorizer"
	"github.com/harshakonda/heapcheck/internal/parser"
)

func sampleResults() *categorizer.Results {
	return &categorizer.Results{
		Summary: categorizer.Summary{
			TotalVariables: 3,
			StackAllocated: 1,
			HeapAllocated:  2,
			Inlined:        0,
			ByFile: map[string]int{
				"main.go":    1,
				"handler.go": 1,
			},
		},
		ByCategory: map[categorizer.Category]int{
			categorizer.CategoryReturnPointer:   1,
			categorizer.CategoryInterfaceBoxing: 1,
		},
		Escapes: []categorizer.CategorizedEscape{
			{
				Info: parser.EscapeInfo{
					File:       "main.go",
					Line:       10,
					Column:     5,
					Variable:   "x",
					EscapeType: parser.EscapesToHeap,
					Reason:     "x escapes to heap",
				},
				Category: categorizer.CategoryReturnPointer,
				Suggestion: categorizer.Suggestion{
					Short:   "Return by value",
					Details: "Return by value if struct ≤ 64 bytes",
				},
			},
			{
				Info: parser.EscapeInfo{
					File:       "handler.go",
					Line:       25,
					Column:     12,
					Variable:   "req",
					EscapeType: parser.EscapesToHeap,
					Reason:     "req escapes to heap",
				},
				Category: categorizer.CategoryInterfaceBoxing,
				Suggestion: categorizer.Suggestion{
					Short:   "Use concrete types",
					Details: "Use generics or concrete types",
				},
			},
		},
	}
}

func TestTextReporter(t *testing.T) {
	results := sampleResults()
	var buf bytes.Buffer

	reporter := NewTextReporter(&buf, false)
	err := reporter.Report(results)
	if err != nil {
		t.Fatalf("Text reporter failed: %v", err)
	}

	output := buf.String()

	// Check for key elements
	checks := []string{
		"heapcheck",
		"Summary",
		"Total variables analyzed",
		"Stack allocated",
		"Heap allocated",
		"Escape Causes",
		"Hotspots",
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Text output missing: %s", check)
		}
	}
}

func TestTextReporterVerbose(t *testing.T) {
	results := sampleResults()
	var buf bytes.Buffer

	reporter := NewTextReporter(&buf, true)
	err := reporter.Report(results)
	if err != nil {
		t.Fatalf("Text reporter (verbose) failed: %v", err)
	}

	output := buf.String()

	// Verbose should include details
	checks := []string{
		"main.go",
		"handler.go",
		"return-pointer",
		"interface-boxing",
		"💡", // suggestion marker
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Verbose output missing: %s", check)
		}
	}
}

func TestJSONReporter(t *testing.T) {
	results := sampleResults()
	var buf bytes.Buffer

	reporter := NewJSONReporter(&buf)
	err := reporter.Report(results)
	if err != nil {
		t.Fatalf("JSON reporter failed: %v", err)
	}

	// Verify valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// Check structure
	if _, ok := result["summary"]; !ok {
		t.Error("JSON missing 'summary' field")
	}
	if _, ok := result["byCategory"]; !ok {
		t.Error("JSON missing 'byCategory' field")
	}
	if _, ok := result["escapes"]; !ok {
		t.Error("JSON missing 'escapes' field")
	}
}

func TestHTMLReporter(t *testing.T) {
	results := sampleResults()
	var buf bytes.Buffer

	reporter := NewHTMLReporter(&buf)
	err := reporter.Report(results)
	if err != nil {
		t.Fatalf("HTML reporter failed: %v", err)
	}

	output := buf.String()

	// Check for HTML structure
	checks := []string{
		"<!DOCTYPE html>",
		"<html",
		"</html>",
		"heapcheck Report",
		"chart.js",
		"main.go",
		"handler.go",
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("HTML output missing: %s", check)
		}
	}
}

func TestSARIFReporter(t *testing.T) {
	results := sampleResults()
	var buf bytes.Buffer

	reporter := NewSARIFReporter(&buf)
	err := reporter.Report(results)
	if err != nil {
		t.Fatalf("SARIF reporter failed: %v", err)
	}

	// Verify valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid SARIF JSON: %v", err)
	}

	// Check SARIF structure
	if result["$schema"] == nil {
		t.Error("SARIF missing '$schema' field")
	}
	if result["version"] != "2.1.0" {
		t.Errorf("Expected SARIF version 2.1.0, got %v", result["version"])
	}
	if result["runs"] == nil {
		t.Error("SARIF missing 'runs' field")
	}
}

func TestEmptyResults(t *testing.T) {
	results := &categorizer.Results{
		Summary: categorizer.Summary{
			ByFile: make(map[string]int),
		},
		ByCategory: make(map[categorizer.Category]int),
		Escapes:    []categorizer.CategorizedEscape{},
	}

	t.Run("Text", func(t *testing.T) {
		var buf bytes.Buffer
		reporter := NewTextReporter(&buf, false)
		err := reporter.Report(results)
		if err != nil {
			t.Errorf("Text failed with empty results: %v", err)
		}
	})

	t.Run("JSON", func(t *testing.T) {
		var buf bytes.Buffer
		reporter := NewJSONReporter(&buf)
		err := reporter.Report(results)
		if err != nil {
			t.Errorf("JSON failed with empty results: %v", err)
		}
	})

	t.Run("HTML", func(t *testing.T) {
		var buf bytes.Buffer
		reporter := NewHTMLReporter(&buf)
		err := reporter.Report(results)
		if err != nil {
			t.Errorf("HTML failed with empty results: %v", err)
		}
	})

	t.Run("SARIF", func(t *testing.T) {
		var buf bytes.Buffer
		reporter := NewSARIFReporter(&buf)
		err := reporter.Report(results)
		if err != nil {
			t.Errorf("SARIF failed with empty results: %v", err)
		}
	})
}
