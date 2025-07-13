package concatenate

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/semaphore"
)

type Config struct {
	Language       string
	Include        []string
	Exclude        []string
	Recursive      bool
	Depth          int
	Jobs           int
	Verbose        bool
	RemoveTests    bool
	RemoveComments bool
	AddLineNumbers bool
	AddHeaders     bool
	OutputFile     string
}

type FileProcessor interface {
	GetExtensions() []string
	IsTestFile(path string) bool
	RemoveComments(content string) string
	RemoveTestCode(content string) string
	SupportsSpecialFiles() map[string]bool
	IsHeaderFile(path string) bool
}

func Run(config Config) error {
	logInfo(config.Verbose, "Starting code concatenation")

	processor := getProcessor(config.Language)
	if processor == nil {
		return fmt.Errorf("unsupported language: %s", config.Language)
	}

	files, err := collectFiles(config, processor)
	if err != nil {
		logError(fmt.Sprintf("Failed to collect files: %v", err))
		return err
	}

	if len(files) == 0 {
		logWarning("No files found matching criteria")
		return nil
	}

	logInfo(config.Verbose, fmt.Sprintf("Found %d files to process", len(files)))

	var output strings.Builder
	
	bar := progressbar.NewOptions(len(files),
		progressbar.OptionSetDescription("Processing files"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionClearOnFinish(),
	)

	sem := semaphore.NewWeighted(int64(config.Jobs))
	var mu sync.Mutex
	var wg sync.WaitGroup

	results := make([]string, len(files))
	
	for i, file := range files {
		wg.Add(1)
		go func(idx int, filePath string) {
			defer wg.Done()
			sem.Acquire(context.Background(), 1)
			defer sem.Release(1)

			content, err := processFile(filePath, config, processor)
			if err != nil {
				logError(fmt.Sprintf("Error processing %s: %v", filePath, err))
				return
			}

			mu.Lock()
			results[idx] = content
			bar.Add(1)
			mu.Unlock()
		}(i, file)
	}

	wg.Wait()
	bar.Finish()

	for _, content := range results {
		if content != "" {
			output.WriteString(content)
		}
	}

	finalOutput := output.String()
	
	if config.OutputFile != "" {
		err := os.WriteFile(config.OutputFile, []byte(finalOutput), 0644)
		if err != nil {
			logError(fmt.Sprintf("Failed to write output file: %v", err))
			return err
		}
		logSuccess(fmt.Sprintf("Output written to %s", config.OutputFile))
	} else {
		fmt.Print(finalOutput)
	}

	logSuccess("Code concatenation completed")
	return nil
}

func getProcessor(language string) FileProcessor {
	switch language {
	case "python":
		return &PythonProcessor{}
	case "rust":
		return &RustProcessor{}
	case "go":
		return &GoProcessor{}
	case "c":
		return &CProcessor{}
	case "cpp":
		return &CppProcessor{}
	default:
		return &GenericProcessor{}
	}
}

func collectFiles(config Config, processor FileProcessor) ([]string, error) {
	var files []string
	extensions := processor.GetExtensions()
	specialFiles := processor.SupportsSpecialFiles()

	startDir := "."
	if len(config.Include) > 0 {
		for _, path := range config.Include {
			matches, err := filepath.Glob(path)
			if err != nil {
				return nil, err
			}
			for _, match := range matches {
				if isValidFile(match, extensions) || isSpecialFile(match, specialFiles) {
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

		if (isValidFile(path, extensions) || isSpecialFile(path, specialFiles)) && !shouldExcludeFile(path, config, processor) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func isValidFile(path string, extensions []string) bool {
	ext := filepath.Ext(path)
	for _, validExt := range extensions {
		if ext == validExt {
			return true
		}
	}
	return false
}

func isSpecialFile(path string, specialFiles map[string]bool) bool {
	filename := filepath.Base(path)
	return specialFiles[filename]
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

func shouldExcludeFile(path string, config Config, processor FileProcessor) bool {
	if config.RemoveTests && processor.IsTestFile(path) {
		return true
	}
	
	for _, excludePattern := range config.Exclude {
		if matched, _ := filepath.Match(excludePattern, path); matched {
			return true
		}
	}
	
	return false
}

func processFile(filePath string, config Config, processor FileProcessor) (string, error) {
	logDebug(config.Verbose, fmt.Sprintf("Processing file: %s", filePath))
	
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	contentStr := string(content)
	
	if config.RemoveComments {
		contentStr = processor.RemoveComments(contentStr)
	}
	
	if config.RemoveTests {
		contentStr = processor.RemoveTestCode(contentStr)
	}

	var result strings.Builder
	
	if config.AddHeaders {
		result.WriteString(fmt.Sprintf("// === %s ===\n", filePath))
		result.WriteString(fmt.Sprintf("// Path: %s\n\n", filePath))
	}

	if config.AddLineNumbers {
		scanner := bufio.NewScanner(strings.NewReader(contentStr))
		lineNum := 1
		for scanner.Scan() {
			result.WriteString(fmt.Sprintf("%4d: %s\n", lineNum, scanner.Text()))
			lineNum++
		}
	} else {
		result.WriteString(contentStr)
	}
	
	if config.AddHeaders {
		result.WriteString("\n\n")
	}

	return result.String(), nil
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

func logDebug(verbose bool, msg string) {
	if os.Getenv("DEBUG") != "" || verbose {
		fmt.Printf("\033[33m%s - DEBUG: %s\033[0m\n", getCurrentTime(), msg)
	}
}

func getCurrentTime() string {
	now := time.Now()
	return fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())
}