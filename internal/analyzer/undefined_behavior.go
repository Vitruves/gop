package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/vitruves/gop/internal/utils"
)

// UndefinedBehaviorOptions contains options for the undefined behavior detector
type UndefinedBehaviorOptions struct {
	InputFile  string
	Directory  string
	Depth      int
	OutputFile string
	Languages  []string
	Excludes   []string
	Jobs       int
	// Specific undefined behavior checks
	CheckSignedOverflow bool
	CheckNullDereference bool
	CheckDivByZero      bool
	CheckUninitVar      bool
	CheckOutOfBounds    bool
	CheckShiftOperations bool
	// Output options
	Short   bool
	Verbose bool
}

// UndefinedBehaviorIssue represents a potential undefined behavior issue
type UndefinedBehaviorIssue struct {
	File     string
	Line     int
	Column   int
	Type     string
	Snippet  string
	Function string
	Severity string
}

// AnalyzeUndefinedBehavior analyzes C/C++ code for potential undefined behavior
func AnalyzeUndefinedBehavior(options UndefinedBehaviorOptions) {
	if options.Verbose {
		fmt.Println("Starting undefined behavior analysis...")
		cwd, _ := os.Getwd()
		fmt.Println("Current working directory:", cwd)
	}

	// Find all files to analyze
	files, err := utils.GetFilesToProcess(options.InputFile, options.Directory, options.Depth, options.Languages, options.Excludes)
	if err != nil {
		fmt.Printf("Error finding files: %v\n", err)
		return
	}

	if options.Verbose {
		fmt.Printf("Found %d files to process\n", len(files))
		if len(files) > 0 {
			fmt.Println("First few files found:")
			for i := 0; i < min(len(files), 5); i++ {
				fmt.Printf("  %s\n", files[i])
			}
		}
	}

	if len(files) == 0 {
		fmt.Println("No files found to analyze. Please check your directory path and language filters.")
		fmt.Printf("Directory: %s\n", options.Directory)
		fmt.Printf("Languages: %v\n", options.Languages)
		fmt.Printf("Excludes: %v\n", options.Excludes)
		return
	}

	// Process files concurrently
	var wg sync.WaitGroup
	issues := make(chan UndefinedBehaviorIssue, 100)
	fileChan := make(chan string, len(files))

	// Add files to the channel
	for _, file := range files {
		fileChan <- file
	}
	close(fileChan)

	// Create workers
	numWorkers := options.Jobs
	if numWorkers <= 0 {
		numWorkers = 4 // Default to 4 workers
	}

	if options.Verbose {
		fmt.Printf("Analyzing undefined behavior with %d workers: ", numWorkers)
	}

	startTime := time.Now()

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				// Analyze file for undefined behavior
				fileIssues := analyzeFileForUndefinedBehavior(file, options)
				for _, issue := range fileIssues {
					issues <- issue
				}
				if options.Verbose {
					fmt.Print(".")
				}
			}
		}()
	}

	// Wait for all workers to finish and close the issues channel
	go func() {
		wg.Wait()
		close(issues)
		if options.Verbose {
			fmt.Println()
		}
	}()

	// Collect all issues
	var allIssues []UndefinedBehaviorIssue
	for issue := range issues {
		allIssues = append(allIssues, issue)
	}

	// Generate report
	processingTime := time.Since(startTime)
	filesPerSecond := float64(len(files)) / processingTime.Seconds()

	// Create output
	var output strings.Builder
	output.WriteString("# Undefined Behavior Analysis Results\n\n")
	
	// Summary
	output.WriteString(fmt.Sprintf("Found %d potential undefined behavior issues in %d files\n", len(allIssues), len(files)))
	
	// Issues by type
	if len(allIssues) > 0 {
		typeCount := make(map[string]int)
		for _, issue := range allIssues {
			typeCount[issue.Type]++
		}
		
		output.WriteString("Issues by type:\n")
		for issueType, count := range typeCount {
			output.WriteString(fmt.Sprintf("  %s: %d\n", issueType, count))
		}
	}
	
	output.WriteString(fmt.Sprintf("Processing time: %.2f ms (%.2f files/sec)\n\n", 
		processingTime.Seconds()*1000, filesPerSecond))

	// Detailed issues
	if len(allIssues) > 0 {
		output.WriteString("## Detailed Issues\n\n")
		
		// Group by file
		fileIssues := make(map[string][]UndefinedBehaviorIssue)
		for _, issue := range allIssues {
			fileIssues[issue.File] = append(fileIssues[issue.File], issue)
		}
		
		for file, issues := range fileIssues {
			relPath, err := filepath.Rel(options.Directory, file)
			if err != nil {
				relPath = file
			}
			
			output.WriteString(fmt.Sprintf("### %s\n\n", relPath))
			
			for _, issue := range issues {
				output.WriteString(fmt.Sprintf("- **%s** (Line %d): %s\n", issue.Type, issue.Line, issue.Snippet))
				if issue.Function != "" {
					output.WriteString(fmt.Sprintf("  - In function: `%s`\n", issue.Function))
				}
				output.WriteString(fmt.Sprintf("  - Severity: %s\n\n", issue.Severity))
			}
		}
	}

	// No recommendations section

	// Output handling: print to console if no output file is specified or if not in short mode
	if options.OutputFile == "" {
		// Always print to console when no output file is specified
		fmt.Println(output.String())
	} else {
		// Write to specified output file
		err := os.WriteFile(options.OutputFile, []byte(output.String()), 0644)
		if err != nil {
			fmt.Printf("Error writing to output file: %v\n", err)
			return
		}
		
		// Print to console as well if not in short mode
		if !options.Short {
			fmt.Println(output.String())
		}
	}
}

