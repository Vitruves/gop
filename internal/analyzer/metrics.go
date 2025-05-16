package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/vitruves/gop/internal/utils"
)

// MetricsOptions contains options for the metrics command
type MetricsOptions struct {
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
	Short   bool // Whether to use short output format
	Verbose bool // Whether to enable verbose output
}

// FileMetrics represents metrics for a single file
type FileMetrics struct {
	FilePath       string // Path to the file
	TotalLines     int    // Total number of lines
	CodeLines      int    // Number of code lines
	CommentLines   int    // Number of comment lines
	BlankLines     int    // Number of blank lines
	Functions      int    // Number of functions
	Classes        int    // Number of classes/structs
	Complexity     int    // Cyclomatic complexity (sum of all functions)
	MaxComplexity  int    // Maximum complexity of any function
	AvgComplexity  float64 // Average complexity per function
	HeaderRatio    float64 // Ratio of header to implementation (if applicable)
	CommentRatio   float64 // Ratio of comments to code
}

// ProjectMetrics represents metrics for the entire project
type ProjectMetrics struct {
	Files            []FileMetrics // Metrics for each file
	TotalFiles       int           // Total number of files
	TotalLines       int           // Total number of lines
	TotalCodeLines   int           // Total number of code lines
	TotalCommentLines int          // Total number of comment lines
	TotalBlankLines  int           // Total number of blank lines
	TotalFunctions   int           // Total number of functions
	TotalClasses     int           // Total number of classes/structs
	AvgComplexity    float64       // Average complexity across all functions
	MaxComplexity    int           // Maximum complexity of any function
	AvgCommentRatio  float64       // Average comment ratio
}

// CalculateMetrics calculates code metrics for the given options
func CalculateMetrics(options MetricsOptions) {
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

	// Process files and extract metrics
	var processedFiles atomic.Int64
	metrics := extractFileMetrics(files, options, &processedFiles)

	// Calculate project-wide metrics
	projectMetrics := calculateProjectMetrics(metrics)

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
	writeMetricsOutput(writer, projectMetrics, files, options, elapsedTime, processingSpeed)
}

// extractFileMetrics extracts metrics from the given files
func extractFileMetrics(files []string, options MetricsOptions, processedFiles *atomic.Int64) []FileMetrics {
	var metrics []FileMetrics
	var mutex sync.Mutex
	var wg sync.WaitGroup

	// Create a channel for files to process
	filesChan := make(chan string, len(files))
	for _, file := range files {
		filesChan <- file
	}
	close(filesChan)

	// Create a progress bar
	progress := utils.NewProgressBar(len(files), "Calculating metrics")
	progress.Start()

	// Create a channel for results
	resultsChan := make(chan FileMetrics, options.Jobs)

	// Process files concurrently
	for i := 0; i < options.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for file := range filesChan {
				// Calculate metrics for this file
				fileMetrics := calculateFileMetrics(file)

				// Send results to the channel
				resultsChan <- fileMetrics

				// Update progress
				processedFiles.Add(1)
				progress.Increment()
			}
		}()
	}

	// Start a goroutine to collect results
	go func() {
		for fileMetrics := range resultsChan {
			mutex.Lock()
			metrics = append(metrics, fileMetrics)
			mutex.Unlock()
		}
	}()

	// Wait for all processing to complete
	wg.Wait()
	close(resultsChan)
	progress.Finish()

	return metrics
}

// calculateFileMetrics calculates metrics for a single file
func calculateFileMetrics(filePath string) FileMetrics {
	metrics := FileMetrics{
		FilePath: filePath,
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return metrics
	}

	// Count lines
	lines := strings.Split(string(content), "\n")
	metrics.TotalLines = len(lines)

	// Regular expressions for detecting comments and function/class definitions
	singleLineCommentRegex := regexp.MustCompile(`^\s*//.*$`)
	multiLineCommentStartRegex := regexp.MustCompile(`^\s*/\*.*$`)
	multiLineCommentEndRegex := regexp.MustCompile(`^.*\*/\s*$`)
	functionRegex := regexp.MustCompile(`^\s*([\w\*]+\s+)+\w+\s*\([^)]*\)\s*({|;)`)
	classRegex := regexp.MustCompile(`^\s*(class|struct)\s+\w+.*$`)
	blankLineRegex := regexp.MustCompile(`^\s*$`)
	
	inMultiLineComment := false
	
	// Analyze each line
	for _, line := range lines {
		if inMultiLineComment {
			metrics.CommentLines++
			if multiLineCommentEndRegex.MatchString(line) {
				inMultiLineComment = false
			}
		} else if singleLineCommentRegex.MatchString(line) {
			metrics.CommentLines++
		} else if multiLineCommentStartRegex.MatchString(line) {
			metrics.CommentLines++
			if !multiLineCommentEndRegex.MatchString(line) {
				inMultiLineComment = true
			}
		} else if blankLineRegex.MatchString(line) {
			metrics.BlankLines++
		} else {
			metrics.CodeLines++
			
			// Check for function definitions
			if functionRegex.MatchString(line) {
				metrics.Functions++
				
				// Estimate complexity (this is a simplification)
				// A more accurate implementation would parse the function body
				metrics.Complexity++
			}
			
			// Check for class/struct definitions
			if classRegex.MatchString(line) {
				metrics.Classes++
			}
		}
	}
	
	// Calculate ratios
	if metrics.CodeLines > 0 {
		metrics.CommentRatio = float64(metrics.CommentLines) / float64(metrics.CodeLines)
	}
	
	// Estimate header ratio (if applicable)
	ext := filepath.Ext(filePath)
	if ext == ".h" || ext == ".hpp" {
		// This is a header file, so we'll set the header ratio to 1
		metrics.HeaderRatio = 1.0
	} else if ext == ".c" || ext == ".cpp" {
		// This is an implementation file, so we'll set the header ratio to 0
		metrics.HeaderRatio = 0.0
	}
	
	// Calculate average complexity
	if metrics.Functions > 0 {
		metrics.AvgComplexity = float64(metrics.Complexity) / float64(metrics.Functions)
	}
	
	// For simplicity, we'll set max complexity equal to average complexity
	// A more accurate implementation would track complexity per function
	metrics.MaxComplexity = metrics.Complexity
	
	return metrics
}

