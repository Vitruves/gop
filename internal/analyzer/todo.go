package analyzer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/vitruves/gop/internal/utils"
)

// TodoOptions contains options for the todo command
type TodoOptions struct {
	// Input/Output options
	InputFile  string   // Path to input file containing list of files to process
	Directory  string   // Root directory to analyze
	Depth      int      // Maximum depth for directory traversal
	OutputFile string   // Path to output file for results
	Languages  []string // Languages to analyze (e.g., "c", "cpp")
	Excludes   []string // Directories or files to exclude

	// Processing options
	Jobs        int      // Number of concurrent jobs for processing
	MaxContext  int      // Maximum number of context lines to include (before/after)
	GroupByType bool     // Whether to group TODOs by type (TODO, FIXME, etc.)
	Filter      []string // Filter TODOs by type (e.g., "TODO", "FIXME")

	// Output options
	JSONOutput bool // Whether to output in JSON format for machine processing
	Short      bool // Whether to use short output format
	Verbose    bool // Whether to enable verbose output
}

// TodoType represents the type of a TODO item
type TodoType string

// Predefined TODO types
const (
	TypeTODO        TodoType = "TODO"
	TypeFIXME       TodoType = "FIXME"
	TypeHACK        TodoType = "HACK"
	TypeNOTE        TodoType = "NOTE"
	TypeBUG         TodoType = "BUG"
	TypeOPTIMIZE    TodoType = "OPTIMIZE"
	TypeWORKAROUND  TodoType = "WORKAROUND"
	TypePLACEHOLDER TodoType = "PLACEHOLDER"
	TypeSIMPLIFY    TodoType = "SIMPLIFY"
	TypeHEURISTIC   TodoType = "HEURISTIC"
	TypeOTHER       TodoType = "OTHER"
)

// TodoItem represents a TODO item found in the code
type TodoItem struct {
	FilePath    string    // Path to the file containing the TODO
	LineNumber  int       // Line number where the TODO is located
	Content     string    // Content of the line containing the TODO
	Context     string    // Context (surrounding lines)
	Function    string    // Function/method containing the TODO
	Type        TodoType  // Type of TODO (TODO, FIXME, etc.)
	Description string    // Extracted description of the TODO
	Priority    int       // Priority (1-5, where 1 is highest)
	Author      string    // Author of the TODO (if specified)
	CreatedAt   time.Time // Creation time (if specified)
}

