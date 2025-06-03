package analyzer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/vitruves/gop/internal/parser"
	"github.com/vitruves/gop/internal/utils"
)

// RegistryOptions contains options for the registry command
type RegistryOptions struct {
	// Input/Output options
	InputFile  string   // Path to input file containing list of files to process
	Directory  string   // Root directory to analyze
	Depth      int      // Maximum depth for directory traversal
	OutputFile string   // Path to output file for results
	Languages  []string // Languages to analyze (e.g., "c", "cpp")
	Excludes   []string // Directories or files to exclude

	// Processing options
	Jobs  int      // Number of concurrent jobs for processing
	Types []string // Types of code elements to extract (e.g., "function", "class")

	// Output format options
	Short     bool   // Whether to use condensed output format
	Relations bool   // Whether to include relationships between elements
	Stats     bool   // Whether to include statistics in output
	IAOutput  bool   // Whether to format output for AI processing (JSON)
	Format    string // Output format: md, json, csv, txt

	// Filtering and sorting options
	SortBy          string // Sort elements by: name, type, file, line
	FilterNamespace string // Filter by specific namespace or class
	MinLineNumber   int    // Minimum line number to include
	MaxLineNumber   int    // Maximum line number to include (0 for no limit)
	HidePrivate     bool   // Hide private and static members
	OnlyPublic      bool   // Show only public members
	NamesOnly       bool   // Show only element names (one per line)

	// Misc options
	Verbose bool // Whether to enable verbose output
}

// ElementType represents the type of a code element
type ElementType string

// Predefined element types
const (
	TypeFunction  ElementType = "function"
	TypeMethod    ElementType = "method"
	TypeClass     ElementType = "class"
	TypeStruct    ElementType = "struct"
	TypeInterface ElementType = "interface"
	TypeConstant  ElementType = "constant"
	TypeVariable  ElementType = "variable"
	TypeEnum      ElementType = "enum"
	TypeNamespace ElementType = "namespace"
	TypeTemplate  ElementType = "template"
	TypeMacro     ElementType = "macro"
	TypeTypedef   ElementType = "typedef"
	TypeUnion     ElementType = "union"
	TypeUnknown   ElementType = "unknown"
)

// CodeElement represents a code element (constant, method, etc.)
type CodeElement struct {
	Type        ElementType // Type of the element (function, class, etc.)
	Name        string      // Name of the element
	Signature   string      // Full signature (for functions/methods)
	FilePath    string      // Path to the file containing the element
	LineNumber  int         // Line number where the element is defined
	Description string      // Description/comment for the element
	Relations   []string    // Related elements (references)
	Visibility  string      // Visibility (public, private, etc.)
	Namespace   string      // Namespace/package the element belongs to
	ReturnType  string      // Return type (for functions/methods)
	Parameters  []Parameter // Parameters (for functions/methods)
}

// Parameter represents a function/method parameter
type Parameter struct {
	Name string // Parameter name
	Type string // Parameter type
}

