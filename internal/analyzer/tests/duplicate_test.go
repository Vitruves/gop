package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitruves/gop/internal/analyzer"
)

func TestDuplicateCommand(t *testing.T) {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	
	// Setup test directory
	testDir := filepath.Join(cwd, "test_files")
	outputFile := filepath.Join(t.TempDir(), "duplicate_output.md")
	monitorFile := filepath.Join(t.TempDir(), "duplication_history.json")

	// Create a filtered test directory with only simple files
	filteredTestDir := filepath.Join(t.TempDir(), "filtered_test_files")
	err = os.MkdirAll(filteredTestDir, 0755)
	if err != nil {
		t.Fatalf("Error creating filtered test directory: %v", err)
	}

	// Copy only the original test files to avoid timeout with complex files
	originalFiles := []string{"duplicate.c", "sample.c", "sample.cpp", "sample.h", "sample_cpp.h"}
	for _, file := range originalFiles {
		srcPath := filepath.Join(testDir, file)
		dstPath := filepath.Join(filteredTestDir, file)
		
		data, err := os.ReadFile(srcPath)
		if err != nil {
			t.Fatalf("Error reading file %s: %v", srcPath, err)
		}
		
		err = os.WriteFile(dstPath, data, 0644)
		if err != nil {
			t.Fatalf("Error writing file %s: %v", dstPath, err)
		}
	}

	// Test basic duplicate detection
	t.Run("BasicDuplicateDetection", func(t *testing.T) {
		options := analyzer.DuplicateOptions{
			Directory:           filteredTestDir,
			Depth:               5,
			OutputFile:          outputFile,
			Languages:           []string{"c", "cpp"},
			Jobs:                2,
			SimilarityThreshold: 0.8,
			MinLineCount:        3,
			Short:               false,
			NamesOnly:           false,
			Verbose:             false,
			Monitor:             false,
		}

		analyzer.FindDuplicates(options)

		// Verify output file exists
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFile)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output contains duplicate code blocks
		contentStr := string(content)
		
		// Just check that the file contains duplicate information
		if !strings.Contains(contentStr, "duplicate") && !strings.Contains(contentStr, "Duplicate") {
			t.Errorf("Output does not contain duplicate information")
		}
	})

	// Test with names-only option
	t.Run("DuplicateNamesOnly", func(t *testing.T) {
		outputFileNames := filepath.Join(t.TempDir(), "duplicate_names.md")
		options := analyzer.DuplicateOptions{
			Directory:           filteredTestDir,
			Depth:               5,
			OutputFile:          outputFileNames,
			Languages:           []string{"c", "cpp"},
			Jobs:                2,
			SimilarityThreshold: 0.8,
			MinLineCount:        3,
			Short:               false,
			NamesOnly:           true,
			Verbose:             false,
			Monitor:             false,
		}

		analyzer.FindDuplicates(options)

		// Verify output file exists
		if _, err := os.Stat(outputFileNames); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFileNames)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFileNames)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output contains duplicate method names
		contentStr := string(content)
		
		// Just check that the file contains duplicate information
		if !strings.Contains(contentStr, "duplicate") && !strings.Contains(contentStr, "Duplicate") {
			t.Errorf("Output does not contain duplicate information")
		}
		
		// Should NOT contain code blocks since we're only showing names
		if strings.Contains(contentStr, "for (int i = 0;") {
			t.Errorf("Output contains code blocks when it should only show method names")
		}
	})

	// Test with different similarity threshold
	t.Run("DuplicateWithDifferentThreshold", func(t *testing.T) {
		outputFileThreshold := filepath.Join(t.TempDir(), "duplicate_threshold.md")
		options := analyzer.DuplicateOptions{
			Directory:           filteredTestDir,
			Depth:               5,
			OutputFile:          outputFileThreshold,
			Languages:           []string{"c", "cpp"},
			Jobs:                2,
			SimilarityThreshold: 0.95, // Higher threshold should detect fewer duplicates
			MinLineCount:        3,
			Short:               false,
			NamesOnly:           false,
			Verbose:             false,
			Monitor:             false,
		}

		analyzer.FindDuplicates(options)

		// Verify output file exists
		if _, err := os.Stat(outputFileThreshold); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFileThreshold)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFileThreshold)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// With a higher threshold, we should detect fewer duplicates
		contentStr := string(content)
		
		// Count the number of duplicate blocks
		duplicateCount := strings.Count(contentStr, "Duplicate")
		
		// Run again with lower threshold
		outputFileLowerThreshold := filepath.Join(t.TempDir(), "duplicate_lower_threshold.md")
		optionsLower := options
		optionsLower.OutputFile = outputFileLowerThreshold
		optionsLower.SimilarityThreshold = 0.6 // Lower threshold should detect more duplicates
		
		analyzer.FindDuplicates(optionsLower)
		
		contentLower, err := os.ReadFile(outputFileLowerThreshold)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}
		
		// Count the number of duplicate blocks with lower threshold
		duplicateCountLower := strings.Count(string(contentLower), "Duplicate")
		
		// Lower threshold should find more duplicates
		if duplicateCountLower <= duplicateCount {
			t.Errorf("Lower similarity threshold did not detect more duplicates")
		}
	})

	// Test with monitoring enabled
	t.Run("DuplicateWithMonitoring", func(t *testing.T) {
		options := analyzer.DuplicateOptions{
			Directory:           filteredTestDir,
			Depth:               5,
			OutputFile:          outputFile,
			Languages:           []string{"c", "cpp"},
			Jobs:                2,
			SimilarityThreshold: 0.8,
			MinLineCount:        3,
			Short:               false,
			NamesOnly:           false,
			Verbose:             false,
			Monitor:             true,
			MonitorFile:         monitorFile,
			MonitorComment:      "Test monitoring run",
		}

		analyzer.FindDuplicates(options)

		// Verify monitor file exists
		if _, err := os.Stat(monitorFile); os.IsNotExist(err) {
			t.Errorf("Monitor file was not created: %s", monitorFile)
		}

		// Read monitor file and verify content
		content, err := os.ReadFile(monitorFile)
		if err != nil {
			t.Errorf("Failed to read monitor file: %v", err)
		}

		// Check that the monitor file contains valid JSON
		var monitorData interface{}
		if err := json.Unmarshal(content, &monitorData); err != nil {
			t.Errorf("Monitor file does not contain valid JSON: %v", err)
		}

		// Check that the monitor file contains our comment
		if !strings.Contains(string(content), "Test monitoring run") {
			t.Errorf("Monitor file does not contain our comment")
		}
	})

	// Test with short output format
	t.Run("DuplicateWithShortOutput", func(t *testing.T) {
		outputFileShort := filepath.Join(t.TempDir(), "duplicate_short.md")
		options := analyzer.DuplicateOptions{
			Directory:           filteredTestDir,
			Depth:               5,
			OutputFile:          outputFileShort,
			Languages:           []string{"c", "cpp"},
			Jobs:                2,
			SimilarityThreshold: 0.8,
			MinLineCount:        3,
			Short:               true,
			NamesOnly:           false,
			Verbose:             false,
			Monitor:             false,
		}

		analyzer.FindDuplicates(options)

		// Verify output file exists
		if _, err := os.Stat(outputFileShort); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFileShort)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFileShort)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output contains some content
		contentStr := string(content)
		
		// Just check that the file contains duplicate information
		if !strings.Contains(contentStr, "duplicate") && !strings.Contains(contentStr, "Duplicate") {
			t.Errorf("Output does not contain duplicate information")
		}
	})
}
