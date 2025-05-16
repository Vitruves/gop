package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitruves/gop/internal/analyzer"
)

func TestTodoCommand(t *testing.T) {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	
	// Setup test directory
	testDir := filepath.Join(cwd, "test_files")
	outputFile := filepath.Join(t.TempDir(), "todo_output.txt")

	// Test basic todo finding
	t.Run("BasicTodoFinding", func(t *testing.T) {
		options := analyzer.TodoOptions{
			Directory:  testDir,
			Depth:      5,
			OutputFile: outputFile,
			Languages:  []string{"c", "cpp"},
			Jobs:       2,
			Short:      false,
			Verbose:    false,
		}

		analyzer.FindTodos(options)

		// Verify output file exists
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFile)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output contains expected TODOs
		contentStr := string(content)
		if !strings.Contains(contentStr, "TODO: Implement error handling") {
			t.Errorf("Output does not contain expected TODO comment")
		}
		if !strings.Contains(contentStr, "FIXME: This function needs optimization") {
			t.Errorf("Output does not contain expected FIXME comment")
		}
		if !strings.Contains(contentStr, "TODO: Add proper error handling") {
			t.Errorf("Output does not contain expected TODO comment from C++ file")
		}
	})

	// Test with specific language
	t.Run("TodoWithSpecificLanguage", func(t *testing.T) {
		outputFileCOnly := filepath.Join(t.TempDir(), "todo_c_only.txt")
		options := analyzer.TodoOptions{
			Directory:  testDir,
			Depth:      5,
			OutputFile: outputFileCOnly,
			Languages:  []string{"c"},
			Jobs:       2,
			Short:      false,
			Verbose:    false,
		}

		analyzer.FindTodos(options)

		// Verify output file exists
		if _, err := os.Stat(outputFileCOnly); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFileCOnly)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFileCOnly)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output contains C TODOs but not C++ TODOs
		contentStr := string(content)
		if !strings.Contains(contentStr, "TODO: Implement error handling") {
			t.Errorf("Output does not contain expected TODO comment from C file")
		}
		if strings.Contains(contentStr, "Calculator") {
			t.Errorf("Output contains C++ content when it should only contain C content")
		}
	})

	// Test with short output format
	t.Run("TodoWithShortOutput", func(t *testing.T) {
		outputFileShort := filepath.Join(t.TempDir(), "todo_short.txt")
		options := analyzer.TodoOptions{
			Directory:  testDir,
			Depth:      5,
			OutputFile: outputFileShort,
			Languages:  []string{"c", "cpp"},
			Jobs:       2,
			Short:      true,
			Verbose:    false,
		}

		analyzer.FindTodos(options)

		// Verify output file exists
		if _, err := os.Stat(outputFileShort); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFileShort)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFileShort)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output is in short format (should be more concise)
		contentStr := string(content)
		// Check that the output contains some content
		if len(contentStr) == 0 {
			t.Errorf("Output file is empty")
		}
	})

	// Test with verbose output
	t.Run("TodoWithVerboseOutput", func(t *testing.T) {
		outputFileVerbose := filepath.Join(t.TempDir(), "todo_verbose.txt")
		options := analyzer.TodoOptions{
			Directory:  testDir,
			Depth:      5,
			OutputFile: outputFileVerbose,
			Languages:  []string{"c", "cpp"},
			Jobs:       2,
			Short:      false,
			Verbose:    true,
		}

		analyzer.FindTodos(options)

		// Verify output file exists
		if _, err := os.Stat(outputFileVerbose); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFileVerbose)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFileVerbose)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output contains verbose information
		contentStr := string(content)
		if !strings.Contains(contentStr, "Found") && !strings.Contains(contentStr, "Processing") {
			t.Errorf("Output does not contain expected verbose information")
		}
	})
}
