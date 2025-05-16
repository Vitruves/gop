package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/vitruves/gop/internal/utils"
)

// ComplexityOptions contains options for the complexity command
type ComplexityOptions struct {
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
	Cyclomatic bool  // Whether to calculate cyclomatic complexity
	Cognitive  bool  // Whether to calculate cognitive complexity
	Threshold  int   // Threshold for highlighting complex functions
	
	// Output options
	Short   bool   // Whether to use short output format
	IAOutput bool   // Whether to output in a format suitable for AI tools
	Verbose bool   // Whether to enable verbose output
}

// FunctionComplexity represents complexity metrics for a function
type FunctionComplexity struct {
	FilePath            string // Path to the file containing the function
	FunctionName        string // Name of the function
	StartLine           int    // Starting line number
	EndLine             int    // Ending line number
	CyclomaticComplexity int   // McCabe's cyclomatic complexity
	CognitiveComplexity int    // Cognitive complexity
	LineCount           int    // Number of lines in the function
}

// AnalyzeComplexity analyzes code complexity metrics
func AnalyzeComplexity(options ComplexityOptions) {
	if options.Verbose {
		fmt.Println(color.CyanString("Starting complexity analysis..."))
	}

	// Set default values for options
	if options.Jobs <= 0 {
		options.Jobs = 4 // Default to 4 concurrent jobs
	}

	if options.Threshold <= 0 {
		options.Threshold = 10 // Default threshold for complex functions
	}

	// If neither complexity metric is specified, enable both
	if !options.Cyclomatic && !options.Cognitive {
		options.Cyclomatic = true
		options.Cognitive = true
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

	// Analyze complexity of functions in files
	metrics := analyzeCodeComplexity(files, options)

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

	// Write results
	if options.IAOutput {
		writeComplexityResultsAI(writer, metrics, options)
	} else {
		writeComplexityResults(writer, metrics, options)
	}

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)
	
	// Print summary
	fmt.Println(color.GreenString("Complexity Analysis Results"))
	fmt.Printf("\nAnalyzed %d functions in %d files\n", len(metrics), len(files))
	
	// Count functions above threshold
	highComplexityCount := 0
	for _, metric := range metrics {
		if (options.Cyclomatic && metric.CyclomaticComplexity > options.Threshold) ||
		   (options.Cognitive && metric.CognitiveComplexity > options.Threshold) {
			highComplexityCount++
		}
	}
	
	fmt.Printf("Found %d functions with high complexity (threshold: %d)\n", highComplexityCount, options.Threshold)
	fmt.Printf("Processing time: %s (%.2f files/sec)\n\n", utils.FormatDuration(elapsedTime), float64(len(files))/elapsedTime.Seconds())
	
	// No recommendation prints
}