// CreateRegistry creates a registry of code elements
func CreateRegistry(options RegistryOptions) {
	// Only add extension if output file is specified but doesn't have one
	if options.OutputFile != "" && !strings.HasSuffix(options.OutputFile, ".md") && !strings.HasSuffix(options.OutputFile, ".json") {
		// Add .md extension if not specified
		options.OutputFile = options.OutputFile + ".md"
	}
	if options.Verbose {
		fmt.Println(color.CyanString("Starting registry creation..."))
	}

	// Get list of files to process
	files, err := utils.GetFilesToProcess(options.InputFile, options.Directory, options.Depth, options.Languages, options.Excludes)
	if err != nil {
		fmt.Println(color.RedString("Error:"), err)
		return
	}

	// Get current working directory for debugging
	cwd, _ := os.Getwd()
	fmt.Printf("Current working directory: %s\n", cwd)

	// Always show the number of files found
	fmt.Printf(color.CyanString("Found %d files to process\n"), len(files))

	// Print the first 5 files for debugging
	if len(files) > 0 {
		fmt.Println("First few files found:")
		for i, file := range files {
			if i >= 5 {
				break
			}
			relPath, _ := filepath.Rel(cwd, file)
			fmt.Printf("  %s\n", relPath)
		}
	}

	// If no files found, print a helpful message
	if len(files) == 0 {
		fmt.Println(color.YellowString("No files found to analyze. Please check your directory path and language filters."))
		fmt.Printf("Directory: %s\n", options.Directory)
		fmt.Printf("Languages: %v\n", options.Languages)
		fmt.Printf("Excludes: %v\n", options.Excludes)
		return
	}

	// Process files and extract code elements
	elements := extractCodeElements(files, options)

	// Apply filtering based on options
	elements = applyFilters(elements, options)

	// Apply sorting based on options
	elements = applySorting(elements, options)

	// Determine if we should output to file or console
	var outputWriter io.Writer
	var outputFile *os.File

	if options.OutputFile == "" {
		// Output to console
		outputWriter = os.Stdout
	} else {
		// Create output file
		outputFile, err = os.Create(options.OutputFile)
		if err != nil {
			fmt.Println(color.RedString("Error creating output file:"), err)
			return
		}
		defer outputFile.Close()
		outputWriter = outputFile
	}

	// Write registry
	if options.IAOutput {
		writeIARegistry(outputWriter, elements, options)
	} else if options.Short {
		writeShortRegistry(outputWriter, elements, options)
	} else {
		// Choose output format based on options
		switch strings.ToLower(options.Format) {
		case "json":
			writeJSONRegistry(outputWriter, elements, options)
		case "csv":
			writeCSVRegistry(outputWriter, elements, options)
		case "txt":
			writeTXTRegistry(outputWriter, elements, options)
		case "md", "markdown", "":
			writeFullRegistry(outputWriter, elements, options)
		default:
			fmt.Printf(color.YellowString("Warning: Unknown format '%s', using markdown\n"), options.Format)
			writeFullRegistry(outputWriter, elements, options)
		}
	}

	// No recommendation prints
}

// extractCodeElements extracts code elements from the given files
func extractCodeElements(files []string, options RegistryOptions) []CodeElement {
	var elements []CodeElement
	var mutex sync.Mutex
	var wg sync.WaitGroup

	// Create a channel for files to process
	filesChan := make(chan string, len(files))
	for _, file := range files {
		filesChan <- file
	}
	close(filesChan)

	// Create a progress bar
	progress := utils.NewProgressBar(len(files), "Extracting code elements")
	progress.Start()

	// Process files concurrently
	for i := 0; i < options.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for file := range filesChan {
				// Parse file and extract elements
				fileElements := parser.ExtractCodeElements(file, options.Types)

				// Debug output for each file
				if options.Verbose {
					relPath, _ := filepath.Rel(options.Directory, file)
					fmt.Printf("File %s: found %d code elements\n", relPath, len(fileElements))

					// Show the first few elements for debugging
					for i, elem := range fileElements {
						if i >= 3 { // Show at most 3 elements per file
							break
						}
						fmt.Printf("  - %s: %s (line %d)\n", elem.Type, elem.Name, elem.LineNumber)
					}
				}

				// Add elements to the global list
				// Convert parser.CodeElement to analyzer.CodeElement
				var analyzerElements []CodeElement
				for _, elem := range fileElements {
					// Convert string type to ElementType
					elementType := ElementType(elem.Type)

					analyzerElements = append(analyzerElements, CodeElement{
						Type:        elementType,
						Name:        elem.Name,
						Signature:   elem.Signature,
						FilePath:    file,
						LineNumber:  elem.LineNumber,
						Description: elem.Description,
						// Initialize new fields with empty values
						Visibility: "",
						Namespace:  "",
						ReturnType: "",
						Parameters: []Parameter{},
					})
				}

				mutex.Lock()
				elements = append(elements, analyzerElements...)
				mutex.Unlock()

				progress.Increment()
			}
		}()
	}

	wg.Wait()
	progress.Finish()

	// Deduplicate elements (prioritize implementations over declarations)
	elements = deduplicateElements(elements)

	// Build relations if requested
	if options.Relations {
		buildElementRelations(elements)
	}

	return elements
}

// deduplicateElements removes duplicate code elements, prioritizing implementations over declarations
func deduplicateElements(elements []CodeElement) []CodeElement {
	// Create maps to track elements by normalized signature
	elementMap := make(map[string][]CodeElement)

	// Group elements by normalized signature
	for _, elem := range elements {
		// Create a normalized key for better matching
		key := createNormalizedKey(elem)
		elementMap[key] = append(elementMap[key], elem)
	}

	var deduplicatedElements []CodeElement

	// For each unique signature
	for _, elemList := range elementMap {
		if len(elemList) == 1 {
			// Only one element, keep it
			deduplicatedElements = append(deduplicatedElements, elemList[0])
		} else {
			// Multiple elements with same normalized signature - need to deduplicate
			chosen := chooseBestElement(elemList)
			deduplicatedElements = append(deduplicatedElements, chosen)
		}
	}

	return deduplicatedElements
}

