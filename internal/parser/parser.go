package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ElementType represents the type of code element
type ElementType string

// Element types
const (
	TypeFunction  ElementType = "function"
	TypeMethod    ElementType = "method"
	TypeClass     ElementType = "class"
	TypeStruct    ElementType = "struct"
	TypeEnum      ElementType = "enum"
	TypeConstant  ElementType = "constant"
	TypeVariable  ElementType = "variable"
	TypeNamespace ElementType = "namespace"
	TypeTemplate  ElementType = "template"
	TypeMacro     ElementType = "macro"
	TypeTypedef   ElementType = "typedef"
	TypeUnion     ElementType = "union"
	TypeUnknown   ElementType = "unknown"
)

// CodeElement represents a code element extracted from a file
type CodeElement struct {
	Type        ElementType // Type of the element (function, class, etc.)
	Name        string      // Name of the element
	Signature   string      // Full signature or declaration
	LineNumber  int         // Line number in the file
	Description string      // Documentation or comments
	Namespace   string      // Namespace or class containing this element
	Parameters  []string    // Function/method parameters
	ReturnType  string      // Return type for functions/methods
	Visibility  string      // Public, private, protected
	IsStatic    bool        // Whether the element is static
	IsConst     bool        // Whether the element is const
	IsInline    bool        // Whether the element is inline
	IsTemplate  bool        // Whether the element is a template
	IsMacro     bool        // Whether the element is a macro
}

// ParseOptions contains options for parsing code elements
type ParseOptions struct {
	ExtractAll      bool
	ExtractTypes    map[string]bool
	IncludeComments bool
	ParseParameters bool
	ParseNamespaces bool
	Verbose         bool
}

// NewParseOptions creates a new ParseOptions from a list of types
func NewParseOptions(types []string) ParseOptions {
	options := ParseOptions{
		ExtractTypes:    make(map[string]bool),
		IncludeComments: true,
		ParseParameters: true,
		ParseNamespaces: true,
	}

	// Check if we should extract all types
	for _, t := range types {
		if t == "all" {
			options.ExtractAll = true
			break
		}
		options.ExtractTypes[t] = true
	}

	return options
}

// ExtractCodeElements extracts code elements from a file
func ExtractCodeElements(filePath string, types []string) []CodeElement {
	// Create parse options
	options := NewParseOptions(types)

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	// Parse the file content
	return ParseCodeContent(string(content), options)
}

