package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/vitruves/gop/internal/utils"
)

// IncludeGraphOptions contains options for the include-graph command
type IncludeGraphOptions struct {
	// Input/Output options
	InputFile  string   // Path to input file containing list of files to process
	Directory  string   // Root directory to analyze
	Depth      int      // Maximum depth for directory traversal
	OutputFile string   // Path to output file for results
	Languages  []string // Languages to analyze (e.g., "c", "cpp")
	Excludes   []string // Directories or files to exclude
	
	// Processing options
	Jobs int // Number of concurrent jobs for processing
	
	// Output options
	Format  string // Output format (dot, json, md)
	Short   bool   // Whether to use short output format
	Verbose bool   // Whether to enable verbose output
}

// IncludeRelation represents a relationship between files via includes
type IncludeRelation struct {
	SourceFile  string // File that contains the include statement
	IncludedFile string // File that is included
	Line        int    // Line number where the include appears
	IsSystem    bool   // Whether it's a system include (<>) or local include ("")
}

// GenerateIncludeGraph generates a graph of include dependencies
func GenerateIncludeGraph(options IncludeGraphOptions) {
	if options.Verbose {
		fmt.Println(color.CyanString("Starting include dependency analysis..."))
	}

	// Set default values for options
	if options.Jobs <= 0 {
		options.Jobs = 4 // Default to 4 concurrent jobs
	}

	if options.Format == "" {
		options.Format = "md" // Default to markdown format
	}

	// Record start time for performance metrics
	startTime := time.Now()

	// Get list of files to process
	files, err := utils.GetFilesToProcess(options.InputFile, options.Directory, options.Depth, options.Languages, options.Excludes)
	if err != nil {
		fmt.Println(color.RedString("Error:"), err)
		return
	}

	// Get current working directory for debugging
	cwd, _ := os.Getwd()
	if options.Verbose {
		fmt.Printf("Current working directory: %s\n", cwd)
	}
	
	// Always show the number of files found
	fmt.Printf(color.CyanString("Found %d files to process\n"), len(files))
	
	// Print the first 5 files for debugging
	if len(files) > 0 && options.Verbose {
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

	// Extract include relationships from files
	relations := extractIncludeRelations(files, options)

	// Determine if we should output to file or console
	var writer *bufio.Writer
	var file *os.File
	
	if options.OutputFile == "" {
		// Output to console
		writer = bufio.NewWriter(os.Stdout)
	} else {
		// Create output file
		file, err = os.Create(options.OutputFile)
		if err != nil {
			fmt.Println(color.RedString("Error creating output file:"), err)
			return
		}
		defer file.Close()
		writer = bufio.NewWriter(file)
	}
	defer writer.Flush()

	// Write results based on format
	switch options.Format {
	case "dot":
		writeIncludeGraphGraphvizOutput(writer, relations, options)
	case "json":
		writeIncludeGraphJSONOutput(writer, relations, options)
	default:
		writeIncludeGraphMarkdownOutput(writer, relations, options)
	}

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)
	
	// Print summary
	fmt.Println(color.GreenString("Include Graph Analysis Results"))
	fmt.Printf("\nFound %d include relationships in %d files\n", len(relations), len(files))
	fmt.Printf("Processing time: %s (%.2f files/sec)\n\n", utils.FormatDuration(elapsedTime), float64(len(files))/elapsedTime.Seconds())
	
	// No recommendation prints
}

// extractIncludeRelations extracts include relationships from files
func extractIncludeRelations(files []string, options IncludeGraphOptions) []IncludeRelation {
	var relations []IncludeRelation
	var mutex sync.Mutex
	var wg sync.WaitGroup
	
	// Set up a worker pool
	fileChan := make(chan string, len(files))
	
	// Create a progress bar
	progress := utils.NewProgressBar(len(files), "Analyzing includes")
	progress.Start()
	
	// Start workers
	for i := 0; i < options.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for file := range fileChan {
				// Extract includes from this file
				fileRelations := extractIncludesFromFile(file)
				
				// Add to global list
				if len(fileRelations) > 0 {
					mutex.Lock()
					relations = append(relations, fileRelations...)
					mutex.Unlock()
				}
				
				// Update progress
				progress.Increment()
			}
		}()
	}
	
	// Feed files to workers
	for _, file := range files {
		fileChan <- file
	}
	close(fileChan)
	
	// Wait for all workers to finish
	wg.Wait()
	progress.Finish()
	
	return relations
}

// extractIncludesFromFile extracts include statements from a single file
func extractIncludesFromFile(filePath string) []IncludeRelation {
	var relations []IncludeRelation
	
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return relations
	}
	defer file.Close()
	
	// Regular expressions for include statements
	systemIncludeRegex := regexp.MustCompile(`^\s*#\s*include\s*<([^>]+)>`)
	localIncludeRegex := regexp.MustCompile(`^\s*#\s*include\s*"([^"]+)"`)
	
	// Scan file line by line
	scanner := bufio.NewScanner(file)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		
		// Check for system includes
		if match := systemIncludeRegex.FindStringSubmatch(line); len(match) > 1 {
			relations = append(relations, IncludeRelation{
				SourceFile:   filePath,
				IncludedFile: match[1],
				Line:         lineNum,
				IsSystem:     true,
			})
		}
		
		// Check for local includes
		if match := localIncludeRegex.FindStringSubmatch(line); len(match) > 1 {
			relations = append(relations, IncludeRelation{
				SourceFile:   filePath,
				IncludedFile: match[1],
				Line:         lineNum,
				IsSystem:     false,
			})
		}
	}
	
	return relations
}