// createNormalizedKey creates a normalized key for element matching
func createNormalizedKey(elem CodeElement) string {
	// For functions and methods, create a key based on name and normalized signature
	if elem.Type == TypeFunction || elem.Type == TypeMethod {
		// Remove extra whitespace and normalize signature
		normalizedSig := strings.TrimSpace(elem.Signature)
		normalizedSig = strings.ReplaceAll(normalizedSig, "  ", " ")
		normalizedSig = strings.ReplaceAll(normalizedSig, "\t", " ")

		// Remove return type for more flexible matching
		// Look for function name pattern: type function_name(params)
		if idx := strings.Index(normalizedSig, elem.Name+"("); idx != -1 {
			// Extract just the function name and parameters part
			funcPart := normalizedSig[idx:]
			if endIdx := strings.Index(funcPart, ")"); endIdx != -1 {
				normalizedSig = funcPart[:endIdx+1]
			}
		}

		return fmt.Sprintf("%s:%s:%s", elem.Type, elem.Name, normalizedSig)
	}

	// For other types, use type:name:signature
	return fmt.Sprintf("%s:%s:%s", elem.Type, elem.Name, elem.Signature)
}

// chooseBestElement chooses the best element from a list of duplicates
func chooseBestElement(elemList []CodeElement) CodeElement {
	if len(elemList) == 1 {
		return elemList[0]
	}

	elementType := elemList[0].Type

	// For functions and methods, prioritize implementation files over header files
	if elementType == TypeFunction || elementType == TypeMethod {
		var implementationFiles []CodeElement
		var headerFiles []CodeElement

		for _, elem := range elemList {
			ext := strings.ToLower(filepath.Ext(elem.FilePath))
			if ext == ".c" || ext == ".cpp" || ext == ".cc" || ext == ".cxx" {
				implementationFiles = append(implementationFiles, elem)
			} else if ext == ".h" || ext == ".hpp" || ext == ".hxx" {
				headerFiles = append(headerFiles, elem)
			}
		}

		// Prefer implementation files
		if len(implementationFiles) > 0 {
			// If multiple implementations, prefer the one with more complete signature or longer description
			return chooseBestFromSimilar(implementationFiles)
		}

		// If no implementations, use header files
		if len(headerFiles) > 0 {
			return chooseBestFromSimilar(headerFiles)
		}

		// Fallback to first element
		return elemList[0]
	} else if elementType == TypeClass || elementType == TypeStruct || elementType == TypeUnion || elementType == TypeEnum {
		// For type definitions, prioritize header files as they're typically the canonical definition
		var headerFiles []CodeElement
		var implementationFiles []CodeElement

		for _, elem := range elemList {
			ext := strings.ToLower(filepath.Ext(elem.FilePath))
			if ext == ".h" || ext == ".hpp" || ext == ".hxx" {
				headerFiles = append(headerFiles, elem)
			} else if ext == ".c" || ext == ".cpp" || ext == ".cc" || ext == ".cxx" {
				implementationFiles = append(implementationFiles, elem)
			}
		}

		// Prefer header files for type definitions
		if len(headerFiles) > 0 {
			return chooseBestFromSimilar(headerFiles)
		}

		// If no headers, use implementation files
		if len(implementationFiles) > 0 {
			return chooseBestFromSimilar(implementationFiles)
		}

		// Fallback to first element
		return elemList[0]
	} else if elementType == TypeConstant || elementType == TypeMacro || elementType == TypeTypedef {
		// For constants, macros, and typedefs, prioritize header files
		return chooseBestByFileType(elemList, true) // true = prefer headers
	} else {
		// For other types (variables, namespaces, templates), prefer implementation files
		return chooseBestByFileType(elemList, false) // false = prefer implementations
	}
}

// chooseBestFromSimilar chooses the best element from a list of similar elements
func chooseBestFromSimilar(elemList []CodeElement) CodeElement {
	if len(elemList) == 1 {
		return elemList[0]
	}

	// Prefer elements with longer, more descriptive signatures or descriptions
	best := elemList[0]
	for _, elem := range elemList[1:] {
		// Prefer element with longer signature (more complete)
		if len(elem.Signature) > len(best.Signature) {
			best = elem
		} else if len(elem.Signature) == len(best.Signature) {
			// If signatures are same length, prefer element with description
			if len(elem.Description) > len(best.Description) {
				best = elem
			}
		}
	}

	return best
}

