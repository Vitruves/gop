package concatenate

import (
	"path/filepath"
	"regexp"
	"strings"
)

type GenericProcessor struct{}

func (g *GenericProcessor) GetExtensions() []string {
	return []string{".py", ".rs", ".go", ".c", ".cpp", ".cxx", ".cc", ".h", ".hpp", ".hxx", ".hh", ".h++", ".c++"}
}

func (g *GenericProcessor) IsTestFile(path string) bool {
	filename := filepath.Base(path)
	
	testPatterns := []string{
		"test_*", "*_test.*", "test*.*",
		"*Test.*", "*Tests.*",
	}
	
	for _, pattern := range testPatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
	}
	
	testDirs := []string{"tests", "test", "__tests__", "unit_tests", "integration_tests"}
	for _, testDir := range testDirs {
		if strings.Contains(path, testDir) {
			return true
		}
	}
	
	return false
}

func (g *GenericProcessor) RemoveComments(content string) string {
	ext := filepath.Ext(strings.ToLower(content))
	
	switch ext {
	case ".py":
		return g.removePythonComments(content)
	case ".rs", ".go", ".c", ".cpp", ".cxx", ".cc", ".h", ".hpp", ".hxx", ".hh", ".h++", ".c++":
		return g.removeCStyleComments(content)
	default:
		return g.removeCStyleComments(content)
	}
}

func (g *GenericProcessor) RemoveTestCode(content string) string {
	testFunctionRegex := regexp.MustCompile(`(?s)(def|func|void|int)\s+(test_|Test)\w*.*?\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}`)
	content = testFunctionRegex.ReplaceAllString(content, "")
	
	testClassRegex := regexp.MustCompile(`(?s)class\s+(Test|.*Test)\w*.*?\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}`)
	content = testClassRegex.ReplaceAllString(content, "")
	
	return content
}

func (g *GenericProcessor) SupportsSpecialFiles() map[string]bool {
	return map[string]bool{
		"Makefile":         true,
		"CMakeLists.txt":   true,
		"Cargo.toml":       true,
		"go.mod":           true,
		"requirements.txt": true,
		"setup.py":         true,
		"package.json":     true,
	}
}

func (g *GenericProcessor) IsHeaderFile(path string) bool {
	ext := filepath.Ext(path)
	headerExts := []string{".h", ".hpp", ".hxx", ".hh"}
	for _, headerExt := range headerExts {
		if ext == headerExt {
			return true
		}
	}
	return false
}

func (g *GenericProcessor) removePythonComments(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	
	inDocstring := false
	docstringDelim := ""
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		if inDocstring {
			if strings.Contains(line, docstringDelim) {
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
			
			if strings.Count(trimmed, docstringDelim) == 1 {
				inDocstring = true
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

func (g *GenericProcessor) removeCStyleComments(content string) string {
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