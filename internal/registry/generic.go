package registry

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type GenericParser struct{}

func (g *GenericParser) GetExtensions() []string {
	return []string{".py", ".rs", ".go", ".c", ".cpp", ".cxx", ".cc", ".h", ".hpp", ".hxx", ".hh"}
}

func (g *GenericParser) IsHeaderFile(filePath string) bool {
	ext := filepath.Ext(filePath)
	headerExts := []string{".h", ".hpp", ".hxx", ".hh"}
	for _, headerExt := range headerExts {
		if ext == headerExt {
			return true
		}
	}
	return false
}

func (g *GenericParser) ParseFile(filePath string) ([]Function, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var functions []Function
	lines := strings.Split(string(content), "\n")
	
	// Generic patterns for different languages
	patterns := []struct {
		regex    *regexp.Regexp
		language string
	}{
		{regexp.MustCompile(`^\s*(def|async def)\s+(\w+)\s*\(`), "python"},
		{regexp.MustCompile(`^\s*(pub\s+)?fn\s+(\w+)\s*\(`), "rust"},
		{regexp.MustCompile(`^\s*func\s+(\w+)\s*\(`), "go"},
		{regexp.MustCompile(`^\s*(\w+)\s+(\w+)\s*\(.*\)\s*[{;]`), "c/cpp"},
	}
	
	ext := filepath.Ext(filePath)
	detectedLang := detectLanguageFromExtension(ext)
	
	for i, line := range lines {
		for _, pattern := range patterns {
			if matches := pattern.regex.FindStringSubmatch(line); matches != nil {
				var name string
				
				switch pattern.language {
				case "python":
					name = matches[2]
				case "rust", "go":
					if len(matches) > 2 {
						name = matches[2]
					} else {
						name = matches[1]
					}
				case "c/cpp":
					if len(matches) > 2 {
						name = matches[2]
					} else {
						name = matches[1]
					}
				}
				
				if name == "" {
					continue
				}
				
				// Skip obvious non-functions
				if isGenericKeyword(name) {
					continue
				}
				
				fn := Function{
					Name:       name,
					File:       filePath,
					Line:       i + 1,
					Visibility: determineGenericVisibility(name, line),
					Language:   detectedLang,
					Signature:  strings.TrimSpace(line),
					Size:       1, // Simplified size calculation
					IsTest:     isGenericTestFunction(name),
					IsMain:     name == "main" || name == "__main__",
				}
				
				functions = append(functions, fn)
				break // Only match one pattern per line
			}
		}
	}
	
	return functions, nil
}

func (g *GenericParser) FindFunctionCalls(content string) []string {
	// Generic function call patterns
	callRegex := regexp.MustCompile(`(\w+)\s*\(`)
	matches := callRegex.FindAllStringSubmatch(content, -1)
	
	var calls []string
	seen := make(map[string]bool)
	
	for _, match := range matches {
		call := match[1]
		if !seen[call] && !isGenericBuiltin(call) && !isGenericKeyword(call) {
			calls = append(calls, call)
			seen[call] = true
		}
	}
	
	return calls
}

func detectLanguageFromExtension(ext string) string {
	switch ext {
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".go":
		return "go"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cxx", ".cc", ".hpp", ".hxx", ".hh", ".h++", ".c++":
		return "cpp"
	default:
		return "unknown"
	}
}

func determineGenericVisibility(name, line string) string {
	// Generic visibility determination
	if strings.HasPrefix(name, "_") {
		return "private"
	}
	
	// Check for explicit visibility keywords
	if strings.Contains(line, "private") {
		return "private"
	}
	if strings.Contains(line, "protected") {
		return "protected"
	}
	if strings.Contains(line, "public") || strings.Contains(line, "pub") {
		return "public"
	}
	
	// Default to public for most cases
	return "public"
}

func isGenericTestFunction(name string) bool {
	testPatterns := []string{
		"test_", "_test", "Test", "TEST",
	}
	
	for _, pattern := range testPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}
	
	return false
}

func isGenericBuiltin(name string) bool {
	// Common built-in functions across languages
	builtins := []string{
		"print", "printf", "println", "len", "size", "count", "max", "min",
		"sort", "map", "filter", "reduce", "sum", "abs", "round",
		"open", "close", "read", "write", "file", "input", "output",
		"assert", "expect", "panic", "error", "throw", "catch",
		"new", "delete", "malloc", "free", "alloc",
		"true", "false", "null", "nil", "undefined",
	}
	
	for _, builtin := range builtins {
		if name == builtin {
			return true
		}
	}
	
	return false
}

func isGenericKeyword(name string) bool {
	// Common keywords across programming languages
	keywords := []string{
		"if", "else", "elif", "while", "for", "do", "switch", "case", "default",
		"break", "continue", "return", "goto", "try", "catch", "finally",
		"class", "struct", "enum", "interface", "trait", "impl", "type",
		"var", "let", "const", "static", "extern", "inline", "virtual",
		"public", "private", "protected", "internal",
		"import", "export", "include", "use", "from", "namespace", "package",
		"int", "float", "double", "char", "string", "bool", "void",
		"this", "self", "super", "base",
	}
	
	for _, keyword := range keywords {
		if name == keyword {
			return true
		}
	}
	
	return false
}