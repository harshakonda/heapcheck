package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// getHeapcheckBinary builds and returns path to heapcheck binary
func getHeapcheckBinary(t *testing.T) string {
	t.Helper()

	// Build heapcheck
	tmpDir := t.TempDir()
	binary := filepath.Join(tmpDir, "heapcheck")

	cmd := exec.Command("go", "build", "-o", binary, "../../cmd/heapcheck")
	cmd.Dir = filepath.Join(getProjectRoot(t), "tests", "integration")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build heapcheck: %v\n%s", err, output)
	}

	return binary
}

func getProjectRoot(t *testing.T) string {
	t.Helper()

	// Get current directory
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Walk up to find go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("Could not find project root (go.mod)")
		}
		dir = parent
	}
}

func TestHeapcheckOnExamples(t *testing.T) {
	binary := getHeapcheckBinary(t)
	projectRoot := getProjectRoot(t)

	examples := []struct {
		name string
		path string
	}{
		{"basic-patterns", "examples/basic-patterns/..."},
		{"testdata", "testdata/..."},
	}

	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			cmd := exec.Command(binary, ex.path)
			cmd.Dir = projectRoot

			output, err := cmd.CombinedOutput()
			// heapcheck should succeed even if there are escapes
			if err != nil {
				t.Logf("Output: %s", output)
				// Don't fail - heapcheck exits 0 even with escapes
			}

			outputStr := string(output)

			// Verify output structure
			checks := []string{
				"heapcheck",
				"Summary",
				"Total variables analyzed",
			}

			for _, check := range checks {
				if !strings.Contains(outputStr, check) {
					t.Errorf("Output missing expected content: %s", check)
				}
			}
		})
	}
}

func TestHeapcheckFormats(t *testing.T) {
	binary := getHeapcheckBinary(t)
	projectRoot := getProjectRoot(t)

	formats := []struct {
		name     string
		flag     string
		contains []string
	}{
		{
			name: "text",
			flag: "text",
			contains: []string{
				"heapcheck",
				"Summary",
			},
		},
		{
			name: "json",
			flag: "json",
			contains: []string{
				`"summary"`,
				`"byCategory"`,
				`"escapes"`,
			},
		},
		{
			name: "html",
			flag: "html",
			contains: []string{
				"<!DOCTYPE html>",
				"heapcheck Report",
			},
		},
		{
			name: "sarif",
			flag: "sarif",
			contains: []string{
				`"$schema"`,
				`"version": "2.1.0"`,
				`"runs"`,
			},
		},
	}

	for _, f := range formats {
		t.Run(f.name, func(t *testing.T) {
			cmd := exec.Command(binary, "--format="+f.flag, "./testdata/...")
			cmd.Dir = projectRoot

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("Warning: %v", err)
			}

			outputStr := string(output)

			for _, check := range f.contains {
				if !strings.Contains(outputStr, check) {
					t.Errorf("Format %s missing: %s\nOutput: %s", f.name, check, outputStr[:min(500, len(outputStr))])
				}
			}
		})
	}
}

func TestHeapcheckVerbose(t *testing.T) {
	binary := getHeapcheckBinary(t)
	projectRoot := getProjectRoot(t)

	// Run with verbose on examples which have more escapes
	cmdVerbose := exec.Command(binary, "-v", "./examples/basic-patterns/...")
	cmdVerbose.Dir = projectRoot
	verboseOutput, _ := cmdVerbose.CombinedOutput()

	// Verbose should contain detailed info
	outputStr := string(verboseOutput)
	if !strings.Contains(outputStr, "heapcheck") {
		t.Error("Verbose output should contain heapcheck header")
	}
}

func TestHeapcheckEscapesOnly(t *testing.T) {
	binary := getHeapcheckBinary(t)
	projectRoot := getProjectRoot(t)

	cmd := exec.Command(binary, "--escapes-only", "--format=json", "./testdata/...")
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Warning: %v", err)
	}

	// With escapes-only, "does not escape" entries should be filtered
	if strings.Contains(string(output), "does not escape") {
		t.Error("--escapes-only should filter out 'does not escape' entries")
	}
}

func TestHeapcheckVersion(t *testing.T) {
	binary := getHeapcheckBinary(t)

	cmd := exec.Command(binary, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--version failed: %v", err)
	}

	if !strings.Contains(string(output), "heapcheck version") {
		t.Errorf("Version output unexpected: %s", output)
	}
}

func TestHeapcheckHelp(t *testing.T) {
	binary := getHeapcheckBinary(t)

	cmd := exec.Command(binary, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v", err)
	}

	outputStr := string(output)

	checks := []string{
		"heapcheck",
		"Usage",
		"--format",
		"--escapes-only",
	}

	for _, check := range checks {
		if !strings.Contains(outputStr, check) {
			t.Errorf("Help missing: %s", check)
		}
	}
}

func TestHeapcheckOnItself(t *testing.T) {
	binary := getHeapcheckBinary(t)
	projectRoot := getProjectRoot(t)

	// heapcheck analyzing itself!
	cmd := exec.Command(binary, "./...")
	cmd.Dir = projectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Logf("Stderr: %s", stderr.String())
	}

	output := stdout.String()

	// Should produce valid output
	if !strings.Contains(output, "heapcheck") {
		t.Error("Self-analysis should produce valid output")
	}

	if !strings.Contains(output, "Summary") {
		t.Error("Self-analysis should include Summary section")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
