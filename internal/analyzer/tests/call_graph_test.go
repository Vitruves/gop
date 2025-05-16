package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitruves/gop/internal/analyzer"
)

func TestCallGraphGeneration(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "call_graph_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test C files
	testFiles := map[string]string{
		"main.c": `
#include <stdio.h>
#include "utils.h"

void printMessage(const char* message);

int main() {
    int x = 10;
    int y = 20;
    
    int sum = add(x, y);
    printMessage("Sum calculated");
    
    printf("Sum: %d\n", sum);
    return 0;
}

void printMessage(const char* message) {
    printf("Message: %s\n", message);
}
`,
		"utils.h": `
#ifndef UTILS_H
#define UTILS_H

int add(int a, int b);
int subtract(int a, int b);

#endif
`,
		"utils.c": `
#include "utils.h"

int add(int a, int b) {
    return a + b;
}

int subtract(int a, int b) {
    return a - b;
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
	outputFile := filepath.Join(tempDir, "call_graph.md")

	// Run call graph generation
	options := analyzer.CallGraphOptions{
		Directory:  tempDir,
		Languages:  []string{"c", "h"},  // Include header files
		Depth:      1,                    // Only search in the temp directory
		OutputFile: outputFile,
		Format:     "md",
		Verbose:    true,                 // Enable verbose output for debugging
	}

	// Generate call graph
	analyzer.GenerateCallGraph(options)

	// Check if output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Output file was not created")
	}

	// Read output file content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Check for expected content
	expectedStrings := []string{
		"Call Graph Analysis",
		"main",
		"add",
		"printMessage",
		"printf",
	}

	for _, str := range expectedStrings {
		if !contains(string(content), str) {
			t.Errorf("Expected string '%s' not found in output", str)
		}
	}
}

// Helper function to check if a string contains another string
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
