package concatenate

import (
	"path/filepath"
	"regexp"
	"strings"
)

type RustProcessor struct{}

func (r *RustProcessor) GetExtensions() []string {
	return []string{".rs"}
}

func (r *RustProcessor) IsTestFile(path string) bool {
	filename := filepath.Base(path)
	
	testPatterns := []string{
		"*_test.rs", "test_*.rs", "tests.rs",
	}
	
	for _, pattern := range testPatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
	}
	
	testDirs := []string{"tests", "test"}
	for _, testDir := range testDirs {
		if strings.Contains(path, testDir) {
			return true
		}
	}
	
	return false
}

func (r *RustProcessor) RemoveComments(content string) string {
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
	
	docCommentRegex := regexp.MustCompile(`(?m)^[ \t]*///.*\n`)
	content = docCommentRegex.ReplaceAllString(content, "")
	
	return content
}

func (r *RustProcessor) RemoveTestCode(content string) string {
	testModuleRegex := regexp.MustCompile(`(?s)#\[cfg\(test\)\].*?mod\s+\w+\s*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}`)
	content = testModuleRegex.ReplaceAllString(content, "")
	
	testFunctionRegex := regexp.MustCompile(`(?s)#\[test\].*?fn\s+\w+\(\)\s*(?:->\s*\w+\s*)?\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}`)
	content = testFunctionRegex.ReplaceAllString(content, "")
	
	benchmarkFunctionRegex := regexp.MustCompile(`(?s)#\[bench\].*?fn\s+\w+\(.*?\)\s*(?:->\s*\w+\s*)?\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}`)
	content = benchmarkFunctionRegex.ReplaceAllString(content, "")
	
	testImportRegex := regexp.MustCompile(`(?m)^[ \t]*use\s+.*test.*;\n`)
	content = testImportRegex.ReplaceAllString(content, "")
	
	assertMacroRegex := regexp.MustCompile(`(?m)^[ \t]*assert.*!.*;\n`)
	content = assertMacroRegex.ReplaceAllString(content, "")
	
	return content
}

func (r *RustProcessor) SupportsSpecialFiles() map[string]bool {
	return map[string]bool{
		"Cargo.toml": true,
		"Cargo.lock": true,
		"build.rs":   true,
	}
}

func (r *RustProcessor) IsHeaderFile(path string) bool {
	return false
}