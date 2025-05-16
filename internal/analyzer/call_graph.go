package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/vitruves/gop/internal/utils"
)

// CallGraphOptions contains options for the call-graph command
type CallGraphOptions struct {
	// Input/Output options
	InputFile  string   // Path to input file containing list of files to process
	Directory  string   // Root directory to analyze
	Depth      int      // Maximum depth for directory traversal
	OutputFile string   // Path to output file for results
	Languages  []string // Languages to analyze (e.g., "c", "cpp")
	Excludes   []string // Directories or files to exclude

	// Processing options
	Jobs int // Number of concurrent jobs for processing

	// Analysis options
	Format     string // Output format (dot, json, md)
	MaxDepth   int    // Maximum depth for call graph traversal
	TargetFunc string // Target function to analyze (empty means all)
	ExcludeSys bool   // Whether to exclude system functions

	// Output options
	Short   bool // Whether to use short output format
	Verbose bool // Whether to enable verbose output
}

// FunctionCall represents a function call in code
type FunctionCall struct {
	Caller     string // Name of the calling function
	Callee     string // Name of the called function
	FilePath   string // Path to the file containing the call
	Line       int    // Line number where the call appears
	IsExternal bool   // Whether the call is to an external function
}

// CallGraphFunction represents a function definition in code for call graph analysis
type CallGraphFunction struct {
	Name       string   // Name of the function
	FilePath   string   // Path to the file containing the function
	StartLine  int      // Starting line number
	EndLine    int      // Ending line number
	Calls      []string // Names of functions called by this function
	CalledBy   []string // Names of functions that call this function
	IsExternal bool     // Whether the function is external (no definition found)
}

// GenerateCallGraph generates a call graph for functions in code
func GenerateCallGraph(options CallGraphOptions) {
	if options.Verbose {
		fmt.Println(color.CyanString("Starting call graph generation..."))
	}

	// Set default values for options
	if options.Jobs <= 0 {
		options.Jobs = 4 // Default to 4 concurrent jobs
	}

	if options.Format == "" {
		options.Format = "md" // Default to markdown format
	}

	if options.MaxDepth <= 0 {
		options.MaxDepth = 5 // Default to maximum depth of 5
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

	// Extract functions and their calls
	functions, calls := extractFunctionsAndCalls(files, options)

	// Build the call graph
	callGraph := buildCallGraph(functions, calls, options)

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
		writeCallGraphGraphvizOutput(writer, callGraph, options)
	case "json":
		writeCallGraphJSONOutput(writer, callGraph, options)
	default:
		writeCallGraphMarkdownOutput(writer, callGraph, options)
	}

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)

	// Print summary
	fmt.Println(color.GreenString("Call Graph Generation Results"))
	fmt.Printf("\nAnalyzed %d functions with %d calls in %d files\n", len(functions), len(calls), len(files))
	fmt.Printf("Processing time: %s (%.2f files/sec)\n\n", utils.FormatDuration(elapsedTime), float64(len(files))/elapsedTime.Seconds())
	
	// No recommendation prints
}