// chooseBestByFileType chooses the best element based on file type preference
func chooseBestByFileType(elemList []CodeElement, preferHeaders bool) CodeElement {
	var preferredFiles []CodeElement
	var otherFiles []CodeElement

	for _, elem := range elemList {
		ext := strings.ToLower(filepath.Ext(elem.FilePath))
		isHeader := ext == ".h" || ext == ".hpp" || ext == ".hxx"

		if (preferHeaders && isHeader) || (!preferHeaders && !isHeader) {
			preferredFiles = append(preferredFiles, elem)
		} else {
			otherFiles = append(otherFiles, elem)
		}
	}

	// Use preferred files if available
	if len(preferredFiles) > 0 {
		return chooseBestFromSimilar(preferredFiles)
	}

	// Otherwise use other files
	if len(otherFiles) > 0 {
		return chooseBestFromSimilar(otherFiles)
	}

	// Fallback to first element
	return elemList[0]
}

// buildElementRelations builds relationships between code elements
func buildElementRelations(elements []CodeElement) {
	// Map of element names to their indices
	nameToIndex := make(map[string]int)
	for i, elem := range elements {
		nameToIndex[elem.Name] = i
	}

	// For each element, check if it references other elements
	for i, elem := range elements {
		// Check signature and description for references to other elements
		for name, idx := range nameToIndex {
			// Skip self-references
			if name == elem.Name {
				continue
			}

			// Check if this element references the other element
			if strings.Contains(elem.Signature, name) || strings.Contains(elem.Description, name) {
				elements[i].Relations = append(elements[i].Relations, name)
				elements[idx].Relations = append(elements[idx].Relations, elem.Name)
			}
		}

		// Remove duplicates
		elements[i].Relations = utils.RemoveDuplicates(elements[i].Relations)
	}
}

// writeFullRegistry writes a full registry to the output writer
func writeFullRegistry(writer io.Writer, elements []CodeElement, options RegistryOptions) {
	// Group elements by type
	elementsByType := make(map[string][]CodeElement)
	for _, elem := range elements {
		// Convert ElementType to string for map key
		typeStr := string(elem.Type)
		elementsByType[typeStr] = append(elementsByType[typeStr], elem)
	}

	// Get all types
	var types []string
	for t := range elementsByType {
		types = append(types, t)
	}
	sort.Strings(types)

	// Write header with stylized title
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(writer, "# Code Registry\n\n")
	fmt.Fprintf(writer, "Generated by gop registry on %s\n\n", timeStr)

	// Write statistics if requested
	if options.Stats {
		fmt.Fprintf(writer, "## Statistics\n\n")
		fmt.Fprintf(writer, "Total elements: **%d**\n\n", len(elements))

		// Create a table for statistics
		fmt.Fprintf(writer, "| Type | Count | Percentage |\n")
		fmt.Fprintf(writer, "|------|-------|------------|\n")

		for _, t := range types {
			count := len(elementsByType[t])
			percentage := float64(count) / float64(len(elements)) * 100
			fmt.Fprintf(writer, "| %s | %d | %.2f%% |\n",
				strings.Title(t),
				count,
				percentage)
		}

		fmt.Fprintf(writer, "\n")
	}

	// Write elements by type
	for _, t := range types {
		fmt.Fprintf(writer, "## %s\n\n", strings.Title(t))

		// Sort elements by name
		sort.Slice(elementsByType[t], func(i, j int) bool {
			return elementsByType[t][i].Name < elementsByType[t][j].Name
		})

		// Write each element with improved formatting
		for _, elem := range elementsByType[t] {
			// Get relative path for better readability
			relPath, err := filepath.Rel(options.Directory, elem.FilePath)
			if err != nil {
				relPath = elem.FilePath
			}

			fmt.Fprintf(writer, "### %s\n\n", elem.Name)
			fmt.Fprintf(writer, "- **File:** `%s`\n", relPath)
			fmt.Fprintf(writer, "- **Line:** %d\n", elem.LineNumber)
			fmt.Fprintf(writer, "- **Signature:** ```c\n%s\n```\n", elem.Signature)

			if elem.Description != "" {
				// Clean up the description
				description := strings.TrimSpace(elem.Description)
				description = strings.ReplaceAll(description, "/*", "")
				description = strings.ReplaceAll(description, "*/", "")
				description = strings.ReplaceAll(description, "//", "")
				description = strings.TrimSpace(description)

				fmt.Fprintf(writer, "- **Description:** %s\n", description)
			}

			if options.Relations && len(elem.Relations) > 0 {
				// Create a bullet list of relations
				fmt.Fprintf(writer, "- **Relations:**\n")
				for _, rel := range elem.Relations {
					fmt.Fprintf(writer, "  - `%s`\n", rel)
				}
			}

			fmt.Fprintf(writer, "\n---\n\n")
		}
	}
}

