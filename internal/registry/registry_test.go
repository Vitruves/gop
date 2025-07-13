package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPythonParser(t *testing.T) {
	parser := &PythonParser{}
	
	if !contains(parser.GetExtensions(), ".py") {
		t.Error("Python parser should support .py files")
	}
	
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.py")
	content := `
def hello_world():
    """A simple function"""
    print("Hello, World!")

class TestClass:
    def test_method(self):
        pass

async def async_function():
    await something()
`
	
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	functions, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}
	
	if len(functions) < 2 {
		t.Errorf("Expected at least 2 functions, got %d", len(functions))
	}
	
	foundHelloWorld := false
	foundAsyncFunction := false
	
	for _, fn := range functions {
		if fn.Name == "hello_world" {
			foundHelloWorld = true
			if fn.Language != "python" {
				t.Error("Function should be identified as Python")
			}
			if fn.Visibility != "public" {
				t.Error("Function should be public")
			}
		}
		if fn.Name == "async_function" {
			foundAsyncFunction = true
			if fn.Metadata["async"] != "true" {
				t.Error("Async function should have async metadata")
			}
		}
	}
	
	if !foundHelloWorld {
		t.Error("Should find hello_world function")
	}
	if !foundAsyncFunction {
		t.Error("Should find async_function")
	}
}

func TestGoParser(t *testing.T) {
	parser := &GoParser{}
	
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")
	content := `package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}

func HelperFunction() string {
    return "helper"
}

func TestSomething(t *testing.T) {
    // test code
}
`
	
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	functions, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}
	
	if len(functions) != 3 {
		t.Errorf("Expected 3 functions, got %d", len(functions))
	}
	
	for _, fn := range functions {
		if fn.Name == "main" && !fn.IsMain {
			t.Error("main function should be identified as main")
		}
		if fn.Name == "HelperFunction" && fn.Visibility != "public" {
			t.Error("HelperFunction should be public")
		}
		if fn.Name == "TestSomething" && !fn.IsTest {
			t.Error("TestSomething should be identified as test")
		}
	}
}

func TestRustParser(t *testing.T) {
	parser := &RustParser{}
	
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.rs")
	content := `
pub fn public_function() -> i32 {
    42
}

fn private_function() {
    println!("private");
}

#[test]
fn test_function() {
    assert_eq!(1, 1);
}

async fn async_function() {
    // async code
}
`
	
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	functions, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}
	
	if len(functions) < 3 {
		t.Errorf("Expected at least 3 functions, got %d", len(functions))
	}
	
	for _, fn := range functions {
		if fn.Name == "public_function" && fn.Visibility != "public" {
			t.Error("public_function should be public")
		}
		if fn.Name == "private_function" && fn.Visibility != "private" {
			t.Error("private_function should be private")
		}
		if fn.Name == "test_function" && !fn.IsTest {
			t.Error("test_function should be identified as test")
		}
	}
}

func TestConfigValidation(t *testing.T) {
	config := Config{
		Language: "python",
		Jobs:     4,
	}
	
	if config.Language != "python" {
		t.Error("Config language should be set correctly")
	}
	
	if config.Jobs != 4 {
		t.Error("Config jobs should be set correctly")
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