// calculateProjectMetrics calculates project-wide metrics
func calculateProjectMetrics(fileMetrics []FileMetrics) ProjectMetrics {
	projectMetrics := ProjectMetrics{
		Files:      fileMetrics,
		TotalFiles: len(fileMetrics),
	}
	
	// Sum up metrics across all files
	for _, metrics := range fileMetrics {
		projectMetrics.TotalLines += metrics.TotalLines
		projectMetrics.TotalCodeLines += metrics.CodeLines
		projectMetrics.TotalCommentLines += metrics.CommentLines
		projectMetrics.TotalBlankLines += metrics.BlankLines
		projectMetrics.TotalFunctions += metrics.Functions
		projectMetrics.TotalClasses += metrics.Classes
		
		if metrics.MaxComplexity > projectMetrics.MaxComplexity {
			projectMetrics.MaxComplexity = metrics.MaxComplexity
		}
	}
	
	// Calculate averages
	if projectMetrics.TotalFunctions > 0 {
		totalComplexity := 0
		for _, metrics := range fileMetrics {
			totalComplexity += metrics.Complexity
		}
		projectMetrics.AvgComplexity = float64(totalComplexity) / float64(projectMetrics.TotalFunctions)
	}
	
	if projectMetrics.TotalCodeLines > 0 {
		projectMetrics.AvgCommentRatio = float64(projectMetrics.TotalCommentLines) / float64(projectMetrics.TotalCodeLines)
	}
	
	return projectMetrics
}

// writeMetricsOutput writes metrics output to the given writer
func writeMetricsOutput(writer *bufio.Writer, metrics ProjectMetrics, files []string, options MetricsOptions, elapsedTime time.Duration, processingSpeed float64) {
	// Write header
	fmt.Fprintf(writer, "# Code Metrics Analysis\n\n")
	fmt.Fprintf(writer, "*Generated on %s*\n\n", time.Now().Format("2006-01-02 15:04:05"))
	
	// Write summary
	fmt.Fprintf(writer, "## Summary\n\n")
	fmt.Fprintf(writer, "| Metric | Value |\n")
	fmt.Fprintf(writer, "|--------|-------|\n")
	fmt.Fprintf(writer, "| Files Analyzed | %d |\n", metrics.TotalFiles)
	fmt.Fprintf(writer, "| Total Lines | %d |\n", metrics.TotalLines)
	fmt.Fprintf(writer, "| Code Lines | %d |\n", metrics.TotalCodeLines)
	fmt.Fprintf(writer, "| Comment Lines | %d |\n", metrics.TotalCommentLines)
	fmt.Fprintf(writer, "| Blank Lines | %d |\n", metrics.TotalBlankLines)
	fmt.Fprintf(writer, "| Functions | %d |\n", metrics.TotalFunctions)
	fmt.Fprintf(writer, "| Classes/Structs | %d |\n", metrics.TotalClasses)
	fmt.Fprintf(writer, "| Avg. Complexity | %.2f |\n", metrics.AvgComplexity)
	fmt.Fprintf(writer, "| Max Complexity | %d |\n", metrics.MaxComplexity)
	fmt.Fprintf(writer, "| Comment Ratio | %.2f |\n", metrics.AvgCommentRatio)
	
	// Write file metrics if not in short mode
	if !options.Short {
		fmt.Fprintf(writer, "\n## File Metrics\n\n")
		fmt.Fprintf(writer, "| File | Lines | Code | Comments | Blank | Functions | Classes | Complexity | Comment Ratio |\n")
		fmt.Fprintf(writer, "|------|-------|------|----------|-------|-----------|---------|------------|---------------|\n")
		
		for _, fileMetrics := range metrics.Files {
			fmt.Fprintf(writer, "| `%s` | %d | %d | %d | %d | %d | %d | %.2f | %.2f |\n",
				fileMetrics.FilePath,
				fileMetrics.TotalLines,
				fileMetrics.CodeLines,
				fileMetrics.CommentLines,
				fileMetrics.BlankLines,
				fileMetrics.Functions,
				fileMetrics.Classes,
				fileMetrics.AvgComplexity,
				fileMetrics.CommentRatio)
		}
	}
	
	// Write processing info
	fmt.Fprintf(writer, "\n## Processing Information\n\n")
	fmt.Fprintf(writer, "- Processing time: %s\n", elapsedTime.Round(time.Millisecond))
	fmt.Fprintf(writer, "- Processing speed: %.2f files/sec\n", processingSpeed)
}
