package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/vitruves/gop/internal/parser"
	"github.com/vitruves/gop/internal/utils"
)

// CoherenceOptions contains options for the coherence command
type CoherenceOptions struct {
	// Input/Output options
	InputFile  string   // Path to input file containing list of files to process
	Directory  string   // Root directory to analyze
	Depth      int      // Maximum depth for directory traversal
	OutputFile string   // Path to output file for results
	Languages  []string // Languages to analyze (e.g., "c", "cpp")
	Excludes   []string // Directories or files to exclude

	// Processing options
	Jobs                int     // Number of concurrent jobs for processing
	SimilarityThreshold float64 // Threshold for function similarity detection (0.0-1.0)

	// Filter options
	CheckHeaders      bool // Whether to check header files
	CheckFiles        bool // Whether to check implementation files
	ShowDiscrepancies bool // Whether to show all discrepancies
	NonImplemented    bool // Whether to show only non-implemented declarations
	NotDeclared       bool // Whether to show only non-declared implementations

	// Output options
	IAOutput bool // Whether to output in a format optimized for AI processing
	Short    bool // Whether to use short output format
	Verbose  bool // Whether to enable verbose output
}

// Declaration represents a function or method declaration
type Declaration struct {
	Name       string // Function/method name
	Signature  string // Full function signature
	FilePath   string // Absolute path to the file containing the declaration
	LineNumber int    // Line number in the file
	IsHeader   bool   // Whether this is from a header file
	BaseFile   string // Base filename (used for matching headers and implementations)
}

// GetRelativePath returns the file path relative to the specified directory
func (d *Declaration) GetRelativePath(baseDir string) string {
	relPath, err := filepath.Rel(baseDir, d.FilePath)
	if err != nil {
		return d.FilePath
	}
	return relPath
}

// GetFormattedSignature returns a formatted signature suitable for display
func (d *Declaration) GetFormattedSignature() string {
	// Clean up the signature for display
	return strings.TrimSpace(d.Signature)
}

// Discrepancy represents a discrepancy between header and implementation
type Discrepancy struct {
	Type        string        // "not_implemented" or "not_declared"
	Declaration Declaration   // The declaration with the discrepancy
	Similar     []Declaration // Similar declarations if similarity threshold is enabled
	Severity    string        // "high", "medium", or "low" based on similarity matches
}

// GetSeverity calculates the severity of the discrepancy
func (d *Discrepancy) GetSeverity() string {
	if len(d.Similar) == 0 {
		return "high" // No similar declarations found
	} else if len(d.Similar) > 3 {
		return "low" // Many similar declarations found, likely a false positive
	}
	return "medium" // Some similar declarations found
}

// HasSimilarDeclarations returns true if there are similar declarations
func (d *Discrepancy) HasSimilarDeclarations() bool {
	return len(d.Similar) > 0
}

// CheckCoherence checks coherence between headers and implementations
func CheckCoherence(options CoherenceOptions) {
	if options.Verbose {
		fmt.Println(color.CyanString("Starting coherence check..."))
	}

	// Get list of files to process
	files, err := utils.GetFilesToProcess(options.InputFile, options.Directory, options.Depth, options.Languages, options.Excludes)
	if err != nil {
		fmt.Println(color.RedString("Error:"), err)
		return
	}

	if options.Verbose {
		fmt.Printf(color.CyanString("Found %d files to process\n"), len(files))
	}

	// Extract declarations from all files
	declarations := extractDeclarations(files, options)

	// Group declarations by their base names
	headerDecls, sourceDecls := groupDeclarationsByBase(declarations)

	// Find discrepancies
	discrepancies := findDiscrepancies(headerDecls, sourceDecls, options)

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

	// Write results
	if options.IAOutput {
		writeIACoherenceResults(writer, discrepancies, options)
	} else {
		writeCoherenceResults(writer, discrepancies, options)
	}

	// Print summary with more detailed information
	totalDiscrepancies := len(discrepancies)

	// Count types of discrepancies
	notImplementedCount := 0
	notDeclaredCount := 0
	for _, d := range discrepancies {
		if d.Type == "not_implemented" {
			notImplementedCount++
		} else {
			notDeclaredCount++
		}
	}

	// Print colorful summary
	fmt.Println()
	titleStyle := color.New(color.Bold, color.FgGreen).SprintFunc()
	fmt.Printf("%s\n\n", titleStyle("Coherence Check Results"))

	// Print overall results
	if totalDiscrepancies == 0 {
		fmt.Println(color.GreenString("✓ Perfect coherence! No discrepancies found."))
	} else {
		fmt.Printf("Found %s discrepancies:\n",
			color.New(color.Bold, color.FgYellow).Sprintf("%d", totalDiscrepancies))

		// Print breakdown
		fmt.Printf("  %s: %s\n",
			color.RedString("Not implemented"),
			color.New(color.Bold).Sprintf("%d", notImplementedCount))

		fmt.Printf("  %s: %s\n",
			color.YellowString("Not declared"),
			color.New(color.Bold).Sprintf("%d", notDeclaredCount))
	}

	// Print similarity info if enabled
	if options.SimilarityThreshold > 0 {
		similarCount := 0
		for _, d := range discrepancies {
			if len(d.Similar) > 0 {
				similarCount++
			}
		}

		if similarCount > 0 {
			fmt.Printf("\n%s similar functions found that might be related.\n",
				color.CyanString("%d", similarCount))
			fmt.Printf("Similarity threshold: %.2f\n", options.SimilarityThreshold)
		}
	}

	fmt.Println()
}

