package registry

import (
	"os"
	"regexp"
	"strings"
)

type RustParser struct{}

func (r *RustParser) GetExtensions() []string {
	return []string{".rs"}
}

func (r *RustParser) IsHeaderFile(filePath string) bool {
	return false
}

func (r *RustParser) ParseFile(filePath string) ([]Function, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var functions []Function
	lines := strings.Split(string(content), "\n")
	
	fnRegex := regexp.MustCompile(`^\s*(pub\s+)?(unsafe\s+)?(extern\s+"[^"]+"\s+)?(async\s+)?fn\s+(\w+)\s*(<[^>]*>)?\s*\((.*?)\)(?:\s*->\s*([^{]+))?\s*\{`)
	implRegex := regexp.MustCompile(`^\s*impl\s*(<[^>]*>)?\s*(\w+)`)
	traitRegex := regexp.MustCompile(`^\s*(pub\s+)?trait\s+(\w+)`)
	attrRegex := regexp.MustCompile(`^\s*#\[([^\]]+)\]`)
	
	var currentImpl string
	var currentTrait string
	var currentAttributes []string
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Track attributes
		if attrMatch := attrRegex.FindStringSubmatch(line); attrMatch != nil {
			currentAttributes = append(currentAttributes, attrMatch[1])
			continue
		}
		
		// Track impl blocks
		if implMatch := implRegex.FindStringSubmatch(line); implMatch != nil {
			currentImpl = implMatch[2]
			currentTrait = ""
			currentAttributes = nil
			continue
		}
		
		// Track trait definitions
		if traitMatch := traitRegex.FindStringSubmatch(line); traitMatch != nil {
			currentTrait = traitMatch[2]
			currentImpl = ""
			currentAttributes = nil
			continue
		}
		
		// Parse function definitions
		if fnMatch := fnRegex.FindStringSubmatch(line); fnMatch != nil {
			pubMod := strings.TrimSpace(fnMatch[1])
			unsafeMod := strings.TrimSpace(fnMatch[2])
			externMod := strings.TrimSpace(fnMatch[3])
			asyncMod := strings.TrimSpace(fnMatch[4])
			name := fnMatch[5]
			generics := fnMatch[6]
			params := fnMatch[7]
			returnType := strings.TrimSpace(fnMatch[8])
			
			if returnType == "" {
				returnType = "()"
			}
			
			fullName := name
			if currentImpl != "" {
				fullName = currentImpl + "::" + name
			} else if currentTrait != "" {
				fullName = currentTrait + "::" + name
			}
			
			visibility := "private"
			if pubMod == "pub" {
				visibility = "public"
			}
			
			paramList := parseRustParameters(params)
			comments := extractRustComments(lines, i)
			
			fn := Function{
				Name:       fullName,
				File:       filePath,
				Line:       i + 1,
				Visibility: visibility,
				ReturnType: returnType,
				Parameters: paramList,
				Language:   "rust",
				Signature:  strings.TrimSpace(line),
				IsTest:     isRustTestFunction(currentAttributes),
				IsMain:     name == "main",
				Size:       calculateRustFunctionSize(lines, i),
				Comments:   comments,
				Complexity: calculateRustComplexity(lines, i),
			}
			
			// Set metadata
			fn.Metadata = make(map[string]string)
			if asyncMod != "" {
				fn.Metadata["async"] = "true"
			}
			if unsafeMod != "" {
				fn.Metadata["unsafe"] = "true"
			}
			if externMod != "" {
				fn.Metadata["extern"] = "true"
			}
			if generics != "" {
				fn.Metadata["generic"] = "true"
			}
			if len(currentAttributes) > 0 {
				fn.Metadata["attributes"] = strings.Join(currentAttributes, ",")
			}
			
			functions = append(functions, fn)
			currentAttributes = nil
		} else if trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "#") {
			if !strings.Contains(trimmed, "impl") && !strings.Contains(trimmed, "trait") {
				currentAttributes = nil
			}
		}
	}
	
	return functions, nil
}

