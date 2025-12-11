// Package reporter provides various output formatters for escape analysis results.
package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/harshakonda/heapcheck/internal/categorizer"
)

// Reporter interface for different output formats
type Reporter interface {
	Report(results *categorizer.Results) error
}

// =============================================================================
// Text Reporter
// =============================================================================

// TextReporter outputs human-readable text
type TextReporter struct {
	w       io.Writer
	verbose bool
}

// NewTextReporter creates a new text reporter
func NewTextReporter(w io.Writer, verbose bool) *TextReporter {
	return &TextReporter{w: w, verbose: verbose}
}

// Report generates a human-readable report
func (r *TextReporter) Report(results *categorizer.Results) error {
	w := r.w

	// Header
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "📊 heapcheck - Escape Analysis Report")
	fmt.Fprintln(w, strings.Repeat("─", 50))
	fmt.Fprintln(w, "")

	// Summary
	fmt.Fprintln(w, "Summary:")
	total := results.Summary.TotalVariables
	stack := results.Summary.StackAllocated
	heap := results.Summary.HeapAllocated
	inlined := results.Summary.Inlined

	stackPct := float64(0)
	heapPct := float64(0)
	if total > 0 {
		stackPct = float64(stack) / float64(total) * 100
		heapPct = float64(heap) / float64(total) * 100
	}

	fmt.Fprintf(w, "  Total variables analyzed: %d\n", total)
	fmt.Fprintf(w, "  Stack allocated:          %d (%.1f%%)\n", stack, stackPct)
	fmt.Fprintf(w, "  Heap allocated:           %d (%.1f%%) ⚠️\n", heap, heapPct)
	if inlined > 0 {
		fmt.Fprintf(w, "  Inlined calls:            %d\n", inlined)
	}
	fmt.Fprintln(w, "")

	if heap == 0 {
		fmt.Fprintln(w, "✅ No heap escapes found! Your code is well-optimized.")
		return nil
	}

	// Escapes by category
	fmt.Fprintln(w, "Escape Causes:")
	categories := sortCategories(results.ByCategory)
	for i, cat := range categories {
		count := results.ByCategory[cat]
		pct := float64(count) / float64(heap) * 100
		fmt.Fprintf(w, "  %d. %-20s %3d (%5.1f%%)\n", i+1, cat, count, pct)
	}
	fmt.Fprintln(w, "")

	// Hotspots (files with most escapes)
	if len(results.Summary.ByFile) > 0 {
		fmt.Fprintln(w, "Hotspots (files with most escapes):")
		files := sortFilesByCount(results.Summary.ByFile)
		for i, f := range files {
			if i >= 5 {
				break
			}
			fmt.Fprintf(w, "  %-40s %3d escapes\n", truncatePath(f.name, 40), f.count)
		}
		fmt.Fprintln(w, "")
	}

	// Detailed escapes (if verbose or few escapes)
	if r.verbose || len(results.Escapes) <= 10 {
		fmt.Fprintln(w, "Details:")
		fmt.Fprintln(w, strings.Repeat("─", 50))

		for _, e := range results.Escapes {
			printEscapeDetail(w, e)
		}
	} else {
		fmt.Fprintf(w, "Run with -v for detailed breakdown of all %d escapes.\n", len(results.Escapes))
	}

	return nil
}

func printEscapeDetail(w io.Writer, e categorizer.CategorizedEscape) {
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "📍 %s:%d:%d\n", e.Info.File, e.Info.Line, e.Info.Column)
	fmt.Fprintf(w, "   Variable: %s\n", e.Info.Variable)
	fmt.Fprintf(w, "   Type:     %s\n", e.Info.EscapeType)
	fmt.Fprintf(w, "   Category: %s\n", e.Category)
	fmt.Fprintf(w, "   💡 %s\n", e.Suggestion.Short)

	if len(e.Info.FlowInfo) > 0 {
		fmt.Fprintln(w, "   Flow:")
		for _, flow := range e.Info.FlowInfo {
			fmt.Fprintf(w, "     %s\n", flow)
		}
	}
}

// =============================================================================
// JSON Reporter
// =============================================================================

// JSONReporter outputs JSON format
type JSONReporter struct {
	w io.Writer
}

// NewJSONReporter creates a new JSON reporter
func NewJSONReporter(w io.Writer) *JSONReporter {
	return &JSONReporter{w: w}
}

