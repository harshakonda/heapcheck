// Package parser handles running the Go compiler with escape analysis flags
// and parsing the output into structured data.
package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// EscapeType represents the type of escape analysis result
type EscapeType int

const (
	Unknown EscapeType = iota
	MovedToHeap      // "moved to heap: x"
	EscapesToHeap    // "x escapes to heap"
	DoesNotEscape    // "x does not escape"
	LeakingParam     // "leaking param: x"
	CanInline        // "can inline foo"
	InliningCall     // "inlining call to foo"
)

func (e EscapeType) String() string {
	switch e {
	case MovedToHeap:
		return "moved-to-heap"
	case EscapesToHeap:
		return "escapes-to-heap"
	case DoesNotEscape:
		return "does-not-escape"
	case LeakingParam:
		return "leaking-param"
	case CanInline:
		return "can-inline"
	case InliningCall:
		return "inlining-call"
	default:
		return "unknown"
	}
}

// EscapeInfo represents a single escape analysis result
type EscapeInfo struct {
	File       string     `json:"file"`
	Line       int        `json:"line"`
	Column     int        `json:"column"`
	Variable   string     `json:"variable"`
	EscapeType EscapeType `json:"escapeType"`
	Reason     string     `json:"reason"`
	FlowInfo   []string   `json:"flowInfo,omitempty"` // Additional flow details from -m=2
}

// Patterns for matching escape analysis output
var (
	// ./file.go:10:2: moved to heap: x
	movedToHeapRe = regexp.MustCompile(`^(.+):(\d+):(\d+): moved to heap: (.+)$`)

	// ./file.go:10:2: x escapes to heap
	escapesToHeapRe = regexp.MustCompile(`^(.+):(\d+):(\d+): (.+) escapes to heap`)

	// ./file.go:10:2: x does not escape
	doesNotEscapeRe = regexp.MustCompile(`^(.+):(\d+):(\d+): (.+) does not escape$`)

	// ./file.go:10:2: leaking param: x
	leakingParamRe = regexp.MustCompile(`^(.+):(\d+):(\d+): leaking param: (.+)`)

	// ./file.go:10:2: can inline foo
	canInlineRe = regexp.MustCompile(`^(.+):(\d+):(\d+): can inline (.+)$`)

	// ./file.go:10:2: inlining call to foo
	inliningCallRe = regexp.MustCompile(`^(.+):(\d+):(\d+): inlining call to (.+)$`)

	// ./file.go:10:2:   flow: ~r0 = &x:
	flowRe = regexp.MustCompile(`^(.+):(\d+):(\d+):\s+flow: (.+)$`)

	// ./file.go:10:2:     from &x (address-of) at ./file.go:10:9
	fromRe = regexp.MustCompile(`^(.+):(\d+):(\d+):\s+from (.+)$`)
)

// RunCompiler executes `go build` with escape analysis flags and returns the output
func RunCompiler(patterns []string) (string, error) {
	// Build the command
	// -gcflags="-m=2" gives detailed escape analysis
	// -l disables inlining for clearer escape info (optional, we include both)
	args := []string{"build", "-gcflags=-m=2", "-o", "/dev/null"}
	args = append(args, patterns...)

	cmd := exec.Command("go", args...)

	// Escape analysis output goes to stderr
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// We don't care about stdout for this
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// Run the command - it may return non-zero if there are build errors
	err := cmd.Run()

	// If there's output in stderr, we got escape analysis data
	// Even if cmd failed (build errors), we might have partial data
	output := stderr.String()

	// If we have no output and an error, something went wrong
	if output == "" && err != nil {
		return "", fmt.Errorf("go build failed: %w", err)
	}

	return output, nil
}

