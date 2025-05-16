package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitruves/gop/internal/analyzer"
)

func TestRefactorCommand(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "refactor_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files with content to refactor
	testFiles := map[string]string{
		"test1.c": `
// This is a test function
int oldFunction(int a, int b) {
    // Call oldFunction recursively
    if (a > 1) {
        return oldFunction(a-1, b);
    }
    return a + b;
}

// Another function that calls oldFunction
int caller() {
    return oldFunction(5, 3);
}
`,
		"test2.cpp": `
#include <iostream>

// Class that uses oldFunction
class Calculator {
private:
    int value;

public:
    Calculator(int v) : value(v) {}

    int calculate(int x) {
        return oldFunction(value, x);
    }

    // Print using oldFunction
    void print() {
        std::cout << "Result: " << oldFunction(value, 10) << std::endl;
    }
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
	outputFile := filepath.Join(tempDir, "refactor_output.md")

	// Test cases for different refactoring options
	testCases := []struct {
		name           string
		options        analyzer.RefactorOptions
		expectedFiles  map[string]string
		expectedOutput []string
	}{
		{
			name: "LiteralReplacement",
			options: analyzer.RefactorOptions{
				Directory:     tempDir,
				Depth:         1,
				OutputFile:    outputFile,
				Languages:     []string{"c", "cpp"},
				Pattern:       "oldFunction",
				Replacement:   "newFunction",
				RegexMode:     false,
				WholeWord:     true,
				CaseSensitive: true,
				Jobs:          1,
				DryRun:        false,
				Backup:        false,
				Verbose:       false,
			},
			expectedFiles: map[string]string{
				"test1.c":   "newFunction",
				"test2.cpp": "newFunction",
			},
			expectedOutput: []string{
				"Refactoring Report",
				"Pattern",
				"oldFunction",
				"Replacement",
				"newFunction",
				"Files with Matches",
			},
		},
		{
			name: "RegexReplacement",
			options: analyzer.RefactorOptions{
				Directory:     tempDir,
				Depth:         1,
				OutputFile:    outputFile,
				Languages:     []string{"c", "cpp"},
				Pattern:       "old([A-Z][a-z]+)",
				Replacement:   "new$1",
				RegexMode:     true,
				WholeWord:     false,
				CaseSensitive: true,
				Jobs:          1,
				DryRun:        true,
				Backup:        false,
				Verbose:       false,
			},
			expectedFiles: map[string]string{}, // No files should be modified in dry run
			expectedOutput: []string{
				"Refactoring Report",
				"Pattern",
				"old([A-Z][a-z]+)",
				"Replacement",
				"new$1",
				"Mode",
				"Regex",
				"dry run",
				"No files were modified",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset test files for each test case
			for filename, content := range testFiles {
				filePath := filepath.Join(tempDir, filename)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write test file %s: %v", filename, err)
				}
			}

			// Run refactoring
			analyzer.RunRefactor(tc.options)

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
			for _, expected := range tc.expectedOutput {
				if !strings.Contains(contentStr, expected) {
					t.Errorf("Output does not contain expected content: %s", expected)
				}
			}

			// Verify file modifications if not in dry run mode
			if !tc.options.DryRun {
				for filename, expectedContent := range tc.expectedFiles {
					filePath := filepath.Join(tempDir, filename)
					fileContent, err := os.ReadFile(filePath)
					if err != nil {
						t.Fatalf("Failed to read modified file %s: %v", filename, err)
					}

					if !strings.Contains(string(fileContent), expectedContent) {
						t.Errorf("File %s does not contain expected content: %s", filename, expectedContent)
					}
				}
			}
		})
	}
}
