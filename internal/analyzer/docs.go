package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/vitruves/gop/internal/utils"
)

// DocsOptions contains options for the docs command
type DocsOptions struct {
	// Input/Output options
	InputFile  string   // Path to input file containing list of files to process
	Directory  string   // Root directory to analyze
	Depth      int      // Maximum depth for directory traversal
	OutputFile string   // Path to output file for results
	Languages  []string // Languages to analyze (e.g., "c", "cpp")
	Excludes   []string // Directories or files to exclude

	// Processing options
	Jobs        int  // Number of concurrent jobs for processing
	IncludeCode bool // Whether to include code snippets in documentation

	// Output options
	Short   bool // Whether to use short output format
	Verbose bool // Whether to enable verbose output
}

// DocItem represents a documentation item extracted from code
type DocItem struct {
	Type        string   // Type of item (function, class, struct, enum, etc.)
	Name        string   // Name of the item
	FilePath    string   // Path to the file containing the item
	LineNumber  int      // Line number where the item is defined
	Description string   // Description extracted from comments
	Params      []string // Parameter descriptions
	Returns     string   // Return value description
	Example     string   // Example usage
	Code        string   // Code snippet (if IncludeCode is true)
}

// GenerateDocs generates documentation for the given options
func GenerateDocs(options DocsOptions) {
	startTime := time.Now()

	// Get files to process
	files, err := utils.GetFilesToProcess(options.InputFile, options.Directory, options.Depth, options.Languages, options.Excludes)
	if err != nil {
		fmt.Println(color.RedString("Error getting files to process:"), err)
		return
	}

	if options.Verbose {
		fmt.Printf(color.CyanString("Found %d files to process\n"), len(files))
	}

	// Process files and extract documentation items
	var processedFiles atomic.Int64
	docItems := extractDocItems(files, options, &processedFiles)

	// Calculate elapsed time and processing speed
	elapsedTime := time.Since(startTime)
	processingSpeed := float64(processedFiles.Load()) / elapsedTime.Seconds()

	// Determine if we should output to file or console
	var writer *bufio.Writer
	var outputFile *os.File

	if options.OutputFile != "" {
		// Create output file
		outputFile, err = os.Create(options.OutputFile)
		if err != nil {
			fmt.Println(color.RedString("Error creating output file:"), err)
			return
		}
		defer outputFile.Close()
		writer = bufio.NewWriter(outputFile)
		defer writer.Flush()
	} else {
		// Output to console
		writer = bufio.NewWriter(os.Stdout)
		defer writer.Flush()
	}

	// Write output
	writeDocsOutput(writer, docItems, files, options, elapsedTime, processingSpeed)
}

// extractDocItems extracts documentation items from the given files
func extractDocItems(files []string, options DocsOptions, processedFiles *atomic.Int64) []DocItem {
	var docItems []DocItem
	var mutex sync.Mutex
	var wg sync.WaitGroup

	// Create a channel for files to process
	filesChan := make(chan string, len(files))
	for _, file := range files {
		filesChan <- file
	}
	close(filesChan)

	// Create a progress bar
	progress := utils.NewProgressBar(len(files), "Extracting documentation")
	progress.Start()

	// Create a channel for results
	resultsChan := make(chan []DocItem, options.Jobs)

	// Process files concurrently
	for i := 0; i < options.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for file := range filesChan {
				// Extract documentation items from this file
				fileDocItems := extractFileDocItems(file, options)

				// Send results to the channel
				if len(fileDocItems) > 0 {
					resultsChan <- fileDocItems
				}

				// Update progress
				processedFiles.Add(1)
				progress.Increment()
			}
		}()
	}

	// Start a goroutine to collect results
	go func() {
		for fileDocItems := range resultsChan {
			mutex.Lock()
			docItems = append(docItems, fileDocItems...)
			mutex.Unlock()
		}
	}()

	// Wait for all processing to complete
	wg.Wait()
	close(resultsChan)
	progress.Finish()

	return docItems
}