// ParseCodeContent parses code content and extracts code elements
func ParseCodeContent(content string, options ParseOptions) []CodeElement {
	var elements []CodeElement

	// Split content into lines
	lines := strings.Split(content, "\n")

	// Current state
	var currentNamespace string
	var currentClass string
	var commentBlock strings.Builder
	var inCommentBlock bool

	// Regular expressions for different code elements
	regexes := getRegexPatterns()

	// Process each line
	for lineNumber, line := range lines {
		// Handle comment blocks
		if strings.HasPrefix(strings.TrimSpace(line), "/*") {
			inCommentBlock = true
			if options.IncludeComments {
				commentBlock.WriteString(line + "\n")
			}
			continue
		}

		if inCommentBlock {
			if options.IncludeComments {
				commentBlock.WriteString(line + "\n")
			}
			if strings.Contains(line, "*/") {
				inCommentBlock = false
			}
			continue
		}

		// Handle single-line comments
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			if options.IncludeComments {
				commentBlock.WriteString(line + "\n")
			}
			continue
		}

		// Extract namespace
		if options.ParseNamespaces {
			if matches := regexes.namespaceRegex.FindStringSubmatch(line); len(matches) > 1 {
				currentNamespace = matches[1]
				if options.ExtractAll || options.ExtractTypes["namespaces"] {
					elements = append(elements, CodeElement{
						Type:        TypeNamespace,
						Name:        currentNamespace,
						Signature:   strings.TrimSpace(line),
						LineNumber:  lineNumber + 1,
						Description: commentBlock.String(),
					})
				}
				commentBlock.Reset()
				continue
			}
		}

		// Extract class or struct
		if options.ExtractAll || options.ExtractTypes["classes"] || options.ExtractTypes["structs"] {
			if matches := regexes.classRegex.FindStringSubmatch(line); len(matches) > 1 {
				classType := TypeClass
				if matches[1] == "struct" {
					classType = TypeStruct
				}
				currentClass = matches[2]
				elements = append(elements, CodeElement{
					Type:        classType,
					Name:        currentClass,
					Signature:   strings.TrimSpace(line),
					LineNumber:  lineNumber + 1,
					Description: commentBlock.String(),
					Namespace:   currentNamespace,
				})
				commentBlock.Reset()
				continue
			}
		}

		// Extract constants (defines, const variables)
		if options.ExtractAll || options.ExtractTypes["constants"] {
			// Check for #define
			if matches := regexes.defineRegex.FindStringSubmatch(line); len(matches) > 1 {
				elements = append(elements, CodeElement{
					Type:        TypeConstant,
					Name:        matches[1],
					Signature:   strings.TrimSpace(line),
					LineNumber:  lineNumber + 1,
					Description: commentBlock.String(),
					Namespace:   currentNamespace,
					IsMacro:     true,
				})
				commentBlock.Reset()
				continue
			}

			// Check for const variables
			if matches := regexes.constVarRegex.FindStringSubmatch(line); len(matches) > 1 {
				elements = append(elements, CodeElement{
					Type:        TypeConstant,
					Name:        matches[2],
					Signature:   strings.TrimSpace(line),
					LineNumber:  lineNumber + 1,
					Description: commentBlock.String(),
					Namespace:   currentNamespace,
					IsConst:     true,
					ReturnType:  matches[1],
				})
				commentBlock.Reset()
				continue
			}
		}

		// Extract enums
		if options.ExtractAll || options.ExtractTypes["enums"] {
			if matches := regexes.enumRegex.FindStringSubmatch(line); len(matches) > 1 {
				elements = append(elements, CodeElement{
					Type:        TypeEnum,
					Name:        matches[1],
					Signature:   strings.TrimSpace(line),
					LineNumber:  lineNumber + 1,
					Description: commentBlock.String(),
					Namespace:   currentNamespace,
				})
				commentBlock.Reset()
				continue
			}
		}

		// Extract methods (class member functions)
		if options.ExtractAll || options.ExtractTypes["methods"] {
			if matches := regexes.methodRegex.FindStringSubmatch(line); len(matches) > 1 {
				// Extract return type, class name, and method name
				returnType := matches[1]
				// Use class name for namespace if no namespace is set
				if currentNamespace == "" {
					currentNamespace = matches[2]
				}
				methodName := matches[3]

				// Extract parameters if requested
				var parameters []string
				if options.ParseParameters && len(matches) > 4 && matches[4] != "" {
					paramStr := matches[4]
					paramList := strings.Split(paramStr, ",")
					for _, p := range paramList {
						parameters = append(parameters, strings.TrimSpace(p))
					}
				}

				// Determine visibility and other modifiers
				visibility := "public"
				isStatic := false
				isConst := false

				if strings.Contains(line, "static") {
					isStatic = true
				}
				if strings.Contains(line, "const") {
					isConst = true
				}

				// Create the element
				elements = append(elements, CodeElement{
					Type:        TypeMethod,
					Name:        methodName,
					Signature:   strings.TrimSpace(line),
					LineNumber:  lineNumber + 1,
					Description: commentBlock.String(),
					Namespace:   currentNamespace,
					Parameters:  parameters,
					ReturnType:  returnType,
					Visibility:  visibility,
					IsStatic:    isStatic,
					IsConst:     isConst,
				})

				commentBlock.Reset()
				continue
			}
		}

		// Extract functions (non-class member functions)
		if options.ExtractAll || options.ExtractTypes["functions"] {
			if matches := regexes.functionRegex.FindStringSubmatch(line); len(matches) > 1 {
				// Extract return type and function name
				returnType := matches[1]
				functionName := matches[2]

				// Extract parameters if requested
				var parameters []string
				if options.ParseParameters && len(matches) > 3 && matches[3] != "" {
					paramStr := matches[3]
					paramList := strings.Split(paramStr, ",")
					for _, p := range paramList {
						parameters = append(parameters, strings.TrimSpace(p))
					}
				}

				// Determine modifiers
				isStatic := false
				isInline := false

				if strings.Contains(line, "static") {
					isStatic = true
				}
				if strings.Contains(line, "inline") {
					isInline = true
				}

				// Create the element
				elements = append(elements, CodeElement{
					Type:        TypeFunction,
					Name:        functionName,
					Signature:   strings.TrimSpace(line),
					LineNumber:  lineNumber + 1,
					Description: commentBlock.String(),
					Namespace:   currentNamespace,
					Parameters:  parameters,
					ReturnType:  returnType,
					IsStatic:    isStatic,
					IsInline:    isInline,
				})

				commentBlock.Reset()
				continue
			}
		}

		// Extract templates
		if options.ExtractAll || options.ExtractTypes["templates"] {
			if matches := regexes.templateRegex.FindStringSubmatch(line); len(matches) > 1 {
				templateName := ""
				if len(matches) > 3 && matches[3] != "" {
					templateName = matches[3]
				} else if len(matches) > 5 && matches[5] != "" {
					templateName = matches[5]
				}

				elements = append(elements, CodeElement{
					Type:        TypeTemplate,
					Name:        templateName,
					Signature:   strings.TrimSpace(line),
					LineNumber:  lineNumber + 1,
					Description: commentBlock.String(),
					Namespace:   currentNamespace,
					IsTemplate:  true,
				})
				commentBlock.Reset()
				continue
			}
		}

		// Extract typedefs
		if options.ExtractAll || options.ExtractTypes["typedefs"] {
			if matches := regexes.typedefRegex.FindStringSubmatch(line); len(matches) > 2 {
				elements = append(elements, CodeElement{
					Type:        TypeTypedef,
					Name:        matches[2],
					Signature:   strings.TrimSpace(line),
					LineNumber:  lineNumber + 1,
					Description: commentBlock.String(),
					Namespace:   currentNamespace,
					ReturnType:  matches[1], // Original type
				})
				commentBlock.Reset()
				continue
			}
		}

		// Extract unions
		if options.ExtractAll || options.ExtractTypes["unions"] {
			if matches := regexes.unionRegex.FindStringSubmatch(line); len(matches) > 1 {
				elements = append(elements, CodeElement{
					Type:        TypeUnion,
					Name:        matches[1],
					Signature:   strings.TrimSpace(line),
					LineNumber:  lineNumber + 1,
					Description: commentBlock.String(),
					Namespace:   currentNamespace,
				})
				commentBlock.Reset()
				continue
			}
		}

		// Extract variables (global/static variables)
		if options.ExtractAll || options.ExtractTypes["variables"] {
			// Skip if it looks like a function declaration
			if !strings.Contains(line, "(") {
				if matches := regexes.variableRegex.FindStringSubmatch(line); len(matches) > 2 {
					// Skip common keywords that aren't variable declarations
					varName := matches[2]
					if varName != "main" && varName != "if" && varName != "for" && varName != "while" {
						isStatic := strings.Contains(line, "static")
						isExtern := strings.Contains(line, "extern")

						elements = append(elements, CodeElement{
							Type:        TypeVariable,
							Name:        varName,
							Signature:   strings.TrimSpace(line),
							LineNumber:  lineNumber + 1,
							Description: commentBlock.String(),
							Namespace:   currentNamespace,
							ReturnType:  matches[1],
							IsStatic:    isStatic,
							Visibility: func() string {
								if isExtern {
									return "extern"
								} else if isStatic {
									return "static"
								}
								return "global"
							}(),
						})
						commentBlock.Reset()
						continue
					}
				}
			}
		}

		// Extract function-like macros
		if options.ExtractAll || options.ExtractTypes["macros"] {
			if matches := regexes.macroRegex.FindStringSubmatch(line); len(matches) > 2 {
				// Extract parameters
				var parameters []string
				if len(matches) > 2 && matches[2] != "" {
					paramStr := matches[2]
					paramList := strings.Split(paramStr, ",")
					for _, p := range paramList {
						parameters = append(parameters, strings.TrimSpace(p))
					}
				}

				elements = append(elements, CodeElement{
					Type:        TypeMacro,
					Name:        matches[1],
					Signature:   strings.TrimSpace(line),
					LineNumber:  lineNumber + 1,
					Description: commentBlock.String(),
					Namespace:   currentNamespace,
					Parameters:  parameters,
					IsMacro:     true,
				})
				commentBlock.Reset()
				continue
			}
		}

		// Reset comment block if we didn't find any code element
		commentBlock.Reset()
	}

	return elements
}

