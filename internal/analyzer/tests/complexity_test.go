package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitruves/gop/internal/analyzer"
)

func TestComplexityAnalysis(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "complexity_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test C files with functions of varying complexity
	testFiles := map[string]string{
		"simple.c": `
#include <stdio.h>

// Simple function with low complexity
int add(int a, int b) {
    return a + b;
}

// Function with moderate complexity
int factorial(int n) {
    if (n <= 1) {
        return 1;
    }
    return n * factorial(n - 1);
}
`,
		"complex.c": `
#include <stdio.h>
#include <stdlib.h>

// Complex function with high cyclomatic complexity
int processData(int* data, int size, int threshold) {
    int result = 0;
    
    if (data == NULL) {
        return -1;
    }
    
    for (int i = 0; i < size; i++) {
        if (data[i] > threshold) {
            if (data[i] % 2 == 0) {
                result += data[i] * 2;
            } else {
                result += data[i];
            }
        } else if (data[i] < 0) {
            result -= data[i];
        } else {
            switch (data[i]) {
                case 0:
                    result += 10;
                    break;
                case 1:
                    result += 20;
                    break;
                default:
                    result += 5;
                    break;
            }
        }
    }
    
    return result;
}

// Function with nested loops (high cognitive complexity)
void processMatrix(int** matrix, int rows, int cols) {
    for (int i = 0; i < rows; i++) {
        for (int j = 0; j < cols; j++) {
            if (i == j) {
                matrix[i][j] *= 2;
            } else if (i > j) {
                matrix[i][j] += matrix[j][i];
            } else {
                for (int k = 0; k < j; k++) {
                    matrix[i][j] += k;
                }
            }
        }
    }
}
`,
	}

	// Write test files
	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		// Create parent directories if needed
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", filename, err)
		}
	}
	
	// Print the files for debugging
	t.Logf("Created test files in %s:", tempDir)
	files, _ := filepath.Glob(filepath.Join(tempDir, "*"))
	for _, f := range files {
		t.Logf("  %s", f)
	}

	// Create a temporary output file
	outputFile := filepath.Join(tempDir, "complexity_metrics.md")

	// Run complexity analysis
	options := analyzer.ComplexityOptions{
		Directory:  tempDir,
		Languages:  []string{"c", "h"},  // Include header files
		Depth:      1,                    // Only search in the temp directory
		OutputFile: outputFile,
		Cyclomatic: true,
		Cognitive:  true,
		Threshold:  5,
		Verbose:    true,
	}

	// Analyze complexity
	analyzer.AnalyzeComplexity(options)

	// Check if output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Output file was not created")
	}

	// Read output file content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	contentStr := string(content)

	// Check for expected function names
	expectedFunctions := []string{
		"add",
		"factorial",
		"processData",
		"processMatrix",
	}

	for _, funcName := range expectedFunctions {
		if !strings.Contains(contentStr, funcName) {
			t.Errorf("Expected function '%s' not found in output", funcName)
		}
	}

	// Check for complexity metrics
	if !strings.Contains(contentStr, "Cyclomatic") || !strings.Contains(contentStr, "Cognitive") {
		t.Errorf("Complexity metrics not found in output")
	}

	// Check that processData has higher complexity than add
	// This is a basic check - in a real test we might parse the output more carefully
	processDataPos := strings.Index(contentStr, "processData")
	addPos := strings.Index(contentStr, "add")
	
	if processDataPos < 0 || addPos < 0 {
		t.Errorf("Could not find positions of functions in output")
	} else {
		// Check that processData appears in the output before add (assuming sorted by complexity)
		// This is a simplistic check and might need adjustment based on actual output format
		if options.Threshold > 0 && processDataPos > addPos {
			t.Errorf("Expected processData (high complexity) to be listed before add (low complexity)")
		}
	}

	// Recommendations section has been removed
}