// FindTodos finds all TODO items in the codebase
func FindTodos(options TodoOptions) {
	// Record start time for performance metrics
	startTime := time.Now()

	if options.Verbose {
		fmt.Println(color.CyanString("Starting TODO search..."))
	}

	// Set default values for options
	if options.MaxContext <= 0 {
		options.MaxContext = 2 // Default to 2 lines of context before/after
	}

	if options.Jobs <= 0 {
		options.Jobs = 4 // Default to 4 concurrent jobs
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

	// Process files and extract TODO items
	var processedFiles atomic.Int64
	todos := extractTodoItems(files, options, &processedFiles)
	
	// Apply filter if specified
	if len(options.Filter) > 0 {
		filteredTodos := []TodoItem{}
		filterMap := make(map[string]bool)
		
		// Create a map of filter types for faster lookup
		for _, filter := range options.Filter {
			filterMap[strings.ToUpper(filter)] = true
		}
		
		// Only keep TODOs that match the filter
		for _, todo := range todos {
			if filterMap[string(todo.Type)] {
				filteredTodos = append(filteredTodos, todo)
			}
		}
		
		todos = filteredTodos
	}

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

	// Write output based on format
	if options.JSONOutput {
		// Write JSON output
		writeJSONTodoOutput(writer, todos, files, elapsedTime, processingSpeed)
	} else {
		// Write markdown output
		if options.GroupByType {
			writeMarkdownTodoOutputByType(writer, todos, files, options, elapsedTime, processingSpeed)
		} else {
			writeMarkdownTodoOutputByFile(writer, todos, files, options, elapsedTime, processingSpeed)
		}
	}

	// Print a summary of the results to the console only if we're writing to a file
	if options.OutputFile != "" {
		printTodoSummary(todos, files, elapsedTime, processingSpeed)
	}
}

// extractTodoItems extracts TODO items from the given files
func extractTodoItems(files []string, options TodoOptions, processedFiles *atomic.Int64) []TodoItem {
	var todos []TodoItem
	var mutex sync.Mutex
	var wg sync.WaitGroup

	// Create a channel for files to process
	filesChan := make(chan string, len(files))
	for _, file := range files {
		filesChan <- file
	}
	close(filesChan)

	// Compile regex patterns for TODO items once to avoid recompilation
	todoPatterns := []*regexp.Regexp{
		// Core patterns
		regexp.MustCompile(`(?i)TODO`),
		regexp.MustCompile(`(?i)FIXME`),
		regexp.MustCompile(`(?i)BUG`),
		regexp.MustCompile(`(?i)NOTE`),
		
		// Placeholder patterns
		regexp.MustCompile(`(?i)PLACEHOLDER`),
		regexp.MustCompile(`(?i)PLACE[\s-]HOLDER`),
		regexp.MustCompile(`(?i)STUB`),
		
		// Simplification patterns
		regexp.MustCompile(`(?i)SIMPLIF`),
		regexp.MustCompile(`(?i)REFACTOR`),
		regexp.MustCompile(`(?i)SIMPLE`),
		
		// Heuristic patterns
		regexp.MustCompile(`(?i)HEURISTIC`),
		regexp.MustCompile(`(?i)APPROXIMAT`),
		regexp.MustCompile(`(?i)ESTIMATE`),
		
		// Other common patterns
		regexp.MustCompile(`(?i)HACK`),
		regexp.MustCompile(`(?i)WORKAROUND`),
		regexp.MustCompile(`(?i)WORK[\s-]AROUND`),
		regexp.MustCompile(`(?i)TEMPORARY`),
		regexp.MustCompile(`(?i)TEMP`),
		regexp.MustCompile(`(?i)XXX`),
		regexp.MustCompile(`(?i)OPTIMIZE`),
		regexp.MustCompile(`(?i)IMPROVEMENT`),
	}

	// Create a progress bar - only create it once outside the goroutines
	progress := utils.NewProgressBar(len(files), "Searching for TODOs")
	progress.Start()

	// Create a channel for results to avoid lock contention
	resultsChan := make(chan []TodoItem, options.Jobs)

	// Process files concurrently
	for i := 0; i < options.Jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for file := range filesChan {
				// Find TODOs in this file
				fileTodos := findTodosInFile(file, todoPatterns, options.MaxContext)

				// Process each TODO to extract type, description, etc.
				for i := range fileTodos {
					// Set the TODO type
					fileTodos[i].Type = extractTodoTypeEnum(fileTodos[i].Content)

					// Extract description from the TODO line
					fileTodos[i].Description = extractTodoDescription(fileTodos[i].Content, string(fileTodos[i].Type))

					// Extract priority if specified (e.g., TODO(P1): ...)
					fileTodos[i].Priority = extractTodoPriority(fileTodos[i].Content)

					// Extract author if specified (e.g., TODO(john): ...)
					fileTodos[i].Author = extractTodoAuthor(fileTodos[i].Content)
				}

				// Send results to the channel
				if len(fileTodos) > 0 {
					resultsChan <- fileTodos
				}

				// Update progress
				processedFiles.Add(1)
				progress.Increment()
			}
		}()
	}

	// Start a goroutine to collect results
	go func() {
		for fileTodos := range resultsChan {
			mutex.Lock()
			todos = append(todos, fileTodos...)
			mutex.Unlock()
		}
	}()

	// Wait for all processing to complete
	wg.Wait()
	close(resultsChan)
	progress.Finish()

	return todos
}