// extractDeclarations extracts declarations from all files
func extractDeclarations(files []string, options CoherenceOptions) []Declaration {
	// Use a thread-safe slice to collect declarations
	declarationsChan := make(chan []Declaration, options.Jobs)
	processedCount := int64(0)

	// Create a channel for files to process
	filesChan := make(chan string, len(files))
	for _, file := range files {
		filesChan <- file
	}
	close(filesChan)

	// Create a progress indicator
	progress := utils.NewProgressBar(len(files), "Extracting declarations")
	progress.Start()

	// Start a timer to measure performance
	startTime := time.Now()

	// Process files concurrently
	var wg sync.WaitGroup
	for i := 0; i < options.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Local buffer to reduce lock contention
			localDeclarations := make([]Declaration, 0, 100)

			for file := range filesChan {
				// Determine if this is a header file
				ext := filepath.Ext(file)
				isHeader := ext == ".h" || ext == ".hpp" || ext == ".hxx"

				// Only process relevant files based on options
				if (isHeader && !options.CheckHeaders) || (!isHeader && !options.CheckFiles) {
					atomic.AddInt64(&processedCount, 1)
					progress.Increment()
					continue
				}

				// Extract declarations from file
				fileDecls := parser.ExtractDeclarations(file)

				// Get the base filename for matching headers and implementations
				baseFile := filepath.Base(file)
				baseFile = strings.TrimSuffix(baseFile, filepath.Ext(baseFile))

				// Mark declarations as header or implementation
				for i := range fileDecls {
					decl := Declaration{
						Name:       fileDecls[i].Name,
						Signature:  fileDecls[i].Signature,
						FilePath:   file,
						LineNumber: fileDecls[i].LineNumber,
						IsHeader:   isHeader,
						BaseFile:   baseFile,
					}

					localDeclarations = append(localDeclarations, decl)
				}

				atomic.AddInt64(&processedCount, 1)
				progress.Increment()
			}

			// Send local declarations to the channel
			if len(localDeclarations) > 0 {
				declarationsChan <- localDeclarations
			}
		}()
	}

	// Close the declarations channel when all workers are done
	go func() {
		wg.Wait()
		close(declarationsChan)
	}()

	// Collect all declarations
	var declarations []Declaration
	for decls := range declarationsChan {
		declarations = append(declarations, decls...)
	}

	progress.Finish()

	// Log performance metrics if verbose
	if options.Verbose {
		elapsed := time.Since(startTime)
		fmt.Printf("Extracted %d declarations from %d files in %s (%.1f files/sec)\n",
			len(declarations),
			atomic.LoadInt64(&processedCount),
			elapsed.Round(time.Millisecond),
			float64(atomic.LoadInt64(&processedCount))/elapsed.Seconds())
	}

	// Sort declarations by name for consistent output
	sort.Slice(declarations, func(i, j int) bool {
		return declarations[i].Name < declarations[j].Name
	})

	return declarations
}

// groupDeclarationsByBase groups declarations by their base file names
func groupDeclarationsByBase(declarations []Declaration) (map[string][]Declaration, map[string][]Declaration) {
	headerDecls := make(map[string][]Declaration)
	sourceDecls := make(map[string][]Declaration)

	// Pre-allocate maps with estimated capacity
	estimatedFiles := len(declarations) / 10
	if estimatedFiles < 10 {
		estimatedFiles = 10
	}

	// Use the BaseFile field that we already computed in extractDeclarations
	for _, decl := range declarations {
		// Group by header or implementation
		if decl.IsHeader {
			headerDecls[decl.BaseFile] = append(headerDecls[decl.BaseFile], decl)
		} else {
			sourceDecls[decl.BaseFile] = append(sourceDecls[decl.BaseFile], decl)
		}
	}

	return headerDecls, sourceDecls
}

