package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitruves/gop/internal/analyzer"
)

func TestCoherenceCommand(t *testing.T) {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current working directory: %v", err)
	}
	
	// Setup test directory
	testDir := filepath.Join(cwd, "test_files")
	outputFile := filepath.Join(t.TempDir(), "coherence_output.txt")

	// Test basic coherence checking
	t.Run("BasicCoherenceCheck", func(t *testing.T) {
		options := analyzer.CoherenceOptions{
			Directory:           testDir,
			Depth:               5,
			OutputFile:          outputFile,
			Languages:           []string{"c", "cpp"},
			Jobs:                2,
			CheckHeaders:        true,
			CheckFiles:          true,
			ShowDiscrepancies:   true,
			NonImplemented:      false,
			NotDeclared:         false,
			SimilarityThreshold: 0.0,
			Short:               false,
			Verbose:             false,
		}

		analyzer.CheckCoherence(options)

		// Verify output file exists
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFile)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFile)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output contains expected discrepancies
		contentStr := string(content)
		
		// Should detect divide() is declared but not implemented
		if !strings.Contains(contentStr, "divide") {
			t.Errorf("Output does not mention 'divide' function which is declared but not implemented")
		}
		
		// Should detect process_data() is implemented but not declared in header
		if !strings.Contains(contentStr, "process_data") {
			t.Errorf("Output does not mention 'process_data' function which is implemented but not declared")
		}
	})

	// Test with only non-implemented checks
	t.Run("CoherenceNonImplementedOnly", func(t *testing.T) {
		outputFileNonImpl := filepath.Join(t.TempDir(), "coherence_non_impl.txt")
		options := analyzer.CoherenceOptions{
			Directory:           testDir,
			Depth:               5,
			OutputFile:          outputFileNonImpl,
			Languages:           []string{"c", "cpp"},
			Jobs:                2,
			CheckHeaders:        true,
			CheckFiles:          false,
			ShowDiscrepancies:   true,
			NonImplemented:      true,
			NotDeclared:         false,
			SimilarityThreshold: 0.0,
			Short:               false,
			Verbose:             false,
		}

		analyzer.CheckCoherence(options)

		// Verify output file exists
		if _, err := os.Stat(outputFileNonImpl); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFileNonImpl)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFileNonImpl)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output contains only non-implemented functions
		contentStr := string(content)
		
		// Should detect divide() is declared but not implemented
		if !strings.Contains(contentStr, "divide") {
			t.Errorf("Output does not mention 'divide' function which is declared but not implemented")
		}
		
		// Should NOT detect process_data() as we're only checking non-implemented
		if strings.Contains(contentStr, "process_data") {
			t.Errorf("Output mentions 'process_data' when we're only checking for non-implemented functions")
		}
	})

	// Test with only not-declared checks
	t.Run("CoherenceNotDeclaredOnly", func(t *testing.T) {
		outputFileNotDecl := filepath.Join(t.TempDir(), "coherence_not_decl.txt")
		options := analyzer.CoherenceOptions{
			Directory:           testDir,
			Depth:               5,
			OutputFile:          outputFileNotDecl,
			Languages:           []string{"c", "cpp"},
			Jobs:                2,
			CheckHeaders:        false,
			CheckFiles:          true,
			ShowDiscrepancies:   true,
			NonImplemented:      false,
			NotDeclared:         true,
			SimilarityThreshold: 0.0,
			Short:               false,
			Verbose:             false,
		}

		analyzer.CheckCoherence(options)

		// Verify output file exists
		if _, err := os.Stat(outputFileNotDecl); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFileNotDecl)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFileNotDecl)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output contains only not-declared functions
		contentStr := string(content)
		
		// Should NOT detect divide() as we're only checking not-declared
		if strings.Contains(contentStr, "divide") {
			t.Errorf("Output mentions 'divide' when we're only checking for not-declared functions")
		}
		
		// Should detect process_data() is implemented but not declared
		if !strings.Contains(contentStr, "process_data") {
			t.Errorf("Output does not mention 'process_data' function which is implemented but not declared")
		}
	})

	// Test with similarity threshold
	t.Run("CoherenceWithSimilarityThreshold", func(t *testing.T) {
		outputFileSimilarity := filepath.Join(t.TempDir(), "coherence_similarity.txt")
		options := analyzer.CoherenceOptions{
			Directory:           testDir,
			Depth:               5,
			OutputFile:          outputFileSimilarity,
			Languages:           []string{"c", "cpp"},
			Jobs:                2,
			CheckHeaders:        true,
			CheckFiles:          true,
			ShowDiscrepancies:   true,
			NonImplemented:      false,
			NotDeclared:         false,
			SimilarityThreshold: 0.7, // Set a similarity threshold to detect similar functions
			Short:               false,
			Verbose:             false,
		}

		analyzer.CheckCoherence(options)

		// Verify output file exists
		if _, err := os.Stat(outputFileSimilarity); os.IsNotExist(err) {
			t.Errorf("Output file was not created: %s", outputFileSimilarity)
		}

		// Read output file and verify content
		content, err := os.ReadFile(outputFileSimilarity)
		if err != nil {
			t.Errorf("Failed to read output file: %v", err)
		}

		// Check that the output contains discrepancies information
		contentStr := string(content)
		if !strings.Contains(contentStr, "discrepancies") {
			t.Errorf("Output does not contain discrepancies information")
		}
	})

	// Test with AI output format
	t.Run("CoherenceWithAIOutput", func(t *testing.T) {
		outputFileAI := filepath.Join(t.TempDir(), "coherence_ai.txt")
		options := analyzer.CoherenceOptions{
			Directory:           testDir,
			Depth:               5,
			OutputFile:          outputFileAI,
			Languages:           []string{"c", "cpp"},
			Jobs:                2,
			CheckHeaders:        true,
			CheckFiles:          true,
			ShowDiscrepancies:   true,
			NonImplemented:      false,
			NotDeclared:         false,
			SimilarityThreshold: 0.0,
			Short:               false,
			IAOutput:            true,
			Verbose:             false,
		}

		analyzer.CheckCoherence(options)

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
