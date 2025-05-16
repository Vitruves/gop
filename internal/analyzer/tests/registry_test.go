package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitruves/gop/internal/analyzer"
)

func TestRegistryCommand(t *testing.T) {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	
	// Setup test directory
	testDir := filepath.Join(cwd, "test_files")
	outputFile := filepath.Join(t.TempDir(), "registry_output.md")

	// Test basic registry creation
	t.Run("BasicRegistry", func(t *testing.T) {
		options := analyzer.RegistryOptions{
			Directory:  testDir,
			Depth:      5,
			OutputFile: outputFile,
			Languages:  []string{"c", "cpp"},
			Jobs:       2,
			Types:      []string{"all"},
			Short:      false,
			Relations:  false,
			Stats:      false,
			IAOutput:   false,
			Verbose:    false,
		}

		analyzer.CreateRegistry(options)

		// Verify output file exists
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFile)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output contains expected sections and functions
		contentStr := string(content)
		if !strings.Contains(contentStr, "Function") {
			t.Errorf("Output does not contain 'Function' section")
		}
		if !strings.Contains(contentStr, "add") {
			t.Errorf("Output does not contain 'add' function")
		}
		if !strings.Contains(contentStr, "multiply") {
			t.Errorf("Output does not contain 'multiply' function")
		}
		if !strings.Contains(contentStr, "process_data") {
			t.Errorf("Output does not contain 'process_data' function")
		}
	})

	// Test with specific types
	t.Run("RegistryWithSpecificTypes", func(t *testing.T) {
		outputFileTypes := filepath.Join(t.TempDir(), "registry_types.md")
		options := analyzer.RegistryOptions{
			Directory:  testDir,
			Depth:      5,
			OutputFile: outputFileTypes,
			Languages:  []string{"c", "cpp"},
			Jobs:       2,
			Types:      []string{"function", "class"},
			Short:      false,
			Relations:  false,
			Stats:      false,
			IAOutput:   false,
			Verbose:    false,
		}

		analyzer.CreateRegistry(options)

		// Verify output file exists
		if _, err := os.Stat(outputFileTypes); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFileTypes)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFileTypes)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output contains some content
		contentStr := string(content)
		
		// Just check that the file has some content
		if len(contentStr) == 0 {
			t.Errorf("Output file is empty")
		}
	})

	// Test with statistics
	t.Run("RegistryWithStats", func(t *testing.T) {
		outputFileStats := filepath.Join(t.TempDir(), "registry_stats.md")
		options := analyzer.RegistryOptions{
			Directory:  testDir,
			Depth:      5,
			OutputFile: outputFileStats,
			Languages:  []string{"c", "cpp"},
			Jobs:       2,
			Types:      []string{"all"},
			Short:      false,
			Relations:  false,
			Stats:      true,
			IAOutput:   false,
			Verbose:    false,
		}

		analyzer.CreateRegistry(options)

		// Verify output file exists
		if _, err := os.Stat(outputFileStats); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFileStats)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFileStats)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output contains statistics
		contentStr := string(content)
		if !strings.Contains(contentStr, "Statistics") {
			t.Errorf("Output does not contain 'Statistics' section")
		}
	})

	// Test with relations
	t.Run("RegistryWithRelations", func(t *testing.T) {
		outputFileRelations := filepath.Join(t.TempDir(), "registry_relations.md")
		options := analyzer.RegistryOptions{
			Directory:  testDir,
			Depth:      5,
			OutputFile: outputFileRelations,
			Languages:  []string{"c", "cpp"},
			Jobs:       2,
			Types:      []string{"all"},
			Short:      false,
			Relations:  true,
			Stats:      false,
			IAOutput:   false,
			Verbose:    false,
		}

		analyzer.CreateRegistry(options)

		// Verify output file exists
		if _, err := os.Stat(outputFileRelations); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFileRelations)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFileRelations)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output contains some content
		contentStr := string(content)
		if len(contentStr) == 0 {
			t.Errorf("Output file is empty")
		}
	})

	// Test with AI output format
	t.Run("RegistryWithAIOutput", func(t *testing.T) {
		outputFileAI := filepath.Join(t.TempDir(), "registry_ai.md")
		options := analyzer.RegistryOptions{
			Directory:  testDir,
			Depth:      5,
			OutputFile: outputFileAI,
			Languages:  []string{"c", "cpp"},
			Jobs:       2,
			Types:      []string{"all"},
			Short:      false,
			Relations:  false,
			Stats:      false,
			IAOutput:   true,
			Verbose:    false,
		}

		analyzer.CreateRegistry(options)

		// Verify output file exists
		if _, err := os.Stat(outputFileAI); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFileAI)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFileAI)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output is in AI format (JSON-like)
		contentStr := string(content)
		if !strings.Contains(contentStr, "{") || !strings.Contains(contentStr, "}") {
			t.Errorf("Output does not appear to be in AI format")
		}
	})
}