// writeJSONTodoOutput writes TODO items to a file in JSON format
func writeJSONTodoOutput(writer *bufio.Writer, todos []TodoItem, files []string, elapsedTime time.Duration, processingSpeed float64) {
	// Create a JSON-friendly structure
	type JSONTodoItem struct {
		FilePath    string    `json:"file_path"`
		LineNumber  int       `json:"line_number"`
		Content     string    `json:"content"`
		Function    string    `json:"function,omitempty"`
		Type        string    `json:"type"`
		Description string    `json:"description,omitempty"`
		Priority    int       `json:"priority"`
		Author      string    `json:"author,omitempty"`
		CreatedAt   time.Time `json:"created_at"`
	}

	type JSONTodoOutput struct {
		TotalTodos      int            `json:"total_todos"`
		TotalFiles      int            `json:"total_files"`
		ProcessedFiles  int            `json:"processed_files"`
		ElapsedTime     string         `json:"elapsed_time"`
		ProcessingSpeed float64        `json:"processing_speed"`
		TodosByType     map[string]int `json:"todos_by_type"`
		TodoItems       []JSONTodoItem `json:"todo_items"`
	}

	// Convert TodoItems to JSONTodoItems
	jsonTodos := make([]JSONTodoItem, 0, len(todos))
	todosByType := make(map[string]int)

	for _, todo := range todos {
		// Count by type
		todoType := string(todo.Type)
		todosByType[todoType]++

		// Convert to JSON item
		jsonTodos = append(jsonTodos, JSONTodoItem{
			FilePath:    todo.FilePath,
			LineNumber:  todo.LineNumber,
			Content:     todo.Content,
			Function:    todo.Function,
			Type:        todoType,
			Description: todo.Description,
			Priority:    todo.Priority,
			Author:      todo.Author,
			CreatedAt:   todo.CreatedAt,
		})
	}

	// Create the output structure
	output := JSONTodoOutput{
		TotalTodos:      len(todos),
		TotalFiles:      len(files),
		ProcessedFiles:  len(files),
		ElapsedTime:     elapsedTime.String(),
		ProcessingSpeed: processingSpeed,
		TodosByType:     todosByType,
		TodoItems:       jsonTodos,
	}

	// Marshal to JSON with indentation
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Println(color.RedString("Error marshaling to JSON:"), err)
		return
	}

	// Write to file
	_, err = writer.Write(jsonData)
	if err != nil {
		fmt.Println(color.RedString("Error writing JSON to file:"), err)
	}
}

// writeMarkdownTodoOutputByFile writes TODO items to a file in Markdown format, grouped by file
func writeMarkdownTodoOutputByFile(writer *bufio.Writer, todos []TodoItem, files []string, options TodoOptions, elapsedTime time.Duration, processingSpeed float64) {
	// Write header
	fmt.Fprintf(writer, "# TODO Items\n\n")
	fmt.Fprintf(writer, "Found %d TODO items in %d files.\n\n", len(todos), len(files))
	fmt.Fprintf(writer, "Processing time: %s (%.2f files/sec)\n\n", elapsedTime.Round(time.Millisecond), processingSpeed)

	// Group TODOs by file
	todosByFile := make(map[string][]TodoItem)
	for _, todo := range todos {
		todosByFile[todo.FilePath] = append(todosByFile[todo.FilePath], todo)
	}

	// Write TODOs by file
	for file, fileTodos := range todosByFile {
		relPath, err := filepath.Rel(options.Directory, file)
		if err != nil {
			relPath = file
		}

		fmt.Fprintf(writer, "## %s\n\n", relPath)

		for _, todo := range fileTodos {
			// Determine color for TODO type
			todoTypeStr := string(todo.Type)
			fmt.Fprintf(writer, "### [%s] Line %d\n\n", todoTypeStr, todo.LineNumber)

			// Add metadata
			if todo.Function != "" {
				fmt.Fprintf(writer, "**Function:** `%s`\n\n", todo.Function)
			}

			if todo.Priority != 3 { // Only show if not default priority
				fmt.Fprintf(writer, "**Priority:** %d\n\n", todo.Priority)
			}

			if todo.Author != "" {
				fmt.Fprintf(writer, "**Author:** %s\n\n", todo.Author)
			}

			// Show content and description
			fmt.Fprintf(writer, "```\n%s\n```\n\n", todo.Content)

			if todo.Description != "" && todo.Description != todo.Content {
				fmt.Fprintf(writer, "**Description:** %s\n\n", todo.Description)
			}

			// Show context if available
			if todo.Context != "" && todo.Context != todo.Content {
				fmt.Fprintf(writer, "**Context:**\n\n```\n%s\n```\n\n", todo.Context)
			}

			fmt.Fprintf(writer, "---\n\n")
		}
	}
}

