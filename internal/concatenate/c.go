package concatenate

import (
	"path/filepath"
	"regexp"
	"strings"
)

type CProcessor struct{}

func (c *CProcessor) GetExtensions() []string {
	return []string{".c", ".h"}
}

func (c *CProcessor) IsTestFile(path string) bool {
	filename := filepath.Base(path)
	
	testPatterns := []string{
		"test_*.c", "*_test.c", "test*.c",
		"test_*.h", "*_test.h", "test*.h",
	}
	
	for _, pattern := range testPatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
	}
	
	testDirs := []string{"tests", "test", "unit_tests"}
	for _, testDir := range testDirs {
		if strings.Contains(path, testDir) {
			return true
		}
	}
	
	return false
}

func (c *CProcessor) RemoveComments(content string) string {
	singleLineRegex := regexp.MustCompile(`//.*$`)
	lines := strings.Split(content, "\n")
	var result []string
	
	for _, line := range lines {
		processed := singleLineRegex.ReplaceAllString(line, "")
		result = append(result, processed)
	}
	
	content = strings.Join(result, "\n")
	
	multiLineRegex := regexp.MustCompile(`(?s)/\*.*?\*/`)
	content = multiLineRegex.ReplaceAllString(content, "")
	
	return content
}

func (c *CProcessor) RemoveTestCode(content string) string {
	testFunctionRegex := regexp.MustCompile(`(?s)(void|int)\s+test_\w+\s*\([^)]*\)\s*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}`)
	content = testFunctionRegex.ReplaceAllString(content, "")
	
	testMainRegex := regexp.MustCompile(`(?s)int\s+main\s*\([^)]*\)\s*\{[^{}]*test[^{}]*\}`)
	content = testMainRegex.ReplaceAllString(content, "")
	
	assertIncludeRegex := regexp.MustCompile(`(?m)^[ \t]*#include\s+<assert\.h>.*\n`)
	content = assertIncludeRegex.ReplaceAllString(content, "")
	
	unityIncludeRegex := regexp.MustCompile(`(?m)^[ \t]*#include\s+"unity\.h".*\n`)
	content = unityIncludeRegex.ReplaceAllString(content, "")
	
	cunitIncludeRegex := regexp.MustCompile(`(?m)^[ \t]*#include\s+<CUnit/.*\.h>.*\n`)
	content = cunitIncludeRegex.ReplaceAllString(content, "")
	
	assertMacroRegex := regexp.MustCompile(`(?m)^[ \t]*assert\s*\(.*\)\s*;.*\n`)
	content = assertMacroRegex.ReplaceAllString(content, "")
	
	testAssertRegex := regexp.MustCompile(`(?m)^[ \t]*TEST_ASSERT.*\(.*\)\s*;.*\n`)
	content = testAssertRegex.ReplaceAllString(content, "")
	
	return content
}

func (c *CProcessor) SupportsSpecialFiles() map[string]bool {
	return map[string]bool{
		"Makefile":       true,
		"CMakeLists.txt": true,
		"configure.ac":   true,
		"configure.in":   true,
		"config.h.in":    true,
	}
}

func (c *CProcessor) IsHeaderFile(path string) bool {
	return filepath.Ext(path) == ".h"
}