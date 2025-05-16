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
	"github.com/agnivade/levenshtein"
)

// DuplicateOptions contains options for the duplicate command
type DuplicateOptions struct {
	// Input/Output options
	InputFile  string   // Path to input file containing list of files to process
	Directory  string   // Root directory to analyze
	Depth      int      // Maximum depth for directory traversal
	OutputFile string   // Path to output file for results
	Languages  []string // Languages to analyze (e.g., "c", "cpp")
	Excludes   []string // Directories or files to exclude
	
	// Processing options
	Jobs                int     // Number of concurrent jobs for processing
	SimilarityThreshold float64 // Threshold for similarity detection (0.0-1.0)
	MinLineCount        int     // Minimum number of lines for a code block to be considered
	
	// Output options
	Short     bool // Whether to use short output format
	NamesOnly bool // Whether to show only method/function names
	Verbose   bool // Whether to enable verbose output
	
	// Monitoring options
	Monitor        bool   // Whether to enable monitoring of duplication over time
	MonitorFile    string // Path to the monitoring history file
	MonitorComment string // Optional comment for this monitoring run
}

// CodeBlock represents a block of code to be analyzed for duplication
type CodeBlock struct {
	FilePath    string   // Path to the file containing the block
	StartLine   int      // Starting line number
	EndLine     int      // Ending line number
	Content     string   // Content of the block
	ContentHash uint64   // Hash of the content for quick comparison
	Lines       []string // Individual lines of the block
}

// DuplicatePair represents a pair of duplicate code blocks
type DuplicatePair struct {
	Block1     CodeBlock // First code block
	Block2     CodeBlock // Second code block
	Similarity float64   // Similarity score (0.0-1.0)
}

// FindDuplicates finds duplicate code blocks in the codebase
func FindDuplicates(options DuplicateOptions) {
	if options.Verbose {
		fmt.Println(color.CyanString("Starting duplicate code detection..."))
	}

	// Set default values for options
	if options.SimilarityThreshold <= 0 || options.SimilarityThreshold > 1.0 {
		options.SimilarityThreshold = 0.8 // Default to 80% similarity
	}

	if options.MinLineCount <= 0 {
		options.MinLineCount = 5 // Default to minimum 5 lines
	}

	if options.Jobs <= 0 {
		options.Jobs = 4 // Default to 4 concurrent jobs
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

	// If NamesOnly is set, we'll analyze method/function names instead of code blocks
	var duplicates []DuplicatePair
	var blocks []CodeBlock
	
	if options.NamesOnly {
		// Extract and analyze method names
		fmt.Println(color.CyanString("Analyzing method and function names..."))
		duplicates, blocks = findDuplicateMethodNames(files, options)
		
		if options.Verbose {
			fmt.Printf(color.CyanString("Found %d duplicate method/function names\n"), len(duplicates))
		}
	} else {
		// Extract code blocks from files
		blocks = extractCodeBlocks(files, options)

		if options.Verbose {
			fmt.Printf(color.CyanString("Extracted %d code blocks\n"), len(blocks))
		}

		// Find duplicate blocks
		fmt.Println(color.CyanString("Finding duplicate blocks..."))
		duplicates = findDuplicateBlocks(blocks, options)

		if options.Verbose {
			fmt.Printf(color.CyanString("Found %d duplicate pairs\n"), len(duplicates))
		}
	}

	// Determine if we should output to file or console
	var writer *bufio.Writer
	var outputFile *os.File
	
	if options.OutputFile == "" {
		// Output to console
		writer = bufio.NewWriter(os.Stdout)
	} else {
		// Create output file
		outputFile, err = os.Create(options.OutputFile)
		if err != nil {
			fmt.Println(color.RedString("Error creating output file:"), err)
			return
		}
		defer outputFile.Close()
		writer = bufio.NewWriter(outputFile)
	}
	defer writer.Flush()

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)

	// Write results to file
	writeDuplicateResults(writer, duplicates, options, elapsedTime)

	// Print summary to console
	fmt.Println(color.GreenString("Duplicate analysis completed in %s", elapsedTime))
	fmt.Printf("Found %s duplicate code blocks with similarity threshold %.0f%%\n", 
		color.YellowString("%d", len(duplicates)), options.SimilarityThreshold*100)
	
	// No recommendation prints
	
	// Save monitoring data if enabled
	if options.Monitor {
		// Calculate duplication rate
		duplicationRate := 0.0
		if len(blocks) > 0 {
			duplicationRate = float64(len(duplicates)) / float64(len(blocks))
		}
		
		// Create metrics
		metrics := DuplicationMetrics{
			Timestamp:       time.Now(),
			TotalFiles:      len(files),
			TotalBlocks:     len(blocks),
			DuplicatePairs:  len(duplicates),
			DuplicationRate: duplicationRate,
			Comment:         options.MonitorComment,
			Directory:       options.Directory,
			Threshold:       options.SimilarityThreshold,
			MinLineCount:    options.MinLineCount,
		}
		
		// Save metrics
		if err := SaveDuplicationMetrics(metrics, options.MonitorFile); err != nil {
			fmt.Println(color.RedString("Error saving monitoring data: %v", err))
		} else {
			fmt.Printf("Monitoring data saved to %s\n", color.CyanString(options.MonitorFile))
			
			// Print trend
			if err := PrintDuplicationTrend(options.MonitorFile); err != nil {
				fmt.Println(color.YellowString("Could not print duplication trend: %v", err))
			}
		}
	}
}

