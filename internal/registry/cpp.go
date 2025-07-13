package registry

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type CppParser struct{}

func (cpp *CppParser) GetExtensions() []string {
	return []string{".cpp", ".cxx", ".cc", ".hpp", ".hxx", ".hh", ".h++", ".c++"}
}

func (cpp *CppParser) IsHeaderFile(filePath string) bool {
	ext := filepath.Ext(filePath)
	headerExts := []string{".hpp", ".hxx", ".hh", ".h++", ".h"}
	for _, headerExt := range headerExts {
		if ext == headerExt {
			return true
		}
	}
	return false
}

func (cpp *CppParser) ParseFile(filePath string) ([]Function, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var functions []Function
	lines := strings.Split(string(content), "\n")
	
	// Comprehensive C++ function regex patterns
	fnRegex := regexp.MustCompile(`^\s*(template\s*<[^>]*>\s*)?(public|private|protected)?\s*:\s*$|^\s*(virtual\s+)?(static\s+)?(inline\s+)?(explicit\s+)?(\w+(?:\s*::\s*\w+)*(?:\s*<[^>]*>)?(?:\s*\*)*)\s+(\w+(?:::\w+)*)\s*\((.*?)\)\s*(const)?\s*(override)?\s*(final)?\s*[{;]`)
	classRegex := regexp.MustCompile(`^\s*(template\s*<[^>]*>\s*)?(class|struct)\s+(\w+)`)
	namespaceRegex := regexp.MustCompile(`^\s*namespace\s+(\w+)`)
	accessRegex := regexp.MustCompile(`^\s*(public|private|protected)\s*:`)
	
	var currentClass string
	var currentNamespace string
	var currentAccess string = "private" // Default for class
	var templateContext string
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Track template context
		if strings.HasPrefix(trimmed, "template") && strings.Contains(trimmed, "<") {
			templateContext = trimmed
			continue
		}
		
		// Track namespace
		if nsMatch := namespaceRegex.FindStringSubmatch(line); nsMatch != nil {
			currentNamespace = nsMatch[1]
			templateContext = ""
			continue
		}
		
		// Track class/struct context
		if classMatch := classRegex.FindStringSubmatch(line); classMatch != nil {
			currentClass = classMatch[3]
			currentAccess = "private"
			if classMatch[2] == "struct" {
				currentAccess = "public"
			}
			templateContext = ""
			continue
		}
		
		// Track access specifiers
		if accessMatch := accessRegex.FindStringSubmatch(line); accessMatch != nil {
			currentAccess = accessMatch[1]
			templateContext = ""
			continue
		}
		
		// Parse function definitions
		if fnMatch := fnRegex.FindStringSubmatch(line); fnMatch != nil {
			// Skip access specifier lines
			if fnMatch[2] != "" && fnMatch[7] == "" {
				currentAccess = fnMatch[2]
				continue
			}
			
			virtualMod := strings.TrimSpace(fnMatch[3])
			staticMod := strings.TrimSpace(fnMatch[4])
			inlineMod := strings.TrimSpace(fnMatch[5])
			explicitMod := strings.TrimSpace(fnMatch[6])
			returnType := strings.TrimSpace(fnMatch[7])
			name := strings.TrimSpace(fnMatch[8])
			params := fnMatch[9]
			constMod := strings.TrimSpace(fnMatch[10])
			overrideMod := strings.TrimSpace(fnMatch[11])
			finalMod := strings.TrimSpace(fnMatch[12])
			
			// Skip obvious non-functions
			if returnType == "" || name == "" {
				continue
			}
			
			// Handle constructors and destructors
			if name == currentClass || name == "~"+currentClass {
				returnType = ""
			}
			
			fullName := name
			if currentClass != "" {
				fullName = currentClass + "::" + name
			}
			if currentNamespace != "" {
				if currentClass != "" {
					fullName = currentNamespace + "::" + currentClass + "::" + name
				} else {
					fullName = currentNamespace + "::" + name
				}
			}
			
			visibility := currentAccess
			if currentClass == "" {
				visibility = "public" // Free functions are public
			}
			
			// Determine if it's a declaration or definition
			isDeclaration := strings.HasSuffix(trimmed, ";")
			isDefinition := strings.Contains(line, "{")
			
			paramList := parseCppParameters(params)
			comments := extractCppComments(lines, i)
			
			fn := Function{
				Name:       fullName,
				File:       filePath,
				Line:       i + 1,
				Visibility: visibility,
				ReturnType: returnType,
				Parameters: paramList,
				Language:   "cpp",
				Signature:  strings.TrimSpace(line),
				IsTest:     isCppTestFunction(name, fullName),
				IsMain:     name == "main",
				Size:       calculateCppFunctionSize(lines, i, isDefinition),
				Comments:   comments,
			}
			
			// Set metadata
			fn.Metadata = make(map[string]string)
			if virtualMod != "" {
				fn.Metadata["virtual"] = "true"
			}
			if staticMod != "" {
				fn.Metadata["static"] = "true"
			}
			if inlineMod != "" {
				fn.Metadata["inline"] = "true"
			}
			if explicitMod != "" {
				fn.Metadata["explicit"] = "true"
			}
			if constMod != "" {
				fn.Metadata["const"] = "true"
			}
			if overrideMod != "" {
				fn.Metadata["override"] = "true"
			}
			if finalMod != "" {
				fn.Metadata["final"] = "true"
			}
			if templateContext != "" {
				fn.Metadata["template"] = "true"
			}
			if isDeclaration {
				fn.Metadata["declaration"] = "true"
			}
			if isDefinition {
				fn.Metadata["definition"] = "true"
			}
			if name == currentClass {
				fn.Metadata["constructor"] = "true"
			}
			if name == "~"+currentClass {
				fn.Metadata["destructor"] = "true"
			}
			
			functions = append(functions, fn)
			templateContext = ""
		} else if trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") {
			// Reset template context on non-template lines
			if !strings.Contains(trimmed, "template") {
				templateContext = ""
			}
		}
		
		// Reset class context on closing brace
		if strings.Contains(line, "}") && !strings.Contains(line, "{") {
			// This is a simplified check - proper parsing would need brace counting
			currentClass = ""
			currentAccess = "private"
		}
	}
	
	return functions, nil
}

