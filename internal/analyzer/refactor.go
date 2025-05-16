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

// RefactorOptions contains options for the refactor command
type RefactorOptions struct {
	// Input/Output options
	InputFile  string   // Path to input file containing list of files to process
	Directory  string   // Root directory to analyze
	Depth      int      // Maximum depth for directory traversal
	OutputFile string   // Path to output file for results
	Languages  []string // Languages to analyze (e.g., "c", "cpp")
	Excludes   []string // Directories or files to exclude

	// Refactoring options
	Pattern     string // Pattern to search for
	Replacement string // Replacement for the pattern
	RegexMode   bool   // Whether to use regex for pattern matching
	WholeWord   bool   // Whether to match whole words only
	CaseSensitive bool // Whether to use case-sensitive matching
	
	// Processing options
	Jobs     int  // Number of concurrent jobs for processing
	DryRun   bool // Whether to perform a dry run (no actual changes)
	Backup   bool // Whether to create backup files before making changes
	
	// Output options
	Verbose bool // Whether to enable verbose output
}

// RefactorResult represents the result of a refactoring operation
type RefactorResult struct {
	FilePath        string // Path to the file
	MatchCount      int    // Number of matches found
	ReplacementCount int   // Number of replacements made
	LineNumbers     []int  // Line numbers where replacements were made
	Error           error  // Error encountered during refactoring
}

// RunRefactor runs the refactoring operation with the given options
func RunRefactor(options RefactorOptions) {
	startTime := time.Now()

	// Validate options
	if options.Pattern == "" {
		fmt.Println(color.RedString("Error: Pattern cannot be empty"))
		return
	}

	// Get files to process
	files, err := utils.GetFilesToProcess(options.InputFile, options.Directory, options.Depth, options.Languages, options.Excludes)
	if err != nil {
		fmt.Println(color.RedString("Error getting files to process:"), err)
		return
	}

	if options.Verbose {
		fmt.Printf(color.CyanString("Found %d files to process\n"), len(files))
	}

	// Compile regex pattern if in regex mode
	var pattern *regexp.Regexp
	if options.RegexMode {
		flags := ""
		if !options.CaseSensitive {
			flags = "(?i)"
		}
		
		pattern, err = regexp.Compile(flags + options.Pattern)
		if err != nil {
			fmt.Println(color.RedString("Error compiling regex pattern:"), err)
			return
		}
	} else {
		// For non-regex mode, escape special characters
		escapedPattern := regexp.QuoteMeta(options.Pattern)
		
		// Add word boundaries if whole word matching is enabled
		if options.WholeWord {
			escapedPattern = "\\b" + escapedPattern + "\\b"
		}
		
		// Add case insensitivity flag if needed
		flags := ""
		if !options.CaseSensitive {
			flags = "(?i)"
		}
		
		pattern, err = regexp.Compile(flags + escapedPattern)
		if err != nil {
			fmt.Println(color.RedString("Error compiling pattern:"), err)
			return
		}
	}

	// Process files and perform refactoring
	var processedFiles atomic.Int64
	results := performRefactoring(files, pattern, options, &processedFiles)

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
	writeRefactorOutput(writer, results, files, options, elapsedTime, processingSpeed)
}

// performRefactoring performs refactoring on the given files
func performRefactoring(files []string, pattern *regexp.Regexp, options RefactorOptions, processedFiles *atomic.Int64) []RefactorResult {
	var results []RefactorResult
	var mutex sync.Mutex
	var wg sync.WaitGroup

	// Create a channel for files to process
	filesChan := make(chan string, len(files))
	for _, file := range files {
		filesChan <- file
	}
	close(filesChan)

	// Create a progress bar
	progress := utils.NewProgressBar(len(files), "Refactoring files")
	progress.Start()

	// Create a channel for results
	resultsChan := make(chan RefactorResult, options.Jobs)

	// Process files concurrently
	for i := 0; i < options.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for file := range filesChan {
				// Refactor this file
				result := refactorFile(file, pattern, options)

				// Send results to the channel
				resultsChan <- result

				// Update progress
				processedFiles.Add(1)
				progress.Increment()
			}
		}()
	}

	// Start a goroutine to collect results
	go func() {
		for result := range resultsChan {
			mutex.Lock()
			results = append(results, result)
			mutex.Unlock()
		}
	}()

	// Wait for all processing to complete
	wg.Wait()
	close(resultsChan)
	progress.Finish()

	return results
}

// refactorFile refactors a single file
func refactorFile(filePath string, pattern *regexp.Regexp, options RefactorOptions) RefactorResult {
	result := RefactorResult{
		FilePath: filePath,
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		result.Error = err
		return result
	}

	// Split content into lines for line number tracking
	lines := strings.Split(string(content), "\n")
	
	// Track matches and replacements
	matchCount := 0
	replacementCount := 0
	lineNumbers := make([]int, 0)
	
	// Process each line
	for i, line := range lines {
		matches := pattern.FindAllStringIndex(line, -1)
		if len(matches) > 0 {
			matchCount += len(matches)
			lineNumbers = append(lineNumbers, i+1)
			
			// Perform replacement if not in dry run mode
			if !options.DryRun {
				lines[i] = pattern.ReplaceAllString(line, options.Replacement)
				replacementCount += len(matches)
			}
		}
	}
	
	// Update result
	result.MatchCount = matchCount
	result.ReplacementCount = replacementCount
	result.LineNumbers = lineNumbers
	
	// If no matches found, return early
	if matchCount == 0 {
		return result
	}
	
	// If in dry run mode, return without modifying the file
	if options.DryRun {
		return result
	}
	
	// Create backup if requested
	if options.Backup {
		backupPath := filePath + ".bak"
		err = os.WriteFile(backupPath, content, 0644)
		if err != nil {
			result.Error = fmt.Errorf("failed to create backup: %w", err)
			return result
		}
	}
	
	// Write modified content back to file
	modifiedContent := strings.Join(lines, "\n")
	err = os.WriteFile(filePath, []byte(modifiedContent), 0644)
	if err != nil {
		result.Error = fmt.Errorf("failed to write modified content: %w", err)
		return result
	}
	
	return result
}

