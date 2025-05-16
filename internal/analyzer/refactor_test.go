package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRefactor(t *testing.T) {
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
		pattern        string
		replacement    string
		regexMode      bool
		wholeWord      bool
		caseSensitive  bool
		dryRun         bool
		expectedFiles  map[string]string
		expectedOutput []string
	}{
		{
			name:          "LiteralReplacement",
			pattern:       "oldFunction",
			replacement:   "newFunction",
			regexMode:     false,
			wholeWord:     true,
			caseSensitive: true,
			dryRun:        false,
			expectedFiles: map[string]string{
				"test1.c": "newFunction",
				"test2.cpp": "newFunction",
			},
			expectedOutput: []string{
				"# Refactoring Report",
				"| Pattern | `oldFunction` |",
				"| Replacement | `newFunction` |",
				"| Files with Matches | 2 |",
			},
		},
		{
			name:          "RegexReplacement",
			pattern:       "old([A-Z][a-z]+)",
			replacement:   "new$1",
			regexMode:     true,
			wholeWord:     false,
			caseSensitive: true,
			dryRun:        true, // Dry run to not modify files
			expectedFiles: map[string]string{}, // No files should be modified in dry run
			expectedOutput: []string{
				"# Refactoring Report",
				"| Pattern | `old([A-Z][a-z]+)` |",
				"| Replacement | `new$1` |",
				"| Mode | Regex",
				"**Note: This was a dry run. No files were modified.**",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run refactoring
			options := RefactorOptions{
				Directory:     tempDir,
				Depth:         1,
				OutputFile:    outputFile,
				Languages:     []string{"c", "cpp"},
				Pattern:       tc.pattern,
				Replacement:   tc.replacement,
				RegexMode:     tc.regexMode,
				WholeWord:     tc.wholeWord,
				CaseSensitive: tc.caseSensitive,
				Jobs:          1,
				DryRun:        tc.dryRun,
				Backup:        false, // No backup for tests
				Verbose:       false,
			}

			RunRefactor(options)

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
			if !tc.dryRun {
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