// analyzeCodeComplexity analyzes complexity metrics for functions in files
func analyzeCodeComplexity(files []string, options ComplexityOptions) []FunctionComplexity {
	var metrics []FunctionComplexity
	var mutex sync.Mutex
	var wg sync.WaitGroup
	
	// Set up a worker pool
	fileChan := make(chan string, len(files))
	
	// Create a progress bar
	progress := utils.NewProgressBar(len(files), "Analyzing complexity")
	progress.Start()
	
	// Start workers
	for i := 0; i < options.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for file := range fileChan {
				// Extract functions and analyze complexity
				functionMetrics := analyzeFunctionComplexity(file)
				
				// Add to global list
				if len(functionMetrics) > 0 {
					mutex.Lock()
					metrics = append(metrics, functionMetrics...)
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
	
	// Sort metrics by complexity (highest first)
	sort.Slice(metrics, func(i, j int) bool {
		// If cyclomatic complexity is the same, sort by cognitive complexity
		if metrics[i].CyclomaticComplexity == metrics[j].CyclomaticComplexity {
			return metrics[i].CognitiveComplexity > metrics[j].CognitiveComplexity
		}
		return metrics[i].CyclomaticComplexity > metrics[j].CyclomaticComplexity
	})
	
	return metrics
}

// analyzeFunctionComplexity analyzes complexity metrics for functions in a file
func analyzeFunctionComplexity(filePath string) []FunctionComplexity {
	var metrics []FunctionComplexity
	
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return metrics
	}
	defer file.Close()
	
	// Read file content
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	
	// Extract functions
	functions := extractFunctions(filePath, lines)
	
	// Analyze complexity for each function
	for _, function := range functions {
		// Calculate metrics
		cyclomaticComplexity := calculateCyclomaticComplexity(lines[function.StartLine-1:function.EndLine])
		cognitiveComplexity := calculateCognitiveComplexity(lines[function.StartLine-1:function.EndLine])
		
		metrics = append(metrics, FunctionComplexity{
			FilePath:            filePath,
			FunctionName:        function.FunctionName,
			StartLine:           function.StartLine,
			EndLine:             function.EndLine,
			CyclomaticComplexity: cyclomaticComplexity,
			CognitiveComplexity: cognitiveComplexity,
			LineCount:           function.EndLine - function.StartLine + 1,
		})
	}
	
	return metrics
}

// Function represents a function or method in code
type Function struct {
	FilePath     string
	FunctionName string
	StartLine    int
	EndLine      int
}

// extractFunctions extracts functions from a file
func extractFunctions(filePath string, lines []string) []Function {
	var functions []Function
	
	// Regular expressions for function declarations
	// This is a simplified approach - a real implementation would use a proper parser
	funcRegex := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\([^)]*\)\s*(\{?)`)
	methodRegex := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*\s+[a-zA-Z_][a-zA-Z0-9_]*::([a-zA-Z_][a-zA-Z0-9_]*)\s*\([^)]*\)\s*(\{?)`)
	
	// Track open braces to find function end
	var openFunction *Function
	braceCount := 0
	
	for i, line := range lines {
		lineNum := i + 1
		
		if openFunction == nil {
			// Look for function declarations
			if match := funcRegex.FindStringSubmatch(line); len(match) > 1 {
				openFunction = &Function{
					FilePath:     filePath,
					FunctionName: match[1],
					StartLine:    lineNum,
				}
				if len(match) > 2 && match[2] == "{" {
					braceCount = 1
				}
			} else if match := methodRegex.FindStringSubmatch(line); len(match) > 1 {
				openFunction = &Function{
					FilePath:     filePath,
					FunctionName: match[1],
					StartLine:    lineNum,
				}
				if len(match) > 2 && match[2] == "{" {
					braceCount = 1
				}
			}
		} else {
			// Count braces to find function end
			for _, char := range line {
				if char == '{' {
					braceCount++
				} else if char == '}' {
					braceCount--
					if braceCount == 0 {
						openFunction.EndLine = lineNum
						functions = append(functions, *openFunction)
						openFunction = nil
						break
					}
				}
			}
		}
	}
	
	return functions
}

// calculateCyclomaticComplexity calculates McCabe's cyclomatic complexity
func calculateCyclomaticComplexity(lines []string) int {
	// Base complexity is 1
	complexity := 1
	
	// Regular expressions for control flow statements
	ifRegex := regexp.MustCompile(`\bif\s*\(`)
	elseIfRegex := regexp.MustCompile(`\belse\s+if\s*\(`)
	switchRegex := regexp.MustCompile(`\bswitch\s*\(`)
	caseRegex := regexp.MustCompile(`\bcase\s+`)
	forRegex := regexp.MustCompile(`\bfor\s*\(`)
	whileRegex := regexp.MustCompile(`\bwhile\s*\(`)
	doWhileRegex := regexp.MustCompile(`\bdo\s*\{`)
	andRegex := regexp.MustCompile(`&&`)
	orRegex := regexp.MustCompile(`\|\|`)
	ternaryRegex := regexp.MustCompile(`\?`)
	catchRegex := regexp.MustCompile(`\bcatch\s*\(`)
	
	for _, line := range lines {
		// Each control flow statement adds 1 to complexity
		complexity += len(ifRegex.FindAllString(line, -1))
		complexity += len(elseIfRegex.FindAllString(line, -1))
		complexity += len(switchRegex.FindAllString(line, -1))
		complexity += len(caseRegex.FindAllString(line, -1))
		complexity += len(forRegex.FindAllString(line, -1))
		complexity += len(whileRegex.FindAllString(line, -1))
		complexity += len(doWhileRegex.FindAllString(line, -1))
		
		// Each logical operator adds 1 to complexity
		complexity += len(andRegex.FindAllString(line, -1))
		complexity += len(orRegex.FindAllString(line, -1))
		
		// Ternary operators add 1 to complexity
		complexity += len(ternaryRegex.FindAllString(line, -1))
		
		// Exception handling adds 1 to complexity
		complexity += len(catchRegex.FindAllString(line, -1))
	}
	
	return complexity
}