// writeIncludeGraphGraphvizOutput writes the include graph in DOT format for Graphviz
func writeIncludeGraphGraphvizOutput(writer *bufio.Writer, relations []IncludeRelation, options IncludeGraphOptions) {
	// Use options for potential customization
	graphTitle := "Include Graph"
	if options.Short {
		graphTitle = "Include Dependencies"
	}
	// Write DOT header
	fmt.Fprintf(writer, "digraph %s {\n", strings.Replace(graphTitle, " ", "_", -1))
	fmt.Fprintf(writer, "  rankdir=LR;\n")
	fmt.Fprintf(writer, "  node [shape=box, style=filled, fillcolor=lightblue];\n\n")
	
	// Track which files we've seen
	seenFiles := make(map[string]bool)
	
	// Write edges
	for _, rel := range relations {
		sourceFile := filepath.Base(rel.SourceFile)
		includedFile := rel.IncludedFile
		
		// Add nodes
		if !seenFiles[sourceFile] {
			fmt.Fprintf(writer, "  \"%s\" [label=\"%s\"];\n", sourceFile, sourceFile)
			seenFiles[sourceFile] = true
		}
		
		if !seenFiles[includedFile] {
			// Use different color for system includes
			if rel.IsSystem {
				fmt.Fprintf(writer, "  \"%s\" [label=\"%s\", fillcolor=lightgrey];\n", includedFile, includedFile)
			} else {
				fmt.Fprintf(writer, "  \"%s\" [label=\"%s\"];\n", includedFile, includedFile)
			}
			seenFiles[includedFile] = true
		}
		
		// Add edge
		edgeStyle := "solid"
		if rel.IsSystem {
			edgeStyle = "dashed"
		}
		fmt.Fprintf(writer, "  \"%s\" -> \"%s\" [style=%s];\n", sourceFile, includedFile, edgeStyle)
	}
	
	// Write DOT footer
	fmt.Fprintf(writer, "}\n")
}

// writeIncludeGraphJSONOutput writes the include graph in JSON format
func writeIncludeGraphJSONOutput(writer *bufio.Writer, relations []IncludeRelation, options IncludeGraphOptions) {
	// Use options for potential customization
	jsonTitle := "includeRelations"
	if options.Short {
		jsonTitle = "dependencies"
	}
	// Write JSON array header
	fmt.Fprintf(writer, "{\n")
	fmt.Fprintf(writer, "  \"%s\": [\n", jsonTitle)
	
	// Write each relation as a JSON object
	for i, rel := range relations {
		sourceFile := filepath.Base(rel.SourceFile)
		
		fmt.Fprintf(writer, "    {\n")
		fmt.Fprintf(writer, "      \"source\": \"%s\",\n", sourceFile)
		fmt.Fprintf(writer, "      \"target\": \"%s\",\n", rel.IncludedFile)
		fmt.Fprintf(writer, "      \"line\": %d,\n", rel.Line)
		fmt.Fprintf(writer, "      \"isSystem\": %t\n", rel.IsSystem)
		
		if i < len(relations)-1 {
			fmt.Fprintf(writer, "    },\n")
		} else {
			fmt.Fprintf(writer, "    }\n")
		}
	}
	
	// Write JSON array footer
	fmt.Fprintf(writer, "  ]\n")
	fmt.Fprintf(writer, "}\n")
}

// writeIncludeGraphMarkdownOutput writes the include graph in Markdown format
func writeIncludeGraphMarkdownOutput(writer *bufio.Writer, relations []IncludeRelation, options IncludeGraphOptions) {
	// Write header
	fmt.Fprintf(writer, "# Include Dependency Graph\n\n")
	
	// Group by source file
	fileMap := make(map[string][]IncludeRelation)
	for _, rel := range relations {
		fileMap[rel.SourceFile] = append(fileMap[rel.SourceFile], rel)
	}
	
	// Sort files for consistent output
	files := make([]string, 0, len(fileMap))
	for file := range fileMap {
		files = append(files, file)
	}
	
	// Write each file's includes
	for _, file := range files {
		relPath, _ := filepath.Rel(options.Directory, file)
		fmt.Fprintf(writer, "## %s\n\n", relPath)
		
		// Write table header
		fmt.Fprintf(writer, "| Include | Type | Line |\n")
		fmt.Fprintf(writer, "|---------|------|------|\n")
		
		// Write each include
		for _, rel := range fileMap[file] {
			includeType := "Local"
			if rel.IsSystem {
				includeType = "System"
			}
			
			fmt.Fprintf(writer, "| `%s` | %s | %d |\n", rel.IncludedFile, includeType, rel.Line)
		}
		
		fmt.Fprintf(writer, "\n")
	}
	
	// Write summary
	fmt.Fprintf(writer, "## Summary\n\n")
	fmt.Fprintf(writer, "- Total files analyzed: %d\n", len(fileMap))
	fmt.Fprintf(writer, "- Total include relationships: %d\n", len(relations))
	
	// Count system vs local includes
	systemIncludes := 0
	for _, rel := range relations {
		if rel.IsSystem {
			systemIncludes++
		}
	}
	
	fmt.Fprintf(writer, "- System includes: %d\n", systemIncludes)
	fmt.Fprintf(writer, "- Local includes: %d\n", len(relations)-systemIncludes)
}
