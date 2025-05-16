package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitruves/gop/internal/analyzer"
)

func TestDocsCommand(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "docs_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files with documentation comments
	testFiles := map[string]string{
		"functions.c": `
/**
 * Adds two integers and returns the result.
 * @param a First integer to add
 * @param b Second integer to add
 * @return The sum of a and b
 * @example add(5, 3) returns 8
 */
int add(int a, int b) {
    return a + b;
}

/// Multiplies two integers and returns the result.
/// @param a First integer to multiply
/// @param b Second integer to multiply
/// @return The product of a and b
int multiply(int a, int b) {
    return a * b;
}
`,
		"classes.cpp": `
/**
 * A simple rectangle class for geometric calculations.
 */
class Rectangle {
private:
    int width;
    int height;

public:
    /**
     * Constructor for Rectangle.
     * @param w Width of the rectangle
     * @param h Height of the rectangle
     */
    Rectangle(int w, int h) : width(w), height(h) {}

    /**
     * Calculates the area of the rectangle.
     * @return The area (width * height)
     */
    int area() {
        return width * height;
    }

    /**
     * Calculates the perimeter of the rectangle.
     * @return The perimeter (2 * width + 2 * height)
     */
    int perimeter() {
        return 2 * width + 2 * height;
    }
};

/// An enumeration of shapes.
enum Shape {
    CIRCLE,
    RECTANGLE,
    TRIANGLE
};
`,
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", filename, err)
		}
	}

	// Create a temporary output file
	outputFile := filepath.Join(tempDir, "docs_output.md")

	// Test cases for different docs options
	testCases := []struct {
		name     string
		options  analyzer.DocsOptions
		expected []string
	}{
		{
			name: "BasicDocs",
			options: analyzer.DocsOptions{
				Directory:   tempDir,
				Depth:       1,
				OutputFile:  outputFile,
				Languages:   []string{"c", "cpp"},
				Jobs:        1,
				IncludeCode: false,
				Short:       false,
				Verbose:     false,
			},
			expected: []string{
				"API Documentation",
				"Table of Contents",
				"Processing Information",
				"Files processed",
				"Documentation items extracted",
			},
		},
		{
			name: "DocsWithCode",
			options: analyzer.DocsOptions{
				Directory:   tempDir,
				Depth:       1,
				OutputFile:  outputFile,
				Languages:   []string{"c", "cpp"},
				Jobs:        1,
				IncludeCode: true,
				Short:       false,
				Verbose:     false,
			},
			expected: []string{
				"API Documentation",
				"Code:",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run docs generation
			analyzer.GenerateDocs(tc.options)

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