// writeRefactorOutput writes refactoring output to the given writer
func writeRefactorOutput(writer *bufio.Writer, results []RefactorResult, files []string, options RefactorOptions, elapsedTime time.Duration, processingSpeed float64) {
	// Calculate summary statistics
	totalMatches := 0
	totalReplacements := 0
	filesWithMatches := 0
	filesWithErrors := 0
	
	for _, result := range results {
		totalMatches += result.MatchCount
		totalReplacements += result.ReplacementCount
		
		if result.MatchCount > 0 {
			filesWithMatches++
		}
		
		if result.Error != nil {
			filesWithErrors++
		}
	}
	
	// Write header
	fmt.Fprintf(writer, "# Refactoring Report\n\n")
	fmt.Fprintf(writer, "*Generated on %s*\n\n", time.Now().Format("2006-01-02 15:04:05"))
	
	// Write summary
	fmt.Fprintf(writer, "## Summary\n\n")
	fmt.Fprintf(writer, "| Metric | Value |\n")
	fmt.Fprintf(writer, "|--------|-------|\n")
	fmt.Fprintf(writer, "| Pattern | `%s` |\n", options.Pattern)
	fmt.Fprintf(writer, "| Replacement | `%s` |\n", options.Replacement)
	fmt.Fprintf(writer, "| Mode | %s |\n", getRefactorMode(options))
	fmt.Fprintf(writer, "| Files Processed | %d |\n", len(files))
	fmt.Fprintf(writer, "| Files with Matches | %d |\n", filesWithMatches)
	fmt.Fprintf(writer, "| Files with Errors | %d |\n", filesWithErrors)
	fmt.Fprintf(writer, "| Total Matches | %d |\n", totalMatches)
	
	if !options.DryRun {
		fmt.Fprintf(writer, "| Total Replacements | %d |\n", totalReplacements)
	}
	
	// Write details for files with matches
	if filesWithMatches > 0 {
		fmt.Fprintf(writer, "\n## Files with Matches\n\n")
		
		if options.DryRun {
			fmt.Fprintf(writer, "**Note: This was a dry run. No files were modified.**\n\n")
		}
		
		fmt.Fprintf(writer, "| File | Matches | Line Numbers |\n")
		fmt.Fprintf(writer, "|------|---------|-------------|\n")
		
		for _, result := range results {
			if result.MatchCount > 0 {
				// Format line numbers
				lineNumbersStr := formatLineNumbers(result.LineNumbers)
				
				fmt.Fprintf(writer, "| `%s` | %d | %s |\n",
					result.FilePath,
					result.MatchCount,
					lineNumbersStr)
			}
		}
	}
	
	// Write errors if any
	if filesWithErrors > 0 {
		fmt.Fprintf(writer, "\n## Errors\n\n")
		fmt.Fprintf(writer, "| File | Error |\n")
		fmt.Fprintf(writer, "|------|-------|\n")
		
		for _, result := range results {
			if result.Error != nil {
				fmt.Fprintf(writer, "| `%s` | %s |\n",
					result.FilePath,
					result.Error.Error())
			}
		}
	}
	
	// Write processing info
	fmt.Fprintf(writer, "\n## Processing Information\n\n")
	fmt.Fprintf(writer, "- Processing time: %s\n", elapsedTime.Round(time.Millisecond))
	fmt.Fprintf(writer, "- Processing speed: %.2f files/sec\n", processingSpeed)
}

// getRefactorMode returns a string describing the refactoring mode
func getRefactorMode(options RefactorOptions) string {
	mode := ""
	
	if options.RegexMode {
		mode += "Regex"
	} else {
		mode += "Literal"
	}
	
	if options.WholeWord {
		mode += ", Whole Word"
	}
	
	if options.CaseSensitive {
		mode += ", Case Sensitive"
	} else {
		mode += ", Case Insensitive"
	}
	
	return mode
}

// formatLineNumbers formats a slice of line numbers for display
func formatLineNumbers(lineNumbers []int) string {
	if len(lineNumbers) == 0 {
		return ""
	}
	
	// If there are too many line numbers, show a summary
	if len(lineNumbers) > 10 {
		return fmt.Sprintf("%d, %d, %d, ... (%d more)",
			lineNumbers[0],
			lineNumbers[1],
			lineNumbers[2],
			len(lineNumbers)-3)
	}
	
	// Convert line numbers to strings
	lineNumbersStr := make([]string, len(lineNumbers))
	for i, lineNumber := range lineNumbers {
		lineNumbersStr[i] = fmt.Sprintf("%d", lineNumber)
	}
	
	return strings.Join(lineNumbersStr, ", ")
}