// extractFunctionsAndCalls extracts functions and their calls from files
func extractFunctionsAndCalls(files []string, options CallGraphOptions) (map[string]CallGraphFunction, []FunctionCall) {
	functions := make(map[string]CallGraphFunction)
	var calls []FunctionCall

	var mutex sync.Mutex
	var wg sync.WaitGroup

	// Set up a worker pool
	fileChan := make(chan string, len(files))

	// Create a progress bar
	progress := utils.NewProgressBar(len(files), "Extracting functions and calls")
	progress.Start()

	// Start workers
	for i := 0; i < options.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for file := range fileChan {
				// Extract functions and calls from this file
				fileFunctions, fileCalls := extractFromFile(file, options)

				// Add to global maps
				mutex.Lock()
				for name, function := range fileFunctions {
					functions[name] = function
				}
				calls = append(calls, fileCalls...)
				mutex.Unlock()

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

	return functions, calls
}

// extractFromFile extracts functions and their calls from a single file
func extractFromFile(filePath string, options CallGraphOptions) (map[string]CallGraphFunction, []FunctionCall) {
	functions := make(map[string]CallGraphFunction)
	var calls []FunctionCall

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return functions, calls
	}
	defer file.Close()

	// Read file content
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Extract function definitions
	extractedFunctions := extractFunctions(filePath, lines)

	// Convert to our Function type
	for _, fn := range extractedFunctions {
		functions[fn.FunctionName] = CallGraphFunction{
			Name:      fn.FunctionName,
			FilePath:  fn.FilePath,
			StartLine: fn.StartLine,
			EndLine:   fn.EndLine,
			Calls:     []string{},
			CalledBy:  []string{},
		}
	}

	// Extract function calls
	for _, fn := range extractedFunctions {
		fnName := fn.FunctionName

		// Analyze the function body for calls
		for i := fn.StartLine - 1; i < fn.EndLine && i < len(lines); i++ {
			lineNum := i + 1
			line := lines[i]

			// Extract function calls from this line
			lineCalls := extractCallsFromLine(line, options)

			for _, callee := range lineCalls {
				// Skip if the callee is the same as the caller (recursive call)
				if callee == fnName {
					continue
				}

				// Add to calls list
				calls = append(calls, FunctionCall{
					Caller:     fnName,
					Callee:     callee,
					FilePath:   filePath,
					Line:       lineNum,
					IsExternal: true, // Will be updated later
				})

				// Add to the function's calls list if not already there
				if fn, ok := functions[fnName]; ok {
					if !contains(fn.Calls, callee) {
						fn.Calls = append(fn.Calls, callee)
						functions[fnName] = fn
					}
				}
			}
		}
	}

	return functions, calls
}

// extractCallsFromLine extracts function calls from a line of code
func extractCallsFromLine(line string, options CallGraphOptions) []string {
	var calls []string

	// Regular expression for function calls
	// This is a simplified approach - a real implementation would use a proper parser
	callRegex := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)

	// Find all function calls in the line
	matches := callRegex.FindAllStringSubmatch(line, -1)

	for _, match := range matches {
		if len(match) > 1 {
			funcName := match[1]

			// Skip common keywords that might be mistaken for function calls
			if isKeyword(funcName) {
				continue
			}

			// Skip system functions if requested
			if options.ExcludeSys && isSystemFunction(funcName) {
				continue
			}

			calls = append(calls, funcName)
		}
	}

	return calls
}

// isKeyword checks if a string is a C/C++ keyword
func isKeyword(word string) bool {
	keywords := map[string]bool{
		"if":      true,
		"else":    true,
		"for":     true,
		"while":   true,
		"do":      true,
		"switch":  true,
		"case":    true,
		"return":  true,
		"sizeof":  true,
		"typedef": true,
		"struct":  true,
		"class":   true,
		"enum":    true,
		"union":   true,
		"goto":    true,
	}

	return keywords[word]
}

// isSystemFunction checks if a function is a system function
func isSystemFunction(funcName string) bool {
	systemFuncs := map[string]bool{
		"printf":  true,
		"sprintf": true,
		"fprintf": true,
		"scanf":   true,
		"fscanf":  true,
		"malloc":  true,
		"calloc":  true,
		"realloc": true,
		"free":    true,
		"memcpy":  true,
		"memset":  true,
		"strcpy":  true,
		"strncpy": true,
		"strcmp":  true,
		"strncmp": true,
		"strlen":  true,
		"fopen":   true,
		"fclose":  true,
		"fread":   true,
		"fwrite":  true,
		"exit":    true,
		"abort":   true,
		"assert":  true,
	}

	return systemFuncs[funcName]
}