// writeShortRegistry writes a condensed registry to the output writer
func writeShortRegistry(writer io.Writer, elements []CodeElement, options RegistryOptions) {
	// Group elements by type
	elementsByType := make(map[string][]CodeElement)
	for _, elem := range elements {
		// Convert ElementType to string for map key
		typeStr := string(elem.Type)
		elementsByType[typeStr] = append(elementsByType[typeStr], elem)
	}

	// Get all types and sort them in a logical order
	var types []string
	for t := range elementsByType {
		types = append(types, t)
	}

	// Custom sort order: functions first, then classes/structs, then others
	sort.Slice(types, func(i, j int) bool {
		// Define priority order
		typePriority := map[string]int{
			"function":  1,
			"method":    2,
			"class":     3,
			"struct":    4,
			"enum":      5,
			"namespace": 6,
			"constant":  7,
			"variable":  8,
			"typedef":   9,
			"union":     10,
			"template":  11,
			"macro":     12,
			"unknown":   13,
		}

		// Get priority for each type, default to 100 if not found
		priI, okI := typePriority[types[i]]
		priJ, okJ := typePriority[types[j]]

		if !okI {
			priI = 100
		}
		if !okJ {
			priJ = 100
		}

		return priI < priJ
	})

	// Write compact output
	for _, t := range types {
		// Skip empty types
		if len(elementsByType[t]) == 0 {
			continue
		}

		// Sort elements by name
		sort.Slice(elementsByType[t], func(i, j int) bool {
			return elementsByType[t][i].Name < elementsByType[t][j].Name
		})

		// Write type header (very compact)
		fmt.Fprintf(writer, "%s:\n", strings.Title(t))

		// Write elements in a very compact format
		for _, elem := range elementsByType[t] {
			fileName := filepath.Base(elem.FilePath)

			// Super compact format: name (file:line)
			fmt.Fprintf(writer, "  %s (%s:%d)\n", elem.Name, fileName, elem.LineNumber)
		}
	}
}