// findDiscrepancies finds discrepancies between header and implementation declarations
func findDiscrepancies(headerDecls, sourceDecls map[string][]Declaration, options CoherenceOptions) []Discrepancy {
	// Use a channel to collect discrepancies from multiple goroutines
	discrepancyChan := make(chan []Discrepancy, options.Jobs)

	// Create a progress indicator
	totalBases := len(headerDecls) + len(sourceDecls)
	progress := utils.NewProgressBar(totalBases, "Analyzing coherence")
	progress.Start()

	// Start a timer to measure performance
	startTime := time.Now()

	// Create channels for work distribution
	headerWorkChan := make(chan workItem, len(headerDecls))
	sourceWorkChan := make(chan workItem, len(sourceDecls))

	// Prepare work items
	if options.CheckHeaders {
		for baseName, decls := range headerDecls {
			sourceDeclsForBase, hasSource := sourceDecls[baseName]
			if !hasSource {
				sourceDeclsForBase = nil
			}

			headerWorkChan <- workItem{
				baseName:   baseName,
				decls:      decls,
				otherDecls: sourceDeclsForBase,
				isHeader:   true,
			}
		}
	}
	close(headerWorkChan)

	if options.CheckFiles {
		for baseName, decls := range sourceDecls {
			headerDeclsForBase, hasHeader := headerDecls[baseName]
			if !hasHeader {
				headerDeclsForBase = nil
			}

			sourceWorkChan <- workItem{
				baseName:   baseName,
				decls:      decls,
				otherDecls: headerDeclsForBase,
				isHeader:   false,
			}
		}
	}
	close(sourceWorkChan)

	// Track progress
	processedCount := int64(0)

	// Process work items concurrently
	var wg sync.WaitGroup
	for i := 0; i < options.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Local buffer to reduce lock contention
			localDiscrepancies := make([]Discrepancy, 0, 100)

			// Process header work items
			for work := range headerWorkChan {
				for _, headerDecl := range work.decls {
					implemented := false

					if work.otherDecls != nil {
						// Check if this declaration is implemented
						for _, sourceDecl := range work.otherDecls {
							if declarationsMatch(headerDecl, sourceDecl) {
								implemented = true
								break
							}
						}
					}

					// If not implemented and we're showing non-implemented declarations
					if !implemented && (options.ShowDiscrepancies || options.NonImplemented) {
						discrepancy := Discrepancy{
							Type:        "not_implemented",
							Declaration: headerDecl,
						}

						// Find similar declarations if similarity threshold is enabled
						if options.SimilarityThreshold > 0 && work.otherDecls != nil {
							for _, sourceDecl := range work.otherDecls {
								if declarationsSimilar(headerDecl, sourceDecl, options.SimilarityThreshold) {
									discrepancy.Similar = append(discrepancy.Similar, sourceDecl)
								}
							}
						}

						// Set severity based on similar declarations
						discrepancy.Severity = discrepancy.GetSeverity()

						localDiscrepancies = append(localDiscrepancies, discrepancy)
					}
				}

				atomic.AddInt64(&processedCount, 1)
				progress.Increment()
			}

			// Process source work items
			for work := range sourceWorkChan {
				for _, sourceDecl := range work.decls {
					declared := false

					if work.otherDecls != nil {
						// Check if this implementation is declared
						for _, headerDecl := range work.otherDecls {
							if declarationsMatch(sourceDecl, headerDecl) {
								declared = true
								break
							}
						}
					}

					// If not declared and we're showing non-declared implementations
					if !declared && (options.ShowDiscrepancies || options.NotDeclared) {
						discrepancy := Discrepancy{
							Type:        "not_declared",
							Declaration: sourceDecl,
						}

						// Find similar declarations if similarity threshold is enabled
						if options.SimilarityThreshold > 0 && work.otherDecls != nil {
							for _, headerDecl := range work.otherDecls {
								if declarationsSimilar(sourceDecl, headerDecl, options.SimilarityThreshold) {
									discrepancy.Similar = append(discrepancy.Similar, headerDecl)
								}
							}
						}

						// Set severity based on similar declarations
						discrepancy.Severity = discrepancy.GetSeverity()

						localDiscrepancies = append(localDiscrepancies, discrepancy)
					}
				}

				atomic.AddInt64(&processedCount, 1)
				progress.Increment()
			}

			// Send local discrepancies to the channel
			if len(localDiscrepancies) > 0 {
				discrepancyChan <- localDiscrepancies
			}
		}()
	}

	// Close the discrepancy channel when all workers are done
	go func() {
		wg.Wait()
		close(discrepancyChan)
	}()

	// Collect all discrepancies
	var discrepancies []Discrepancy
	for d := range discrepancyChan {
		discrepancies = append(discrepancies, d...)
	}

	progress.Finish()

	// Log performance metrics if verbose
	if options.Verbose {
		elapsed := time.Since(startTime)
		fmt.Printf("Analyzed %d files for coherence in %s (%.1f files/sec)\n",
			atomic.LoadInt64(&processedCount),
			elapsed.Round(time.Millisecond),
			float64(atomic.LoadInt64(&processedCount))/elapsed.Seconds())
		fmt.Printf("Found %d discrepancies between header and source files\n", len(discrepancies))
		fmt.Println("Coherence analysis completed successfully")
	}

	// Sort discrepancies by name for consistent output
	sort.Slice(discrepancies, func(i, j int) bool {
		return discrepancies[i].Declaration.Name < discrepancies[j].Declaration.Name
	})

	return discrepancies
}