// writeMarkdownTodoOutputByType writes TODO items to a file in Markdown format, grouped by type
func writeMarkdownTodoOutputByType(writer *bufio.Writer, todos []TodoItem, files []string, options TodoOptions, elapsedTime time.Duration, processingSpeed float64) {
	// Write header
	fmt.Fprintf(writer, "# TODO Items by Type\n\n")
	fmt.Fprintf(writer, "Found %d TODO items in %d files.\n\n", len(todos), len(files))
	fmt.Fprintf(writer, "Processing time: %s (%.2f files/sec)\n\n", elapsedTime.Round(time.Millisecond), processingSpeed)

	// Group TODOs by type
	todosByType := make(map[TodoType][]TodoItem)
	for _, todo := range todos {
		todosByType[todo.Type] = append(todosByType[todo.Type], todo)
	}

	// Define the order of types to display
	typeOrder := []TodoType{
		TypeFIXME, // Critical issues first
		TypeBUG,
		TypeTODO,
		TypeHACK,
		TypeOPTIMIZE,
		TypeWORKAROUND,
		TypePLACEHOLDER,
		TypeSIMPLIFY,
		TypeHEURISTIC,
		TypeNOTE,
		TypeOTHER,
	}

	// Write summary of counts by type
	fmt.Fprintf(writer, "## Summary\n\n")
	for _, todoType := range typeOrder {
		if items, exists := todosByType[todoType]; exists && len(items) > 0 {
			fmt.Fprintf(writer, "- **%s**: %d\n", todoType, len(items))
		}
	}
	fmt.Fprintf(writer, "\n")

	// Write TODOs by type
	for _, todoType := range typeOrder {
		items, exists := todosByType[todoType]
		if !exists || len(items) == 0 {
			continue
		}

		fmt.Fprintf(writer, "## %s (%d)\n\n", todoType, len(items))

		// Sort items by priority
		sort.Slice(items, func(i, j int) bool {
			// Lower priority number = higher priority
			return items[i].Priority < items[j].Priority
		})

		for _, todo := range items {
			// Get relative path for the file
			relPath, err := filepath.Rel(options.Directory, todo.FilePath)
			if err != nil {
				relPath = todo.FilePath
			}

			// Show file and line info
			fmt.Fprintf(writer, "### %s (Line %d)\n\n", relPath, todo.LineNumber)

			// Add metadata
			if todo.Function != "" {
				fmt.Fprintf(writer, "**Function:** `%s`\n\n", todo.Function)
			}

			if todo.Priority != 3 { // Only show if not default priority
				fmt.Fprintf(writer, "**Priority:** %d\n\n", todo.Priority)
			}

			if todo.Author != "" {
				fmt.Fprintf(writer, "**Author:** %s\n\n", todo.Author)
			}

			// Show content and description
			fmt.Fprintf(writer, "```\n%s\n```\n\n", todo.Content)

			if todo.Description != "" && todo.Description != todo.Content {
				fmt.Fprintf(writer, "**Description:** %s\n\n", todo.Description)
			}

			fmt.Fprintf(writer, "---\n\n")
		}
	}
}

