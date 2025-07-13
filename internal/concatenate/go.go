package concatenate

import (
	"path/filepath"
	"regexp"
	"strings"
)

type GoProcessor struct{}

func (g *GoProcessor) GetExtensions() []string {
	return []string{".go"}
}

func (g *GoProcessor) IsTestFile(path string) bool {
	filename := filepath.Base(path)
	
	testPatterns := []string{
		"*_test.go",
	}
	
	for _, pattern := range testPatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
	}
	
	return false
}

func (g *GoProcessor) RemoveComments(content string) string {
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

func (g *GoProcessor) RemoveTestCode(content string) string {
	testFunctionRegex := regexp.MustCompile(`(?s)func\s+Test\w*\(\s*t\s+\*testing\.T\s*\)\s*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}`)
	content = testFunctionRegex.ReplaceAllString(content, "")
	
	benchmarkFunctionRegex := regexp.MustCompile(`(?s)func\s+Benchmark\w*\(\s*b\s+\*testing\.B\s*\)\s*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}`)
	content = benchmarkFunctionRegex.ReplaceAllString(content, "")
	
	exampleFunctionRegex := regexp.MustCompile(`(?s)func\s+Example\w*\(\s*\)\s*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}`)
	content = exampleFunctionRegex.ReplaceAllString(content, "")
	
	testingImportRegex := regexp.MustCompile(`(?m)^[ \t]*"testing"\n`)
	content = testingImportRegex.ReplaceAllString(content, "")
	
	testifyImportRegex := regexp.MustCompile(`(?m)^[ \t]*".*testify.*"\n`)
	content = testifyImportRegex.ReplaceAllString(content, "")
	
	return content
}

func (g *GoProcessor) SupportsSpecialFiles() map[string]bool {
	return map[string]bool{
		"go.mod":  true,
		"go.sum":  true,
		"go.work": true,
	}
}

func (g *GoProcessor) IsHeaderFile(path string) bool {
	return false
}