// buildCallGraph builds a call graph from functions and calls
func buildCallGraph(functions map[string]CallGraphFunction, calls []FunctionCall, options CallGraphOptions) map[string]CallGraphFunction {
	// Create a copy of the functions map to build the call graph
	callGraph := make(map[string]CallGraphFunction)
	for name, fn := range functions {
		callGraph[name] = fn
	}

	// Update the IsExternal flag for each call
	for _, call := range calls {
		// Check if the callee exists in our functions
		if _, ok := functions[call.Callee]; !ok {
			// Add external function to the call graph
			if _, exists := callGraph[call.Callee]; !exists {
				callGraph[call.Callee] = CallGraphFunction{
					Name:       call.Callee,
					IsExternal: true,
				}
			}
		}
	}

	// Build the CalledBy relationships
	for _, call := range calls {
		// Update the callee's CalledBy list
		if callee, ok := callGraph[call.Callee]; ok {
			if !contains(callee.CalledBy, call.Caller) {
				callee.CalledBy = append(callee.CalledBy, call.Caller)
				callGraph[call.Callee] = callee
			}
		}
	}

	// If a target function is specified, filter the call graph
	if options.TargetFunc != "" {
		filteredGraph := make(map[string]CallGraphFunction)

		// Start with the target function
		if targetFn, ok := callGraph[options.TargetFunc]; ok {
			filteredGraph[options.TargetFunc] = targetFn

			// Add functions called by the target function (up to MaxDepth)
			addCallees(filteredGraph, callGraph, options.TargetFunc, 1, options.MaxDepth)

			// Add functions that call the target function
			addCallers(filteredGraph, callGraph, options.TargetFunc, 1, options.MaxDepth)
		}

		return filteredGraph
	}

	return callGraph
}

// addCallees adds functions called by a function to the filtered graph
func addCallees(filteredGraph, callGraph map[string]CallGraphFunction, funcName string, depth, maxDepth int) {
	if depth > maxDepth {
		return
	}

	if fn, ok := callGraph[funcName]; ok {
		for _, callee := range fn.Calls {
			if calleeFn, ok := callGraph[callee]; ok {
				filteredGraph[callee] = calleeFn
				addCallees(filteredGraph, callGraph, callee, depth+1, maxDepth)
			}
		}
	}
}

// addCallers adds functions that call a function to the filtered graph
func addCallers(filteredGraph, callGraph map[string]CallGraphFunction, funcName string, depth, maxDepth int) {
	if depth > maxDepth {
		return
	}

	if fn, ok := callGraph[funcName]; ok {
		for _, caller := range fn.CalledBy {
			if callerFn, ok := callGraph[caller]; ok {
				filteredGraph[caller] = callerFn
				addCallers(filteredGraph, callGraph, caller, depth+1, maxDepth)
			}
		}
	}
}

// writeCallGraphGraphvizOutput writes the call graph in DOT format for Graphviz
func writeCallGraphGraphvizOutput(writer *bufio.Writer, callGraph map[string]CallGraphFunction, options CallGraphOptions) {
	// Use options for potential customization
	graphTitle := "Call Graph"
	if options.TargetFunc != "" {
		graphTitle = "Call Graph for " + options.TargetFunc
	}
	// Write DOT header
	fmt.Fprintf(writer, "digraph %s {\n", strings.Replace(graphTitle, " ", "_", -1))
	fmt.Fprintf(writer, "  rankdir=LR;\n")
	fmt.Fprintf(writer, "  node [shape=box, style=filled, fillcolor=lightblue];\n\n")

	// Write nodes
	for name, fn := range callGraph {
		if fn.IsExternal {
			fmt.Fprintf(writer, "  \"%s\" [label=\"%s\", fillcolor=lightgrey];\n", name, name)
		} else {
			fmt.Fprintf(writer, "  \"%s\" [label=\"%s\"];\n", name, name)
		}
	}

	fmt.Fprintf(writer, "\n")

	// Write edges
	for name, fn := range callGraph {
		for _, callee := range fn.Calls {
			// Skip if the callee is not in our filtered graph
			if _, ok := callGraph[callee]; !ok {
				continue
			}

			fmt.Fprintf(writer, "  \"%s\" -> \"%s\";\n", name, callee)
		}
	}

	// Write DOT footer
	fmt.Fprintf(writer, "}\n")
}