// printTodoSummary prints a summary of the TODO search results to the console
func printTodoSummary(todos []TodoItem, files []string, elapsedTime time.Duration, processingSpeed float64) {
	fmt.Println()
	titleStyle := color.New(color.Bold, color.FgGreen).SprintFunc()
	fmt.Printf("%s\n\n", titleStyle("TODO Search Results"))

	// Group TODOs by type
	todosByType := make(map[TodoType]int)
	for _, todo := range todos {
		todosByType[todo.Type]++
	}

	// Print summary by type
	fmt.Printf("Found %s TODO items in %s files:\n",
		color.New(color.Bold, color.FgCyan).Sprintf("%d", len(todos)),
		color.New(color.Bold, color.FgCyan).Sprintf("%d", len(files)))

	// Define the order of types to display
	typeOrder := []TodoType{
		TypeFIXME, // Critical issues first
		TypeBUG,
		TypeTODO,
		TypeHACK,
		TypeOPTIMIZE,
		TypeWORKAROUND,
		TypePLACEHOLDER,
		TypeSIMPLIFY,
		TypeHEURISTIC,
		TypeNOTE,
		TypeOTHER,
	}

	for _, todoType := range typeOrder {
		count, exists := todosByType[todoType]
		if !exists || count == 0 {
			continue
		}

		var typeColor func(string, ...interface{}) string
		switch todoType {
		case TypeTODO:
			typeColor = color.YellowString
		case TypeFIXME, TypeBUG:
			typeColor = color.RedString
		case TypeHACK, TypeWORKAROUND:
			typeColor = color.MagentaString
		case TypeOPTIMIZE:
			typeColor = color.GreenString
		default:
			typeColor = color.CyanString
		}

		fmt.Printf("  %s: %s\n",
			typeColor("%-12s", todoType),
			color.New(color.Bold).Sprintf("%d", count))
	}

	// Print performance metrics
	fmt.Printf("\nProcessing time: %s (%.2f files/sec)\n",
		color.New(color.Bold).Sprintf("%s", elapsedTime.Round(time.Millisecond)),
		processingSpeed)

	fmt.Println()
}

// findTodosInFile finds TODO items in a single file
func findTodosInFile(filePath string, patterns []*regexp.Regexp, maxContext int) []TodoItem {
	var todos []TodoItem

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return todos
	}

	lines := strings.Split(string(content), "\n")

	// Current function context
	currentFunction := ""
	functionRegex := regexp.MustCompile(`(?m)^(?:\w+\s+)+(\w+)\s*\([^)]*\)\s*(?:{|\n\s*{)`)

	// Track processed comment blocks to avoid duplicates
	processedBlocks := make(map[int]bool)

	// Scan each line for TODO patterns
	for i, line := range lines {
		lineNumber := i + 1

		// Skip if this line is part of a comment block we've already processed
		if processedBlocks[lineNumber] {
			continue
		}

		// Check if this line defines a function
		if matches := functionRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentFunction = matches[1]
		}

		// Check for TODO patterns
		for _, pattern := range patterns {
			if pattern.MatchString(line) {
				// Found a TODO
				todo := TodoItem{
					FilePath:   filePath,
					LineNumber: lineNumber,
					Content:    line,
					Function:   currentFunction,
					Type:       TypeOTHER,  // Default type, will be refined later
					Priority:   3,          // Default medium priority
					CreatedAt:  time.Now(), // Default to current time
				}

				// Add context (a few lines before and after)
				contextStart := utils.Max(0, i-maxContext)
				contextEnd := utils.Min(len(lines)-1, i+maxContext)
				contextLines := lines[contextStart : contextEnd+1]
				todo.Context = strings.Join(contextLines, "\n")

				todos = append(todos, todo)

				// Mark this comment block as processed
				// Consider lines in the context as part of the same comment block
				for j := contextStart; j <= contextEnd; j++ {
					processedBlocks[j+1] = true // +1 because line numbers are 1-indexed
				}

				break // Only add this line once, even if it matches multiple patterns
			}
		}
	}

	return todos
}

