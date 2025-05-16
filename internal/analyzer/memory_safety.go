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

// MemorySafetyOptions contains options for the memory-safety command
type MemorySafetyOptions struct {
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
	CheckLeaks      bool // Whether to check for memory leaks
	CheckUseAfter   bool // Whether to check for use-after-free
	CheckDoubleFree bool // Whether to check for double-free
	CheckOverflow   bool // Whether to check for buffer overflows
	
	// Output options
	Short   bool   // Whether to use short output format
	Verbose bool   // Whether to enable verbose output
}

// MemoryIssue represents a potential memory safety issue
type MemoryIssue struct {
	FilePath    string // Path to the file containing the issue
	Line        int    // Line number where the issue appears
	Column      int    // Column number where the issue appears
	IssueType   string // Type of issue (leak, use-after-free, etc.)
	Description string // Description of the issue
	Severity    string // Severity of the issue (high, medium, low)
	Code        string // The code snippet containing the issue
}

// AnalyzeMemorySafety analyzes code for memory safety issues
func AnalyzeMemorySafety(options MemorySafetyOptions) {
	if options.Verbose {
		fmt.Println(color.CyanString("Starting memory safety analysis..."))
	}

	// Set default values for options
	if options.Jobs <= 0 {
		options.Jobs = 4 // Default to 4 concurrent jobs
	}

	// If no specific checks are enabled, enable all
	if !options.CheckLeaks && !options.CheckUseAfter && !options.CheckDoubleFree && !options.CheckOverflow {
		options.CheckLeaks = true
		options.CheckUseAfter = true
		options.CheckDoubleFree = true
		options.CheckOverflow = true
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

	// Analyze memory safety issues in files
	issues := analyzeMemorySafetyIssues(files, options)

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
	writeMemorySafetyResults(writer, issues, options)

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)
	
	// Print summary
	fmt.Println(color.GreenString("Memory Safety Analysis Results"))
	fmt.Printf("\nFound %d potential memory safety issues in %d files\n", len(issues), len(files))
	
	// Count issues by type
	issuesByType := make(map[string]int)
	for _, issue := range issues {
		issuesByType[issue.IssueType]++
	}
	
	// Print issue counts by type
	if len(issuesByType) > 0 {
		fmt.Println("Issues by type:")
		for issueType, count := range issuesByType {
			fmt.Printf("  %s: %d\n", issueType, count)
		}
	}
	
	fmt.Printf("Processing time: %s (%.2f files/sec)\n\n", utils.FormatDuration(elapsedTime), float64(len(files))/elapsedTime.Seconds())
	
	// No recommendation prints
}

// analyzeMemorySafetyIssues analyzes memory safety issues in files
func analyzeMemorySafetyIssues(files []string, options MemorySafetyOptions) []MemoryIssue {
	var issues []MemoryIssue
	var mutex sync.Mutex
	var wg sync.WaitGroup
	
	// Set up a worker pool
	fileChan := make(chan string, len(files))
	
	// Create a progress bar
	progress := utils.NewProgressBar(len(files), "Analyzing memory safety")
	progress.Start()
	
	// Start workers
	for i := 0; i < options.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for file := range fileChan {
				// Analyze memory safety issues in this file
				fileIssues := analyzeFileMemorySafety(file, options)
				
				// Add to global list
				if len(fileIssues) > 0 {
					mutex.Lock()
					issues = append(issues, fileIssues...)
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
	
	// Sort issues by file path and line number
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].FilePath == issues[j].FilePath {
			return issues[i].Line < issues[j].Line
		}
		return issues[i].FilePath < issues[j].FilePath
	})
	
	return issues
}