// RegexPatterns contains all regex patterns for code parsing
type RegexPatterns struct {
	namespaceRegex *regexp.Regexp
	classRegex     *regexp.Regexp
	defineRegex    *regexp.Regexp
	constVarRegex  *regexp.Regexp
	enumRegex      *regexp.Regexp
	methodRegex    *regexp.Regexp
	functionRegex  *regexp.Regexp
	templateRegex  *regexp.Regexp
	typedefRegex   *regexp.Regexp
	unionRegex     *regexp.Regexp
	variableRegex  *regexp.Regexp
	macroRegex     *regexp.Regexp
}

// getRegexPatterns returns compiled regex patterns for code parsing
func getRegexPatterns() RegexPatterns {
	return RegexPatterns{
		// Namespace pattern - improved to handle nested namespaces
		namespaceRegex: regexp.MustCompile(`^\s*namespace\s+([a-zA-Z_][a-zA-Z0-9_]*(?:::[a-zA-Z_][a-zA-Z0-9_]*)*)\s*\{`),

		// Class/struct pattern - improved to handle inheritance and templates
		classRegex: regexp.MustCompile(`^\s*(class|struct)\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*(?:<[^>]*>)?\s*(?:\s*:\s*(?:public|private|protected)?\s*[a-zA-Z_][a-zA-Z0-9_:]*(?:\s*,\s*(?:public|private|protected)?\s*[a-zA-Z_][a-zA-Z0-9_:]*)*)?`),

		// Define pattern - improved to handle multi-line defines and function-like macros
		defineRegex: regexp.MustCompile(`^\s*#define\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*(?:\([^)]*\))?\s+(.*?)(?:\\s*$|$)`),

		// Const variable pattern - improved to handle pointers and references
		constVarRegex: regexp.MustCompile(`^\s*(?:static\s+)?const\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\s*[*&]+\s*)*)\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*=`),

		// Enum pattern - improved to handle enum classes and typed enums
		enumRegex: regexp.MustCompile(`^\s*enum\s*(?:class\s+)?([a-zA-Z_][a-zA-Z0-9_]*)\s*(?:\s*:\s*[a-zA-Z_][a-zA-Z0-9_]*)?`),

		// Method pattern - improved to handle const, static, virtual, and templates
		methodRegex: regexp.MustCompile(`^\s*(?:virtual\s+|static\s+|inline\s+)*([a-zA-Z_][a-zA-Z0-9_]*(?:\s*[*&]+\s*)*)\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*::\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\(([^)]*)\)\s*(?:const)?\s*(?:override)?\s*(?:final)?\s*(?:\{|;)`),

		// Function pattern - improved to handle storage classes and attributes
		functionRegex: regexp.MustCompile(`^\s*(?:extern\s+|static\s+|inline\s+)*([a-zA-Z_][a-zA-Z0-9_]*(?:\s*[*&]+\s*)*)\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(([^)]*)\)\s*(?:\{|;)`),

		// Template pattern - improved to handle complex template declarations
		templateRegex: regexp.MustCompile(`^\s*template\s*<([^>]*)>\s*(?:(class|struct|typename)\s+([a-zA-Z_][a-zA-Z0-9_]*)|([a-zA-Z_][a-zA-Z0-9_]*(?:\s*[*&]+\s*)*)\s+([a-zA-Z_][a-zA-Z0-9_]*))`),

		// Add new patterns for additional element types
		typedefRegex:  regexp.MustCompile(`^\s*typedef\s+(.+?)\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*;`),
		unionRegex:    regexp.MustCompile(`^\s*union\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\{`),
		variableRegex: regexp.MustCompile(`^\s*(?:extern\s+|static\s+)?([a-zA-Z_][a-zA-Z0-9_]*(?:\s*[*&]+\s*)*)\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*(?:=.*)?;`),
		macroRegex:    regexp.MustCompile(`^\s*#define\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(([^)]*)\)\s+(.*?)(?:\\s*$|$)`),
	}
}