// extractFileDocItems extracts documentation items from a single file
func extractFileDocItems(filePath string, options DocsOptions) []DocItem {
	var docItems []DocItem

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return docItems
	}

	lines := strings.Split(string(content), "\n")
	
	// Regular expressions for detecting documentation comments and code elements
	docCommentRegex := regexp.MustCompile(`^\s*/\*\*|\s*\*\s*@|\s*///`)
	functionRegex := regexp.MustCompile(`^\s*([\w\*]+\s+)+(\w+)\s*\(([^)]*)\)\s*({|;)`)
	classRegex := regexp.MustCompile(`^\s*(class|struct)\s+(\w+).*$`)
	enumRegex := regexp.MustCompile(`^\s*enum\s+(\w+).*$`)
	
	var currentComment strings.Builder
	var inDocComment bool
	var docItem *DocItem
	
	for i, line := range lines {
		// Check for documentation comment
		if docCommentRegex.MatchString(line) {
			inDocComment = true
			// Clean up the comment line
			cleanLine := strings.TrimSpace(line)
			cleanLine = strings.TrimPrefix(cleanLine, "/**")
			cleanLine = strings.TrimPrefix(cleanLine, "///")
			cleanLine = strings.TrimPrefix(cleanLine, "*")
			cleanLine = strings.TrimSpace(cleanLine)
			
			// Check for special tags
			if strings.HasPrefix(cleanLine, "@param") {
				if docItem != nil {
					docItem.Params = append(docItem.Params, cleanLine[6:])
				}
			} else if strings.HasPrefix(cleanLine, "@return") {
				if docItem != nil {
					docItem.Returns = cleanLine[7:]
				}
			} else if strings.HasPrefix(cleanLine, "@example") {
				if docItem != nil {
					docItem.Example = cleanLine[8:]
				}
			} else if len(cleanLine) > 0 {
				currentComment.WriteString(cleanLine)
				currentComment.WriteString(" ")
			}
		} else if strings.TrimSpace(line) == "*/" {
			// End of doc comment
			inDocComment = false
		} else if !inDocComment {
			// Check for code elements
			if functionMatch := functionRegex.FindStringSubmatch(line); functionMatch != nil {
				// Found a function
				docItem = &DocItem{
					Type:        "Function",
					Name:        functionMatch[2],
					FilePath:    filePath,
					LineNumber:  i + 1,
					Description: currentComment.String(),
				}
				
				// Extract code snippet if requested
				if options.IncludeCode {
					docItem.Code = extractCodeSnippet(lines, i, 10)
				}
				
				docItems = append(docItems, *docItem)
				currentComment.Reset()
				docItem = nil
			} else if classMatch := classRegex.FindStringSubmatch(line); classMatch != nil {
				// Found a class/struct
				docItem = &DocItem{
					Type:        classMatch[1], // "class" or "struct"
					Name:        classMatch[2],
					FilePath:    filePath,
					LineNumber:  i + 1,
					Description: currentComment.String(),
				}
				
				// Extract code snippet if requested
				if options.IncludeCode {
					docItem.Code = extractCodeSnippet(lines, i, 10)
				}
				
				docItems = append(docItems, *docItem)
				currentComment.Reset()
				docItem = nil
			} else if enumMatch := enumRegex.FindStringSubmatch(line); enumMatch != nil {
				// Found an enum
				docItem = &DocItem{
					Type:        "Enum",
					Name:        enumMatch[1],
					FilePath:    filePath,
					LineNumber:  i + 1,
					Description: currentComment.String(),
				}
				
				// Extract code snippet if requested
				if options.IncludeCode {
					docItem.Code = extractCodeSnippet(lines, i, 10)
				}
				
				docItems = append(docItems, *docItem)
				currentComment.Reset()
				docItem = nil
			} else {
				// If we reach a non-empty line that's not a code element, reset the comment
				if len(strings.TrimSpace(line)) > 0 {
					currentComment.Reset()
				}
			}
		}
	}
	
	return docItems
}

// extractCodeSnippet extracts a code snippet from the given lines
func extractCodeSnippet(lines []string, startLine, maxLines int) string {
	var snippet strings.Builder
	
	// Extract up to maxLines lines
	for i := startLine; i < startLine+maxLines && i < len(lines); i++ {
		snippet.WriteString(lines[i])
		snippet.WriteString("\n")
		
		// Stop at the end of a function/class/struct
		if strings.TrimSpace(lines[i]) == "}" {
			break
		}
	}
	
	return snippet.String()
}