// analyzeFileMemorySafety analyzes memory safety issues in a single file
func analyzeFileMemorySafety(filePath string, options MemorySafetyOptions) []MemoryIssue {
	var issues []MemoryIssue
	
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return issues
	}
	defer file.Close()
	
	// Read file content
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	
	// Check for memory leaks
	if options.CheckLeaks {
		issues = append(issues, checkMemoryLeaks(filePath, lines)...)
	}
	
	// Check for use-after-free
	if options.CheckUseAfter {
		issues = append(issues, checkUseAfterFree(filePath, lines)...)
	}
	
	// Check for double-free
	if options.CheckDoubleFree {
		issues = append(issues, checkDoubleFree(filePath, lines)...)
	}
	
	// Check for buffer overflows
	if options.CheckOverflow {
		issues = append(issues, checkBufferOverflow(filePath, lines)...)
	}
	
	return issues
}

// checkMemoryLeaks checks for potential memory leaks
func checkMemoryLeaks(filePath string, lines []string) []MemoryIssue {
	var issues []MemoryIssue
	
	// Track allocations and deallocations
	allocations := make(map[string][]int) // Variable name -> line numbers
	deallocations := make(map[string]bool) // Variable name -> deallocated
	
	// Regular expressions for memory allocations and deallocations
	mallocRegex := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*(malloc|calloc|realloc)\b`)
	newRegex := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*new\b`)
	freeRegex := regexp.MustCompile(`\bfree\s*\(\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\)`)
	deleteRegex := regexp.MustCompile(`\bdelete\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	deleteArrayRegex := regexp.MustCompile(`\bdelete\s*\[\]\s*([a-zA-Z_][a-zA-Z0-9_]*)`)
	
	// Scan for allocations and deallocations
	for i, line := range lines {
		lineNum := i + 1
		
		// Check for C-style allocations
		if matches := mallocRegex.FindStringSubmatch(line); len(matches) > 2 {
			varName := matches[1]
			allocations[varName] = append(allocations[varName], lineNum)
		}
		
		// Check for C++-style allocations
		if matches := newRegex.FindStringSubmatch(line); len(matches) > 1 {
			varName := matches[1]
			allocations[varName] = append(allocations[varName], lineNum)
		}
		
		// Check for C-style deallocations
		if matches := freeRegex.FindStringSubmatch(line); len(matches) > 1 {
			varName := matches[1]
			deallocations[varName] = true
		}
		
		// Check for C++-style deallocations
		if matches := deleteRegex.FindStringSubmatch(line); len(matches) > 1 {
			varName := matches[1]
			deallocations[varName] = true
		}
		
		// Check for C++-style array deallocations
		if matches := deleteArrayRegex.FindStringSubmatch(line); len(matches) > 1 {
			varName := matches[1]
			deallocations[varName] = true
		}
	}
	
	// Find allocations without corresponding deallocations
	for varName, allocs := range allocations {
		if !deallocations[varName] {
			for _, lineNum := range allocs {
				// Get the code snippet
				startLine := utils.Max(0, lineNum-1)
				endLine := utils.Min(len(lines), lineNum+1)
				codeSnippet := strings.Join(lines[startLine:endLine], "\n")
				
				issues = append(issues, MemoryIssue{
					FilePath:    filePath,
					Line:        lineNum,
					Column:      1, // Simplified - would need more parsing for exact column
					IssueType:   "Memory Leak",
					Description: fmt.Sprintf("Potential memory leak: '%s' is allocated but never freed", varName),
					Severity:    "High",
					Code:        codeSnippet,
				})
			}
		}
	}
	
	return issues
}

// checkUseAfterFree checks for potential use-after-free issues
func checkUseAfterFree(filePath string, lines []string) []MemoryIssue {
	var issues []MemoryIssue
	
	// Track deallocations and subsequent uses
	deallocations := make(map[string]int) // Variable name -> line number
	
	// Regular expressions for deallocations and variable uses
	freeRegex := regexp.MustCompile(`\bfree\s*\(\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\)`)
	deleteRegex := regexp.MustCompile(`\bdelete\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	deleteArrayRegex := regexp.MustCompile(`\bdelete\s*\[\]\s*([a-zA-Z_][a-zA-Z0-9_]*)`)
	
	// First pass: find deallocations
	for i, line := range lines {
		lineNum := i + 1
		
		// Check for C-style deallocations
		if matches := freeRegex.FindStringSubmatch(line); len(matches) > 1 {
			varName := matches[1]
			deallocations[varName] = lineNum
		}
		
		// Check for C++-style deallocations
		if matches := deleteRegex.FindStringSubmatch(line); len(matches) > 1 {
			varName := matches[1]
			deallocations[varName] = lineNum
		}
		
		// Check for C++-style array deallocations
		if matches := deleteArrayRegex.FindStringSubmatch(line); len(matches) > 1 {
			varName := matches[1]
			deallocations[varName] = lineNum
		}
	}
	
	// Second pass: find uses after deallocations
	for varName, freeLine := range deallocations {
		varRegex := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(varName)))
		
		for i := freeLine; i < len(lines); i++ {
			lineNum := i + 1
			
			// Skip the deallocation line itself
			if lineNum == freeLine {
				continue
			}
			
			// Check if the variable is used after being freed
			if varRegex.MatchString(lines[i]) {
				// Get the code snippet
				startLine := utils.Max(freeLine-1, 0)
				endLine := utils.Min(len(lines), lineNum+1)
				codeSnippet := strings.Join(lines[startLine:endLine], "\n")
				
				issues = append(issues, MemoryIssue{
					FilePath:    filePath,
					Line:        lineNum,
					Column:      1, // Simplified
					IssueType:   "Use-After-Free",
					Description: fmt.Sprintf("Potential use-after-free: '%s' is used after being freed at line %d", varName, freeLine),
					Severity:    "High",
					Code:        codeSnippet,
				})
				
				// Only report the first use after free for this variable
				break
			}
		}
	}
	
	return issues
}