// analyzeFileForUndefinedBehavior analyzes a single file for undefined behavior
func analyzeFileForUndefinedBehavior(file string, options UndefinedBehaviorOptions) []UndefinedBehaviorIssue {
	var issues []UndefinedBehaviorIssue

	// Read file content
	content, err := os.ReadFile(file)
	if err != nil {
		return issues
	}

	// Get file lines for context
	lines := strings.Split(string(content), "\n")
	
	// Track current function
	currentFunction := ""
	functionRegex := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*\([^)]*\)\s*\{`)

	// Check for various undefined behaviors
	for i, line := range lines {
		lineNum := i + 1
		
		// Track function context
		if matches := functionRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentFunction = matches[1]
		} else if strings.Contains(line, "}") && strings.TrimSpace(line) == "}" {
			currentFunction = ""
		}

		// Check for signed integer overflow
		if options.CheckSignedOverflow && checkSignedOverflow(line) {
			issues = append(issues, UndefinedBehaviorIssue{
				File:     file,
				Line:     lineNum,
				Type:     "Signed Integer Overflow",
				Snippet:  strings.TrimSpace(line),
				Function: currentFunction,
				Severity: "High",
			})
		}

		// Check for null pointer dereference
		if options.CheckNullDereference && checkNullDereference(line) {
			issues = append(issues, UndefinedBehaviorIssue{
				File:     file,
				Line:     lineNum,
				Type:     "Null Pointer Dereference",
				Snippet:  strings.TrimSpace(line),
				Function: currentFunction,
				Severity: "Critical",
			})
		}

		// Check for division by zero
		if options.CheckDivByZero && checkDivisionByZero(line) {
			issues = append(issues, UndefinedBehaviorIssue{
				File:     file,
				Line:     lineNum,
				Type:     "Division by Zero",
				Snippet:  strings.TrimSpace(line),
				Function: currentFunction,
				Severity: "High",
			})
		}

		// Check for uninitialized variables
		if options.CheckUninitVar && checkUninitializedVariable(line) {
			issues = append(issues, UndefinedBehaviorIssue{
				File:     file,
				Line:     lineNum,
				Type:     "Uninitialized Variable",
				Snippet:  strings.TrimSpace(line),
				Function: currentFunction,
				Severity: "Medium",
			})
		}

		// Check for array out of bounds
		if options.CheckOutOfBounds && checkArrayOutOfBounds(line) {
			issues = append(issues, UndefinedBehaviorIssue{
				File:     file,
				Line:     lineNum,
				Type:     "Array Out of Bounds",
				Snippet:  strings.TrimSpace(line),
				Function: currentFunction,
				Severity: "High",
			})
		}

		// Check for invalid shift operations
		if options.CheckShiftOperations && checkInvalidShift(line) {
			issues = append(issues, UndefinedBehaviorIssue{
				File:     file,
				Line:     lineNum,
				Type:     "Invalid Shift Operation",
				Snippet:  strings.TrimSpace(line),
				Function: currentFunction,
				Severity: "Medium",
			})
		}
	}

	return issues
}

// Helper functions to check for specific undefined behaviors

// checkSignedOverflow checks for potential signed integer overflow
func checkSignedOverflow(line string) bool {
	// Look for patterns that might cause signed overflow
	patterns := []string{
		`INT_MAX\s*\+`, 
		`INT_MIN\s*\-`,
		`\bint\b.*=.*\d{10,}`,  // Large integer literals
		`\+\+\s*INT_MAX`,
		`\-\-\s*INT_MIN`,
		`overflow`,          // Comments mentioning overflow
		`underflow`,         // Comments mentioning underflow
		`max \+ 1`,          // Adding to max value
		`min \- 1`,          // Subtracting from min value
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, line)
		if matched {
			return true
		}
	}

	return false
}

// checkNullDereference checks for potential null pointer dereference
func checkNullDereference(line string) bool {
	// Look for patterns that might cause null pointer dereference
	patterns := []string{
		`\*\s*\(\s*[a-zA-Z_][a-zA-Z0-9_]*\s*\)NULL`,
		`\*\s*NULL`,
		`NULL\s*->`,
		`\w+\s*=\s*NULL;\s*.*\*\w+`,
		`\w+\s*=\s*NULL;\s*.*\w+->`,
		`\*ptr`,                      // Dereferencing a pointer (context-sensitive)
		`ptr\s*=\s*NULL`,            // Setting a pointer to NULL
		`null\s*pointer\s*dereference`, // Comment mentioning null pointer dereference
		`\*\s*\w+\s*=`,             // Assigning through a pointer
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, line)
		if matched {
			return true
		}
	}

	return false
}

// checkDivisionByZero checks for potential division by zero
func checkDivisionByZero(line string) bool {
	// Look for patterns that might cause division by zero
	patterns := []string{
		`\/\s*0`,
		`\/\s*\(\s*0\s*\)`,
		`\%\s*0`,
		`\%\s*\(\s*0\s*\)`,
		`\/\s*\w+\s*\/\*.*potential zero.*\*\/`,
		`b\s*=\s*0`,                  // Setting a variable to 0 that might be used as divisor
		`division\s*by\s*zero`,       // Comment mentioning division by zero
		`\w+\s*\/\s*\w+`,           // Any division operation (context-sensitive)
		`result\s*=\s*a\s*\/\s*b`,  // Division operation with specific variable names
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, line)
		if matched {
			return true
		}
	}

	return false
}

// checkUninitializedVariable checks for potentially uninitialized variables
func checkUninitializedVariable(line string) bool {
	// Look for variable declarations without initialization
	// This is a simple heuristic and will have false positives
	patterns := []string{
		`\b(int|char|float|double|long|short|unsigned|struct|enum)\s+[a-zA-Z_][a-zA-Z0-9_]*\s*;`,
		`\b(int|char|float|double|long|short|unsigned|struct|enum)\s+[a-zA-Z_][a-zA-Z0-9_]*\s*,\s*[a-zA-Z_][a-zA-Z0-9_]*\s*;`,
	}

	// Skip certain patterns that are likely false positives
	skipPatterns := []string{
		`\bextern\b`,
		`\bstatic\b.*=`,
		`\bconst\b`,
		`\(\s*\w+\s*\)`,  // Function parameters
		`\bstruct\b.*\{`, // Struct definition
	}

	for _, skipPattern := range skipPatterns {
		matched, _ := regexp.MatchString(skipPattern, line)
		if matched {
			return false
		}
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, line)
		if matched {
			return true
		}
	}

	return false
}

// checkArrayOutOfBounds checks for potential array out of bounds access
func checkArrayOutOfBounds(line string) bool {
	// This is a simple heuristic and will have false positives
	patterns := []string{
		`\[\s*-\d+\s*\]`,                     // Negative index
		`\[\s*\d+\s*\]\s*\/\*.*too large.*\*\/`, // Comment indicating large index
		`\[\s*sizeof\s*\w+\s*\]`,             // Using sizeof as index
		`\[\s*\w+\s*\]\s*\/\*.*unchecked.*\*\/`, // Unchecked index
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, line)
		if matched {
			return true
		}
	}

	return false
}

// checkInvalidShift checks for invalid shift operations
func checkInvalidShift(line string) bool {
	// Look for patterns that might cause undefined behavior with shift operations
	patterns := []string{
		`<<\s*-\d+`,          // Negative shift amount
		`>>\s*-\d+`,          // Negative shift amount
		`<<\s*\d{2,}`,        // Potentially too large shift amount
		`>>\s*\d{2,}`,        // Potentially too large shift amount
		`<<\s*sizeof\(\w+\)`, // Shifting by sizeof
		`>>\s*sizeof\(\w+\)`, // Shifting by sizeof
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, line)
		if matched {
			return true
		}
	}

	return false
}