// writeCallGraphJSONOutput writes the call graph in JSON format
func writeCallGraphJSONOutput(writer *bufio.Writer, callGraph map[string]CallGraphFunction, options CallGraphOptions) {
	// Write JSON header
	fmt.Fprintf(writer, "{\n")
	fmt.Fprintf(writer, "  \"functions\": [\n")

	// Convert map to slice for consistent output
	functions := make([]CallGraphFunction, 0, len(callGraph))
	for _, fn := range callGraph {
		functions = append(functions, fn)
	}

	// Sort functions by name
	sort.Slice(functions, func(i, j int) bool {
		return functions[i].Name < functions[j].Name
	})

	// Write functions
	for i, fn := range functions {
		fmt.Fprintf(writer, "    {\n")
		fmt.Fprintf(writer, "      \"name\": \"%s\",\n", fn.Name)

		if fn.FilePath != "" {
			relPath, _ := filepath.Rel(options.Directory, fn.FilePath)
			fmt.Fprintf(writer, "      \"file\": \"%s\",\n", relPath)
		}

		if fn.StartLine > 0 {
			fmt.Fprintf(writer, "      \"startLine\": %d,\n", fn.StartLine)
		}

		if fn.EndLine > 0 {
			fmt.Fprintf(writer, "      \"endLine\": %d,\n", fn.EndLine)
		}

		fmt.Fprintf(writer, "      \"isExternal\": %t,\n", fn.IsExternal)

		// Write calls
		fmt.Fprintf(writer, "      \"calls\": [")
		for j, callee := range fn.Calls {
			if j > 0 {
				fmt.Fprintf(writer, ", ")
			}
			fmt.Fprintf(writer, "\"%s\"", callee)
		}
		fmt.Fprintf(writer, "],\n")

		// Write calledBy
		fmt.Fprintf(writer, "      \"calledBy\": [")
		for j, caller := range fn.CalledBy {
			if j > 0 {
				fmt.Fprintf(writer, ", ")
			}
			fmt.Fprintf(writer, "\"%s\"", caller)
		}
		fmt.Fprintf(writer, "]\n")

		if i < len(functions)-1 {
			fmt.Fprintf(writer, "    },\n")
		} else {
			fmt.Fprintf(writer, "    }\n")
		}
	}

	// Write JSON footer
	fmt.Fprintf(writer, "  ]\n")
	fmt.Fprintf(writer, "}\n")
}

// writeCallGraphMarkdownOutput writes the call graph in Markdown format
func writeCallGraphMarkdownOutput(writer *bufio.Writer, callGraph map[string]CallGraphFunction, options CallGraphOptions) {
	// Write header
	fmt.Fprintf(writer, "# Call Graph Analysis\n\n")

	// Write summary
	fmt.Fprintf(writer, "## Summary\n\n")

	// Count internal and external functions
	internalCount := 0
	externalCount := 0
	for _, fn := range callGraph {
		if fn.IsExternal {
			externalCount++
		} else {
			internalCount++
		}
	}

	fmt.Fprintf(writer, "- Total functions: %d\n", len(callGraph))
	fmt.Fprintf(writer, "- Internal functions: %d\n", internalCount)
	fmt.Fprintf(writer, "- External functions: %d\n\n", externalCount)

	if options.TargetFunc != "" {
		fmt.Fprintf(writer, "Analysis centered on function: `%s`\n", options.TargetFunc)
		fmt.Fprintf(writer, "Maximum call depth: %d\n\n", options.MaxDepth)
	}

	// Write function details
	fmt.Fprintf(writer, "## Function Details\n\n")

	// Convert map to slice for consistent output
	functions := make([]CallGraphFunction, 0, len(callGraph))
	for _, fn := range callGraph {
		functions = append(functions, fn)
	}

	// Sort functions by name
	sort.Slice(functions, func(i, j int) bool {
		return functions[i].Name < functions[j].Name
	})

	// Write each function
	for _, fn := range functions {
		fmt.Fprintf(writer, "### %s\n\n", fn.Name)

		if fn.IsExternal {
			fmt.Fprintf(writer, "**External function**\n\n")
		} else {
			relPath, _ := filepath.Rel(options.Directory, fn.FilePath)
			fmt.Fprintf(writer, "**Defined in**: `%s` (lines %d-%d)\n\n", relPath, fn.StartLine, fn.EndLine)
		}

		// Write calls
		if len(fn.Calls) > 0 {
			fmt.Fprintf(writer, "#### Calls:\n\n")
			for _, callee := range fn.Calls {
				// Skip if the callee is not in our filtered graph
				if _, ok := callGraph[callee]; !ok {
					continue
				}

				fmt.Fprintf(writer, "- `%s`\n", callee)
			}
			fmt.Fprintf(writer, "\n")
		}

		// Write calledBy
		if len(fn.CalledBy) > 0 {
			fmt.Fprintf(writer, "#### Called by:\n\n")
			for _, caller := range fn.CalledBy {
				// Skip if the caller is not in our filtered graph
				if _, ok := callGraph[caller]; !ok {
					continue
				}

				fmt.Fprintf(writer, "- `%s`\n", caller)
			}
			fmt.Fprintf(writer, "\n")
		}
	}

	// Write call hierarchy
	if options.TargetFunc != "" && !options.Short {
		fmt.Fprintf(writer, "## Call Hierarchy\n\n")

		if targetFn, ok := callGraph[options.TargetFunc]; ok {
			fmt.Fprintf(writer, "### Functions called by `%s`\n\n", options.TargetFunc)
			writeCallHierarchy(writer, callGraph, targetFn.Name, 0, options.MaxDepth, make(map[string]bool), true)

			fmt.Fprintf(writer, "\n### Functions that call `%s`\n\n", options.TargetFunc)
			writeCallerHierarchy(writer, callGraph, targetFn.Name, 0, options.MaxDepth, make(map[string]bool), true)
		}
	}
}

