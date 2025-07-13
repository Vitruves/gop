package concatenate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPythonProcessor(t *testing.T) {
	processor := &PythonProcessor{}
	
	if !contains(processor.GetExtensions(), ".py") {
		t.Error("Python processor should support .py files")
	}
	
	if processor.IsTestFile("test_example.py") != true {
		t.Error("Should identify test_example.py as test file")
	}
	
	if processor.IsTestFile("example.py") != false {
		t.Error("Should not identify example.py as test file")
	}
}

func TestRustProcessor(t *testing.T) {
	processor := &RustProcessor{}
	
	if !contains(processor.GetExtensions(), ".rs") {
		t.Error("Rust processor should support .rs files")
	}
	
	content := `
	#[cfg(test)]
	mod tests {
		#[test]
		fn test_something() {
			assert_eq!(1, 1);
		}
	}
	
	fn main() {
		println!("Hello");
	}
	`
	
	result := processor.RemoveTestCode(content)
	if strings.Contains(result, "#[test]") {
		t.Error("Should remove test code from Rust")
	}
}

func TestGoProcessor(t *testing.T) {
	processor := &GoProcessor{}
	
	if !contains(processor.GetExtensions(), ".go") {
		t.Error("Go processor should support .go files")
	}
	
	if processor.IsTestFile("example_test.go") != true {
		t.Error("Should identify example_test.go as test file")
	}
}

func TestCProcessor(t *testing.T) {
	processor := &CProcessor{}
	
	extensions := processor.GetExtensions()
	if !contains(extensions, ".c") || !contains(extensions, ".h") {
		t.Error("C processor should support .c and .h files")
	}
}

func TestCppProcessor(t *testing.T) {
	processor := &CppProcessor{}
	
	extensions := processor.GetExtensions()
	if !contains(extensions, ".cpp") || !contains(extensions, ".hpp") {
		t.Error("C++ processor should support .cpp and .hpp files")
	}
	
	if processor.IsHeaderFile("example.hpp") != true {
		t.Error("Should identify .hpp as header file")
	}
}

func TestConfigProcessing(t *testing.T) {
	tempDir := t.TempDir()
	
	testFile := filepath.Join(tempDir, "test.py")
	content := `# This is a test file
def hello():
    print("Hello, World!")

def test_hello():
    hello()
`
	
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	config := Config{
		Language:       "python",
		Include:        []string{testFile},
		RemoveTests:    true,
		RemoveComments: true,
		AddHeaders:     true,
		AddLineNumbers: false,
	}
	
	processor := &PythonProcessor{}
	files, err := collectFiles(config, processor)
	if err != nil {
		t.Fatalf("Failed to collect files: %v", err)
	}
	
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}