func (r *RustParser) FindFunctionCalls(content string) []string {
	// Rust function calls and macro invocations
	callRegex := regexp.MustCompile(`(\w+)!\s*\(|(\w+)\s*\(`)
	methodRegex := regexp.MustCompile(`\.(\w+)\s*\(`)
	
	var calls []string
	seen := make(map[string]bool)
	
	matches := callRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		var call string
		if match[1] != "" { // Macro call
			call = match[1]
		} else if match[2] != "" { // Function call
			call = match[2]
		}
		
		if call != "" && !seen[call] && !isRustBuiltin(call) {
			calls = append(calls, call)
			seen[call] = true
		}
	}
	
	// Method calls
	methodMatches := methodRegex.FindAllStringSubmatch(content, -1)
	for _, match := range methodMatches {
		call := match[1]
		if !seen[call] && !isRustBuiltin(call) {
			calls = append(calls, call)
			seen[call] = true
		}
	}
	
	return calls
}

func parseRustParameters(params string) []string {
	if strings.TrimSpace(params) == "" {
		return []string{}
	}
	
	var result []string
	parts := strings.Split(params, ",")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		// Handle self parameters
		if part == "self" || part == "&self" || part == "&mut self" || strings.HasPrefix(part, "mut self") {
			result = append(result, "self")
			continue
		}
		
		// Handle typed parameters: name: type
		if colonIndex := strings.Index(part, ":"); colonIndex != -1 {
			paramName := strings.TrimSpace(part[:colonIndex])
			// Remove mut keyword
			paramName = strings.TrimPrefix(paramName, "mut ")
			result = append(result, paramName)
		} else {
			// Fallback for unusual patterns
			words := strings.Fields(part)
			if len(words) > 0 {
				paramName := words[len(words)-1]
				result = append(result, paramName)
			}
		}
	}
	
	return result
}

func extractRustComments(lines []string, fnLine int) string {
	var comments []string
	
	// Look for documentation comments above the function
	for i := fnLine - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "///") {
			comment := strings.TrimPrefix(line, "///")
			comments = append([]string{strings.TrimSpace(comment)}, comments...)
		} else if strings.HasPrefix(line, "#[") {
			// Skip attributes
			continue
		} else {
			break
		}
	}
	
	return strings.Join(comments, " ")
}

func calculateRustFunctionSize(lines []string, startLine int) int {
	if startLine >= len(lines) {
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

func calculateRustComplexity(lines []string, startLine int) int {
	complexity := 1 // Base complexity
	braceCount := 0
	
	braceCount += strings.Count(lines[startLine], "{") - strings.Count(lines[startLine], "}")
	
	for i := startLine; i < len(lines); i++ {
		line := lines[i]
		braceCount += strings.Count(line, "{") - strings.Count(line, "}")
		
		// Count complexity-increasing constructs
		complexity += strings.Count(line, "if ")
		complexity += strings.Count(line, "else if ")
		complexity += strings.Count(line, "match ")
		complexity += strings.Count(line, "for ")
		complexity += strings.Count(line, "while ")
		complexity += strings.Count(line, "loop ")
		complexity += strings.Count(line, "?") // Error propagation
		
		if braceCount == 0 && i > startLine {
			break
		}
	}
	
	return complexity
}

func isRustTestFunction(attributes []string) bool {
	for _, attr := range attributes {
		if attr == "test" || strings.Contains(attr, "test") {
			return true
		}
	}
	return false
}

func isRustBuiltin(name string) bool {
	builtins := []string{
		"println", "print", "eprintln", "eprint", "panic", "assert", "assert_eq", "assert_ne",
		"format", "write", "writeln", "vec", "Some", "None", "Ok", "Err", "Box", "Rc", "Arc",
		"clone", "copy", "drop", "len", "is_empty", "push", "pop", "insert", "remove",
		"iter", "into_iter", "collect", "map", "filter", "fold", "reduce", "find",
		"unwrap", "expect", "unwrap_or", "unwrap_or_else", "is_some", "is_none", "is_ok", "is_err",
	}
	
	for _, builtin := range builtins {
		if name == builtin {
			return true
		}
	}
	
	return false
}