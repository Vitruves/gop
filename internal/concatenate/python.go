package concatenate

import (
	"path/filepath"
	"regexp"
	"strings"
)

type PythonProcessor struct{}

func (p *PythonProcessor) GetExtensions() []string {
	return []string{".py"}
}

func (p *PythonProcessor) IsTestFile(path string) bool {
	filename := filepath.Base(path)
	
	testPatterns := []string{
		"test_*.py", "*_test.py", "test*.py", "conftest.py",
	}
	
	for _, pattern := range testPatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
	}
	
	testDirs := []string{"tests", "test", "__tests__"}
	for _, testDir := range testDirs {
		if strings.Contains(path, testDir) {
			return true
		}
	}
	
	return false
}

func (p *PythonProcessor) RemoveComments(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	
	inDocstring := false
	docstringDelim := ""
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		if inDocstring {
			if strings.Contains(line, docstringDelim) {
				parts := strings.Split(line, docstringDelim)
				if len(parts) > 1 {
					remaining := strings.Join(parts[1:], docstringDelim)
					if strings.TrimSpace(remaining) != "" {
						result = append(result, remaining)
					}
				}
				inDocstring = false
				docstringDelim = ""
			}
			continue
		}
		
		if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, `'''`) {
			if strings.HasPrefix(trimmed, `"""`) {
				docstringDelim = `"""`
			} else {
				docstringDelim = `'''`
			}
			
			occurrences := strings.Count(trimmed, docstringDelim)
			if occurrences == 1 {
				inDocstring = true
				continue
			} else if occurrences >= 2 {
				continue
			}
		}
		
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		
		if commentIndex := strings.Index(line, "#"); commentIndex != -1 {
			beforeComment := line[:commentIndex]
			if !isInsideString(beforeComment) {
				line = strings.TrimRight(beforeComment, " \t")
			}
		}
		
		result = append(result, line)
	}
	
	return strings.Join(result, "\n")
}

func (p *PythonProcessor) RemoveTestCode(content string) string {
	testFunctionRegex := regexp.MustCompile(`(?m)^[ \t]*def test_.*?(?=^[ \t]*def|\n[ \t]*class|\n[ \t]*if|\n[ \t]*$|\z)`)
	content = testFunctionRegex.ReplaceAllString(content, "")
	
	testClassRegex := regexp.MustCompile(`(?s)^[ \t]*class Test.*?(?=^[ \t]*class|\n[ \t]*def|\n[ \t]*if|\n[ \t]*$|\z)`)
	content = testClassRegex.ReplaceAllString(content, "")
	
	unittestImportRegex := regexp.MustCompile(`(?m)^[ \t]*import unittest.*\n`)
	content = unittestImportRegex.ReplaceAllString(content, "")
	
	pytestImportRegex := regexp.MustCompile(`(?m)^[ \t]*import pytest.*\n`)
	content = pytestImportRegex.ReplaceAllString(content, "")
	
	return content
}

func (p *PythonProcessor) SupportsSpecialFiles() map[string]bool {
	return map[string]bool{
		"requirements.txt": true,
		"setup.py":         true,
		"pyproject.toml":   true,
		"setup.cfg":        true,
		"Pipfile":          true,
		"poetry.lock":      true,
	}
}

func (p *PythonProcessor) IsHeaderFile(path string) bool {
	return false
}

func isInsideString(code string) bool {
	inSingle := false
	inDouble := false
	inTripleSingle := false
	inTripleDouble := false
	escaped := false
	
	for i, char := range code {
		if escaped {
			escaped = false
			continue
		}
		
		if i < len(code)-2 {
			if code[i:i+3] == `"""` && !inSingle && !inTripleSingle {
				inTripleDouble = !inTripleDouble
				continue
			}
			if code[i:i+3] == `'''` && !inDouble && !inTripleDouble {
				inTripleSingle = !inTripleSingle
				continue
			}
		}
		
		if inTripleSingle || inTripleDouble {
			continue
		}
		
		switch char {
		case '\\':
			escaped = true
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		}
	}
	
	return inSingle || inDouble || inTripleSingle || inTripleDouble
}