// writeCallHierarchy writes the call hierarchy for a function
func writeCallHierarchy(writer *bufio.Writer, callGraph map[string]CallGraphFunction, funcName string, depth, maxDepth int, visited map[string]bool, isRoot bool) {
	if depth > maxDepth {
		return
	}

	// Check for cycles
	if visited[funcName] && !isRoot {
		fmt.Fprintf(writer, "%s- `%s` (recursive call)\n", strings.Repeat("  ", depth), funcName)
		return
	}

	// Mark as visited
	visited[funcName] = true

	// Print the function name
	if !isRoot {
		fmt.Fprintf(writer, "%s- `%s`", strings.Repeat("  ", depth), funcName)

		// Add external indicator
		if fn, ok := callGraph[funcName]; ok && fn.IsExternal {
			fmt.Fprintf(writer, " (external)")
		}

		fmt.Fprintf(writer, "\n")
	}

	// Print calls
	if fn, ok := callGraph[funcName]; ok {
		for _, callee := range fn.Calls {
			// Skip if the callee is not in our filtered graph
			if _, ok := callGraph[callee]; !ok {
				continue
			}

			writeCallHierarchy(writer, callGraph, callee, depth+1, maxDepth, visited, false)
		}
	}

	// Unmark as visited if we're backtracking
	if depth == 0 {
		delete(visited, funcName)
	}
}

// writeCallerHierarchy writes the caller hierarchy for a function
func writeCallerHierarchy(writer *bufio.Writer, callGraph map[string]CallGraphFunction, funcName string, depth, maxDepth int, visited map[string]bool, isRoot bool) {
	if depth > maxDepth {
		return
	}

	// Check for cycles
	if visited[funcName] && !isRoot {
		fmt.Fprintf(writer, "%s- `%s` (recursive call)\n", strings.Repeat("  ", depth), funcName)
		return
	}

	// Mark as visited
	visited[funcName] = true

	// Print the function name
	if !isRoot {
		fmt.Fprintf(writer, "%s- `%s`", strings.Repeat("  ", depth), funcName)

		// Add external indicator
		if fn, ok := callGraph[funcName]; ok && fn.IsExternal {
			fmt.Fprintf(writer, " (external)")
		}

		fmt.Fprintf(writer, "\n")
	}

	// Print callers
	if fn, ok := callGraph[funcName]; ok {
		for _, caller := range fn.CalledBy {
			// Skip if the caller is not in our filtered graph
			if _, ok := callGraph[caller]; !ok {
				continue
			}

			writeCallerHierarchy(writer, callGraph, caller, depth+1, maxDepth, visited, false)
		}
	}

	// Unmark as visited if we're backtracking
	if depth == 0 {
		delete(visited, funcName)
	}
}

// Helper function to check if a slice contains a string
func contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}
