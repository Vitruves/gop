package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitruves/gop/internal/analyzer"
)

func TestUndefinedBehaviorAnalysis(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "undefined_behavior_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test C files with various undefined behavior issues
	testFiles := map[string]string{
		"signed_overflow.c": `
#include <limits.h>
#include <stdio.h>

void signedOverflow() {
    int max = INT_MAX;
    int overflow = max + 1;  // Signed overflow
    printf("Overflow: %d\n", overflow);
    
    int min = INT_MIN;
    int underflow = min - 1;  // Signed underflow
    printf("Underflow: %d\n", underflow);
}

int main() {
    signedOverflow();
    return 0;
}
`,
		"null_deref.c": `
#include <stdlib.h>
#include <stdio.h>

void nullDereference() {
    int* ptr = NULL;
    *ptr = 10;  // Null pointer dereference
    printf("Value: %d\n", *ptr);
}

int main() {
    // Uncommenting this would crash the program
    // nullDereference();
    return 0;
}
`,
		"div_zero.c": `
#include <stdio.h>

void divisionByZero() {
    int a = 10;
    int b = 0;
    int result = a / b;  // Division by zero
    printf("Result: %d\n", result);
}

int main() {
    // Uncommenting this would crash the program
    // divisionByZero();
    return 0;
}
`,
		"uninit_var.c": `
#include <stdio.h>

void uninitializedVariable() {
    int x;  // Uninitialized variable
    printf("Value of x: %d\n", x);
    
    int a, b, c;  // Multiple uninitialized variables
    int sum = a + b + c;
    printf("Sum: %d\n", sum);
}

int main() {
    uninitializedVariable();
    return 0;
}
`,
		"out_of_bounds.c": `
#include <stdio.h>

void arrayOutOfBounds() {
    int arr[5];
    arr[10] = 5;  // Array index out of bounds
    printf("Value: %d\n", arr[10]);
    
    arr[-1] = 10;  // Negative index
    printf("Value: %d\n", arr[-1]);
}

int main() {
    arrayOutOfBounds();
    return 0;
}
`,
		"invalid_shift.c": `
#include <stdio.h>

void invalidShift() {
    int x = 1;
    int y = x << 32;  // Shift amount too large
    printf("Value: %d\n", y);
    
    int z = x << -1;  // Negative shift amount
    printf("Value: %d\n", z);
}

int main() {
    invalidShift();
    return 0;
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
	outputFile := filepath.Join(tempDir, "undefined_behavior.md")

	// Run undefined behavior analysis
	options := analyzer.UndefinedBehaviorOptions{
		Directory:          tempDir,
		Languages:          []string{"c", "h"},
		Depth:              1,
		OutputFile:         outputFile,
		CheckSignedOverflow: true,
		CheckNullDereference: true,
		CheckDivByZero:      true,
		CheckUninitVar:      true,
		CheckOutOfBounds:    true,
		CheckShiftOperations: true,
		Verbose:            true,
	}

	// Analyze undefined behavior
	analyzer.AnalyzeUndefinedBehavior(options)

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

	// Check for expected issue types
	expectedIssues := []string{
		"Signed Integer Overflow",
		"Null Pointer Dereference",
		"Division by Zero",
		"Uninitialized Variable",
		"Array Out of Bounds",
		"Invalid Shift Operation",
	}

	for _, issue := range expectedIssues {
		if !strings.Contains(contentStr, issue) {
			t.Errorf("Expected issue type '%s' not found in output", issue)
		}
	}

	// Check for specific file names
	expectedNames := []string{
		"signed_overflow.c",
		"null_deref.c",
		"div_zero.c",
		"uninit_var.c",
		"out_of_bounds.c",
		"invalid_shift.c",
	}

	for _, name := range expectedNames {
		if !strings.Contains(contentStr, name) {
			t.Errorf("Expected file '%s' not found in output", name)
		}
	}

	// Check for specific patterns in the output
	expectedPatterns := []string{
		"overflow",
		"NULL",
		"zero",
		"uninitialized",
		"bounds",
		"shift",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(strings.ToLower(contentStr), strings.ToLower(pattern)) {
			t.Errorf("Expected pattern '%s' not found in output", pattern)
		}
	}

	// Recommendations section has been removed
}
