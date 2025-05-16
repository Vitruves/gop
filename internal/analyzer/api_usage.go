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

// APIUsageOptions contains options for the api-usage command
type APIUsageOptions struct {
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
	APIDefinitionFile string   // Path to file containing API definitions
	CheckDeprecated   bool     // Whether to check for deprecated API usage
	CheckConsistency  bool     // Whether to check for consistent API usage
	TargetAPIs        []string // Specific APIs to analyze (empty means all)
	
	// Output options
	Short   bool   // Whether to use short output format
	Verbose bool   // Whether to enable verbose output
}

// APIUsage represents usage of an API function or method
type APIUsage struct {
	APIName     string // Name of the API function or method
	FilePath    string // Path to the file containing the usage
	Line        int    // Line number where the usage appears
	Context     string // Context of the usage (e.g., function name)
	IsDeprecated bool  // Whether the API is deprecated
	Alternatives []string // Alternative APIs to use instead
}

// APIDefinition represents a definition of an API function or method
type APIDefinition struct {
	Name         string   // Name of the API function or method
	Pattern      string   // Regex pattern to match the API usage
	IsDeprecated bool     // Whether the API is deprecated
	Alternatives []string // Alternative APIs to use instead
	Category     string   // Category of the API (e.g., "Memory", "String", "IO")
}

// AnalyzeAPIUsage analyzes API usage in code
func AnalyzeAPIUsage(options APIUsageOptions) {
	if options.Verbose {
		fmt.Println(color.CyanString("Starting API usage analysis..."))
	}

	// Set default values for options
	if options.Jobs <= 0 {
		options.Jobs = 4 // Default to 4 concurrent jobs
	}

	// If neither check is specified, enable both
	if !options.CheckDeprecated && !options.CheckConsistency {
		options.CheckDeprecated = true
		options.CheckConsistency = true
	}

	// Record start time for performance metrics
	startTime := time.Now()

	// Load API definitions
	apiDefs, err := loadAPIDefinitions(options.APIDefinitionFile)
	if err != nil {
		fmt.Println(color.RedString("Error loading API definitions:"), err)
		return
	}

	// Filter API definitions if specific APIs are requested
	if len(options.TargetAPIs) > 0 {
		targetAPIs := make(map[string]bool)
		for _, api := range options.TargetAPIs {
			targetAPIs[api] = true
		}

		filteredDefs := make([]APIDefinition, 0)
		for _, def := range apiDefs {
			if targetAPIs[def.Name] {
				filteredDefs = append(filteredDefs, def)
			}
		}
		apiDefs = filteredDefs
	}

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

	// Analyze API usage in files
	usages := analyzeAPIUsageInFiles(files, apiDefs, options)

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
	writeAPIUsageResults(writer, usages, apiDefs, options)

	// Calculate elapsed time
	elapsedTime := time.Since(startTime)
	
	// Print summary
	fmt.Println(color.GreenString("API Usage Analysis Results"))
	fmt.Printf("\nFound %d API usages in %d files\n", len(usages), len(files))
	
	// Count deprecated API usages
	deprecatedCount := 0
	for _, usage := range usages {
		if usage.IsDeprecated {
			deprecatedCount++
		}
	}
	
	if options.CheckDeprecated {
		fmt.Printf("Found %d uses of deprecated APIs\n", deprecatedCount)
	}
	
	fmt.Printf("Processing time: %s (%.2f files/sec)\n\n", utils.FormatDuration(elapsedTime), float64(len(files))/elapsedTime.Seconds())
	
	// No recommendation prints
}

// loadAPIDefinitions loads API definitions from a file
func loadAPIDefinitions(filePath string) ([]APIDefinition, error) {
	// If no file is specified, use built-in definitions
	if filePath == "" {
		return getBuiltInAPIDefinitions(), nil
	}
	
	// Load definitions from file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var definitions []APIDefinition
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		
		// Parse definition line
		// Format: name|pattern|deprecated|alternatives|category
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		
		name := strings.TrimSpace(parts[0])
		pattern := strings.TrimSpace(parts[1])
		isDeprecated := strings.TrimSpace(parts[2]) == "true"
		
		var alternatives []string
		if len(parts) > 3 {
			alternatives = strings.Split(strings.TrimSpace(parts[3]), ",")
			for i := range alternatives {
				alternatives[i] = strings.TrimSpace(alternatives[i])
			}
		}
		
		category := ""
		if len(parts) > 4 {
			category = strings.TrimSpace(parts[4])
		}
		
		definitions = append(definitions, APIDefinition{
			Name:         name,
			Pattern:      pattern,
			IsDeprecated: isDeprecated,
			Alternatives: alternatives,
			Category:     category,
		})
	}
	
	return definitions, nil
}

