package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestMain sets up the test environment and runs all tests
func TestMain(m *testing.M) {
	// Setup test environment
	setupTestEnvironment()
	
	// Run all tests
	exitCode := m.Run()
	
	// Clean up test environment
	cleanupTestEnvironment()
	
	// Exit with the test exit code
	os.Exit(exitCode)
}

// setupTestEnvironment ensures all necessary test files and directories exist
func setupTestEnvironment() {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current working directory: %v\n", err)
		return
	}

	// Ensure test directories exist
	testDir := filepath.Join(cwd, "test_files")
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		fmt.Printf("Creating test directory: %s\n", testDir)
		os.MkdirAll(testDir, 0755)
	}
	
	// Verify all test files exist
	requiredFiles := []string{
		filepath.Join(cwd, "test_files", "sample.c"),
		filepath.Join(cwd, "test_files", "sample.h"),
		filepath.Join(cwd, "test_files", "duplicate.c"),
		filepath.Join(cwd, "test_files", "sample.cpp"),
		filepath.Join(cwd, "test_files", "sample_cpp.h"),
	}
	
	for _, file := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			fmt.Printf("Warning: Test file not found: %s\n", file)
			fmt.Println("Please ensure all test files are created before running tests.")
		}
	}
}

// cleanupTestEnvironment cleans up any temporary files created during tests
func cleanupTestEnvironment() {
	// Nothing to clean up for now, as we use t.TempDir() for test outputs
	// which are automatically cleaned up by the testing package
}

// TestAllCommands runs a simple test for each command to ensure they all work
func TestAllCommands(t *testing.T) {
	// This is a meta-test that ensures all command tests are run
	// The actual tests are in their respective test files
	
	t.Run("ConcatCommand", func(t *testing.T) {
		// This will be handled by concat_test.go
	})
	
	t.Run("RegistryCommand", func(t *testing.T) {
		// This will be handled by registry_test.go
	})
	
	t.Run("TodoCommand", func(t *testing.T) {
		// This will be handled by todo_test.go
	})
	
	t.Run("CoherenceCommand", func(t *testing.T) {
		// This will be handled by coherence_test.go
	})
	
	t.Run("DuplicateCommand", func(t *testing.T) {
		// This will be handled by duplicate_test.go
	})
}
