package registry

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type CParser struct{}

func (c *CParser) GetExtensions() []string {
	return []string{".c", ".h"}
}

func (c *CParser) IsHeaderFile(filePath string) bool {
	return filepath.Ext(filePath) == ".h"
}

func (c *CParser) ParseFile(filePath string) ([]Function, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var functions []Function
	lines := strings.Split(string(content), "\n")
	
	// More comprehensive C function regex
	fnRegex := regexp.MustCompile(`^\s*(static\s+)?(extern\s+)?(inline\s+)?(\w+(?:\s*\*)*)\s+(\w+)\s*\((.*?)\)\s*[{;]`)
	structRegex := regexp.MustCompile(`^\s*struct\s+(\w+)`)
	preprocessorRegex := regexp.MustCompile(`^\s*#(\w+)`)
	
	var currentStruct string
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip preprocessor directives
		if preprocessorRegex.MatchString(line) {
			continue
		}
		
		// Track struct context
		if structMatch := structRegex.FindStringSubmatch(line); structMatch != nil {
			currentStruct = structMatch[1]
			continue
		}
		
		// Parse function definitions and declarations
		if fnMatch := fnRegex.FindStringSubmatch(line); fnMatch != nil {
			staticMod := strings.TrimSpace(fnMatch[1])
			externMod := strings.TrimSpace(fnMatch[2])
			inlineMod := strings.TrimSpace(fnMatch[3])
			returnType := strings.TrimSpace(fnMatch[4])
			name := fnMatch[5]
			params := fnMatch[6]
			
			// Skip if this looks like a variable declaration
			if strings.Contains(line, "=") && !strings.Contains(line, "{") {
				continue
			}
			
			visibility := "public"
			if staticMod == "static" {
				visibility = "private"
			}
			
			// Determine if it's a declaration or definition
			isDeclaration := strings.HasSuffix(trimmed, ";")
			isDefinition := strings.Contains(line, "{")
			
			paramList := parseCParameters(params)
			comments := extractCComments(lines, i)
			
			fn := Function{
				Name:       name,
				File:       filePath,
				Line:       i + 1,
				Visibility: visibility,
				ReturnType: returnType,
				Parameters: paramList,
				Language:   "c",
				Signature:  strings.TrimSpace(line),
				IsTest:     isCTestFunction(name),
				IsMain:     name == "main",
				Size:       calculateCFunctionSize(lines, i, isDefinition),
				Comments:   comments,
			}
			
			// Set metadata
			fn.Metadata = make(map[string]string)
			if externMod != "" {
				fn.Metadata["extern"] = "true"
			}
			if inlineMod != "" {
				fn.Metadata["inline"] = "true"
			}
			if isDeclaration {
				fn.Metadata["declaration"] = "true"
			}
			if isDefinition {
				fn.Metadata["definition"] = "true"
			}
			if currentStruct != "" {
				fn.Metadata["struct_context"] = currentStruct
			}
			
			functions = append(functions, fn)
		}
		
		// Reset struct context on closing brace
		if strings.Contains(line, "}") && !strings.Contains(line, "{") {
			currentStruct = ""
		}
	}
	
	return functions, nil
}

func (c *CParser) FindFunctionCalls(content string) []string {
	callRegex := regexp.MustCompile(`(\w+)\s*\(`)
	matches := callRegex.FindAllStringSubmatch(content, -1)
	
	var calls []string
	seen := make(map[string]bool)
	
	for _, match := range matches {
		call := match[1]
		if !seen[call] && !isCBuiltin(call) && !isCKeyword(call) {
			calls = append(calls, call)
			seen[call] = true
		}
	}
	
	return calls
}

func parseCParameters(params string) []string {
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
		
		// Handle function pointers and complex types
		if strings.Contains(part, "(") && strings.Contains(part, ")") {
			// Function pointer parameter
			if idx := strings.LastIndex(part, ")"); idx != -1 {
				remaining := part[idx+1:]
				words := strings.Fields(remaining)
				if len(words) > 0 {
					result = append(result, words[0])
				}
			}
			continue
		}
		
		// Regular parameter: type name or type *name
		words := strings.Fields(part)
		if len(words) > 0 {
			// Take the last word as the parameter name
			paramName := words[len(words)-1]
			// Remove pointer asterisks
			paramName = strings.TrimLeft(paramName, "*")
			// Remove array brackets
			if bracketIdx := strings.Index(paramName, "["); bracketIdx != -1 {
				paramName = paramName[:bracketIdx]
			}
			result = append(result, paramName)
		}
	}
	
	return result
}

func extractCComments(lines []string, fnLine int) string {
	var comments []string
	
	// Look for comments above the function
	for i := fnLine - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		
		if strings.HasPrefix(line, "/*") && strings.HasSuffix(line, "*/") {
			// Single line block comment
			comment := strings.TrimSuffix(strings.TrimPrefix(line, "/*"), "*/")
			comments = append([]string{strings.TrimSpace(comment)}, comments...)
		} else if strings.HasPrefix(line, "/*") {
			// Multi-line block comment start
			comment := strings.TrimPrefix(line, "/*")
			comments = append([]string{strings.TrimSpace(comment)}, comments...)
			
			// Continue reading until */
			for j := i + 1; j < len(lines); j++ {
				commentLine := lines[j]
				if strings.Contains(commentLine, "*/") {
					finalPart := strings.Split(commentLine, "*/")[0]
					if strings.TrimSpace(finalPart) != "" {
						comments = append(comments, strings.TrimSpace(finalPart))
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

func calculateCFunctionSize(lines []string, startLine int, isDefinition bool) int {
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

func isCTestFunction(name string) bool {
	return strings.HasPrefix(name, "test_") || 
	       strings.HasSuffix(name, "_test") ||
	       strings.Contains(name, "Test")
}

func isCBuiltin(name string) bool {
	builtins := []string{
		"printf", "scanf", "fprintf", "fscanf", "sprintf", "sscanf",
		"malloc", "calloc", "realloc", "free",
		"strlen", "strcpy", "strncpy", "strcat", "strncat", "strcmp", "strncmp",
		"memcpy", "memmove", "memset", "memcmp",
		"fopen", "fclose", "fread", "fwrite", "fseek", "ftell", "rewind",
		"getchar", "putchar", "gets", "puts", "fgets", "fputs",
		"atoi", "atof", "atol", "strtol", "strtof", "strtod",
		"abs", "labs", "fabs", "ceil", "floor", "sqrt", "pow", "sin", "cos", "tan",
		"exit", "abort", "atexit", "system", "getenv",
		"assert",
	}
	
	for _, builtin := range builtins {
		if name == builtin {
			return true
		}
	}
	
	return false
}

func isCKeyword(name string) bool {
	keywords := []string{
		"if", "else", "while", "for", "do", "switch", "case", "default",
		"break", "continue", "return", "goto",
		"sizeof", "typedef", "struct", "union", "enum",
		"static", "extern", "register", "auto", "volatile", "const",
		"signed", "unsigned", "short", "long",
		"int", "char", "float", "double", "void",
	}
	
	for _, keyword := range keywords {
		if name == keyword {
			return true
		}
	}
	
	return false
}