// getBuiltInAPIDefinitions returns built-in API definitions
func getBuiltInAPIDefinitions() []APIDefinition {
	return []APIDefinition{
		// C Standard Library - String functions
		{
			Name:         "strcpy",
			Pattern:      `\bstrcpy\s*\(`,
			IsDeprecated: true,
			Alternatives: []string{"strncpy", "strlcpy"},
			Category:     "String",
		},
		{
			Name:         "strcat",
			Pattern:      `\bstrcat\s*\(`,
			IsDeprecated: true,
			Alternatives: []string{"strncat", "strlcat"},
			Category:     "String",
		},
		{
			Name:         "gets",
			Pattern:      `\bgets\s*\(`,
			IsDeprecated: true,
			Alternatives: []string{"fgets"},
			Category:     "IO",
		},
		{
			Name:         "sprintf",
			Pattern:      `\bsprintf\s*\(`,
			IsDeprecated: true,
			Alternatives: []string{"snprintf"},
			Category:     "String",
		},
		
		// C Standard Library - Memory functions
		{
			Name:         "malloc",
			Pattern:      `\bmalloc\s*\(`,
			IsDeprecated: false,
			Alternatives: []string{},
			Category:     "Memory",
		},
		{
			Name:         "calloc",
			Pattern:      `\bcalloc\s*\(`,
			IsDeprecated: false,
			Alternatives: []string{},
			Category:     "Memory",
		},
		{
			Name:         "realloc",
			Pattern:      `\brealloc\s*\(`,
			IsDeprecated: false,
			Alternatives: []string{},
			Category:     "Memory",
		},
		{
			Name:         "free",
			Pattern:      `\bfree\s*\(`,
			IsDeprecated: false,
			Alternatives: []string{},
			Category:     "Memory",
		},
		
		// C++ - Old style casts
		{
			Name:         "C-style cast",
			Pattern:      `\(\s*[a-zA-Z_][a-zA-Z0-9_]*\s*\*?\s*\)`,
			IsDeprecated: true,
			Alternatives: []string{"static_cast", "dynamic_cast", "const_cast", "reinterpret_cast"},
			Category:     "C++",
		},
		
		// C++ - Memory management
		{
			Name:         "new",
			Pattern:      `\bnew\s+[a-zA-Z_][a-zA-Z0-9_]*`,
			IsDeprecated: false,
			Alternatives: []string{"std::make_unique", "std::make_shared"},
			Category:     "C++",
		},
		{
			Name:         "delete",
			Pattern:      `\bdelete\s+[a-zA-Z_][a-zA-Z0-9_]*`,
			IsDeprecated: false,
			Alternatives: []string{},
			Category:     "C++",
		},
		{
			Name:         "new[]",
			Pattern:      `\bnew\s+[a-zA-Z_][a-zA-Z0-9_]*\s*\[`,
			IsDeprecated: false,
			Alternatives: []string{"std::vector", "std::array"},
			Category:     "C++",
		},
		{
			Name:         "delete[]",
			Pattern:      `\bdelete\s*\[\s*\]\s+[a-zA-Z_][a-zA-Z0-9_]*`,
			IsDeprecated: false,
			Alternatives: []string{},
			Category:     "C++",
		},
		
		// C++ - Deprecated features
		{
			Name:         "auto_ptr",
			Pattern:      `\bauto_ptr\s*<`,
			IsDeprecated: true,
			Alternatives: []string{"std::unique_ptr"},
			Category:     "C++",
		},
		{
			Name:         "std::random_shuffle",
			Pattern:      `\bstd\s*::\s*random_shuffle\s*\(`,
			IsDeprecated: true,
			Alternatives: []string{"std::shuffle"},
			Category:     "C++",
		},
		{
			Name:         "std::binary_function",
			Pattern:      `\bstd\s*::\s*binary_function\s*<`,
			IsDeprecated: true,
			Alternatives: []string{"lambda functions"},
			Category:     "C++",
		},
		{
			Name:         "std::unary_function",
			Pattern:      `\bstd\s*::\s*unary_function\s*<`,
			IsDeprecated: true,
			Alternatives: []string{"lambda functions"},
			Category:     "C++",
		},
	}
}