// extractCodeBlocks extracts code blocks from files
func extractCodeBlocks(files []string, options DuplicateOptions) []CodeBlock {
	var blocks []CodeBlock
	var mutex sync.Mutex
	var wg sync.WaitGroup

	// Create a channel for files to process
	filesChan := make(chan string, len(files))
	for _, file := range files {
		filesChan <- file
	}
	close(filesChan)

	// Set up progress tracking
	progress := utils.NewProgressBar(len(files), "Extracting code blocks")
	progress.Start()

	// Process files concurrently
	for i := 0; i < options.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for file := range filesChan {
				// Read file content
				content, err := os.ReadFile(file)
				if err != nil {
					if options.Verbose {
						fmt.Printf(color.RedString("Error reading file %s: %v\n"), file, err)
					}
					progress.Increment()
					continue
				}

				// Split content into lines
				lines := strings.Split(string(content), "\n")

				// Extract code blocks
				fileBlocks := extractBlocksFromFile(file, lines, options)

				// Add blocks to the global list
				mutex.Lock()
				blocks = append(blocks, fileBlocks...)
				mutex.Unlock()
				
				// Update progress - one dot per file
				progress.Increment()
			}
		}()
	}
	
	// Wait for all files to be processed
	wg.Wait()
	
	// Finish the progress indicator
	progress.Finish()

	return blocks
}

// extractBlocksFromFile extracts code blocks from a single file
func extractBlocksFromFile(filePath string, lines []string, options DuplicateOptions) []CodeBlock {
	var blocks []CodeBlock
	
	// Simple sliding window approach
	for i := 0; i <= len(lines)-options.MinLineCount; i++ {
		// Create blocks of different sizes
		for size := options.MinLineCount; size <= 30 && i+size <= len(lines); size += 5 {
			blockLines := lines[i : i+size]
			
			// Skip blocks with too many empty lines
			emptyLines := 0
			for _, line := range blockLines {
				if strings.TrimSpace(line) == "" {
					emptyLines++
				}
			}
			
			if float64(emptyLines)/float64(len(blockLines)) > 0.3 {
				continue // Skip if more than 30% of lines are empty
			}
			
			// Create a code block
			content := strings.Join(blockLines, "\n")
			block := CodeBlock{
				FilePath:  filePath,
				StartLine: i + 1, // 1-indexed line numbers
				EndLine:   i + size,
				Content:   content,
				Lines:     blockLines,
			}
			
			blocks = append(blocks, block)
		}
	}
	
	return blocks
}

