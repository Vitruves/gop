package registry

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/semaphore"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Language        string
	Include         []string
	Exclude         []string
	Recursive       bool
	Depth           int
	Jobs            int
	Verbose         bool
	OutputFile      string
	ByScript        bool
	OnlyHeaderFiles bool
	AddRelations    bool
	OnlyDeadCode    bool
}

type Function struct {
	Name       string            `json:"name" yaml:"name"`
	File       string            `json:"file" yaml:"file"`
	Line       int               `json:"line" yaml:"line"`
	Visibility string            `json:"visibility" yaml:"visibility"`
	ReturnType string            `json:"return_type" yaml:"return_type"`
	Parameters []string          `json:"parameters" yaml:"parameters"`
	Language   string            `json:"language" yaml:"language"`
	CallCount  int               `json:"call_count" yaml:"call_count"`
	CalledBy   []string          `json:"called_by,omitempty" yaml:"called_by,omitempty"`
	Calls      []string          `json:"calls,omitempty" yaml:"calls,omitempty"`
	Comments   string            `json:"comments,omitempty" yaml:"comments,omitempty"`
	Signature  string            `json:"signature" yaml:"signature"`
	IsTest     bool              `json:"is_test" yaml:"is_test"`
	IsMain     bool              `json:"is_main" yaml:"is_main"`
	Complexity int               `json:"complexity,omitempty" yaml:"complexity,omitempty"`
	Size       int               `json:"size" yaml:"size"`
	Metadata   map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type Registry struct {
	Functions []Function            `json:"functions" yaml:"functions"`
	Scripts   map[string][]Function `json:"scripts,omitempty" yaml:"scripts,omitempty"`
	Summary   Summary               `json:"summary" yaml:"summary"`
}

type Summary struct {
	TotalFunctions   int `json:"total_functions" yaml:"total_functions"`
	TotalFiles       int `json:"total_files" yaml:"total_files"`
	PublicFunctions  int `json:"public_functions" yaml:"public_functions"`
	PrivateFunctions int `json:"private_functions" yaml:"private_functions"`
	DeadFunctions    int `json:"dead_functions" yaml:"dead_functions"`
	TestFunctions    int `json:"test_functions" yaml:"test_functions"`
}

type LanguageParser interface {
	GetExtensions() []string
	ParseFile(filePath string) ([]Function, error)
	IsHeaderFile(filePath string) bool
	FindFunctionCalls(content string) []string
}

func Run(config Config) error {
	logInfo(config.Verbose, "Starting function registry generation")

	parser := getParser(config.Language)
	if parser == nil {
		return fmt.Errorf("unsupported language: %s", config.Language)
	}

	files, err := collectFiles(config, parser)
	if err != nil {
		logError(fmt.Sprintf("Failed to collect files: %v", err))
		return err
	}

	if len(files) == 0 {
		logWarning("No files found matching criteria")
		return nil
	}

	logInfo(config.Verbose, fmt.Sprintf("Found %d files to analyze", len(files)))

	registry := &Registry{
		Functions: []Function{},
		Scripts:   make(map[string][]Function),
	}

	bar := progressbar.NewOptions(len(files),
		progressbar.OptionSetDescription("Analyzing functions"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionClearOnFinish(),
	)

	sem := semaphore.NewWeighted(int64(config.Jobs))
	var mu sync.Mutex
	var wg sync.WaitGroup

	allFunctions := make([][]Function, len(files))

	for i, file := range files {
		wg.Add(1)
		go func(idx int, filePath string) {
			defer wg.Done()
			sem.Acquire(context.Background(), 1)
			defer sem.Release(1)

			functions, err := parser.ParseFile(filePath)
			if err != nil {
				logError(fmt.Sprintf("Error parsing %s: %v", filePath, err))
				return
			}

			mu.Lock()
			allFunctions[idx] = functions
			bar.Add(1)
			mu.Unlock()
		}(i, file)
	}

	wg.Wait()
	bar.Finish()

	functionMap := make(map[string]*Function)

	for i, functions := range allFunctions {
		if functions == nil {
			continue
		}

		fileName := files[i]

		for _, fn := range functions {
			if config.OnlyDeadCode && fn.CallCount > 0 {
				continue
			}

			registry.Functions = append(registry.Functions, fn)
			functionMap[fn.Name] = &fn

			if config.ByScript {
				registry.Scripts[fileName] = append(registry.Scripts[fileName], fn)
			}
		}
	}

	if config.AddRelations {
		addCallRelations(registry, files, parser, config)
	}

	registry.Summary = generateSummary(registry.Functions, len(files))

	err = writeOutput(registry, config)
	if err != nil {
		logError(fmt.Sprintf("Failed to write output: %v", err))
		return err
	}

	logSuccess("Function registry generated successfully")
	return nil
}

func getParser(language string) LanguageParser {
	switch language {
	case "python":
		return &PythonParser{}
	case "rust":
		return &RustParser{}
	case "go":
		return &GoParser{}
	case "c":
		return &CParser{}
	case "cpp":
		return &CppParser{}
	default:
		return &GenericParser{}
	}
}

func collectFiles(config Config, parser LanguageParser) ([]string, error) {
	var files []string
	extensions := parser.GetExtensions()

	startDir := "."
	if len(config.Include) > 0 {
		for _, path := range config.Include {
			matches, err := filepath.Glob(path)
			if err != nil {
				return nil, err
			}
			for _, match := range matches {
				if isValidFile(match, extensions, config, parser) {
					files = append(files, match)
				}
			}
		}
		return files, nil
	}

	err := filepath.WalkDir(startDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if shouldExcludeDir(path, config.Exclude) {
				return filepath.SkipDir
			}
			if !config.Recursive && path != startDir {
				return filepath.SkipDir
			}
			if config.Depth > 0 {
				relPath, _ := filepath.Rel(startDir, path)
				if strings.Count(relPath, string(filepath.Separator)) >= config.Depth {
					return filepath.SkipDir
				}
			}
			return nil
		}

		if isValidFile(path, extensions, config, parser) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func isValidFile(path string, extensions []string, config Config, parser LanguageParser) bool {
	ext := filepath.Ext(path)

	for _, validExt := range extensions {
		if ext == validExt {
			if config.OnlyHeaderFiles && !parser.IsHeaderFile(path) {
				return false
			}
			return !shouldExcludeFile(path, config.Exclude)
		}
	}

	return false
}

func shouldExcludeDir(path string, exclude []string) bool {
	excludeDirs := []string{".git", "node_modules", "__pycache__", ".pytest_cache", "target", "build", "dist", "vendor"}

	for _, excludePattern := range exclude {
		if matched, _ := filepath.Match(excludePattern, path); matched {
			return true
		}
	}

	for _, excludeDir := range excludeDirs {
		if strings.Contains(path, excludeDir) {
			return true
		}
	}

	return false
}

func shouldExcludeFile(path string, exclude []string) bool {
	for _, excludePattern := range exclude {
		if matched, _ := filepath.Match(excludePattern, path); matched {
			return true
		}
	}

	return false
}

func addCallRelations(registry *Registry, files []string, parser LanguageParser, config Config) {
	logInfo(config.Verbose, "Analyzing function call relationships")

	functionMap := make(map[string]*Function)
	for i := range registry.Functions {
		functionMap[registry.Functions[i].Name] = &registry.Functions[i]
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		calls := parser.FindFunctionCalls(string(content))

		for _, call := range calls {
			if fn, exists := functionMap[call]; exists {
				fn.CallCount++
			}
		}
	}
}

func generateSummary(functions []Function, totalFiles int) Summary {
	summary := Summary{
		TotalFunctions: len(functions),
		TotalFiles:     totalFiles,
	}

	for _, fn := range functions {
		if fn.Visibility == "public" {
			summary.PublicFunctions++
		} else {
			summary.PrivateFunctions++
		}

		if fn.CallCount == 0 {
			summary.DeadFunctions++
		}

		if fn.IsTest {
			summary.TestFunctions++
		}
	}

	return summary
}

func writeOutput(registry *Registry, config Config) error {
	var output []byte
	var err error

	ext := filepath.Ext(config.OutputFile)

	switch ext {
	case ".yaml", ".yml":
		output, err = yaml.Marshal(registry)
	case ".json":
		output, err = json.MarshalIndent(registry, "", "  ")
	case ".csv":
		output, err = formatCSV(registry)
	default:
		output = []byte(formatText(registry, config))
	}

	if err != nil {
		return err
	}

	if config.OutputFile != "" {
		return os.WriteFile(config.OutputFile, output, 0644)
	} else {
		fmt.Print(string(output))
		return nil
	}
}

func formatText(registry *Registry, config Config) string {
	var sb strings.Builder

	sb.WriteString("# Function Registry\n\n")

	sb.WriteString("## Summary\n")
	sb.WriteString(fmt.Sprintf("- Total Functions: %d\n", registry.Summary.TotalFunctions))
	sb.WriteString(fmt.Sprintf("- Total Files: %d\n", registry.Summary.TotalFiles))
	sb.WriteString(fmt.Sprintf("- Public Functions: %d\n", registry.Summary.PublicFunctions))
	sb.WriteString(fmt.Sprintf("- Private Functions: %d\n", registry.Summary.PrivateFunctions))
	sb.WriteString(fmt.Sprintf("- Dead Functions: %d\n", registry.Summary.DeadFunctions))
	sb.WriteString(fmt.Sprintf("- Test Functions: %d\n", registry.Summary.TestFunctions))
	sb.WriteString("\n")

	if config.ByScript {
		for file, functions := range registry.Scripts {
			sb.WriteString(fmt.Sprintf("## %s\n\n", file))

			sort.Slice(functions, func(i, j int) bool {
				return functions[i].Line < functions[j].Line
			})

			for _, fn := range functions {
				sb.WriteString(formatFunction(fn))
			}
			sb.WriteString("\n")
		}
	} else {
		sort.Slice(registry.Functions, func(i, j int) bool {
			if registry.Functions[i].File == registry.Functions[j].File {
				return registry.Functions[i].Line < registry.Functions[j].Line
			}
			return registry.Functions[i].File < registry.Functions[j].File
		})

		sb.WriteString("## Functions\n\n")
		for _, fn := range registry.Functions {
			sb.WriteString(formatFunction(fn))
		}
	}

	return sb.String()
}

func formatFunction(fn Function) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("### %s\n", fn.Name))
	sb.WriteString(fmt.Sprintf("- **File**: %s:%d\n", fn.File, fn.Line))
	sb.WriteString(fmt.Sprintf("- **Visibility**: %s\n", fn.Visibility))
	sb.WriteString(fmt.Sprintf("- **Return Type**: %s\n", fn.ReturnType))
	sb.WriteString(fmt.Sprintf("- **Parameters**: %s\n", strings.Join(fn.Parameters, ", ")))
	sb.WriteString(fmt.Sprintf("- **Language**: %s\n", fn.Language))
	sb.WriteString(fmt.Sprintf("- **Call Count**: %d\n", fn.CallCount))
	sb.WriteString(fmt.Sprintf("- **Size**: %d lines\n", fn.Size))

	if fn.IsTest {
		sb.WriteString("- **Type**: Test Function\n")
	}

	if fn.IsMain {
		sb.WriteString("- **Type**: Main Function\n")
	}

	if fn.Complexity > 0 {
		sb.WriteString(fmt.Sprintf("- **Complexity**: %d\n", fn.Complexity))
	}

	if len(fn.CalledBy) > 0 {
		sb.WriteString(fmt.Sprintf("- **Called By**: %s\n", strings.Join(fn.CalledBy, ", ")))
	}

	if len(fn.Calls) > 0 {
		sb.WriteString(fmt.Sprintf("- **Calls**: %s\n", strings.Join(fn.Calls, ", ")))
	}

	if fn.Comments != "" {
		sb.WriteString(fmt.Sprintf("- **Comments**: %s\n", fn.Comments))
	}

	sb.WriteString(fmt.Sprintf("- **Signature**: `%s`\n", fn.Signature))
	sb.WriteString("\n")

	return sb.String()
}

func formatCSV(registry *Registry) ([]byte, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	
	// Write header
	header := []string{
		"Name", "File", "Line", "Visibility", "ReturnType", "Parameters",
		"Language", "CallCount", "Size", "IsTest", "IsMain", "Comments", "Signature",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	
	// Sort functions for consistent output
	sort.Slice(registry.Functions, func(i, j int) bool {
		if registry.Functions[i].File == registry.Functions[j].File {
			return registry.Functions[i].Line < registry.Functions[j].Line
		}
		return registry.Functions[i].File < registry.Functions[j].File
	})
	
	// Write function data
	for _, fn := range registry.Functions {
		record := []string{
			fn.Name,
			fn.File,
			strconv.Itoa(fn.Line),
			fn.Visibility,
			fn.ReturnType,
			strings.Join(fn.Parameters, ";"), // Use semicolon to separate parameters
			fn.Language,
			strconv.Itoa(fn.CallCount),
			strconv.Itoa(fn.Size),
			strconv.FormatBool(fn.IsTest),
			strconv.FormatBool(fn.IsMain),
			strings.ReplaceAll(fn.Comments, "\n", " "), // Replace newlines with spaces
			strings.ReplaceAll(fn.Signature, "\n", " "), // Replace newlines with spaces
		}
		
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	
	return []byte(buf.String()), nil
}

func logInfo(verbose bool, msg string) {
	if verbose {
		fmt.Printf("\033[34m%s - INFO: %s\033[0m\n", getCurrentTime(), msg)
	}
}

func logSuccess(msg string) {
	fmt.Printf("\033[32m%s - SUCCESS: %s\033[0m\n", getCurrentTime(), msg)
}

func logWarning(msg string) {
	fmt.Printf("\033[33m%s - WARNING: %s\033[0m\n", getCurrentTime(), msg)
}

func logError(msg string) {
	fmt.Printf("\033[31m%s - ERROR: %s\033[0m\n", getCurrentTime(), msg)
}

func getCurrentTime() string {
	now := time.Now()
	return fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
}