// calculateCognitiveComplexity calculates cognitive complexity
func calculateCognitiveComplexity(lines []string) int {
	complexity := 0
	nestingLevel := 0
	
	// Regular expressions for control flow statements
	ifRegex := regexp.MustCompile(`\bif\s*\(`)
	elseIfRegex := regexp.MustCompile(`\belse\s+if\s*\(`)
	elseRegex := regexp.MustCompile(`\belse\s*\{`)
	switchRegex := regexp.MustCompile(`\bswitch\s*\(`)
	forRegex := regexp.MustCompile(`\bfor\s*\(`)
	whileRegex := regexp.MustCompile(`\bwhile\s*\(`)
	doWhileRegex := regexp.MustCompile(`\bdo\s*\{`)
	catchRegex := regexp.MustCompile(`\bcatch\s*\(`)
	
	// Logical operators
	andRegex := regexp.MustCompile(`&&`)
	orRegex := regexp.MustCompile(`\|\|`)
	
	// Track open and close braces for nesting
	openBraceRegex := regexp.MustCompile(`\{`)
	closeBraceRegex := regexp.MustCompile(`\}`)
	
	for _, line := range lines {
		// Basic control flow statements
		if ifRegex.MatchString(line) && !elseIfRegex.MatchString(line) {
			complexity += 1 + nestingLevel
			nestingLevel++
		}
		
		if elseIfRegex.MatchString(line) {
			complexity += 1 + nestingLevel
		}
		
		if elseRegex.MatchString(line) {
			complexity += 1
		}
		
		if switchRegex.MatchString(line) {
			complexity += 1 + nestingLevel
			nestingLevel++
		}
		
		if forRegex.MatchString(line) || whileRegex.MatchString(line) || doWhileRegex.MatchString(line) {
			complexity += 1 + nestingLevel
			nestingLevel++
		}
		
		if catchRegex.MatchString(line) {
			complexity += 1 + nestingLevel
		}
		
		// Logical operators
		complexity += len(andRegex.FindAllString(line, -1))
		complexity += len(orRegex.FindAllString(line, -1))
		
		// Track nesting level
		openBraces := len(openBraceRegex.FindAllString(line, -1))
		closeBraces := len(closeBraceRegex.FindAllString(line, -1))
		
		// Adjust nesting level based on braces
		// This is a simplified approach - a real implementation would track nesting more accurately
		if closeBraces > openBraces {
			nestingLevel -= (closeBraces - openBraces)
			if nestingLevel < 0 {
				nestingLevel = 0
			}
		}
	}
	
	return complexity
}

// writeComplexityResults writes complexity metrics to the output file
func writeComplexityResults(writer *bufio.Writer, metrics []FunctionComplexity, options ComplexityOptions) {
	// Write header
	fmt.Fprintf(writer, "# Code Complexity Metrics\n\n")
	
	// Write summary
	fmt.Fprintf(writer, "## Summary\n\n")
	fmt.Fprintf(writer, "- Total functions analyzed: %d\n", len(metrics))
	
	// Count functions above threshold
	highCyclomaticCount := 0
	highCognitiveCount := 0
	
	for _, metric := range metrics {
		if options.Cyclomatic && metric.CyclomaticComplexity > options.Threshold {
			highCyclomaticCount++
		}
		if options.Cognitive && metric.CognitiveComplexity > options.Threshold {
			highCognitiveCount++
		}
	}
	
	if options.Cyclomatic {
		fmt.Fprintf(writer, "- Functions with high cyclomatic complexity (> %d): %d\n", options.Threshold, highCyclomaticCount)
	}
	
	if options.Cognitive {
		fmt.Fprintf(writer, "- Functions with high cognitive complexity (> %d): %d\n", options.Threshold, highCognitiveCount)
	}
	
	// Write table of metrics
	fmt.Fprintf(writer, "\n## Function Complexity Metrics\n\n")
	
	// Table header
	fmt.Fprintf(writer, "| Function | File | Lines | ")
	if options.Cyclomatic {
		fmt.Fprintf(writer, "Cyclomatic | ")
	}
	if options.Cognitive {
		fmt.Fprintf(writer, "Cognitive | ")
	}
	fmt.Fprintf(writer, "\n")
	
	// Table separator
	fmt.Fprintf(writer, "|----------|------|-------|")
	if options.Cyclomatic {
		fmt.Fprintf(writer, "-----------|")
	}
	if options.Cognitive {
		fmt.Fprintf(writer, "-----------|")
	}
	fmt.Fprintf(writer, "\n")
	
	// Table rows
	for _, metric := range metrics {
		// Skip functions below threshold if not in verbose mode
		if !options.Verbose && 
		   !(options.Cyclomatic && metric.CyclomaticComplexity > options.Threshold) && 
		   !(options.Cognitive && metric.CognitiveComplexity > options.Threshold) {
			continue
		}
		
		relPath, _ := filepath.Rel(options.Directory, metric.FilePath)
		
		fmt.Fprintf(writer, "| `%s` | `%s` | %d-%d | ", 
			metric.FunctionName, 
			relPath, 
			metric.StartLine, 
			metric.EndLine)
		
		if options.Cyclomatic {
			// Highlight high complexity
			if metric.CyclomaticComplexity > options.Threshold {
				fmt.Fprintf(writer, "**%d** | ", metric.CyclomaticComplexity)
			} else {
				fmt.Fprintf(writer, "%d | ", metric.CyclomaticComplexity)
			}
		}
		
		if options.Cognitive {
			// Highlight high complexity
			if metric.CognitiveComplexity > options.Threshold {
				fmt.Fprintf(writer, "**%d** | ", metric.CognitiveComplexity)
			} else {
				fmt.Fprintf(writer, "%d | ", metric.CognitiveComplexity)
			}
		}
		
		fmt.Fprintf(writer, "\n")
	}
	
	// No recommendations section
}