// writeIARegistry writes a registry optimized for AI processing
func writeIARegistry(writer io.Writer, elements []CodeElement, options RegistryOptions) {
	// Group elements by type for better organization
	elementsByType := make(map[string][]CodeElement)
	for _, elem := range elements {
		typeStr := string(elem.Type)
		elementsByType[typeStr] = append(elementsByType[typeStr], elem)
	}

	// Get all types and sort them in a logical order
	var types []string
	for t := range elementsByType {
		types = append(types, t)
	}

	// Custom sort order: namespace first, then classes/structs, then functions/methods, then others
	sort.Slice(types, func(i, j int) bool {
		// Define priority order
		typePriority := map[string]int{
			"namespace": 1,
			"class":     2,
			"struct":    3,
			"interface": 4,
			"enum":      5,
			"function":  6,
			"method":    7,
			"constant":  8,
			"variable":  9,
			"unknown":   10,
		}

		// Get priority for each type, default to 100 if not found
		priI, okI := typePriority[types[i]]
		priJ, okJ := typePriority[types[j]]

		if !okI {
			priI = 100
		}
		if !okJ {
			priJ = 100
		}

		return priI < priJ
	})

	// Start JSON output
	fmt.Fprintf(writer, "{\n")

	// Add metadata
	fmt.Fprintf(writer, "  \"metadata\": {\n")
	fmt.Fprintf(writer, "    \"generated_at\": \"%s\",\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(writer, "    \"total_elements\": %d\n", len(elements))
	fmt.Fprintf(writer, "  },\n")

	// Write elements by type for better organization
	fmt.Fprintf(writer, "  \"code\": {\n")

	for i, t := range types {
		// Skip empty types
		if len(elementsByType[t]) == 0 {
			continue
		}

		// Sort elements by name
		sort.Slice(elementsByType[t], func(i, j int) bool {
			return elementsByType[t][i].Name < elementsByType[t][j].Name
		})

		// Write type header
		fmt.Fprintf(writer, "    \"%s\": [\n", t)

		// Write elements of this type
		for j, elem := range elementsByType[t] {
			fmt.Fprintf(writer, "      {\n")

			// Always include name, file, line
			fmt.Fprintf(writer, "        \"name\": \"%s\",\n", elem.Name)
			fmt.Fprintf(writer, "        \"file\": \"%s\",\n", filepath.Base(elem.FilePath))
			fmt.Fprintf(writer, "        \"line\": %d,\n", elem.LineNumber)

			// Include signature for functions, methods, etc.
			if elem.Signature != "" {
				// Clean up signature - remove extra whitespace and newlines
				cleanSig := strings.TrimSpace(elem.Signature)
				cleanSig = strings.Replace(cleanSig, "\n", " ", -1)
				cleanSig = strings.Replace(cleanSig, "\t", " ", -1)
				// Replace multiple spaces with single space
				for strings.Contains(cleanSig, "  ") {
					cleanSig = strings.Replace(cleanSig, "  ", " ", -1)
				}
				// Escape quotes
				cleanSig = strings.Replace(cleanSig, "\"", "\\\"", -1)
				fmt.Fprintf(writer, "        \"signature\": \"%s\",\n", cleanSig)

				// Extract parameters separately for better usability
				if t == "function" || t == "method" {
					// Extract just the parameters
					paramStart := strings.Index(cleanSig, "(")
					paramEnd := strings.LastIndex(cleanSig, ")")
					if paramStart > 0 && paramEnd > paramStart {
						params := cleanSig[paramStart+1 : paramEnd]
						fmt.Fprintf(writer, "        \"parameters_string\": \"%s\",\n", params)
					}
				}
			}

			// Include return type for functions and methods
			if elem.ReturnType != "" {
				fmt.Fprintf(writer, "        \"return_type\": \"%s\",\n", elem.ReturnType)
			}

			// Include parameters for functions and methods
			if len(elem.Parameters) > 0 {
				fmt.Fprintf(writer, "        \"parameters\": [\n")
				for k, param := range elem.Parameters {
					if k < len(elem.Parameters)-1 {
						fmt.Fprintf(writer, "          {\"name\": \"%s\", \"type\": \"%s\"},\n",
							param.Name, param.Type)
					} else {
						fmt.Fprintf(writer, "          {\"name\": \"%s\", \"type\": \"%s\"}\n",
							param.Name, param.Type)
					}
				}
				fmt.Fprintf(writer, "        ],\n")
			}

			// Include namespace/visibility if available
			if elem.Namespace != "" {
				fmt.Fprintf(writer, "        \"namespace\": \"%s\",\n", elem.Namespace)
			}
			if elem.Visibility != "" {
				fmt.Fprintf(writer, "        \"visibility\": \"%s\",\n", elem.Visibility)
			}

			// Include description if available (cleaned up)
			if elem.Description != "" {
				// Clean up description
				cleanDesc := strings.TrimSpace(elem.Description)
				cleanDesc = strings.Replace(cleanDesc, "\n", " ", -1)
				cleanDesc = strings.Replace(cleanDesc, "\t", " ", -1)
				// Replace multiple spaces with single space
				for strings.Contains(cleanDesc, "  ") {
					cleanDesc = strings.Replace(cleanDesc, "  ", " ", -1)
				}
				// Escape quotes
				cleanDesc = strings.Replace(cleanDesc, "\"", "\\\"", -1)
				fmt.Fprintf(writer, "        \"description\": \"%s\",\n", cleanDesc)
			}

			// Include relations if available and requested
			if options.Relations && len(elem.Relations) > 0 {
				fmt.Fprintf(writer, "        \"relations\": [")
				for k, rel := range elem.Relations {
					if k > 0 {
						fmt.Fprintf(writer, ", ")
					}
					fmt.Fprintf(writer, "\"%s\"", rel)
				}
				fmt.Fprintf(writer, "]\n")
			} else {
				// Remove trailing comma from last property
				fmt.Fprintf(writer, "        \"relations\": []\n")
			}

			// Close element
			if j < len(elementsByType[t])-1 {
				fmt.Fprintf(writer, "      },\n")
			} else {
				fmt.Fprintf(writer, "      }\n")
			}
		}

		// Close type array
		if i < len(types)-1 {
			fmt.Fprintf(writer, "    ],\n")
		} else {
			fmt.Fprintf(writer, "    ]\n")
		}
	}

	// Close code object
	fmt.Fprintf(writer, "  }\n")

	// Add statistics if requested
	if options.Stats {
		fmt.Fprintf(writer, "  ,\"stats\": {\n")

		// Count elements by type
		for i, t := range types {
			if i < len(types)-1 {
				fmt.Fprintf(writer, "    \"%s\": %d,\n", t, len(elementsByType[t]))
			} else {
				fmt.Fprintf(writer, "    \"%s\": %d\n", t, len(elementsByType[t]))
			}
		}

		fmt.Fprintf(writer, "  }\n")
	}

	// Close root object
	fmt.Fprintf(writer, "}\n")
}