func (cpp *CppParser) FindFunctionCalls(content string) []string {
	callRegex := regexp.MustCompile(`(\w+(?:::\w+)*)\s*\(`)
	methodRegex := regexp.MustCompile(`\.(\w+)\s*\(|->(\w+)\s*\(`)
	
	var calls []string
	seen := make(map[string]bool)
	
	// Function calls
	matches := callRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		call := match[1]
		// Remove namespace qualifiers for simplicity
		if idx := strings.LastIndex(call, "::"); idx != -1 {
			call = call[idx+2:]
		}
		
		if !seen[call] && !isCppBuiltin(call) && !isCppKeyword(call) {
			calls = append(calls, call)
			seen[call] = true
		}
	}
	
	// Method calls
	methodMatches := methodRegex.FindAllStringSubmatch(content, -1)
	for _, match := range methodMatches {
		var call string
		if match[1] != "" {
			call = match[1]
		} else if match[2] != "" {
			call = match[2]
		}
		
		if call != "" && !seen[call] && !isCppBuiltin(call) {
			calls = append(calls, call)
			seen[call] = true
		}
	}
	
	return calls
}

func parseCppParameters(params string) []string {
	if strings.TrimSpace(params) == "" || strings.TrimSpace(params) == "void" {
		return []string{}
	}
	
	var result []string
	parts := strings.Split(params, ",")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "void" {
			continue
		}
		
		// Handle default parameters: type name = default
		if equalIndex := strings.Index(part, "="); equalIndex != -1 {
			part = strings.TrimSpace(part[:equalIndex])
		}
		
		// Handle function pointers and complex types
		if strings.Contains(part, "(") && strings.Contains(part, ")") {
			// Function pointer parameter - extract name after the closing paren
			if idx := strings.LastIndex(part, ")"); idx != -1 {
				remaining := part[idx+1:]
				words := strings.Fields(remaining)
				if len(words) > 0 {
					result = append(result, words[0])
				}
			}
			continue
		}
		
		// Regular parameter: type name, const type& name, type* name, etc.
		words := strings.Fields(part)
		if len(words) > 0 {
			// Take the last word as the parameter name
			paramName := words[len(words)-1]
			// Remove reference and pointer symbols
			paramName = strings.TrimLeft(paramName, "*&")
			// Remove array brackets
			if bracketIdx := strings.Index(paramName, "["); bracketIdx != -1 {
				paramName = paramName[:bracketIdx]
			}
			if paramName != "" {
				result = append(result, paramName)
			}
		}
	}
	
	return result
}

