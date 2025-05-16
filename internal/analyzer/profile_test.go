package analyzer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunProfiler(t *testing.T) {
	// Skip test if not on a supported platform
	if testing.Short() {
		t.Skip("Skipping profiler test in short mode")
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
	
	// Use exec.Command to compile
	cmd := exec.Command("gcc", "-o", executableFile, programFile)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to compile test program: %v", err)
	}

	// Create a temporary output file
	outputFile := filepath.Join(tempDir, "profile_output.md")

	// Run profiler with time profiling (most portable)
	options := ProfileOptions{
		Executable:  executableFile,
		Args:        []string{"10000"},
		OutputFile:  outputFile,
		Format:      "md",
		ProfileType: "time",
		Short:       false,
		Verbose:     false,
	}

	// Run the profiler
	RunProfiler(options)

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
		"# Performance Profile",
		"## Summary",
		"| Executable |",
		"| Profile Type | time |",
	}

	for _, section := range expectedSections {
		if !strings.Contains(string(content), section) {
			t.Errorf("Output does not contain expected section: %s", section)
		}
	}
}