// applyFilters applies filtering based on the given options
func applyFilters(elements []CodeElement, options RegistryOptions) []CodeElement {
	// Filter by namespace
	if options.FilterNamespace != "" {
		elements = filterByNamespace(elements, options.FilterNamespace)
	}

	// Filter by line number
	if options.MinLineNumber > 0 || options.MaxLineNumber > 0 {
		elements = filterByLineNumber(elements, options.MinLineNumber, options.MaxLineNumber)
	}

	// Filter by visibility
	if options.HidePrivate && options.OnlyPublic {
		elements = filterByVisibility(elements, "public")
	} else if options.HidePrivate {
		elements = filterByVisibility(elements, "private")
	} else if options.OnlyPublic {
		elements = filterByVisibility(elements, "public")
	}

	// Filter by name
	if options.NamesOnly {
		elements = filterByName(elements, options.NamesOnly)
	}

	return elements
}

// applySorting applies sorting based on the given options
func applySorting(elements []CodeElement, options RegistryOptions) []CodeElement {
	// Sort by type
	if options.SortBy == "type" {
		sort.Slice(elements, func(i, j int) bool {
			return elements[i].Type < elements[j].Type
		})
	}

	// Sort by name
	if options.SortBy == "name" {
		sort.Slice(elements, func(i, j int) bool {
			return elements[i].Name < elements[j].Name
		})
	}

	// Sort by file
	if options.SortBy == "file" {
		sort.Slice(elements, func(i, j int) bool {
			return elements[i].FilePath < elements[j].FilePath
		})
	}

	// Sort by line
	if options.SortBy == "line" {
		sort.Slice(elements, func(i, j int) bool {
			return elements[i].LineNumber < elements[j].LineNumber
		})
	}

	return elements
}

// filterByNamespace filters elements by namespace
func filterByNamespace(elements []CodeElement, namespace string) []CodeElement {
	var filteredElements []CodeElement
	for _, elem := range elements {
		if strings.Contains(elem.Namespace, namespace) {
			filteredElements = append(filteredElements, elem)
		}
	}
	return filteredElements
}

// filterByLineNumber filters elements by line number
func filterByLineNumber(elements []CodeElement, minLine, maxLine int) []CodeElement {
	var filteredElements []CodeElement
	for _, elem := range elements {
		if elem.LineNumber >= minLine && elem.LineNumber <= maxLine {
			filteredElements = append(filteredElements, elem)
		}
	}
	return filteredElements
}

// filterByVisibility filters elements by visibility
func filterByVisibility(elements []CodeElement, visibility string) []CodeElement {
	var filteredElements []CodeElement
	for _, elem := range elements {
		if elem.Visibility == visibility {
			filteredElements = append(filteredElements, elem)
		}
	}
	return filteredElements
}

// filterByName filters elements by name
func filterByName(elements []CodeElement, namesOnly bool) []CodeElement {
	var filteredElements []CodeElement
	for _, elem := range elements {
		if namesOnly {
			fmt.Println(elem.Name)
		} else {
			filteredElements = append(filteredElements, elem)
		}
	}
	return filteredElements
}