func extractCppComments(lines []string, fnLine int) string {
	var comments []string
	
	// Look for comments above the function
	for i := fnLine - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		
		if strings.HasPrefix(line, "///") {
			// Doxygen comment
			comment := strings.TrimPrefix(line, "///")
			comments = append([]string{strings.TrimSpace(comment)}, comments...)
		} else if strings.HasPrefix(line, "/**") && strings.HasSuffix(line, "*/") {
			// Single line Doxygen block comment
			comment := strings.TrimSuffix(strings.TrimPrefix(line, "/**"), "*/")
			comments = append([]string{strings.TrimSpace(comment)}, comments...)
		} else if strings.HasPrefix(line, "/**") {
			// Multi-line Doxygen block comment
			comment := strings.TrimPrefix(line, "/**")
			comments = append([]string{strings.TrimSpace(comment)}, comments...)
			
			// Continue reading until */
			for j := i + 1; j < len(lines); j++ {
				commentLine := lines[j]
				if strings.Contains(commentLine, "*/") {
					finalPart := strings.Split(commentLine, "*/")[0]
					cleanPart := strings.TrimSpace(finalPart)
					cleanPart = strings.TrimPrefix(cleanPart, "*")
					cleanPart = strings.TrimSpace(cleanPart)
					if cleanPart != "" {
						comments = append(comments, cleanPart)
					}
					break
				} else {
					cleanLine := strings.TrimSpace(commentLine)
					cleanLine = strings.TrimPrefix(cleanLine, "*")
					cleanLine = strings.TrimSpace(cleanLine)
					if cleanLine != "" {
						comments = append(comments, cleanLine)
					}
				}
			}
			break
		} else if strings.HasPrefix(line, "//") {
			// Single line comment
			comment := strings.TrimPrefix(line, "//")
			comments = append([]string{strings.TrimSpace(comment)}, comments...)
		} else {
			break
		}
	}
	
	return strings.Join(comments, " ")
}

func calculateCppFunctionSize(lines []string, startLine int, isDefinition bool) int {
	if !isDefinition || startLine >= len(lines) {
		return 1
	}
	
	braceCount := 0
	size := 1
	
	// Count opening braces in the first line
	braceCount += strings.Count(lines[startLine], "{") - strings.Count(lines[startLine], "}")
	
	for i := startLine + 1; i < len(lines); i++ {
		line := lines[i]
		braceCount += strings.Count(line, "{") - strings.Count(line, "}")
		size++
		
		if braceCount == 0 {
			break
		}
	}
	
	return size
}

func isCppTestFunction(name, fullName string) bool {
	testPatterns := []string{"test", "Test", "TEST"}
	
	for _, pattern := range testPatterns {
		if strings.Contains(name, pattern) || strings.Contains(fullName, pattern) {
			return true
		}
	}
	
	return false
}

func isCppBuiltin(name string) bool {
	builtins := []string{
		// C++ standard library
		"cout", "cin", "cerr", "clog", "endl", "flush",
		"string", "vector", "list", "map", "set", "unordered_map", "unordered_set",
		"shared_ptr", "unique_ptr", "weak_ptr", "make_shared", "make_unique",
		"thread", "mutex", "lock_guard", "unique_lock",
		"begin", "end", "size", "empty", "clear", "push_back", "pop_back",
		"insert", "erase", "find", "count", "at", "front", "back",
		// C standard library (inherited)
		"printf", "scanf", "malloc", "free", "strlen", "strcpy", "strcmp",
		"memcpy", "memset", "assert",
	}
	
	for _, builtin := range builtins {
		if name == builtin {
			return true
		}
	}
	
	return false
}

func isCppKeyword(name string) bool {
	keywords := []string{
		// C++ keywords
		"alignas", "alignof", "and", "and_eq", "asm", "auto", "bitand", "bitor",
		"bool", "break", "case", "catch", "char", "char16_t", "char32_t", "class",
		"compl", "concept", "const", "constexpr", "const_cast", "continue",
		"decltype", "default", "delete", "do", "double", "dynamic_cast",
		"else", "enum", "explicit", "export", "extern", "false", "float",
		"for", "friend", "goto", "if", "inline", "int", "long", "mutable",
		"namespace", "new", "noexcept", "not", "not_eq", "nullptr", "operator",
		"or", "or_eq", "private", "protected", "public", "register", "reinterpret_cast",
		"requires", "return", "short", "signed", "sizeof", "static", "static_assert",
		"static_cast", "struct", "switch", "template", "this", "thread_local",
		"throw", "true", "try", "typedef", "typeid", "typename", "union",
		"unsigned", "using", "virtual", "void", "volatile", "wchar_t", "while",
		"xor", "xor_eq", "override", "final",
	}
	
	for _, keyword := range keywords {
		if name == keyword {
			return true
		}
	}
	
	return false
}