// checkDoubleFree checks for potential double-free issues
func checkDoubleFree(filePath string, lines []string) []MemoryIssue {
	var issues []MemoryIssue
	
	// Track deallocations
	deallocations := make(map[string][]int) // Variable name -> line numbers
	
	// Regular expressions for deallocations
	freeRegex := regexp.MustCompile(`\bfree\s*\(\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\)`)
	deleteRegex := regexp.MustCompile(`\bdelete\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	deleteArrayRegex := regexp.MustCompile(`\bdelete\s*\[\]\s*([a-zA-Z_][a-zA-Z0-9_]*)`)
	
	// Find all deallocations
	for i, line := range lines {
		lineNum := i + 1
		
		// Check for C-style deallocations
		if matches := freeRegex.FindStringSubmatch(line); len(matches) > 1 {
			varName := matches[1]
			deallocations[varName] = append(deallocations[varName], lineNum)
		}
		
		// Check for C++-style deallocations
		if matches := deleteRegex.FindStringSubmatch(line); len(matches) > 1 {
			varName := matches[1]
			deallocations[varName] = append(deallocations[varName], lineNum)
		}
		
		// Check for C++-style array deallocations
		if matches := deleteArrayRegex.FindStringSubmatch(line); len(matches) > 1 {
			varName := matches[1]
			deallocations[varName] = append(deallocations[varName], lineNum)
		}
	}
	
	// Find variables with multiple deallocations
	for varName, deallocs := range deallocations {
		if len(deallocs) > 1 {
			// Get the code snippet for the second deallocation
			secondFree := deallocs[1]
			startLine := utils.Max(0, secondFree-2)
			endLine := utils.Min(len(lines), secondFree+1)
			codeSnippet := strings.Join(lines[startLine:endLine], "\n")
			
			issues = append(issues, MemoryIssue{
				FilePath:    filePath,
				Line:        secondFree,
				Column:      1, // Simplified
				IssueType:   "Double-Free",
				Description: fmt.Sprintf("Potential double-free: '%s' is freed at line %d and again at line %d", varName, deallocs[0], secondFree),
				Severity:    "High",
				Code:        codeSnippet,
			})
		}
	}
	
	return issues
}

// checkBufferOverflow checks for potential buffer overflow issues
func checkBufferOverflow(filePath string, lines []string) []MemoryIssue {
	var issues []MemoryIssue
	
	// Regular expressions for risky functions and patterns
	strcpyRegex := regexp.MustCompile(`\bstrcpy\s*\(\s*([^,]+),\s*([^)]+)\)`)
	strncpyRegex := regexp.MustCompile(`\bstrncpy\s*\(\s*([^,]+),\s*([^,]+),\s*([^)]+)\)`)
	strcatRegex := regexp.MustCompile(`\bstrcat\s*\(\s*([^,]+),\s*([^)]+)\)`)
	sprintfRegex := regexp.MustCompile(`\bsprintf\s*\(\s*([^,]+),`)
	getsRegex := regexp.MustCompile(`\bgets\s*\(\s*([^)]+)\)`)
	
	// Check for risky function calls
	for i, line := range lines {
		lineNum := i + 1
		
		// Check for strcpy (unsafe)
		if strcpyRegex.MatchString(line) {
			// Get the code snippet
			startLine := utils.Max(0, lineNum-1)
			endLine := utils.Min(len(lines), lineNum+1)
			codeSnippet := strings.Join(lines[startLine:endLine], "\n")
			
			issues = append(issues, MemoryIssue{
				FilePath:    filePath,
				Line:        lineNum,
				Column:      1, // Simplified
				IssueType:   "Buffer Overflow",
				Description: "Potential buffer overflow: 'strcpy' used without bounds checking",
				Severity:    "High",
				Code:        codeSnippet,
			})
		}
		
		// Check for strncpy (safer but still risky if size is incorrect)
		if matches := strncpyRegex.FindStringSubmatch(line); len(matches) > 3 {
			// If the size parameter is a literal number, it might be too small
			sizeParam := matches[3]
			if _, err := fmt.Sscanf(sizeParam, "%d", new(int)); err == nil {
				// Get the code snippet
				startLine := utils.Max(0, lineNum-1)
				endLine := utils.Min(len(lines), lineNum+1)
				codeSnippet := strings.Join(lines[startLine:endLine], "\n")
				
				issues = append(issues, MemoryIssue{
					FilePath:    filePath,
					Line:        lineNum,
					Column:      1, // Simplified
					IssueType:   "Buffer Overflow",
					Description: "Potential buffer overflow: 'strncpy' used with hardcoded size",
					Severity:    "Medium",
					Code:        codeSnippet,
				})
			}
		}
		
		// Check for strcat (unsafe)
		if strcatRegex.MatchString(line) {
			// Get the code snippet
			startLine := utils.Max(0, lineNum-1)
			endLine := utils.Min(len(lines), lineNum+1)
			codeSnippet := strings.Join(lines[startLine:endLine], "\n")
			
			issues = append(issues, MemoryIssue{
				FilePath:    filePath,
				Line:        lineNum,
				Column:      1, // Simplified
				IssueType:   "Buffer Overflow",
				Description: "Potential buffer overflow: 'strcat' used without bounds checking",
				Severity:    "High",
				Code:        codeSnippet,
			})
		}
		
		// Check for sprintf (unsafe)
		if sprintfRegex.MatchString(line) {
			// Get the code snippet
			startLine := utils.Max(0, lineNum-1)
			endLine := utils.Min(len(lines), lineNum+1)
			codeSnippet := strings.Join(lines[startLine:endLine], "\n")
			
			issues = append(issues, MemoryIssue{
				FilePath:    filePath,
				Line:        lineNum,
				Column:      1, // Simplified
				IssueType:   "Buffer Overflow",
				Description: "Potential buffer overflow: 'sprintf' used without bounds checking",
				Severity:    "High",
				Code:        codeSnippet,
			})
		}
		
		// Check for gets (very unsafe, deprecated)
		if getsRegex.MatchString(line) {
			// Get the code snippet
			startLine := utils.Max(0, lineNum-1)
			endLine := utils.Min(len(lines), lineNum+1)
			codeSnippet := strings.Join(lines[startLine:endLine], "\n")
			
			issues = append(issues, MemoryIssue{
				FilePath:    filePath,
				Line:        lineNum,
				Column:      1, // Simplified
				IssueType:   "Buffer Overflow",
				Description: "Potential buffer overflow: 'gets' is unsafe and deprecated",
				Severity:    "Critical",
				Code:        codeSnippet,
			})
		}
	}
	
	return issues
}

// writeMemorySafetyResults writes memory safety analysis results to the output file
func writeMemorySafetyResults(writer *bufio.Writer, issues []MemoryIssue, options MemorySafetyOptions) {
	// Write header
	fmt.Fprintf(writer, "# Memory Safety Analysis Results\n\n")
	
	// Write summary
	fmt.Fprintf(writer, "## Summary\n\n")
	fmt.Fprintf(writer, "- Total issues found: %d\n", len(issues))
	
	// Count issues by type
	issuesByType := make(map[string]int)
	for _, issue := range issues {
		issuesByType[issue.IssueType]++
	}
	
	// Write issue counts by type
	if len(issuesByType) > 0 {
		fmt.Fprintf(writer, "### Issues by Type\n\n")
		for issueType, count := range issuesByType {
			fmt.Fprintf(writer, "- %s: %d\n", issueType, count)
		}
		fmt.Fprintf(writer, "\n")
	}
	
	// Count issues by severity
	issuesBySeverity := make(map[string]int)
	for _, issue := range issues {
		issuesBySeverity[issue.Severity]++
	}
	
	// Write issue counts by severity
	if len(issuesBySeverity) > 0 {
		fmt.Fprintf(writer, "### Issues by Severity\n\n")
		
		// Order severities
		severities := []string{"Critical", "High", "Medium", "Low"}
		for _, severity := range severities {
			if count, ok := issuesBySeverity[severity]; ok {
				fmt.Fprintf(writer, "- %s: %d\n", severity, count)
			}
		}
		fmt.Fprintf(writer, "\n")
	}
	
	// Group issues by file
	issuesByFile := make(map[string][]MemoryIssue)
	for _, issue := range issues {
		issuesByFile[issue.FilePath] = append(issuesByFile[issue.FilePath], issue)
	}
	
	// Write detailed issues by file
	fmt.Fprintf(writer, "## Detailed Issues\n\n")
	
	// Sort files for consistent output
	files := make([]string, 0, len(issuesByFile))
	for file := range issuesByFile {
		files = append(files, file)
	}
	sort.Strings(files)
	
	for _, file := range files {
		relPath, _ := filepath.Rel(options.Directory, file)
		fmt.Fprintf(writer, "### %s\n\n", relPath)
		
		// Sort issues by line number
		sort.Slice(issuesByFile[file], func(i, j int) bool {
			return issuesByFile[file][i].Line < issuesByFile[file][j].Line
		})
		
		for _, issue := range issuesByFile[file] {
			// Format severity with appropriate color
			severityStr := issue.Severity
			switch issue.Severity {
			case "Critical":
				severityStr = "**Critical**"
			case "High":
				severityStr = "**High**"
			}
			
			fmt.Fprintf(writer, "#### %s at line %d\n\n", issue.IssueType, issue.Line)
			fmt.Fprintf(writer, "- **Severity**: %s\n", severityStr)
			fmt.Fprintf(writer, "- **Description**: %s\n\n", issue.Description)
			
			// Write code snippet
			fmt.Fprintf(writer, "```c\n%s\n```\n\n", issue.Code)
		}
	}
	
	// No recommendations section
	
	// No buffer overflow recommendations
}

// Helper functions are now in utils package
