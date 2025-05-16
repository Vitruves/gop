package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/vitruves/gop/internal/utils"
)

// ConcatOptions contains options for the concat command
type ConcatOptions struct {
	// Input/Output options
	InputFile  string   // Path to input file containing list of files to process
	Directory  string   // Root directory to analyze
	Depth      int      // Maximum depth for directory traversal
	OutputFile string   // Path to output file for results
	Languages  []string // Languages to analyze (e.g., "c", "cpp")
	Excludes   []string // Directories or files to exclude
	
	// Processing options
	Jobs int // Number of concurrent jobs for processing
	
	// Formatting options
	IncludeHeaders bool // Whether to include file headers in output
	AddLineNumbers bool // Whether to add line numbers to output
	RemoveComments bool // Whether to remove comments from output
	
	// Output options
	Short   bool // Whether to use short output format
	Verbose bool // Whether to enable verbose output
}

// FileProcessResult contains the result of processing a file
type FileProcessResult struct {
	Content     string // Processed content
	LineCount   int    // Number of lines in the file
	ByteCount   int    // Number of bytes in the file
	FilePath    string // Path to the file
	BaseName    string // Base name of the file
	IsHeader    bool   // Whether this is a header file
}

// ConcatenateFiles concatenates source files into a single text file
func ConcatenateFiles(options ConcatOptions) {
	if options.Verbose {
		fmt.Println(color.CyanString("Starting file concatenation..."))
	}

	// Record start time for performance metrics
	startTime := time.Now()

	// Get list of files to process
	files, err := utils.GetFilesToProcess(options.InputFile, options.Directory, options.Depth, options.Languages, options.Excludes)
	if err != nil {
		fmt.Println(color.RedString("Error:"), err)
		return
	}

	if options.Verbose {
		fmt.Printf(color.CyanString("Found %d files to process\n"), len(files))
	}

	// Group files by their base names (without extension)
	fileGroups := groupFilesByBaseName(files)

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

	// Set up concurrency
	var wg sync.WaitGroup
	resultsChan := make(chan FileProcessResult, len(files))
	filesChan := make(chan struct {
		baseName string
		files    []string
	}, len(fileGroups))

	// Fill the files channel
	for baseName, groupFiles := range fileGroups {
		filesChan <- struct {
			baseName string
			files    []string
		}{baseName, groupFiles}
	}
	close(filesChan)

	// Set up progress tracking
	progress := utils.NewProgressBar(len(fileGroups), "Concatenating files")
	progress.Start()

	// Track statistics
	var totalLinesProcessed atomic.Int64
	var totalBytesProcessed atomic.Int64
	var filesProcessed atomic.Int64

	// Process files concurrently
	jobs := options.Jobs
	if jobs <= 0 {
		jobs = 1
	}

	// Start worker goroutines
	for i := 0; i < jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for fileGroup := range filesChan {
				baseName := fileGroup.baseName
				groupFiles := fileGroup.files

				// Sort files so headers come first
				var headerFiles, sourceFiles []string
				for _, file := range groupFiles {
					ext := filepath.Ext(file)
					if ext == ".h" || ext == ".hpp" || ext == ".hxx" {
						headerFiles = append(headerFiles, file)
					} else {
						sourceFiles = append(sourceFiles, file)
					}
				}

				// Process header files first, then source files
				allFiles := append([]string{}, headerFiles...)
				allFiles = append(allFiles, sourceFiles...)

				for _, file := range allFiles {
					isHeader := false
					ext := filepath.Ext(file)
					if ext == ".h" || ext == ".hpp" || ext == ".hxx" {
						isHeader = true
					}

					// Read file content
					content, err := os.ReadFile(file)
					if err != nil {
						resultsChan <- FileProcessResult{
							Content:  fmt.Sprintf("// Error reading file: %s\n", err),
							FilePath: file,
							BaseName: baseName,
							IsHeader: isHeader,
						}
						continue
					}

					lines := strings.Split(string(content), "\n")
					var processedContent strings.Builder

					// Add file header if requested
					if options.IncludeHeaders {
						processedContent.WriteString(fmt.Sprintf("\n// File: %s\n", file))
						processedContent.WriteString(fmt.Sprintf("// Base: %s\n\n", baseName))
					}

					// Process content
					lineCount := 0
					for i, line := range lines {
						lineNum := i + 1

						// Skip line if it's a comment and we're removing comments
						if options.RemoveComments && isRegularComment(line) {
							continue
						}

						// Add line number if requested
						if options.AddLineNumbers {
							processedContent.WriteString(fmt.Sprintf("%5d: ", lineNum))
						}

						processedContent.WriteString(line)
						processedContent.WriteString("\n")
						lineCount++
					}

					// Add empty line between files
					processedContent.WriteString("\n")

					// Update statistics
					totalLinesProcessed.Add(int64(lineCount))
					totalBytesProcessed.Add(int64(len(content)))
					filesProcessed.Add(1)

					// Send result
					resultsChan <- FileProcessResult{
						Content:   processedContent.String(),
						LineCount: lineCount,
						ByteCount: len(content),
						FilePath:  file,
						BaseName:  baseName,
						IsHeader:  isHeader,
					}
				}

				// Update progress
				progress.Increment()
			}
		}()
	}

	// Start a goroutine to collect results and write to file
	go func() {
		for result := range resultsChan {
			// Write processed content to output file
			_, err := writer.WriteString(result.Content)
			if err != nil {
				fmt.Println(color.RedString("Error writing to output file:"), err)
			}
		}
	}()

	// Wait for all processing to complete
	wg.Wait()
	close(resultsChan)

	// Ensure all data is written
	writer.Flush()
	progress.Finish()

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)
	processingSpeed := float64(filesProcessed.Load()) / elapsedTime.Seconds()
	
	// Print summary with more detailed information
	fmt.Println()
	titleStyle := color.New(color.Bold, color.FgGreen).SprintFunc()
	fmt.Printf("%s\n\n", titleStyle("Concatenation Results"))
	
	// Get output stats
	totalLines := 0
	totalBytes := 0
	
	// If we're writing to a file, get stats from the file
	if options.OutputFile != "" {
		stat, err := os.Stat(options.OutputFile)
		if err == nil {
			totalBytes = int(stat.Size())
		}
		
		// Count lines in output file
		if file, err := os.Open(options.OutputFile); err == nil {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				totalLines++
			}
		}
	} else {
		// For console output, use the processed counts
		totalBytes = int(totalBytesProcessed.Load())
		totalLines = int(totalLinesProcessed.Load())
	}
	
	// Print statistics
	fmt.Printf("Concatenated %s files\n", 
		color.New(color.Bold, color.FgCyan).Sprintf("%d", filesProcessed.Load()))
	
	fmt.Printf("Output file contains %s lines and %s\n", 
		color.New(color.Bold).Sprintf("%d", totalLines),
		formatSize(totalBytes))
	
	// Print performance metrics
	fmt.Printf("Processing time: %s (%.2f files/sec)\n",
		color.New(color.Bold).Sprintf("%s", elapsedTime.Round(time.Millisecond)),
		processingSpeed)
	
	// Print options used
	fmt.Println("\nOptions used:")
	fmt.Printf("  Include headers: %s\n", formatBool(options.IncludeHeaders))
	fmt.Printf("  Add line numbers: %s\n", formatBool(options.AddLineNumbers))
	fmt.Printf("  Remove comments: %s\n", formatBool(options.RemoveComments))
	fmt.Printf("  Concurrent jobs: %s\n", color.New(color.Bold).Sprintf("%d", jobs))
}