// workItem represents a unit of work for the discrepancy finder
type workItem struct {
	baseName   string
	decls      []Declaration
	otherDecls []Declaration
	isHeader   bool
}

// declarationsMatch checks if two declarations match (same function/method)
func declarationsMatch(decl1, decl2 Declaration) bool {
	// Simple matching by name
	return decl1.Name == decl2.Name
}

// declarationsSimilar checks if two declarations are similar based on a threshold
func declarationsSimilar(decl1, decl2 Declaration, threshold float64) bool {
	// If names are identical, they're already matched
	if decl1.Name == decl2.Name {
		return false
	}

	// Calculate similarity between names
	similarity := calculateSimilarity(decl1.Name, decl2.Name)

	return similarity >= threshold
}

// calculateSimilarity calculates the similarity between two strings (0.0 to 1.0)
func calculateSimilarity(s1, s2 string) float64 {
	// Use Levenshtein distance for similarity
	distance := levenshteinDistance(s1, s2)
	maxLen := max(len(s1), len(s2))

	if maxLen == 0 {
		return 1.0 // Both strings are empty
	}

	return 1.0 - float64(distance)/float64(maxLen)
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create a matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			// Calculate minimum of three values
			deletion := matrix[i-1][j] + 1
			insertion := matrix[i][j-1] + 1
			substitution := matrix[i-1][j-1] + cost

			// Find minimum value
			matrix[i][j] = deletion
			if insertion < matrix[i][j] {
				matrix[i][j] = insertion
			}
			if substitution < matrix[i][j] {
				matrix[i][j] = substitution
			}
		}
	}

	return matrix[len(s1)][len(s2)]
}

// writeCoherenceResults writes coherence check results to the output file
func writeCoherenceResults(writer *bufio.Writer, discrepancies []Discrepancy, options CoherenceOptions) {
	// Write header
	fmt.Fprintf(writer, "# Coherence Check Results\n\n")

	// Group discrepancies by type
	notImplemented := make([]Discrepancy, 0)
	notDeclared := make([]Discrepancy, 0)

	for _, discrepancy := range discrepancies {
		if discrepancy.Type == "not_implemented" {
			notImplemented = append(notImplemented, discrepancy)
		} else {
			notDeclared = append(notDeclared, discrepancy)
		}
	}

	// Write summary
	fmt.Fprintf(writer, "## Summary\n\n")
	fmt.Fprintf(writer, "- Total discrepancies: %d\n", len(discrepancies))
	fmt.Fprintf(writer, "- Not implemented: %d\n", len(notImplemented))
	fmt.Fprintf(writer, "- Not declared: %d\n\n", len(notDeclared))

	// Write not implemented declarations
	if len(notImplemented) > 0 && (options.ShowDiscrepancies || options.NonImplemented) {
		fmt.Fprintf(writer, "## Not Implemented Declarations\n\n")

		for _, discrepancy := range notImplemented {
			decl := discrepancy.Declaration
			relPath, _ := filepath.Rel(options.Directory, decl.FilePath)

			fmt.Fprintf(writer, "### %s\n\n", decl.Name)
			fmt.Fprintf(writer, "- **File:** %s\n", relPath)
			fmt.Fprintf(writer, "- **Line:** %d\n", decl.LineNumber)
			fmt.Fprintf(writer, "- **Signature:** `%s`\n\n", decl.Signature)

			if len(discrepancy.Similar) > 0 {
				fmt.Fprintf(writer, "**Similar implementations:**\n\n")

				for _, similar := range discrepancy.Similar {
					similarRelPath, _ := filepath.Rel(options.Directory, similar.FilePath)
					fmt.Fprintf(writer, "- `%s` in %s (line %d)\n", similar.Name, similarRelPath, similar.LineNumber)
				}

				fmt.Fprintf(writer, "\n")
			}

			fmt.Fprintf(writer, "---\n\n")
		}
	}

	// Write not declared implementations
	if len(notDeclared) > 0 && (options.ShowDiscrepancies || options.NotDeclared) {
		fmt.Fprintf(writer, "## Not Declared Implementations\n\n")

		for _, discrepancy := range notDeclared {
			decl := discrepancy.Declaration
			relPath, _ := filepath.Rel(options.Directory, decl.FilePath)

			fmt.Fprintf(writer, "### %s\n\n", decl.Name)
			fmt.Fprintf(writer, "- **File:** %s\n", relPath)
			fmt.Fprintf(writer, "- **Line:** %d\n", decl.LineNumber)
			fmt.Fprintf(writer, "- **Signature:** `%s`\n\n", decl.Signature)

			if len(discrepancy.Similar) > 0 {
				fmt.Fprintf(writer, "**Similar declarations:**\n\n")

				for _, similar := range discrepancy.Similar {
					similarRelPath, _ := filepath.Rel(options.Directory, similar.FilePath)
					fmt.Fprintf(writer, "- `%s` in %s (line %d)\n", similar.Name, similarRelPath, similar.LineNumber)
				}

				fmt.Fprintf(writer, "\n")
			}

			fmt.Fprintf(writer, "---\n\n")
		}
	}
}

