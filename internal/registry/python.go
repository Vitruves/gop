package registry

import (
	"os"
	"regexp"
	"strings"
)

type PythonParser struct{}

func (p *PythonParser) GetExtensions() []string {
	return []string{".py"}
}

func (p *PythonParser) IsHeaderFile(filePath string) bool {
	return false
}

func (p *PythonParser) ParseFile(filePath string) ([]Function, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var functions []Function
	lines := strings.Split(string(content), "\n")

	defRegex := regexp.MustCompile(`^\s*(def|async def)\s+(\w+)\s*\((.*?)\)(?:\s*->\s*([^:]+))?\s*:`)
	classRegex := regexp.MustCompile(`^\s*class\s+(\w+)(?:\s*\([^)]*\))?\s*:`)
	decoratorRegex := regexp.MustCompile(`^\s*@(\w+)`)

	var currentClass string
	var currentDecorators []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track decorators
		if decoratorMatch := decoratorRegex.FindStringSubmatch(line); decoratorMatch != nil {
			currentDecorators = append(currentDecorators, decoratorMatch[1])
			continue
		}

		// Track class context
		if classMatch := classRegex.FindStringSubmatch(line); classMatch != nil {
			currentClass = classMatch[1]
			currentDecorators = nil
			continue
		}

		// Parse function definitions
		if defMatch := defRegex.FindStringSubmatch(line); defMatch != nil {
			fnType := defMatch[1]
			name := defMatch[2]
			params := defMatch[3]
			returnType := defMatch[4]

			if returnType == "" {
				returnType = "None"
			}

			fullName := name
			if currentClass != "" {
				fullName = currentClass + "." + name
			}

			visibility := "public"
			if strings.HasPrefix(name, "_") {
				if strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__") {
					visibility = "magic"
				} else {
					visibility = "private"
				}
			}

			paramList := parsePythonParameters(params)
			comments := extractPythonDocstring(lines, i+1)

			fn := Function{
				Name:       fullName,
				File:       filePath,
				Line:       i + 1,
				Visibility: visibility,
				ReturnType: returnType,
				Parameters: paramList,
				Language:   "python",
				Signature:  strings.TrimSpace(line),
				IsTest:     isTestFunction(name, currentDecorators),
				IsMain:     name == "__main__" || (currentClass == "" && name == "main"),
				Size:       calculatePythonFunctionSize(lines, i),
				Comments:   comments,
			}

			if fnType == "async def" {
				fn.Metadata = map[string]string{"async": "true"}
			}

			if len(currentDecorators) > 0 {
				if fn.Metadata == nil {
					fn.Metadata = make(map[string]string)
				}
				fn.Metadata["decorators"] = strings.Join(currentDecorators, ",")
			}

			functions = append(functions, fn)
			currentDecorators = nil
		} else if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			currentDecorators = nil
		}
	}

	return functions, nil
}

func (p *PythonParser) FindFunctionCalls(content string) []string {
	callRegex := regexp.MustCompile(`(\w+)\s*\(`)
	matches := callRegex.FindAllStringSubmatch(content, -1)

	seen := make(map[string]bool)
	var calls []string

	for _, match := range matches {
		call := match[1]
		if !seen[call] && !isPythonBuiltin(call) {
			calls = append(calls, call)
			seen[call] = true
		}
	}

	return calls
}

func parsePythonParameters(params string) []string {
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

		// Handle type annotations: name: type = default
		if colonIndex := strings.Index(part, ":"); colonIndex != -1 {
			paramName := strings.TrimSpace(part[:colonIndex])
			result = append(result, paramName)
		} else if equalIndex := strings.Index(part, "="); equalIndex != -1 {
			// Handle default values: name = default
			paramName := strings.TrimSpace(part[:equalIndex])
			result = append(result, paramName)
		} else {
			// Simple parameter name
			result = append(result, part)
		}
	}

	return result
}

func extractPythonDocstring(lines []string, startLine int) string {
	if startLine >= len(lines) {
		return ""
	}

	// Look for docstring on the next non-empty line
	for i := startLine; i < len(lines) && i < startLine+3; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}

		// Check for triple-quoted strings
		if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, `'''`) {
			quote := `"""`
			if strings.HasPrefix(trimmed, `'''`) {
				quote = `'''`
			}

			// Single line docstring
			if strings.Count(trimmed, quote) >= 2 {
				content := strings.Trim(trimmed, quote)
				return strings.TrimSpace(content)
			}

			// Multi-line docstring
			var docParts []string
			docParts = append(docParts, strings.TrimPrefix(trimmed, quote))

			for j := i + 1; j < len(lines); j++ {
				line := lines[j]
				if strings.Contains(line, quote) {
					finalPart := strings.Split(line, quote)[0]
					if finalPart != "" {
						docParts = append(docParts, finalPart)
					}
					break
				}
				docParts = append(docParts, line)
			}

			return strings.TrimSpace(strings.Join(docParts, " "))
		}
		break
	}

	return ""
}

func calculatePythonFunctionSize(lines []string, startLine int) int {
	if startLine >= len(lines) {
		return 1
	}

	baseIndent := getIndentLevel(lines[startLine])
	size := 1

	for i := startLine + 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Empty lines don't count for indentation
		if trimmed == "" {
			size++
			continue
		}

		currentIndent := getIndentLevel(line)

		// If we're back to the same or less indentation, function is done
		if currentIndent <= baseIndent {
			break
		}

		size++
	}

	return size
}

func getIndentLevel(line string) int {
	count := 0
	for _, char := range line {
		switch char {
		case ' ':
			count++
		case '\t':
			count += 4
		default:
			break
		}
	}
	return count
}

func isTestFunction(name string, decorators []string) bool {
	// Check function name patterns
	if strings.HasPrefix(name, "test_") || strings.HasSuffix(name, "_test") {
		return true
	}

	// Check decorators
	for _, decorator := range decorators {
		if decorator == "pytest" || decorator == "unittest" || strings.Contains(decorator, "test") {
			return true
		}
	}

	return false
}

func isPythonBuiltin(name string) bool {
	builtins := []string{
		"print", "len", "range", "str", "int", "float", "bool", "list", "dict", "tuple", "set",
		"open", "type", "isinstance", "hasattr", "getattr", "setattr", "delattr",
		"min", "max", "sum", "abs", "round", "sorted", "reversed", "enumerate", "zip",
		"map", "filter", "any", "all", "next", "iter", "super", "property", "staticmethod", "classmethod",
	}

	for _, builtin := range builtins {
		if name == builtin {
			return true
		}
	}

	return false
}
