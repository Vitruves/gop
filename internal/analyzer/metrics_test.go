package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCalculateMetrics(t *testing.T) {
	// Create a temporary directory for test files
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

	// Run metrics analysis
	options := MetricsOptions{
		Directory:  tempDir,
		Depth:      1,
		OutputFile: outputFile,
		Languages:  []string{"c", "cpp"},
		Jobs:       1,
		Verbose:    false,
	}

	// Run the metrics calculation
	CalculateMetrics(options)

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
	expectedSections := []string{
		"# Code Metrics Analysis",
		"## Summary",
		"| Files Analyzed | 2 |",
		"## File Metrics",
		"## Processing Information",
	}

	contentStr := string(content)
	for _, section := range expectedSections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("Output does not contain expected section: %s", section)
		}
	}
}
