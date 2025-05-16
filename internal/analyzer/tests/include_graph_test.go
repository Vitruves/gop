package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitruves/gop/internal/analyzer"
)

func TestIncludeGraphGeneration(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "include_graph_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test C files with include relationships
	testFiles := map[string]string{
		"main.c": `
#include <stdio.h>
#include <stdlib.h>
#include "utils.h"
#include "config.h"

int main() {
    // Code here
    return 0;
}
`,
		"utils.h": `
#ifndef UTILS_H
#define UTILS_H

#include <string.h>
#include "helper.h"

// Function declarations
void utilFunction();

#endif
`,
		"helper.h": `
#ifndef HELPER_H
#define HELPER_H

// Helper functions
void helperFunction();

#endif
`,
		"config.h": `
#ifndef CONFIG_H
#define CONFIG_H

#define VERSION "1.0.0"

#endif
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
	outputFile := filepath.Join(tempDir, "include_graph.md")

	// Run include graph generation
	options := analyzer.IncludeGraphOptions{
		Directory:  tempDir,
		Languages:  []string{"c", "h"},
		Depth:      1,                    // Only search in the temp directory
		OutputFile: outputFile,
		Format:     "md",
		Verbose:    true,                 // Enable verbose output for debugging
	}

	// Generate include graph
	analyzer.GenerateIncludeGraph(options)

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

	// Check for expected include relationships
	expectedRelationships := []struct {
		source string
		target string
	}{
		{"main.c", "stdio.h"},
		{"main.c", "stdlib.h"},
		{"main.c", "utils.h"},
		{"main.c", "config.h"},
		{"utils.h", "string.h"},
		{"utils.h", "helper.h"},
	}

	for _, rel := range expectedRelationships {
		// Check if the relationship is mentioned in the output
		if !strings.Contains(contentStr, rel.source) || !strings.Contains(contentStr, rel.target) {
			t.Errorf("Expected include relationship '%s -> %s' not found in output", 
				rel.source, rel.target)
		}
	}

	// Check for summary information
	if !strings.Contains(contentStr, "Summary") {
		t.Errorf("Summary section not found in output")
	}

	// Check for system vs local includes distinction
	if !strings.Contains(contentStr, "System") || !strings.Contains(contentStr, "Local") {
		t.Errorf("System/Local include type distinction not found in output")
	}
}
