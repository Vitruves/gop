package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitruves/gop/internal/analyzer"
)

func TestMetricsCommand(t *testing.T) {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	
	// Setup test directory with existing test files
	testDir := filepath.Join(cwd, "test_files")
	
	// Create a temporary directory for additional test files
	tempDir, err := os.MkdirTemp("", "metrics_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := map[string]string{
		"test1.c": `
// This is a comment
int add(int a, int b) {
    return a + b; // Another comment
}

/*
 * Multi-line comment
 */
struct Point {
    int x;
    int y;
};

int main() {
    int result = add(5, 3);
    return 0;
}
`,
		"test2.cpp": `
#include <iostream>

// Class definition
class Rectangle {
private:
    int width;
    int height;

public:
    Rectangle(int w, int h) : width(w), height(h) {}

    // Calculate area
    int area() {
        return width * height;
    }
};

int main() {
    Rectangle rect(5, 3);
    std::cout << "Area: " << rect.area() << std::endl;
    return 0;
}
`,
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", filename, err)
		}
	}

	// Create a temporary output file
	outputFile := filepath.Join(tempDir, "metrics_output.md")

	// Test cases for different metrics options
	testCases := []struct {
		name     string
		options  analyzer.MetricsOptions
		expected []string
	}{
		{
			name: "BasicMetrics",
			options: analyzer.MetricsOptions{
				Directory:  tempDir,
				Depth:      1,
				OutputFile: outputFile,
				Languages:  []string{"c", "cpp"},
				Jobs:       1,
				Short:      false,
				Verbose:    false,
			},
			expected: []string{
				"Code Metrics Analysis",
				"Summary",
				"Files Analyzed",
				"File Metrics",
			},
		},
		{
			name: "ShortMetrics",
			options: analyzer.MetricsOptions{
				Directory:  tempDir,
				Depth:      1,
				OutputFile: outputFile,
				Languages:  []string{"c", "cpp"},
				Jobs:       1,
				Short:      true,
				Verbose:    false,
			},
			expected: []string{
				"Code Metrics Analysis",
				"Summary",
				"Files Analyzed",
				"Processing Information",
			},
		},
		{
			name: "ComplexMetrics",
			options: analyzer.MetricsOptions{
				Directory:  testDir,
				Depth:      1,
				OutputFile: outputFile,
				Languages:  []string{"c", "cpp"},
				Jobs:       2,
				Short:      false,
				Verbose:    true,
			},
			expected: []string{
				"Code Metrics Analysis",
				"Summary",
				"Files Analyzed",
				"metrics_complex.cpp",
				"Complexity",
				"Comment Ratio",
				"Processing Information",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run metrics calculation
			analyzer.CalculateMetrics(tc.options)

			// Verify output file exists
			if _, err := os.Stat(outputFile); os.IsNotExist(err) {
				t.Errorf("Output file was not created")
			}

			// Read output file content
			content, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			// Verify content contains expected sections
			contentStr := string(content)
			for _, expected := range tc.expected {
				if !strings.Contains(contentStr, expected) {
					t.Errorf("Output does not contain expected content: %s", expected)
				}
			}
		})
	}
}