// CommentType represents the type of a comment
type CommentType int

const (
	NotComment CommentType = iota
	RegularComment
	ImportantComment
)

// Cache of important keywords for faster lookups
var importantKeywords = map[string]bool{
	"TODO":       true,
	"FIXME":      true,
	"NOTE":       true,
	"IMPORTANT":  true,
	"NAMESPACE":  true,
	"CLASS":      true,
	"STRUCT":     true,
	"ENUM":       true,
	"INTERFACE":  true,
	"FUNCTION":   true,
	"METHOD":     true,
	"COPYRIGHT":  true,
	"LICENSE":    true,
	"AUTHOR":     true,
	"DEPRECATED": true,
	"PARAM":      true,
	"RETURN":     true,
	"THROWS":     true,
	"EXAMPLE":    true,
	"SEE":        true,
	"SINCE":      true,
	"VERSION":    true,
}

// isRegularComment checks if a line is a regular comment (not an important one)
func isRegularComment(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Fast path: if it's not a comment at all, return false
	if !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") && !strings.HasSuffix(trimmed, "*/") && !strings.Contains(trimmed, "*/") {
		return false
	}

	// Convert to uppercase for case-insensitive comparison
	upperLine := strings.ToUpper(trimmed)
	
	// Check if it contains any important keywords
	for keyword := range importantKeywords {
		if strings.Contains(upperLine, keyword) {
			return false // It's an important comment, don't remove
		}
	}

	return true // It's a regular comment, can be removed
}

// groupFilesByBaseName groups files by their base names (without extension)
func groupFilesByBaseName(files []string) map[string][]string {
	// Pre-allocate the map with an estimated capacity
	estimatedGroups := len(files) / 2
	if estimatedGroups < 10 {
		estimatedGroups = 10
	}
	groups := make(map[string][]string, estimatedGroups)
	
	// Process files in batches for better memory locality
	const batchSize = 100
	for i := 0; i < len(files); i += batchSize {
		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}
		
		// Process this batch
		for _, file := range files[i:end] {
			baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
			
			// Check if we already have an entry for this base name
			if _, exists := groups[baseName]; exists {
				groups[baseName] = append(groups[baseName], file)
			} else {
				// Allocate a new slice with some capacity for future appends
				groups[baseName] = make([]string, 1, 4)
				groups[baseName][0] = file
			}
		}
	}
	
	return groups
}

// Size units for human-readable output
const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
	PB
)

// formatSize formats a size in bytes to a human-readable string with color
func formatSize(bytes int) string {
	var size float64
	var unit string
	
	switch {
	case bytes >= PB:
		size = float64(bytes) / PB
		unit = "PB"
	case bytes >= TB:
		size = float64(bytes) / TB
		unit = "TB"
	case bytes >= GB:
		size = float64(bytes) / GB
		unit = "GB"
	case bytes >= MB:
		size = float64(bytes) / MB
		unit = "MB"
	case bytes >= KB:
		size = float64(bytes) / KB
		unit = "KB"
	default:
		return color.New(color.FgCyan).Sprintf("%d bytes", bytes)
	}
	
	// Format with different colors based on size
	var sizeColor *color.Color
	switch {
	case bytes >= GB:
		sizeColor = color.New(color.FgMagenta, color.Bold)
	case bytes >= MB:
		sizeColor = color.New(color.FgBlue, color.Bold)
	default:
		sizeColor = color.New(color.FgCyan, color.Bold)
	}
	
	return sizeColor.Sprintf("%.2f %s", size, unit)
}

// formatBool formats a boolean value with color and icon
func formatBool(value bool) string {
	if value {
		return color.New(color.FgGreen, color.Bold).Sprintf("✓ enabled")
	}
	return color.New(color.FgRed, color.Bold).Sprintf("✗ disabled")
}