// writeJSONRegistry writes a registry in JSON format
func writeJSONRegistry(writer io.Writer, elements []CodeElement, options RegistryOptions) {
	// Simple JSON output without metadata
	fmt.Fprintf(writer, "[\n")

	for i, elem := range elements {
		fmt.Fprintf(writer, "  {\n")
		fmt.Fprintf(writer, "    \"type\": \"%s\",\n", elem.Type)
		fmt.Fprintf(writer, "    \"name\": \"%s\",\n", elem.Name)
		fmt.Fprintf(writer, "    \"file\": \"%s\",\n", filepath.Base(elem.FilePath))
		fmt.Fprintf(writer, "    \"line\": %d,\n", elem.LineNumber)

		if elem.Signature != "" {
			// Clean up signature - escape quotes and newlines
			cleanSig := strings.ReplaceAll(elem.Signature, "\"", "\\\"")
			cleanSig = strings.ReplaceAll(cleanSig, "\n", "\\n")
			fmt.Fprintf(writer, "    \"signature\": \"%s\",\n", cleanSig)
		}

		if elem.Namespace != "" {
			fmt.Fprintf(writer, "    \"namespace\": \"%s\",\n", elem.Namespace)
		}

		fmt.Fprintf(writer, "    \"visibility\": \"%s\"\n", elem.Visibility)

		if i < len(elements)-1 {
			fmt.Fprintf(writer, "  },\n")
		} else {
			fmt.Fprintf(writer, "  }\n")
		}
	}

	fmt.Fprintf(writer, "]\n")
}

// writeCSVRegistry writes a registry in CSV format
func writeCSVRegistry(writer io.Writer, elements []CodeElement, options RegistryOptions) {
	// Write CSV header
	fmt.Fprintf(writer, "Type,Name,File,Line,Signature,Namespace,Visibility\n")

	for _, elem := range elements {
		// Escape commas and quotes in fields
		name := strings.ReplaceAll(elem.Name, "\"", "\"\"")
		if strings.Contains(name, ",") || strings.Contains(name, "\"") {
			name = "\"" + name + "\""
		}

		signature := strings.ReplaceAll(elem.Signature, "\"", "\"\"")
		signature = strings.ReplaceAll(signature, "\n", " ")
		if strings.Contains(signature, ",") || strings.Contains(signature, "\"") {
			signature = "\"" + signature + "\""
		}

		namespace := strings.ReplaceAll(elem.Namespace, "\"", "\"\"")
		if strings.Contains(namespace, ",") || strings.Contains(namespace, "\"") {
			namespace = "\"" + namespace + "\""
		}

		fmt.Fprintf(writer, "%s,%s,%s,%d,%s,%s,%s\n",
			elem.Type,
			name,
			filepath.Base(elem.FilePath),
			elem.LineNumber,
			signature,
			namespace,
			elem.Visibility)
	}
}

// writeTXTRegistry writes a registry in plain text format
func writeTXTRegistry(writer io.Writer, elements []CodeElement, options RegistryOptions) {
	// Group elements by type for better organization
	elementsByType := make(map[string][]CodeElement)
	for _, elem := range elements {
		typeStr := string(elem.Type)
		elementsByType[typeStr] = append(elementsByType[typeStr], elem)
	}

	// Get all types and sort them
	var types []string
	for t := range elementsByType {
		types = append(types, t)
	}
	sort.Strings(types)

	// Write header
	fmt.Fprintf(writer, "CODE REGISTRY\n")
	fmt.Fprintf(writer, "=============\n\n")

	// Write statistics if requested
	if options.Stats {
		fmt.Fprintf(writer, "STATISTICS\n")
		fmt.Fprintf(writer, "----------\n")
		fmt.Fprintf(writer, "Total elements: %d\n\n", len(elements))

		for _, t := range types {
			count := len(elementsByType[t])
			percentage := float64(count) / float64(len(elements)) * 100
			fmt.Fprintf(writer, "%-12s: %4d (%.2f%%)\n", strings.Title(t), count, percentage)
		}
		fmt.Fprintf(writer, "\n")
	}

	// Write elements by type
	for _, t := range types {
		if len(elementsByType[t]) == 0 {
			continue
		}

		fmt.Fprintf(writer, "%s\n", strings.ToUpper(t))
		fmt.Fprintf(writer, "%s\n", strings.Repeat("-", len(t)))

		// Sort elements by name
		sort.Slice(elementsByType[t], func(i, j int) bool {
			return elementsByType[t][i].Name < elementsByType[t][j].Name
		})

		for _, elem := range elementsByType[t] {
			fmt.Fprintf(writer, "  %s", elem.Name)

			if elem.Signature != "" && len(elem.Signature) < 80 {
				fmt.Fprintf(writer, " - %s", strings.TrimSpace(strings.Split(elem.Signature, "\n")[0]))
			}

			fmt.Fprintf(writer, " (%s:%d)", filepath.Base(elem.FilePath), elem.LineNumber)

			if elem.Namespace != "" {
				fmt.Fprintf(writer, " [%s]", elem.Namespace)
			}

			fmt.Fprintf(writer, "\n")
		}

		fmt.Fprintf(writer, "\n")
	}
}
