// heapcheck - A developer-friendly Go escape analysis reporter
//
// heapcheck parses Go compiler escape analysis output and transforms it
// into actionable, human-readable reports with optimization suggestions.
//
// Usage:
//
//	heapcheck ./...                    # Analyze current module
//	heapcheck --format=json ./...      # Output as JSON
//	heapcheck --escapes-only ./...     # Show only heap escapes
//	heapcheck --filter=pkg/server ./...# Filter by package path
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/harshakonda/heapcheck/internal/categorizer"
	"github.com/harshakonda/heapcheck/internal/parser"
	"github.com/harshakonda/heapcheck/internal/reporter"
)

// Version information - set at build time via ldflags
var (
	Version = "0.1.4"
	Commit  = "unknown"
	Date    = "unknown"
)

func main() {
	// Define flags
	formatFlag := flag.String("format", "text", "Output format: text, json, html, sarif")
	escapesOnly := flag.Bool("escapes-only", false, "Show only variables that escape to heap")
	filterPkg := flag.String("filter", "", "Filter results by package path prefix")
	verbose := flag.Bool("v", false, "Verbose output (show all compiler messages)")
	version := flag.Bool("version", false, "Print version and exit")
	help := flag.Bool("help", false, "Show help")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `heapcheck - Go escape analysis made human-readable

Usage:
  heapcheck [flags] [packages]

Examples:
  heapcheck ./...                     Analyze all packages
  heapcheck ./pkg/server              Analyze specific package
  heapcheck --format=json ./...       Output as JSON
  heapcheck --escapes-only ./...      Show only heap allocations
  heapcheck --filter=internal ./...   Filter by path

Flags:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Output Formats:
  text   Human-readable summary (default)
  json   Machine-readable JSON
  html   Visual HTML report
  sarif  GitHub Code Scanning compatible

For more information: https://github.com/harshakonda/heapcheck
`)
	}

	flag.Parse()

	if *version {
		fmt.Printf("heapcheck version %s\n", Version)
		if Commit != "unknown" {
			fmt.Printf("  commit: %s\n", Commit)
		}
		if Date != "unknown" {
			fmt.Printf("  built:  %s\n", Date)
		}
		os.Exit(0)
	}

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	// Get package patterns from remaining args
	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	// Run analysis
	config := &Config{
		Format:      *formatFlag,
		EscapesOnly: *escapesOnly,
		FilterPkg:   *filterPkg,
		Verbose:     *verbose,
		Patterns:    patterns,
	}

	if err := run(config); err != nil {
		fmt.Fprintf(os.Stderr, "heapcheck: %v\n", err)
		os.Exit(1)
	}
}

// Config holds the CLI configuration
type Config struct {
	Format      string
	EscapesOnly bool
	FilterPkg   string
	Verbose     bool
	Patterns    []string
}

func run(cfg *Config) error {
	// Step 1: Run compiler and capture escape analysis output
	rawOutput, err := parser.RunCompiler(cfg.Patterns)
	if err != nil {
		return fmt.Errorf("running compiler: %w", err)
	}

	// Step 2: Parse the raw output into structured data
	escapes, err := parser.Parse(rawOutput)
	if err != nil {
		return fmt.Errorf("parsing output: %w", err)
	}

	// Step 3: Categorize and add suggestions
	results := categorizer.Categorize(escapes)

	// Step 4: Apply filters
	if cfg.EscapesOnly {
		results = filterEscapesOnly(results)
	}
	if cfg.FilterPkg != "" {
		results = filterByPackage(results, cfg.FilterPkg)
	}

	// Step 5: Generate report
	var rep reporter.Reporter
	switch cfg.Format {
	case "json":
		rep = reporter.NewJSONReporter(os.Stdout)
	case "html":
		rep = reporter.NewHTMLReporter(os.Stdout)
	case "sarif":
		rep = reporter.NewSARIFReporter(os.Stdout)
	default:
		rep = reporter.NewTextReporter(os.Stdout, cfg.Verbose)
	}

	return rep.Report(results)
}

func filterEscapesOnly(results *categorizer.Results) *categorizer.Results {
	filtered := &categorizer.Results{
		Summary:    results.Summary,
		ByCategory: results.ByCategory,
		Escapes:    make([]categorizer.CategorizedEscape, 0),
	}
	for _, e := range results.Escapes {
		if e.Info.EscapeType == parser.MovedToHeap || e.Info.EscapeType == parser.EscapesToHeap {
			filtered.Escapes = append(filtered.Escapes, e)
		}
	}
	return filtered
}

func filterByPackage(results *categorizer.Results, prefix string) *categorizer.Results {
	filtered := &categorizer.Results{
		Summary:    results.Summary,
		ByCategory: results.ByCategory,
		Escapes:    make([]categorizer.CategorizedEscape, 0),
	}
	for _, e := range results.Escapes {
		if containsPrefix(e.Info.File, prefix) {
			filtered.Escapes = append(filtered.Escapes, e)
		}
	}
	return filtered
}

func containsPrefix(path, prefix string) bool {
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}