// writeDocsOutput writes documentation output to the given writer
func writeDocsOutput(writer *bufio.Writer, docItems []DocItem, files []string, options DocsOptions, elapsedTime time.Duration, processingSpeed float64) {
	// Write header
	fmt.Fprintf(writer, "# API Documentation\n\n")
	fmt.Fprintf(writer, "*Generated on %s*\n\n", time.Now().Format("2006-01-02 15:04:05"))
	
	// Group items by type
	functions := []DocItem{}
	classes := []DocItem{}
	structs := []DocItem{}
	enums := []DocItem{}
	
	for _, item := range docItems {
		switch item.Type {
		case "Function":
			functions = append(functions, item)
		case "class":
			classes = append(classes, item)
		case "struct":
			structs = append(structs, item)
		case "Enum":
			enums = append(enums, item)
		}
	}
	
	// Write table of contents
	fmt.Fprintf(writer, "## Table of Contents\n\n")
	
	if len(functions) > 0 {
		fmt.Fprintf(writer, "- [Functions](#functions)\n")
	}
	
	if len(classes) > 0 {
		fmt.Fprintf(writer, "- [Classes](#classes)\n")
	}
	
	if len(structs) > 0 {
		fmt.Fprintf(writer, "- [Structs](#structs)\n")
	}
	
	if len(enums) > 0 {
		fmt.Fprintf(writer, "- [Enums](#enums)\n")
	}
	
	fmt.Fprintf(writer, "\n")
	
	// Write functions
	if len(functions) > 0 {
		fmt.Fprintf(writer, "## Functions\n\n")
		
		for _, item := range functions {
			fmt.Fprintf(writer, "### %s\n\n", item.Name)
			
			if len(item.Description) > 0 {
				fmt.Fprintf(writer, "%s\n\n", item.Description)
			}
			
			fmt.Fprintf(writer, "**File:** `%s`\n\n", item.FilePath)
			fmt.Fprintf(writer, "**Line:** %d\n\n", item.LineNumber)
			
			if len(item.Params) > 0 {
				fmt.Fprintf(writer, "**Parameters:**\n\n")
				
				for _, param := range item.Params {
					fmt.Fprintf(writer, "- %s\n", param)
				}
				
				fmt.Fprintf(writer, "\n")
			}
			
			if len(item.Returns) > 0 {
				fmt.Fprintf(writer, "**Returns:** %s\n\n", item.Returns)
			}
			
			if len(item.Example) > 0 {
				fmt.Fprintf(writer, "**Example:**\n\n```c\n%s\n```\n\n", item.Example)
			}
			
			if options.IncludeCode && len(item.Code) > 0 {
				fmt.Fprintf(writer, "**Code:**\n\n```c\n%s\n```\n\n", item.Code)
			}
			
			fmt.Fprintf(writer, "---\n\n")
		}
	}
	
	// Write classes
	if len(classes) > 0 {
		fmt.Fprintf(writer, "## Classes\n\n")
		
		for _, item := range classes {
			fmt.Fprintf(writer, "### %s\n\n", item.Name)
			
			if len(item.Description) > 0 {
				fmt.Fprintf(writer, "%s\n\n", item.Description)
			}
			
			fmt.Fprintf(writer, "**File:** `%s`\n\n", item.FilePath)
			fmt.Fprintf(writer, "**Line:** %d\n\n", item.LineNumber)
			
			if options.IncludeCode && len(item.Code) > 0 {
				fmt.Fprintf(writer, "**Code:**\n\n```c\n%s\n```\n\n", item.Code)
			}
			
			fmt.Fprintf(writer, "---\n\n")
		}
	}
	
	// Write structs
	if len(structs) > 0 {
		fmt.Fprintf(writer, "## Structs\n\n")
		
		for _, item := range structs {
			fmt.Fprintf(writer, "### %s\n\n", item.Name)
			
			if len(item.Description) > 0 {
				fmt.Fprintf(writer, "%s\n\n", item.Description)
			}
			
			fmt.Fprintf(writer, "**File:** `%s`\n\n", item.FilePath)
			fmt.Fprintf(writer, "**Line:** %d\n\n", item.LineNumber)
			
			if options.IncludeCode && len(item.Code) > 0 {
				fmt.Fprintf(writer, "**Code:**\n\n```c\n%s\n```\n\n", item.Code)
			}
			
			fmt.Fprintf(writer, "---\n\n")
		}
	}
	
	// Write enums
	if len(enums) > 0 {
		fmt.Fprintf(writer, "## Enums\n\n")
		
		for _, item := range enums {
			fmt.Fprintf(writer, "### %s\n\n", item.Name)
			
			if len(item.Description) > 0 {
				fmt.Fprintf(writer, "%s\n\n", item.Description)
			}
			
			fmt.Fprintf(writer, "**File:** `%s`\n\n", item.FilePath)
			fmt.Fprintf(writer, "**Line:** %d\n\n", item.LineNumber)
			
			if options.IncludeCode && len(item.Code) > 0 {
				fmt.Fprintf(writer, "**Code:**\n\n```c\n%s\n```\n\n", item.Code)
			}
			
			fmt.Fprintf(writer, "---\n\n")
		}
	}
	
	// Write processing info
	fmt.Fprintf(writer, "## Processing Information\n\n")
	fmt.Fprintf(writer, "- Files processed: %d\n", len(files))
	fmt.Fprintf(writer, "- Documentation items extracted: %d\n", len(docItems))
	fmt.Fprintf(writer, "- Processing time: %s\n", elapsedTime.Round(time.Millisecond))
	fmt.Fprintf(writer, "- Processing speed: %.2f files/sec\n", processingSpeed)
}