// analyzeAPIUsageInFiles analyzes API usage in files
func analyzeAPIUsageInFiles(files []string, apiDefs []APIDefinition, options APIUsageOptions) []APIUsage {
	var usages []APIUsage
	var mutex sync.Mutex
	var wg sync.WaitGroup
	
	// Set up a worker pool
	fileChan := make(chan string, len(files))
	
	// Create a progress bar
	progress := utils.NewProgressBar(len(files), "Analyzing API usage")
	progress.Start()
	
	// Compile regex patterns for each API definition
	patterns := make(map[string]*regexp.Regexp)
	for _, def := range apiDefs {
		patterns[def.Name] = regexp.MustCompile(def.Pattern)
	}
	
	// Start workers
	for i := 0; i < options.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for file := range fileChan {
				// Analyze API usage in this file
				fileUsages := analyzeFileAPIUsage(file, apiDefs, patterns, options)
				
				// Add to global list
				if len(fileUsages) > 0 {
					mutex.Lock()
					usages = append(usages, fileUsages...)
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
	
	// Sort usages by file path and line number
	sort.Slice(usages, func(i, j int) bool {
		if usages[i].FilePath == usages[j].FilePath {
			return usages[i].Line < usages[j].Line
		}
		return usages[i].FilePath < usages[j].FilePath
	})
	
	return usages
}

// analyzeFileAPIUsage analyzes API usage in a single file
func analyzeFileAPIUsage(filePath string, apiDefs []APIDefinition, patterns map[string]*regexp.Regexp, options APIUsageOptions) []APIUsage {
	var usages []APIUsage
	
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return usages
	}
	defer file.Close()
	
	// Read file content
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	
	// Extract function context information
	functions := extractFunctions(filePath, lines)
	
	// Create a map of line numbers to function contexts
	lineToContext := make(map[int]string)
	for _, fn := range functions {
		for line := fn.StartLine; line <= fn.EndLine; line++ {
			lineToContext[line] = fn.FunctionName
		}
	}
	
	// Check each line for API usage
	for i, line := range lines {
		lineNum := i + 1
		
		// Get the current function context
		context := lineToContext[lineNum]
		if context == "" {
			context = "global scope"
		}
		
		// Check each API definition
		for _, def := range apiDefs {
			// Skip deprecated API check if not requested
			if def.IsDeprecated && !options.CheckDeprecated {
				continue
			}
			
			// Check if the line contains the API
			if patterns[def.Name].MatchString(line) {
				usages = append(usages, APIUsage{
					APIName:      def.Name,
					FilePath:     filePath,
					Line:         lineNum,
					Context:      context,
					IsDeprecated: def.IsDeprecated,
					Alternatives: def.Alternatives,
				})
			}
		}
	}
	
	return usages
}

// writeAPIUsageResults writes API usage analysis results to the output file
func writeAPIUsageResults(writer *bufio.Writer, usages []APIUsage, apiDefs []APIDefinition, options APIUsageOptions) {
	// Write header
	fmt.Fprintf(writer, "# API Usage Analysis Results\n\n")
	
	// Write summary
	fmt.Fprintf(writer, "## Summary\n\n")
	fmt.Fprintf(writer, "- Total API usages found: %d\n", len(usages))
	
	// Count usages by API
	usagesByAPI := make(map[string]int)
	for _, usage := range usages {
		usagesByAPI[usage.APIName]++
	}
	
	// Count deprecated API usages
	deprecatedCount := 0
	for _, usage := range usages {
		if usage.IsDeprecated {
			deprecatedCount++
		}
	}
	
	if options.CheckDeprecated {
		fmt.Fprintf(writer, "- Deprecated API usages: %d\n", deprecatedCount)
	}
	
	// Group API definitions by category
	apisByCategory := make(map[string][]APIDefinition)
	for _, def := range apiDefs {
		apisByCategory[def.Category] = append(apisByCategory[def.Category], def)
	}
	
	// Write API usage by category
	fmt.Fprintf(writer, "\n## API Usage by Category\n\n")
	
	// Sort categories for consistent output
	categories := make([]string, 0, len(apisByCategory))
	for category := range apisByCategory {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	
	for _, category := range categories {
		fmt.Fprintf(writer, "### %s\n\n", category)
		
		// Table header
		fmt.Fprintf(writer, "| API | Usage Count | Deprecated | Alternatives |\n")
		fmt.Fprintf(writer, "|-----|-------------|------------|-------------|\n")
		
		// Sort APIs within category by name
		apis := apisByCategory[category]
		sort.Slice(apis, func(i, j int) bool {
			return apis[i].Name < apis[j].Name
		})
		
		for _, api := range apis {
			count := usagesByAPI[api.Name]
			if count == 0 && !options.Verbose {
				continue
			}
			
			deprecated := "No"
			if api.IsDeprecated {
				deprecated = "**Yes**"
			}
			
			alternatives := "-"
			if len(api.Alternatives) > 0 {
				alternatives = strings.Join(api.Alternatives, ", ")
			}
			
			fmt.Fprintf(writer, "| `%s` | %d | %s | %s |\n", api.Name, count, deprecated, alternatives)
		}
		
		fmt.Fprintf(writer, "\n")
	}
	
	// If there are deprecated API usages, write a section for them
	if deprecatedCount > 0 && options.CheckDeprecated {
		fmt.Fprintf(writer, "## Deprecated API Usages\n\n")
		
		// Group by file
		deprecatedByFile := make(map[string][]APIUsage)
		for _, usage := range usages {
			if usage.IsDeprecated {
				deprecatedByFile[usage.FilePath] = append(deprecatedByFile[usage.FilePath], usage)
			}
		}
		
		// Sort files for consistent output
		files := make([]string, 0, len(deprecatedByFile))
		for file := range deprecatedByFile {
			files = append(files, file)
		}
		sort.Strings(files)
		
		for _, file := range files {
			relPath, _ := filepath.Rel(options.Directory, file)
			fmt.Fprintf(writer, "### %s\n\n", relPath)
			
			// Table header
			fmt.Fprintf(writer, "| Line | API | Context | Alternatives |\n")
			fmt.Fprintf(writer, "|------|-----|---------|-------------|\n")
			
			// Sort usages by line number
			sort.Slice(deprecatedByFile[file], func(i, j int) bool {
				return deprecatedByFile[file][i].Line < deprecatedByFile[file][j].Line
			})
			
			for _, usage := range deprecatedByFile[file] {
				alternatives := "-"
				if len(usage.Alternatives) > 0 {
					alternatives = strings.Join(usage.Alternatives, ", ")
				}
				
				fmt.Fprintf(writer, "| %d | `%s` | `%s` | %s |\n", usage.Line, usage.APIName, usage.Context, alternatives)
			}
			
			fmt.Fprintf(writer, "\n")
		}
	}
	
	// Write consistency analysis if requested
	if options.CheckConsistency {
		fmt.Fprintf(writer, "## API Usage Consistency\n\n")
		
		// Check for inconsistent API usage patterns
		// For example, using both malloc/free and new/delete in the same codebase
		
		// Check memory management consistency
		cStyleMemory := 0
		cppStyleMemory := 0
		
		for _, usage := range usages {
			if usage.APIName == "malloc" || usage.APIName == "calloc" || usage.APIName == "realloc" || usage.APIName == "free" {
				cStyleMemory++
			} else if usage.APIName == "new" || usage.APIName == "delete" || usage.APIName == "new[]" || usage.APIName == "delete[]" {
				cppStyleMemory++
			}
		}
		
		if cStyleMemory > 0 && cppStyleMemory > 0 {
			fmt.Fprintf(writer, "### Inconsistent Memory Management\n\n")
			fmt.Fprintf(writer, "The codebase uses both C-style memory management (malloc/free) and C++-style memory management (new/delete).\n")
			fmt.Fprintf(writer, "- C-style memory functions: %d usages\n", cStyleMemory)
			fmt.Fprintf(writer, "- C++-style memory operators: %d usages\n\n", cppStyleMemory)
			fmt.Fprintf(writer, "**Recommendation**: Standardize on one approach for better maintainability.\n\n")
		}
		
		// Check string handling consistency
		cStyleStrings := 0
		cppStyleStrings := 0
		
		for _, usage := range usages {
			if usage.APIName == "strcpy" || usage.APIName == "strncpy" || usage.APIName == "strcat" || usage.APIName == "strncat" {
				cStyleStrings++
			}
			// Check for std::string usage (not in our default definitions, but could be added)
		}
		
		if cStyleStrings > 0 && cppStyleStrings > 0 {
			fmt.Fprintf(writer, "### Inconsistent String Handling\n\n")
			fmt.Fprintf(writer, "The codebase uses both C-style strings and C++ string classes.\n")
			fmt.Fprintf(writer, "- C-style string functions: %d usages\n", cStyleStrings)
			fmt.Fprintf(writer, "- C++ string class: %d usages\n\n", cppStyleStrings)
		}
	}
	
	// No recommendations section
}