// Report generates JSON output
func (r *JSONReporter) Report(results *categorizer.Results) error {
	encoder := json.NewEncoder(r.w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// =============================================================================
// HTML Reporter
// =============================================================================

// HTMLReporter outputs an HTML report
type HTMLReporter struct {
	w io.Writer
}

// NewHTMLReporter creates a new HTML reporter
func NewHTMLReporter(w io.Writer) *HTMLReporter {
	return &HTMLReporter{w: w}
}

// Report generates an HTML report
func (r *HTMLReporter) Report(results *categorizer.Results) error {
	html := generateHTML(results)
	_, err := r.w.Write([]byte(html))
	return err
}

func generateHTML(results *categorizer.Results) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>heapcheck Report</title>
    <style>
        * { box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0; padding: 20px; background: #f5f5f5;
        }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: #333; }
        .card { 
            background: white; border-radius: 8px; padding: 20px; 
            margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 15px; }
        .stat { text-align: center; }
        .stat-value { font-size: 2em; font-weight: bold; color: #2563eb; }
        .stat-label { color: #666; font-size: 0.9em; }
        .heap { color: #dc2626 !important; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #eee; }
        th { background: #f9fafb; font-weight: 600; }
        .category-badge {
            display: inline-block; padding: 2px 8px; border-radius: 4px;
            font-size: 0.8em; background: #e5e7eb;
        }
        .suggestion { color: #059669; font-style: italic; }
        .file-link { color: #2563eb; text-decoration: none; }
        .file-link:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <h1>📊 heapcheck Report</h1>
`)

	// Summary card
	sb.WriteString(`<div class="card"><h2>Summary</h2><div class="summary">`)
	sb.WriteString(fmt.Sprintf(`<div class="stat"><div class="stat-value">%d</div><div class="stat-label">Total Variables</div></div>`, results.Summary.TotalVariables))
	sb.WriteString(fmt.Sprintf(`<div class="stat"><div class="stat-value">%d</div><div class="stat-label">Stack Allocated</div></div>`, results.Summary.StackAllocated))
	sb.WriteString(fmt.Sprintf(`<div class="stat"><div class="stat-value heap">%d</div><div class="stat-label">Heap Allocated</div></div>`, results.Summary.HeapAllocated))
	sb.WriteString(`</div></div>`)

	// Categories card
	sb.WriteString(`<div class="card"><h2>Escape Categories</h2><table><tr><th>Category</th><th>Count</th><th>%</th></tr>`)
	categories := sortCategories(results.ByCategory)
	for _, cat := range categories {
		count := results.ByCategory[cat]
		pct := float64(count) / float64(results.Summary.HeapAllocated) * 100
		sb.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%d</td><td>%.1f%%</td></tr>`, cat, count, pct))
	}
	sb.WriteString(`</table></div>`)

	// Escapes table
	sb.WriteString(`<div class="card"><h2>Escapes</h2><table><tr><th>Location</th><th>Variable</th><th>Category</th><th>Suggestion</th></tr>`)
	for _, e := range results.Escapes {
		sb.WriteString(fmt.Sprintf(`<tr>
			<td><span class="file-link">%s:%d</span></td>
			<td><code>%s</code></td>
			<td><span class="category-badge">%s</span></td>
			<td class="suggestion">%s</td>
		</tr>`, e.Info.File, e.Info.Line, e.Info.Variable, e.Category, e.Suggestion.Short))
	}
	sb.WriteString(`</table></div>`)

	sb.WriteString(`</div></body></html>`)

	return sb.String()
}

// =============================================================================
// SARIF Reporter (for GitHub Code Scanning)
// =============================================================================

// SARIFReporter outputs SARIF format for GitHub integration
type SARIFReporter struct {
	w io.Writer
}

// NewSARIFReporter creates a new SARIF reporter
func NewSARIFReporter(w io.Writer) *SARIFReporter {
	return &SARIFReporter{w: w}
}

// Report generates SARIF output
func (r *SARIFReporter) Report(results *categorizer.Results) error {
	sarif := generateSARIF(results)
	encoder := json.NewEncoder(r.w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(sarif)
}

type sarifReport struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name    string      `json:"name"`
	Version string      `json:"version"`
	Rules   []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string           `json:"id"`
	ShortDescription sarifMessage     `json:"shortDescription"`
	Help             sarifMessage     `json:"help"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           sarifRegion   `json:"region"`
}

type sarifArtifact struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
}

func generateSARIF(results *categorizer.Results) sarifReport {
	// Build rules from categories
	rules := make([]sarifRule, 0)
	ruleSet := make(map[categorizer.Category]bool)
	for _, e := range results.Escapes {
		if !ruleSet[e.Category] {
			ruleSet[e.Category] = true
			rules = append(rules, sarifRule{
				ID:               string(e.Category),
				ShortDescription: sarifMessage{Text: e.Suggestion.Short},
				Help:             sarifMessage{Text: e.Suggestion.Details},
			})
		}
	}

	// Build results
	sarifResults := make([]sarifResult, 0, len(results.Escapes))
	for _, e := range results.Escapes {
		sarifResults = append(sarifResults, sarifResult{
			RuleID:  string(e.Category),
			Level:   "warning",
			Message: sarifMessage{Text: fmt.Sprintf("%s escapes to heap: %s", e.Info.Variable, e.Suggestion.Short)},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifact{URI: e.Info.File},
					Region:           sarifRegion{StartLine: e.Info.Line, StartColumn: e.Info.Column},
				},
			}},
		})
	}

	return sarifReport{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []sarifRun{{
			Tool: sarifTool{
				Driver: sarifDriver{
					Name:    "heapcheck",
					Version: "1.0.0",
					Rules:   rules,
				},
			},
			Results: sarifResults,
		}},
	}
}

// =============================================================================
// Helpers
// =============================================================================

type fileCount struct {
	name  string
	count int
}

func sortFilesByCount(m map[string]int) []fileCount {
	result := make([]fileCount, 0, len(m))
	for name, count := range m {
		result = append(result, fileCount{name, count})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].count > result[j].count
	})
	return result
}

func sortCategories(m map[categorizer.Category]int) []categorizer.Category {
	result := make([]categorizer.Category, 0, len(m))
	for cat := range m {
		result = append(result, cat)
	}
	sort.Slice(result, func(i, j int) bool {
		return m[result[i]] > m[result[j]]
	})
	return result
}

func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}