// findDuplicateBlocks finds duplicate code blocks
func findDuplicateBlocks(blocks []CodeBlock, options DuplicateOptions) []DuplicatePair {
	var duplicates []DuplicatePair
	var mutex sync.Mutex
	var wg sync.WaitGroup
	
	// Create work items for comparison
	type workItem struct {
		i, j int
	}
	
	workChan := make(chan workItem, 1000)
	
	// Fill work channel
	go func() {
		for i := 0; i < len(blocks); i++ {
			for j := i + 1; j < len(blocks); j++ {
				// Skip blocks from the same file that overlap
				if blocks[i].FilePath == blocks[j].FilePath &&
					((blocks[i].StartLine <= blocks[j].EndLine && blocks[i].EndLine >= blocks[j].StartLine) ||
						(blocks[j].StartLine <= blocks[i].EndLine && blocks[j].EndLine >= blocks[i].StartLine)) {
					continue
				}
				
				workChan <- workItem{i, j}
			}
		}
		close(workChan)
	}()
	
	// We'll use a different approach for the comparison progress
	// Instead of showing progress for each comparison, we'll show progress for each file
	// This gives a more meaningful progress indicator with one dot per file
	
	// Set up progress tracking - one dot per unique file
	// Use a mutex to protect the uniqueFiles map
	uniqueFilesMutex := &sync.Mutex{}
	uniqueFiles := make(map[string]bool)
	for _, block := range blocks {
		uniqueFilesMutex.Lock()
		uniqueFiles[block.FilePath] = true
		uniqueFilesMutex.Unlock()
	}
	
	// Get the count of unique files for the progress bar
	uniqueFilesMutex.Lock()
	uniqueFilesCount := len(uniqueFiles)
	uniqueFilesMutex.Unlock()
	
	// Create a progress bar with the number of unique files
	progress := utils.NewProgressBar(uniqueFilesCount, "Comparing code blocks")
	progress.Start()
	
	// Track which files we've processed
	processedFilesMutex := &sync.Mutex{}
	processedFiles := make(map[string]bool)
	
	// Create a channel to signal when a new file is processed
	fileProcessedChan := make(chan string, uniqueFilesCount)

	// Process comparisons concurrently
	for i := 0; i < options.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for work := range workChan {
				i, j := work.i, work.j
				
				// Compare blocks
				similarity := calculateBlockSimilarity(blocks[i], blocks[j])
				
				// If similarity is above threshold, add to duplicates
				if similarity >= options.SimilarityThreshold {
					mutex.Lock()
					duplicates = append(duplicates, DuplicatePair{
						Block1:     blocks[i],
						Block2:     blocks[j],
						Similarity: similarity,
					})
					mutex.Unlock()
				}
				
				// Signal that we've processed files if we haven't already
				file1 := blocks[i].FilePath
				file2 := blocks[j].FilePath
				
				// Use mutex to safely access the processedFiles map
				processedFilesMutex.Lock()
				// Send file1 to the channel if it hasn't been processed yet
				if !processedFiles[file1] {
					processedFiles[file1] = true
					fileProcessedChan <- file1
				}
				
				// Send file2 to the channel if it hasn't been processed yet
				if !processedFiles[file2] {
					processedFiles[file2] = true
					fileProcessedChan <- file2
				}
				processedFilesMutex.Unlock()
			}
		}()
	}
	
	// Start a goroutine to update the progress bar
	go func() {
		for range fileProcessedChan {
			progress.Increment()
		}
	}()
	
	wg.Wait()
	
	// Close the file processed channel
	close(fileProcessedChan)
	
	// Finish the progress indicator
	progress.Finish()
	
	// Sort duplicates by similarity (highest first)
	sort.Slice(duplicates, func(i, j int) bool {
		return duplicates[i].Similarity > duplicates[j].Similarity
	})
	
	return duplicates
}

// calculateBlockSimilarity calculates the similarity between two code blocks
func calculateBlockSimilarity(block1, block2 CodeBlock) float64 {
	// If the blocks have very different lengths, they're probably not duplicates
	lenRatio := float64(minInt(len(block1.Content), len(block2.Content))) / float64(maxInt(len(block1.Content), len(block2.Content)))
	if lenRatio < 0.7 {
		return 0.0
	}
	
	// Calculate Levenshtein distance
	distance := levenshtein.ComputeDistance(block1.Content, block2.Content)
	maxLen := maxInt(len(block1.Content), len(block2.Content))
	
	// Convert distance to similarity score (0.0-1.0)
	if maxLen == 0 {
		return 1.0
	}
	
	return 1.0 - float64(distance)/float64(maxLen)
}