// writeComplexityResultsAI writes complexity metrics in a format suitable for AI tools
func writeComplexityResultsAI(writer *bufio.Writer, metrics []FunctionComplexity, options ComplexityOptions) {
	// Write header
	fmt.Fprintf(writer, "# Code Complexity Analysis\n\n")
	
	// Write JSON-like format
	fmt.Fprintf(writer, "```json\n")
	fmt.Fprintf(writer, "{\n")
	fmt.Fprintf(writer, "  \"metrics\": [\n")
	
	// Write each function's metrics
	for i, metric := range metrics {
		relPath, _ := filepath.Rel(options.Directory, metric.FilePath)
		
		fmt.Fprintf(writer, "    {\n")
		fmt.Fprintf(writer, "      \"function\": \"%s\",\n", metric.FunctionName)
		fmt.Fprintf(writer, "      \"file\": \"%s\",\n", relPath)
		fmt.Fprintf(writer, "      \"startLine\": %d,\n", metric.StartLine)
		fmt.Fprintf(writer, "      \"endLine\": %d,\n", metric.EndLine)
		fmt.Fprintf(writer, "      \"lineCount\": %d,\n", metric.LineCount)
		
		if options.Cyclomatic {
			fmt.Fprintf(writer, "      \"cyclomaticComplexity\": %d,\n", metric.CyclomaticComplexity)
		}
		
		if options.Cognitive {
			fmt.Fprintf(writer, "      \"cognitiveComplexity\": %d,\n", metric.CognitiveComplexity)
		}
		
		// Add a flag for high complexity
		highComplexity := (options.Cyclomatic && metric.CyclomaticComplexity > options.Threshold) ||
						  (options.Cognitive && metric.CognitiveComplexity > options.Threshold)
		
		fmt.Fprintf(writer, "      \"highComplexity\": %t\n", highComplexity)
		
		if i < len(metrics)-1 {
			fmt.Fprintf(writer, "    },\n")
		} else {
			fmt.Fprintf(writer, "    }\n")
		}
	}
	
	fmt.Fprintf(writer, "  ]\n")
	fmt.Fprintf(writer, "}\n")
	fmt.Fprintf(writer, "```\n")
	
	// Add explanation
	fmt.Fprintf(writer, "\n## Complexity Metrics Explanation\n\n")
	fmt.Fprintf(writer, "- **Cyclomatic Complexity**: McCabe's cyclomatic complexity measures the number of linearly independent paths through a program's source code.\n")
	fmt.Fprintf(writer, "- **Cognitive Complexity**: Measures how difficult it is to understand the control flow of a function.\n\n")
	
	fmt.Fprintf(writer, "Functions with `highComplexity: true` are candidates for refactoring to improve maintainability.\n")
}
