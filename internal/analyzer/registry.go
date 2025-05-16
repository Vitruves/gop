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
	Short    bool // Whether to use condensed output format
	Relations bool // Whether to include relationships between elements
	Stats     bool // Whether to include statistics in output
	IAOutput  bool // Whether to format output for AI processing (JSON)
	
	// Misc options
	Verbose bool // Whether to enable verbose output
}

// ElementType represents the type of a code element
type ElementType string

// Predefined element types
const (
	TypeFunction   ElementType = "function"
	TypeMethod     ElementType = "method"
	TypeClass      ElementType = "class"
	TypeStruct     ElementType = "struct"
	TypeInterface  ElementType = "interface"
	TypeConstant   ElementType = "constant"
	TypeVariable   ElementType = "variable"
	TypeEnum       ElementType = "enum"
	TypeNamespace  ElementType = "namespace"
	TypeUnknown    ElementType = "unknown"
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
		writeFullRegistry(outputWriter, elements, options)
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
						Visibility:  "",
						Namespace:   "",
						ReturnType:  "",
						Parameters:  []Parameter{},
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
	
	// Build relations if requested
	if options.Relations {
		buildElementRelations(elements)
	}
	
	return elements
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
	
	// Custom sort order: namespace first, then classes/structs, then functions/methods, then others
	sort.Slice(types, func(i, j int) bool {
		// Define priority order
		typePriority := map[string]int{
			"namespace":  1,
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
	
	// Write header with timestamp
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(writer, "# Code Registry\n\n")
	fmt.Fprintf(writer, "*Generated on %s*\n\n", currentTime)
	
	// Write statistics if requested
	if options.Stats {
		fmt.Fprintf(writer, "## Summary\n\n")
		fmt.Fprintf(writer, "| Type | Count |\n|------|-------|\n")
		fmt.Fprintf(writer, "| **Total** | **%d** |\n", len(elements))
		
		for _, t := range types {
			fmt.Fprintf(writer, "| %s | %d |\n", strings.Title(t), len(elementsByType[t]))
		}
		fmt.Fprintf(writer, "\n")
	}
	
	// Write elements by type
	for _, t := range types {
		// Skip types with no elements
		if len(elementsByType[t]) == 0 {
			continue
		}
		
		fmt.Fprintf(writer, "## %s\n\n", strings.Title(t))
		
		// Sort elements by name
		sort.Slice(elementsByType[t], func(i, j int) bool {
			return elementsByType[t][i].Name < elementsByType[t][j].Name
		})
		
		// Group by file for better organization
		if t == "function" || t == "method" || t == "constant" || len(elementsByType[t]) > 20 {
			// For functions, methods, constants, or large groups, use a more compact table format
			fmt.Fprintf(writer, "| Name | Signature | File | Line |\n|------|----------|------|------|\n")
			for _, elem := range elementsByType[t] {
				filePath := filepath.Base(elem.FilePath)
				
				// Extract a clean signature for display
				signature := ""
				if elem.Signature != "" {
					// Extract just the function name and parameters
					sigParts := strings.Split(elem.Signature, "{")
					if len(sigParts) > 0 {
						// Clean up the signature
						sig := strings.TrimSpace(sigParts[0])
						
						// Extract parameters if present
						paramStart := strings.Index(sig, "(")
						if paramStart > 0 {
							// Get everything inside the parentheses
							paramEnd := strings.LastIndex(sig, ")")
							if paramEnd > paramStart {
								signature = sig[paramStart:paramEnd+1]
								// Truncate if too long
								if len(signature) > 40 {
									signature = signature[:37] + "...)"
								}
							}
						}
					}
				}
				
				// Format the name to handle reserved words like 'if'
				formattedName := elem.Name
				
				fmt.Fprintf(writer, "| `%s` | `%s` | %s | %d |\n", 
					formattedName, signature, filePath, elem.LineNumber)
			}
		} else {
			// For other types, use bullet points
			for _, elem := range elementsByType[t] {
				filePath := filepath.Base(elem.FilePath)
				
				// Add signature for structs, classes, etc. if available
				if elem.Signature != "" && len(elem.Signature) < 60 {
					fmt.Fprintf(writer, "- `%s` (%s:%d) - `%s`\n", 
						elem.Name, filePath, elem.LineNumber, 
						strings.TrimSpace(strings.Split(elem.Signature, "\n")[0]))
				} else {
					fmt.Fprintf(writer, "- `%s` (%s:%d)\n", elem.Name, filePath, elem.LineNumber)
				}
			}
		}
		
		fmt.Fprintf(writer, "\n")
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
			"namespace":  1,
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
						params := cleanSig[paramStart+1:paramEnd]
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