// writeDuplicateResults writes duplicate analysis results to the output file
func writeDuplicateResults(writer *bufio.Writer, duplicates []DuplicatePair, options DuplicateOptions, elapsedTime time.Duration) {
	// Write header
	fmt.Fprintf(writer, "# Duplicate Code Analysis\n\n")
	fmt.Fprintf(writer, "*Generated on %s*\n\n", time.Now().Format("2006-01-02 15:04:05"))
	
	// Count unique files in the duplicates
	uniqueFiles := make(map[string]bool)
	for _, dup := range duplicates {
		uniqueFiles[dup.Block1.FilePath] = true
		uniqueFiles[dup.Block2.FilePath] = true
	}
	
	// Write summary
	fmt.Fprintf(writer, "## Summary\n\n")
	fmt.Fprintf(writer, "- **Files Analyzed:** %d\n", len(uniqueFiles))
	fmt.Fprintf(writer, "- **Duplicate Pairs Found:** %d\n", len(duplicates))
	fmt.Fprintf(writer, "- **Similarity Threshold:** %.0f%%\n", options.SimilarityThreshold*100)
	fmt.Fprintf(writer, "- **Minimum Block Size:** %d lines\n", options.MinLineCount)
	fmt.Fprintf(writer, "- **Analysis Time:** %s\n\n", elapsedTime)
	// If names-only mode is enabled, extract and display function/method names
	if options.NamesOnly {
		writeNamesOnlyOutput(writer, duplicates, options)
		return
	}
	
	// Write duplicates
	if len(duplicates) == 0 {
		fmt.Fprintf(writer, "No duplicates found with the current threshold.\n")
		return
	}
	
	fmt.Fprintf(writer, "## Duplicate Blocks\n\n")
	
	// Group duplicates by similarity range
	similarityRanges := []struct {
		min, max float64
		label    string
	}{
		{0.95, 1.0, "Near-Identical (95-100%)"},
		{0.9, 0.95, "Very Similar (90-95%)"},
		{0.8, 0.9, "Similar (80-90%)"},
		{0.7, 0.8, "Moderately Similar (70-80%)"},
	}
	
	for _, sr := range similarityRanges {
		// Filter duplicates in this range
		var rangeMatches []DuplicatePair
		for _, dup := range duplicates {
			if dup.Similarity >= sr.min && dup.Similarity < sr.max {
				rangeMatches = append(rangeMatches, dup)
			}
		}
		
		if len(rangeMatches) == 0 {
			continue
		}
		
		fmt.Fprintf(writer, "### %s\n\n", sr.label)
		fmt.Fprintf(writer, "Found %d duplicate pairs in this range.\n\n", len(rangeMatches))
		
		// Write top duplicates in this range (max 10)
		maxToShow := minInt(10, len(rangeMatches))
		for i := 0; i < maxToShow; i++ {
			dup := rangeMatches[i]
			
			fmt.Fprintf(writer, "#### Duplicate Pair %d (%.1f%% similar)\n\n", i+1, dup.Similarity*100)
			
			// First block
			relPath1, _ := filepath.Rel(options.Directory, dup.Block1.FilePath)
			fmt.Fprintf(writer, "**File 1:** `%s` (lines %d-%d)\n\n", relPath1, dup.Block1.StartLine, dup.Block1.EndLine)
			
			// For short output, don't include the full code blocks
			if !options.Short {
				fmt.Fprintf(writer, "```\n%s\n```\n\n", dup.Block1.Content)
			} else {
				// Just show a snippet for short output
				snippet := getSnippet(dup.Block1.Content, 3)
				fmt.Fprintf(writer, "```\n%s...\n```\n\n", snippet)
			}
			
			// Second block
			relPath2, _ := filepath.Rel(options.Directory, dup.Block2.FilePath)
			fmt.Fprintf(writer, "**File 2:** `%s` (lines %d-%d)\n\n", relPath2, dup.Block2.StartLine, dup.Block2.EndLine)
			
			// For short output, don't include the full code blocks
			if !options.Short {
				fmt.Fprintf(writer, "```\n%s\n```\n\n", dup.Block2.Content)
			} else {
				// Just show a snippet for short output
				snippet := getSnippet(dup.Block2.Content, 3)
				fmt.Fprintf(writer, "```\n%s...\n```\n\n", snippet)
			}
			
			fmt.Fprintf(writer, "---\n\n")
		}
		
		// If there are more duplicates in this range, mention it
		if len(rangeMatches) > maxToShow {
			fmt.Fprintf(writer, "*%d more duplicate pairs in this range not shown.*\n\n", len(rangeMatches)-maxToShow)
		}
	}
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt returns the maximum of two integers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// getSnippet returns the first n lines of a string
func getSnippet(content string, lines int) string {
	split := strings.SplitN(content, "\n", lines+1)
	if len(split) <= lines {
		return content
	}
	return strings.Join(split[:lines], "\n")
}

// findDuplicateMethodNames finds duplicate method and function names in the codebase
func findDuplicateMethodNames(files []string, options DuplicateOptions) ([]DuplicatePair, []CodeBlock) {
	// Set up progress tracking
	progress := utils.NewProgressBar(len(files), "Analyzing method names")
	progress.Start()
	
	// Map to store method names and their locations
	type MethodLocation struct {
		FilePath   string
		LineNumber int
		Content    string
	}
	
	methodMap := make(map[string][]MethodLocation)
	
	// Extract method names from all files
	var mutex sync.Mutex
	var wg sync.WaitGroup
	
	// Create a channel for files to process
	filesChan := make(chan string, len(files))
	for _, file := range files {
		filesChan <- file
	}
	close(filesChan)
	
	// Process files concurrently
	for i := 0; i < options.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			// Regular expressions for finding method/function names
			// Use more specific patterns similar to the registry analyzer
			funcRegex := regexp.MustCompile(`(?m)^\s*(func|method)\s+([a-zA-Z0-9_]+)\s*\(`)
			
			// For C/C++ style functions
			cppFuncRegex := regexp.MustCompile(`(?m)(\w+(?:\s*[*&]\s*)?)\s+(\w+)\s*\(([^)]*)\)\s*(?:{|;)`)
			
			// For C++ methods (class member functions)
			cppMethodRegex := regexp.MustCompile(`(?m)(\w+(?:\s*[*&]\s*)?)\s+(\w+)::(\w+)\s*\(([^)]*)\)\s*(?:const)?\s*(?:{|;)`)
			
			// Generic fallback pattern - use with caution
			genericFuncRegex := regexp.MustCompile(`(?m)([a-zA-Z0-9_]+)\s*\([^)]*\)\s*{`)
			
			// Common keywords and method names to exclude to reduce false positives
commonKeywords := map[string]bool{
	// C/C++ keywords
	"if": true, "for": true, "while": true, "switch": true, "case": true,
	"return": true, "sizeof": true, "typedef": true, "struct": true, "class": true,
	"enum": true, "union": true, "goto": true, "continue": true, "break": true,
	"else": true, "do": true, "static": true, "extern": true, "const": true,
	"volatile": true, "register": true, "auto": true, "inline": true, "virtual": true,
	"explicit": true, "operator": true, "template": true, "typename": true, "namespace": true,
	"using": true, "try": true, "catch": true, "throw": true, "new": true,
	"public": true, "private": true, "protected": true, "friend": true, "this": true,
	"default": true, "NULL": true, "nullptr": true, "true": true, "false": true,

	// Common method names to exclude
	"get": true, "set": true, "add": true, "remove": true, "create": true,
	"update": true, "find": true, "search": true, "init": true, "initialize": true,
	"start": true, "stop": true, "run": true, "execute": true, "parse": true,
	"read": true, "write": true, "open": true, "close": true, "load": true, "save": true,
	"reset": true, "print": true, "main": true, "test": true, "handle": true,
	"process": true, "validate": true, "check": true, "convert": true, "transform": true,
	"calculate": true, "compute": true,

	// Common C++ standard library functions
	"at": true, "data": true, "size": true, "empty": true,
	"push_back": true, "pop_back": true, "front": true, "back": true, "insert": true,
	"erase": true, "clear": true, "swap": true, "emplace": true, "emplace_back": true,

	// Common chemical/molecular function names
	"exp": true, "sqrt": true, "pow": true, "abs": true, "sin": true, "cos": true,
	"tan": true, "floor": true, "ceil": true, "round": true,
	"min": true, "max": true, "clamp": true, "lerp": true, "normalize": true,
	"isnan": true, "isinf": true, "isfinite": true, "constexpr": true,
	"nan": true, "dist": true, "basis": true, "shell": true, "file": true, "t": true,
	
	// Chemical-specific methods
	"cancel": true, "cancelProcessing": true, "cancelParsing": true, "visualize": true,
	"visualizeFromSmiles": true, "visualizeMolecularStructure": true, "draw": true,
	"generate": true, "parser": true, "calculator": true, "density": true, "worker": true,
	"progress": true, "rng": true, "mmffProps": true, "params": true,
	"task": true, "generator": true, "drawer": true, "visualizer": true, "minimizer": true,
	"outPath": true, "svg_file": true, "validation_file": true, "chunkParser": true,
	"parallelParser": true, "producer": true, "signalHandler": true, "uff_minimizer": true,
	"mmff_minimizer": true, "generator_rdkit": true, "generator_with_canon": true,
	"generator_without_canon": true, "saveAsSVG": true, "saveSVG": true, "saveAsImage": true,
	"log10": true,
}
			
			for file := range filesChan {
				// Read file content
				content, err := os.ReadFile(file)
				if err != nil {
					if options.Verbose {
						fmt.Printf(color.RedString("Error reading file %s: %v\n"), file, err)
					}
					progress.Increment()
					continue
				}
				
				// Convert content to string
				contentStr := string(content)
				
				// Split content into lines
				lines := strings.Split(contentStr, "\n")
				
				// Try Go-style functions first
				matches := funcRegex.FindAllStringSubmatchIndex(contentStr, -1)
				for _, match := range matches {
					if len(match) >= 4 {
						// Extract method name
						methodName := contentStr[match[4]:match[5]]
						
						// Find line number
						lineNumber := 1
						for i := 0; i < match[0]; i++ {
							if contentStr[i] == '\n' {
								lineNumber++
							}
						}
						
						// Extract context (the method declaration line)
						contextLine := ""
						if lineNumber-1 < len(lines) {
							contextLine = lines[lineNumber-1]
						}
						
						// Add to method map
						mutex.Lock()
						methodMap[methodName] = append(methodMap[methodName], MethodLocation{
							FilePath:   file,
							LineNumber: lineNumber,
							Content:    contextLine,
						})
						mutex.Unlock()
					}
				}
				
				// Try C++ functions
				matches = cppFuncRegex.FindAllStringSubmatchIndex(contentStr, -1)
				for _, match := range matches {
					if len(match) >= 4 {
						// Extract method name
						methodName := contentStr[match[4]:match[5]]
						
						// Skip common keywords
						if commonKeywords[methodName] {
							continue
						}
						
						// Find line number
						lineNumber := 1
						for i := 0; i < match[0]; i++ {
							if contentStr[i] == '\n' {
								lineNumber++
							}
						}
						
						// Extract context (the method declaration line)
						contextLine := ""
						if lineNumber-1 < len(lines) {
							contextLine = lines[lineNumber-1]
						}
						
						// Add to method map
						mutex.Lock()
						methodMap[methodName] = append(methodMap[methodName], MethodLocation{
							FilePath:   file,
							LineNumber: lineNumber,
							Content:    contextLine,
						})
						mutex.Unlock()
					}
				}
				
				// Try C++ methods
				matches = cppMethodRegex.FindAllStringSubmatchIndex(contentStr, -1)
				for _, match := range matches {
					if len(match) >= 6 {
						// Extract method name
						methodName := contentStr[match[6]:match[7]]
						
						// Skip common keywords
						if commonKeywords[methodName] {
							continue
						}
						
						// Find line number
						lineNumber := 1
						for i := 0; i < match[0]; i++ {
							if contentStr[i] == '\n' {
								lineNumber++
							}
						}
						
						// Extract context (the method declaration line)
						contextLine := ""
						if lineNumber-1 < len(lines) {
							contextLine = lines[lineNumber-1]
						}
						
						// Add to method map
						mutex.Lock()
						methodMap[methodName] = append(methodMap[methodName], MethodLocation{
							FilePath:   file,
							LineNumber: lineNumber,
							Content:    contextLine,
						})
						mutex.Unlock()
					}
				}
				
				// If no methods found with the specific regexes, try the generic one as a last resort
				if len(methodMap) == 0 {
					matches := genericFuncRegex.FindAllStringSubmatchIndex(contentStr, -1)
					for _, match := range matches {
						if len(match) >= 2 {
							// Extract method name
							methodName := contentStr[match[2]:match[3]]
							
							// Skip common keywords that might be mistaken for functions
							if commonKeywords[methodName] {
								continue
							}
							
							// Find line number
							lineNumber := 1
							for i := 0; i < match[0]; i++ {
								if contentStr[i] == '\n' {
									lineNumber++
								}
							}
							
							// Extract context (the method declaration line)
							contextLine := ""
							if lineNumber-1 < len(lines) {
								contextLine = lines[lineNumber-1]
							}
							
							// Add to method map
							mutex.Lock()
							methodMap[methodName] = append(methodMap[methodName], MethodLocation{
								FilePath:   file,
								LineNumber: lineNumber,
								Content:    contextLine,
							})
							mutex.Unlock()
						}
					}
				}
				
				// Update progress
				progress.Increment()
			}
		}()
	}
	
	// Wait for all files to be processed
	wg.Wait()
	
	// Finish the progress indicator
	progress.Finish()
	
	// Create duplicate pairs for methods that appear in multiple locations
	var duplicates []DuplicatePair
	var blocks []CodeBlock
	
	// Convert method map to duplicate pairs
	for methodName, locations := range methodMap {
		// Only include methods that appear in multiple locations
		if len(locations) <= 1 {
			continue
		}
		
		// Create a code block for each location
		for i, loc := range locations {
			block := CodeBlock{
				FilePath:  loc.FilePath,
				StartLine: loc.LineNumber,
				EndLine:   loc.LineNumber,
				Content:   loc.Content,
				Lines:     []string{loc.Content},
			}
			
			blocks = append(blocks, block)
			
			// Create duplicate pairs by comparing each location with others
			for j := i + 1; j < len(locations); j++ {
				block2 := CodeBlock{
					FilePath:  locations[j].FilePath,
					StartLine: locations[j].LineNumber,
					EndLine:   locations[j].LineNumber,
					Content:   locations[j].Content,
					Lines:     []string{locations[j].Content},
				}
				
				// Calculate similarity between method implementations
				// For method names, we'll use Levenshtein distance to calculate similarity
				// between the actual implementations, not just the names
				similarity := calculateMethodSimilarity(block.Content, block2.Content)
				
				// Only add if similarity is above threshold
				if similarity >= options.SimilarityThreshold {
					duplicates = append(duplicates, DuplicatePair{
						Block1:     block,
						Block2:     block2,
						Similarity: similarity,
					})
					
					// Debug output if verbose
					if options.Verbose {
						// Format the similarity percentage as a string first
						similarityStr := color.GreenString("%.2f", similarity*100)
						fmt.Printf("Found duplicate method: %s (similarity: %s%%) in %s and %s\n", 
							color.YellowString(methodName),
							similarityStr,
							color.CyanString(block.FilePath),
							color.CyanString(block2.FilePath))
					}
				}
			}
		}
	}
	
	// Sort duplicates by method name
	sort.Slice(duplicates, func(i, j int) bool {
		// Extract method names from both blocks
		name1 := extractMethodName(duplicates[i].Block1.Content)
		name2 := extractMethodName(duplicates[j].Block1.Content)
		return name1 < name2
	})
	
	return duplicates, blocks
}