// ExtractDeclarations extracts function/method declarations from a file
func ExtractDeclarations(filePath string) []CodeElement {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	// Create parse options focused on functions and methods
	options := ParseOptions{
		ExtractTypes: map[string]bool{
			"functions": true,
			"methods":   true,
		},
		IncludeComments: false,
		ParseParameters: true,
		ParseNamespaces: true,
	}

	// Parse the content
	elements := ParseCodeContent(string(content), options)

	// Filter only function and method declarations
	var declarations []CodeElement
	for _, elem := range elements {
		if elem.Type == TypeFunction || elem.Type == TypeMethod {
			declarations = append(declarations, elem)
		}
	}

	return declarations
}

// FindImplementations finds implementations for a given declaration
func FindImplementations(declaration CodeElement, files []string) []CodeElement {
	var implementations []CodeElement

	// Create regex to find implementations
	pattern := fmt.Sprintf(`\b%s\b.*\([^)]*\)\s*{`, regexp.QuoteMeta(declaration.Name))
	implRegex := regexp.MustCompile(pattern)

	// Search each file for implementations
	for _, file := range files {
		// Skip header files
		if isHeaderFile(file) {
			continue
		}

		// Read file content
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Find matches
		matches := implRegex.FindAllStringIndex(string(content), -1)
		if len(matches) > 0 {
			// Parse the file to get full implementation details
			elements := ExtractCodeElements(file, []string{"functions", "methods"})

			// Find matching elements
			for _, elem := range elements {
				if elem.Name == declaration.Name {
					implementations = append(implementations, elem)
				}
			}
		}
	}

	return implementations
}

// isHeaderFile checks if a file is a C/C++ header file
func isHeaderFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".h" || ext == ".hpp" || ext == ".hxx"
}
