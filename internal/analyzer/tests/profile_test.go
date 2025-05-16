package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitruves/gop/internal/analyzer"
)

func TestProfileCommand(t *testing.T) {
	// Skip test if running in short mode
	if testing.Short() {
		t.Skip("Skipping profile test in short mode")
	}

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "profile_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple test program to profile
	testProgram := `
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

// A simple function that does some work
void do_work(int iterations) {
    int sum = 0;
    for (int i = 0; i < iterations; i++) {
        sum += i;
    }
    printf("Sum: %d\\n", sum);
}

int main(int argc, char *argv[]) {
    int iterations = 1000000;
    if (argc > 1) {
        iterations = atoi(argv[1]);
    }
    
    do_work(iterations);
    
    return 0;
}
`

	// Write the test program to a file
	programFile := filepath.Join(tempDir, "test_program.c")
	if err := os.WriteFile(programFile, []byte(testProgram), 0644); err != nil {
		t.Fatalf("Failed to write test program: %v", err)
	}

	// Compile the test program
	executableFile := filepath.Join(tempDir, "test_program")
	cmd := exec.Command("gcc", "-o", executableFile, programFile)
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping test: Failed to compile test program: %v", err)
		return
	}

	// Make sure the executable exists and is executable
	if _, err := os.Stat(executableFile); os.IsNotExist(err) {
		t.Skipf("Skipping test: Executable was not created")
		return
	}

	// Create a temporary output file
	outputFile := filepath.Join(tempDir, "profile_output.md")

	// Test cases for different profiling options
	testCases := []struct {
		name     string
		options  analyzer.ProfileOptions
		expected []string
	}{
		{
			name: "TimeProfile",
			options: analyzer.ProfileOptions{
				Executable:  executableFile,
				Args:        []string{"10000"},
				OutputFile:  outputFile,
				Format:      "md",
				ProfileType: "time",
				Duration:    1,
				Short:       false,
				Verbose:     false,
			},
			expected: []string{
				"# Performance Profile",
				"## Summary",
				"| Executable |",
				"| Profile Type | time |",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run profiler
			analyzer.RunProfiler(tc.options)

			// Verify output file exists
			if _, err := os.Stat(outputFile); os.IsNotExist(err) {
				t.Errorf("Output file was not created")
				return
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