// calculateMethodSimilarity calculates the similarity between two method implementations
func calculateMethodSimilarity(method1, method2 string) float64 {
	// If either method is empty, return 0 similarity
	if len(method1) == 0 || len(method2) == 0 {
		return 0.0
	}
	
	// Calculate Levenshtein distance between the two methods
	distance := levenshtein.ComputeDistance(method1, method2)
	
	// Convert distance to similarity (0.0-1.0)
	maxLen := maxInt(len(method1), len(method2))
	if maxLen == 0 {
		return 1.0 // Both strings are empty, so they're identical
	}
	
	// Calculate similarity as 1 - (distance / maxLen)
	similarity := 1.0 - float64(distance)/float64(maxLen)
	
	// Ensure similarity is between 0 and 1
	if similarity < 0.0 {
		similarity = 0.0
	} else if similarity > 1.0 {
		similarity = 1.0
	}
	
	return similarity
}

// extractMethodName extracts the method name from a method declaration
func extractMethodName(declaration string) string {
	// Try to match function declaration pattern
	funcRegex := regexp.MustCompile(`func\s+([a-zA-Z0-9_]+)\s*\(`)
	matches := funcRegex.FindStringSubmatch(declaration)
	if len(matches) >= 2 {
		return matches[1]
	}
	
	// Try generic pattern
	genericRegex := regexp.MustCompile(`([a-zA-Z0-9_]+)\s*\(`)
	matches = genericRegex.FindStringSubmatch(declaration)
	if len(matches) >= 2 {
		return matches[1]
	}
	
	// If no pattern matches, return a shortened version of the declaration
	if len(declaration) > 20 {
		return declaration[:20] + "..."
	}
	return declaration
}

