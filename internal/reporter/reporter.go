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

	// Calculate percentages for charts
	stackPct := float64(0)
	heapPct := float64(0)
	if results.Summary.TotalVariables > 0 {
		stackPct = float64(results.Summary.StackAllocated) / float64(results.Summary.TotalVariables) * 100
		heapPct = float64(results.Summary.HeapAllocated) / float64(results.Summary.TotalVariables) * 100
	}

	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>heapcheck Report</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        * { box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0; padding: 20px; background: #f5f5f5;
        }
        .container { max-width: 1400px; margin: 0 auto; }
        h1 { color: #333; margin-bottom: 30px; }
        h2 { color: #444; margin-top: 0; margin-bottom: 20px; border-bottom: 2px solid #e5e7eb; padding-bottom: 10px; }
        .card { 
            background: white; border-radius: 12px; padding: 24px; 
            margin-bottom: 24px; box-shadow: 0 4px 6px rgba(0,0,0,0.07);
        }
        .grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; }
        .grid-3 { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; }
        @media (max-width: 768px) { .grid-2 { grid-template-columns: 1fr; } }
        
        .stat-card {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            border-radius: 12px; padding: 24px; color: white; text-align: center;
        }
        .stat-card.success { background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%); }
        .stat-card.danger { background: linear-gradient(135deg, #eb3349 0%, #f45c43 100%); }
        .stat-card.info { background: linear-gradient(135deg, #2196F3 0%, #21CBF3 100%); }
        .stat-value { font-size: 3em; font-weight: bold; margin-bottom: 5px; }
        .stat-label { font-size: 1em; opacity: 0.9; }
        .stat-pct { font-size: 0.9em; opacity: 0.8; margin-top: 5px; }
        
        .chart-container { position: relative; height: 300px; }
        .chart-container-sm { position: relative; height: 250px; }
        
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 12px 16px; text-align: left; border-bottom: 1px solid #e5e7eb; }
        th { background: #f9fafb; font-weight: 600; color: #374151; }
        tr:hover { background: #f9fafb; }
        
        .category-badge {
            display: inline-block; padding: 4px 12px; border-radius: 20px;
            font-size: 0.85em; font-weight: 500;
        }
        .badge-red { background: #fee2e2; color: #dc2626; }
        .badge-orange { background: #ffedd5; color: #ea580c; }
        .badge-yellow { background: #fef3c7; color: #ca8a04; }
        .badge-green { background: #dcfce7; color: #16a34a; }
        .badge-blue { background: #dbeafe; color: #2563eb; }
        .badge-purple { background: #f3e8ff; color: #9333ea; }
        .badge-gray { background: #f3f4f6; color: #6b7280; }
        
        .suggestion { color: #059669; font-style: italic; font-size: 0.9em; }
        .file-link { color: #2563eb; text-decoration: none; font-family: monospace; }
        .file-link:hover { text-decoration: underline; }
        .var-name { font-family: monospace; background: #f3f4f6; padding: 2px 6px; border-radius: 4px; }
        
        .hotspot-bar {
            background: #e5e7eb; border-radius: 4px; height: 24px; position: relative; overflow: hidden;
        }
        .hotspot-fill {
            background: linear-gradient(90deg, #ef4444 0%, #f97316 100%);
            height: 100%; border-radius: 4px; transition: width 0.3s;
        }
        .hotspot-label {
            position: absolute; right: 8px; top: 50%; transform: translateY(-50%);
            font-size: 0.8em; font-weight: 600; color: #374151;
        }
        
        .legend-item { display: flex; align-items: center; margin-bottom: 8px; }
        .legend-color { width: 16px; height: 16px; border-radius: 4px; margin-right: 10px; }
        .legend-text { font-size: 0.9em; color: #4b5563; }
        
        .no-escapes {
            text-align: center; padding: 60px 20px; color: #059669;
        }
        .no-escapes-icon { font-size: 4em; margin-bottom: 20px; }
        .no-escapes-text { font-size: 1.5em; font-weight: 600; }
        
        .footer { text-align: center; color: #9ca3af; font-size: 0.85em; margin-top: 40px; padding: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>📊 heapcheck Report</h1>
`)

	// Summary cards
	sb.WriteString(`<div class="grid-3" style="margin-bottom: 24px;">`)
	sb.WriteString(fmt.Sprintf(`<div class="stat-card info"><div class="stat-value">%d</div><div class="stat-label">Total Variables</div></div>`, results.Summary.TotalVariables))
	sb.WriteString(fmt.Sprintf(`<div class="stat-card success"><div class="stat-value">%d</div><div class="stat-label">Stack Allocated</div><div class="stat-pct">%.1f%% ✓</div></div>`, results.Summary.StackAllocated, stackPct))
	sb.WriteString(fmt.Sprintf(`<div class="stat-card danger"><div class="stat-value">%d</div><div class="stat-label">Heap Allocated</div><div class="stat-pct">%.1f%% ⚠</div></div>`, results.Summary.HeapAllocated, heapPct))
	sb.WriteString(`</div>`)

	// Check if there are any escapes
	if results.Summary.HeapAllocated == 0 {
		sb.WriteString(`<div class="card no-escapes">
			<div class="no-escapes-icon">🎉</div>
			<div class="no-escapes-text">No heap escapes found!</div>
			<p style="color: #6b7280; margin-top: 10px;">Your code is well-optimized for stack allocation.</p>
		</div>`)
	} else {
		// Charts row
		sb.WriteString(`<div class="grid-2">`)

		// Allocation pie chart
		sb.WriteString(`<div class="card">
			<h2>Allocation Distribution</h2>
			<div class="chart-container">
				<canvas id="allocationChart"></canvas>
			</div>
		</div>`)

		// Categories bar chart
		sb.WriteString(`<div class="card">
			<h2>Escape Categories</h2>
			<div class="chart-container">
				<canvas id="categoriesChart"></canvas>
			</div>
		</div>`)

		sb.WriteString(`</div>`) // end grid-2

		// Hotspots card
		if len(results.Summary.ByFile) > 0 {
			sb.WriteString(`<div class="card"><h2>🔥 Hotspots</h2>`)
			
			// Find max for scaling
			maxEscapes := 0
			for _, count := range results.Summary.ByFile {
				if count > maxEscapes {
					maxEscapes = count
				}
			}
			
			// Sort files by escape count
			type fileCount struct {
				file  string
				count int
			}
			var files []fileCount
			for f, c := range results.Summary.ByFile {
				files = append(files, fileCount{f, c})
			}
			sort.Slice(files, func(i, j int) bool {
				return files[i].count > files[j].count
			})
			
			sb.WriteString(`<table><tr><th>File</th><th style="width: 50%;">Escapes</th><th style="width: 80px;">Count</th></tr>`)
			for i, fc := range files {
				if i >= 10 { // Show top 10 only
					break
				}
				pct := float64(fc.count) / float64(maxEscapes) * 100
				sb.WriteString(fmt.Sprintf(`<tr>
					<td><span class="file-link">%s</span></td>
					<td><div class="hotspot-bar"><div class="hotspot-fill" style="width: %.1f%%;"></div></div></td>
					<td><strong>%d</strong></td>
				</tr>`, fc.file, pct, fc.count))
			}
			sb.WriteString(`</table></div>`)
		}

		// Detailed escapes table
		sb.WriteString(`<div class="card"><h2>📋 All Escapes</h2>`)
		sb.WriteString(`<table><tr><th>Location</th><th>Variable</th><th>Category</th><th>Suggestion</th></tr>`)
		for _, e := range results.Escapes {
			badgeClass := getCategoryBadgeClass(e.Category)
			sb.WriteString(fmt.Sprintf(`<tr>
				<td><span class="file-link">%s:%d</span></td>
				<td><span class="var-name">%s</span></td>
				<td><span class="category-badge %s">%s</span></td>
				<td class="suggestion">%s</td>
			</tr>`, e.Info.File, e.Info.Line, e.Info.Variable, badgeClass, e.Category, e.Suggestion.Short))
		}
		sb.WriteString(`</table></div>`)

		// Chart.js scripts
		sb.WriteString(`<script>
		// Allocation Pie Chart
		new Chart(document.getElementById('allocationChart'), {
			type: 'doughnut',
			data: {
				labels: ['Stack Allocated', 'Heap Allocated'],
				datasets: [{
					data: [`)
		sb.WriteString(fmt.Sprintf("%d, %d", results.Summary.StackAllocated, results.Summary.HeapAllocated))
		sb.WriteString(`],
					backgroundColor: ['#22c55e', '#ef4444'],
					borderWidth: 0,
					hoverOffset: 4
				}]
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				plugins: {
					legend: { position: 'bottom' },
					tooltip: {
						callbacks: {
							label: function(context) {
								let total = context.dataset.data.reduce((a, b) => a + b, 0);
								let pct = ((context.raw / total) * 100).toFixed(1);
								return context.label + ': ' + context.raw + ' (' + pct + '%)';
							}
						}
					}
				}
			}
		});

		// Categories Bar Chart
		new Chart(document.getElementById('categoriesChart'), {
			type: 'bar',
			data: {
				labels: [`)
		
		// Add category labels
		categories := sortCategories(results.ByCategory)
		for i, cat := range categories {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(fmt.Sprintf("'%s'", cat))
		}
		sb.WriteString(`],
				datasets: [{
					label: 'Count',
					data: [`)
		
		// Add category counts
		for i, cat := range categories {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(fmt.Sprintf("%d", results.ByCategory[cat]))
		}
		sb.WriteString(`],
					backgroundColor: [
						'#ef4444', '#f97316', '#f59e0b', '#eab308', '#84cc16',
						'#22c55e', '#14b8a6', '#06b6d4', '#0ea5e9', '#3b82f6',
						'#6366f1', '#8b5cf6', '#a855f7', '#d946ef', '#ec4899'
					],
					borderRadius: 6
				}]
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				indexAxis: 'y',
				plugins: {
					legend: { display: false }
				},
				scales: {
					x: { beginAtZero: true, grid: { display: false } },
					y: { grid: { display: false } }
				}
			}
		});
		</script>`)
	}

	sb.WriteString(`<div class="footer">Generated by <strong>heapcheck</strong> • <a href="https://github.com/harshakonda/heapcheck" style="color: #6b7280;">github.com/harshakonda/heapcheck</a></div>`)
	sb.WriteString(`</div></body></html>`)

	return sb.String()
}

// getCategoryBadgeClass returns the CSS class for a category badge
func getCategoryBadgeClass(cat categorizer.Category) string {
	switch cat {
	case categorizer.CategoryReturnPointer, categorizer.CategoryInterfaceBoxing:
		return "badge-red"
	case categorizer.CategoryClosureCapture, categorizer.CategoryGoroutineEscape:
		return "badge-orange"
	case categorizer.CategorySliceGrow, categorizer.CategoryChannelSend:
		return "badge-yellow"
	case categorizer.CategoryFmtCall, categorizer.CategoryReflection:
		return "badge-blue"
	case categorizer.CategoryUnknownSize, categorizer.CategoryTooLarge:
		return "badge-purple"
	default:
		return "badge-gray"
	}
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
	ID               string       `json:"id"`
	ShortDescription sarifMessage `json:"shortDescription"`
	Help             sarifMessage `json:"help"`
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