// writeIACoherenceResults writes coherence check results in a format optimized for AI processing
func writeIACoherenceResults(writer *bufio.Writer, discrepancies []Discrepancy, _ CoherenceOptions) {
	// Write in JSON format
	fmt.Fprintf(writer, "{\n")
	fmt.Fprintf(writer, "  \"discrepancies\": [\n")

	for i, discrepancy := range discrepancies {
		decl := discrepancy.Declaration

		fmt.Fprintf(writer, "    {\n")
		fmt.Fprintf(writer, "      \"type\": \"%s\",\n", discrepancy.Type)
		fmt.Fprintf(writer, "      \"name\": \"%s\",\n", decl.Name)
		fmt.Fprintf(writer, "      \"signature\": \"%s\",\n", strings.Replace(decl.Signature, "\"", "\\\"", -1))
		fmt.Fprintf(writer, "      \"file\": \"%s\",\n", decl.FilePath)
		fmt.Fprintf(writer, "      \"line\": %d,\n", decl.LineNumber)
		fmt.Fprintf(writer, "      \"is_header\": %t,\n", decl.IsHeader)

		if len(discrepancy.Similar) > 0 {
			fmt.Fprintf(writer, "      \"similar\": [\n")

			for j, similar := range discrepancy.Similar {
				fmt.Fprintf(writer, "        {\n")
				fmt.Fprintf(writer, "          \"name\": \"%s\",\n", similar.Name)
				fmt.Fprintf(writer, "          \"signature\": \"%s\",\n", strings.Replace(similar.Signature, "\"", "\\\"", -1))
				fmt.Fprintf(writer, "          \"file\": \"%s\",\n", similar.FilePath)
				fmt.Fprintf(writer, "          \"line\": %d,\n", similar.LineNumber)
				fmt.Fprintf(writer, "          \"is_header\": %t\n", similar.IsHeader)

				if j < len(discrepancy.Similar)-1 {
					fmt.Fprintf(writer, "        },\n")
				} else {
					fmt.Fprintf(writer, "        }\n")
				}
			}

			fmt.Fprintf(writer, "      ]\n")
		} else {
			fmt.Fprintf(writer, "      \"similar\": []\n")
		}

		if i < len(discrepancies)-1 {
			fmt.Fprintf(writer, "    },\n")
		} else {
			fmt.Fprintf(writer, "    }\n")
		}
	}

	fmt.Fprintf(writer, "  ],\n")

	// Add summary
	notImplemented := 0
	notDeclared := 0

	for _, discrepancy := range discrepancies {
		if discrepancy.Type == "not_implemented" {
			notImplemented++
		} else {
			notDeclared++
		}
	}

	fmt.Fprintf(writer, "  \"summary\": {\n")
	fmt.Fprintf(writer, "    \"total\": %d,\n", len(discrepancies))
	fmt.Fprintf(writer, "    \"not_implemented\": %d,\n", notImplemented)
	fmt.Fprintf(writer, "    \"not_declared\": %d\n", notDeclared)
	fmt.Fprintf(writer, "  }\n")
	fmt.Fprintf(writer, "}\n")
}