// writeNamesOnlyOutput writes duplicate analysis results showing only function/method names
func writeNamesOnlyOutput(writer *bufio.Writer, duplicates []DuplicatePair, options DuplicateOptions) {
	// First, write the duplicate pairs directly
	fmt.Fprintf(writer, "## Duplicate Method/Function Pairs\n\n")
	
	if len(duplicates) == 0 {
		fmt.Fprintf(writer, "No duplicate method/function pairs found.\n\n")
	} else {
		// Write table header for duplicate pairs
		fmt.Fprintf(writer, "| Method 1 | Method 2 | Similarity | \n")
		fmt.Fprintf(writer, "|----------|----------|------------| \n")
		
		// Write each duplicate pair
		for _, dup := range duplicates {
			// Get relative paths for better readability
			relPath1, _ := filepath.Rel(options.Directory, dup.Block1.FilePath)
			relPath2, _ := filepath.Rel(options.Directory, dup.Block2.FilePath)
			
			// Extract method names or use file locations if names can't be extracted
			method1 := extractMethodName(dup.Block1.Content)
			method2 := extractMethodName(dup.Block2.Content)
			
			// Format the method locations
			method1Location := fmt.Sprintf("`%s` in `%s:%d`", method1, relPath1, dup.Block1.StartLine)
			method2Location := fmt.Sprintf("`%s` in `%s:%d`", method2, relPath2, dup.Block2.StartLine)
			
			// Write the pair
			fmt.Fprintf(writer, "| %s | %s | %.1f%% |\n", 
				method1Location, 
				method2Location, 
				dup.Similarity * 100)
		}
	}
	
	// Then, aggregate methods by name for the summary section
	fmt.Fprintf(writer, "\n## Method/Function Name Summary\n\n")
	
	// Extract function/method names from the duplicate blocks
	type MethodInfo struct {
		Name     string
		FilePath string
		Line     int
	}

	// Map to store unique method names and their occurrences
	methodMap := make(map[string][]MethodInfo)
	
	// Common C/C++ keywords to exclude
	commonKeywords := map[string]bool{
		"if": true, "for": true, "while": true, "switch": true, "case": true,
		"return": true, "sizeof": true, "typedef": true, "struct": true, "class": true,
		"enum": true, "union": true, "goto": true, "continue": true, "break": true,
		"else": true, "do": true, "static": true, "extern": true, "const": true,
	}
	
	// Process each duplicate pair
	for _, dup := range duplicates {
		// Extract method name from first block
		method1 := extractMethodName(dup.Block1.Content)
		if method1 != "" && !commonKeywords[method1] {
			methodMap[method1] = append(methodMap[method1], MethodInfo{
				Name:     method1,
				FilePath: dup.Block1.FilePath,
				Line:     dup.Block1.StartLine,
			})
		}
		
		// Extract method name from second block
		method2 := extractMethodName(dup.Block2.Content)
		if method2 != "" && !commonKeywords[method2] {
			methodMap[method2] = append(methodMap[method2], MethodInfo{
				Name:     method2,
				FilePath: dup.Block2.FilePath,
				Line:     dup.Block2.StartLine,
			})
		}
	}
	
	// Check if we found any methods
	if len(methodMap) == 0 {
		fmt.Fprintf(writer, "No function or method names detected in the duplicated code blocks.\n")
		return
	}
	
	// Convert map to slice for sorting
	type MethodEntry struct {
		Name      string
		Locations []MethodInfo
	}
	
	methods := make([]MethodEntry, 0, len(methodMap))
	for name, locations := range methodMap {
		methods = append(methods, MethodEntry{Name: name, Locations: locations})
	}
	
	// Sort by number of occurrences (descending)
	sort.Slice(methods, func(i, j int) bool {
		return len(methods[i].Locations) > len(methods[j].Locations)
	})
	
	// Write table header
	fmt.Fprintf(writer, "| Method/Function Name | Occurrences | Locations |\n")
	fmt.Fprintf(writer, "|-------------------|------------|-----------|\n")
	
	// Write each method
	for _, method := range methods {
		// Only include methods that appear in multiple locations
		if len(method.Locations) <= 1 {
			continue
		}
		
		// Format locations
		locations := ""
		for i, loc := range method.Locations {
			relPath, _ := filepath.Rel(options.Directory, loc.FilePath)
			if i > 0 {
				locations += "<br>"
			}
			locations += fmt.Sprintf("`%s:%d`", relPath, loc.Line)
			
			// Limit to 3 locations if there are many
			if i >= 2 && len(method.Locations) > 3 {
				locations += fmt.Sprintf("<br>*...and %d more*", len(method.Locations)-3)
				break
			}
		}
		
		fmt.Fprintf(writer, "| `%s` | %d | %s |\n", method.Name, len(method.Locations), locations)
	}
}