// extractTodoTypeEnum extracts the TodoType from a line
func extractTodoTypeEnum(line string) TodoType {
	lineUpper := strings.ToUpper(line)

	// Check for each type in order of precedence
	if strings.Contains(lineUpper, string(TypeTODO)) {
		return TypeTODO
	}
	if strings.Contains(lineUpper, string(TypeFIXME)) {
		return TypeFIXME
	}
	if strings.Contains(lineUpper, string(TypeHACK)) {
		return TypeHACK
	}
	if strings.Contains(lineUpper, string(TypeBUG)) {
		return TypeBUG
	}
	if strings.Contains(lineUpper, string(TypeOPTIMIZE)) {
		return TypeOPTIMIZE
	}
	if strings.Contains(lineUpper, string(TypeNOTE)) {
		return TypeNOTE
	}
	if strings.Contains(lineUpper, "WORKAROUND") || strings.Contains(lineUpper, "WORK AROUND") {
		return TypeWORKAROUND
	}
	// Improved detection for PLACEHOLDER
	if strings.Contains(lineUpper, "PLACEHOLDER") || strings.Contains(lineUpper, "PLACE HOLDER") || 
	   strings.Contains(lineUpper, "PLACE-HOLDER") || strings.Contains(lineUpper, "STUB") {
		return TypePLACEHOLDER
	}
	// Improved detection for SIMPLIFY
	if strings.Contains(lineUpper, "SIMPLIF") || strings.Contains(lineUpper, "SIMPLE") || 
	   strings.Contains(lineUpper, "SIMPLIFICATION") || strings.Contains(lineUpper, "REFACTOR") {
		return TypeSIMPLIFY
	}
	// Improved detection for HEURISTIC
	if strings.Contains(lineUpper, "HEURISTIC") || strings.Contains(lineUpper, "APPROXIMAT") || 
	   strings.Contains(lineUpper, "ESTIMATE") {
		return TypeHEURISTIC
	}

	return TypeOTHER
}

// extractTodoDescription extracts the description part of a TODO comment
func extractTodoDescription(line, todoType string) string {
	// Remove leading comment markers
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "//")
	line = strings.TrimPrefix(line, "/*")
	line = strings.TrimSuffix(line, "*/")
	line = strings.TrimSpace(line)

	// Find the TODO type in the line
	index := strings.Index(strings.ToUpper(line), todoType)
	if index == -1 {
		return line // Couldn't find the type, return the whole line
	}

	// Extract the part after the TODO type
	after := line[index+len(todoType):]

	// Remove any parenthetical content like TODO(user): or TODO(P1):
	parenRegex := regexp.MustCompile(`^\s*\([^)]*\)\s*:?`)
	after = parenRegex.ReplaceAllString(after, "")

	// Remove any colon and trim
	after = strings.TrimPrefix(after, ":")
	return strings.TrimSpace(after)
}

// extractTodoPriority extracts the priority from a TODO comment if specified
// Format: TODO(P1): ... or TODO(priority=1): ...
func extractTodoPriority(line string) int {
	// Default priority (3 = medium)
	defaultPriority := 3

	// Look for priority in parentheses
	priorityRegex := regexp.MustCompile(`(?i)\(\s*(?:P|priority\s*=\s*)(\d)\s*\)`)
	matches := priorityRegex.FindStringSubmatch(line)
	if len(matches) < 2 {
		return defaultPriority
	}

	// Parse the priority number
	priorityStr := matches[1]
	priority := defaultPriority

	// Convert to int
	if p, err := strconv.Atoi(priorityStr); err == nil {
		if p >= 1 && p <= 5 {
			priority = p
		}
	}

	return priority
}

// extractTodoAuthor extracts the author from a TODO comment if specified
// Format: TODO(user): ... or TODO(author=user): ...
func extractTodoAuthor(line string) string {
	// Look for author in parentheses
	authorRegex := regexp.MustCompile(`(?i)\(\s*(?:author\s*=\s*)?([a-zA-Z0-9._-]+)\s*\)`)
	matches := authorRegex.FindStringSubmatch(line)
	if len(matches) < 2 {
		return ""
	}

	// Check if what we found is actually a priority marker
	if matches[1] == "P1" || matches[1] == "P2" || matches[1] == "P3" || matches[1] == "P4" || matches[1] == "P5" {
		return ""
	}

	return matches[1]
}

// Helper functions are now in utils package