// Parse parses the raw compiler output into structured EscapeInfo slice
func Parse(output string) ([]EscapeInfo, error) {
	var results []EscapeInfo

	scanner := bufio.NewScanner(strings.NewReader(output))
	var currentEscape *EscapeInfo

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Try to match each pattern
		if info := parseMovedToHeap(line); info != nil {
			if currentEscape != nil {
				results = append(results, *currentEscape)
			}
			currentEscape = info
			continue
		}

		if info := parseEscapesToHeap(line); info != nil {
			if currentEscape != nil {
				results = append(results, *currentEscape)
			}
			currentEscape = info
			continue
		}

		if info := parseDoesNotEscape(line); info != nil {
			if currentEscape != nil {
				results = append(results, *currentEscape)
			}
			currentEscape = info
			continue
		}

		if info := parseLeakingParam(line); info != nil {
			if currentEscape != nil {
				results = append(results, *currentEscape)
			}
			currentEscape = info
			continue
		}

		if info := parseCanInline(line); info != nil {
			if currentEscape != nil {
				results = append(results, *currentEscape)
			}
			currentEscape = info
			continue
		}

		if info := parseInliningCall(line); info != nil {
			if currentEscape != nil {
				results = append(results, *currentEscape)
			}
			currentEscape = info
			continue
		}

		// Check for flow/from lines (additional details for current escape)
		if currentEscape != nil {
			if flowRe.MatchString(line) || fromRe.MatchString(line) {
				currentEscape.FlowInfo = append(currentEscape.FlowInfo, strings.TrimSpace(line))
			}
		}
	}

	// Don't forget the last one
	if currentEscape != nil {
		results = append(results, *currentEscape)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning output: %w", err)
	}

	return results, nil
}

func parseMovedToHeap(line string) *EscapeInfo {
	matches := movedToHeapRe.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}
	lineNum, _ := strconv.Atoi(matches[2])
	colNum, _ := strconv.Atoi(matches[3])
	return &EscapeInfo{
		File:       matches[1],
		Line:       lineNum,
		Column:     colNum,
		Variable:   matches[4],
		EscapeType: MovedToHeap,
		Reason:     line,
	}
}

func parseEscapesToHeap(line string) *EscapeInfo {
	matches := escapesToHeapRe.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}
	lineNum, _ := strconv.Atoi(matches[2])
	colNum, _ := strconv.Atoi(matches[3])
	return &EscapeInfo{
		File:       matches[1],
		Line:       lineNum,
		Column:     colNum,
		Variable:   matches[4],
		EscapeType: EscapesToHeap,
		Reason:     line,
	}
}

func parseDoesNotEscape(line string) *EscapeInfo {
	matches := doesNotEscapeRe.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}
	lineNum, _ := strconv.Atoi(matches[2])
	colNum, _ := strconv.Atoi(matches[3])
	return &EscapeInfo{
		File:       matches[1],
		Line:       lineNum,
		Column:     colNum,
		Variable:   matches[4],
		EscapeType: DoesNotEscape,
		Reason:     line,
	}
}

func parseLeakingParam(line string) *EscapeInfo {
	matches := leakingParamRe.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}
	lineNum, _ := strconv.Atoi(matches[2])
	colNum, _ := strconv.Atoi(matches[3])
	return &EscapeInfo{
		File:       matches[1],
		Line:       lineNum,
		Column:     colNum,
		Variable:   matches[4],
		EscapeType: LeakingParam,
		Reason:     line,
	}
}

func parseCanInline(line string) *EscapeInfo {
	matches := canInlineRe.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}
	lineNum, _ := strconv.Atoi(matches[2])
	colNum, _ := strconv.Atoi(matches[3])
	return &EscapeInfo{
		File:       matches[1],
		Line:       lineNum,
		Column:     colNum,
		Variable:   matches[4],
		EscapeType: CanInline,
		Reason:     line,
	}
}

func parseInliningCall(line string) *EscapeInfo {
	matches := inliningCallRe.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}
	lineNum, _ := strconv.Atoi(matches[2])
	colNum, _ := strconv.Atoi(matches[3])
	return &EscapeInfo{
		File:       matches[1],
		Line:       lineNum,
		Column:     colNum,
		Variable:   matches[4],
		EscapeType: InliningCall,
		Reason:     line,
	}
}
