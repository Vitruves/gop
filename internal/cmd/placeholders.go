package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
)

type Placeholder struct {
	File    string
	Line    int
	Column  int
	Content string
	Type    string
}

var placeholdersCmd = &cobra.Command{
	Use:   "placeholders",
	Short: "Search and highlight placeholders in code",
	Long:  `Find and highlight various types of placeholders in your codebase including TODO comments, hardcoded values, and temporary code.`,
	RunE:  runPlaceholders,
}

func runPlaceholders(cmd *cobra.Command, args []string) error {
	if verbose {
		logInfo("Starting placeholder search")
	}

	files, err := collectSourceFiles()
	if err != nil {
		logError(fmt.Sprintf("Failed to collect files: %v", err))
		return err
	}

	if len(files) == 0 {
		logWarning("No files found")
		return nil
	}

	if verbose {
		logInfo(fmt.Sprintf("Scanning %d files for placeholders", len(files)))
	}

	var allPlaceholders []Placeholder
	var mu sync.Mutex

	bar := progressbar.NewOptions(len(files),
		progressbar.OptionSetDescription("Scanning for placeholders"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionClearOnFinish(),
	)

	sem := semaphore.NewWeighted(int64(jobs))
	var wg sync.WaitGroup

	for _, file := range files {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			sem.Acquire(context.Background(), 1)
			defer sem.Release(1)

			placeholders, err := scanFileForPlaceholders(filePath)
			if err != nil {
				logError(fmt.Sprintf("Error scanning %s: %v", filePath, err))
				return
			}

			mu.Lock()
			allPlaceholders = append(allPlaceholders, placeholders...)
			bar.Add(1)
			mu.Unlock()
		}(file)
	}

	wg.Wait()
	bar.Finish()

	if len(allPlaceholders) == 0 {
		logSuccess("No placeholders found")
		return nil
	}

	displayPlaceholders(allPlaceholders)
	logSuccess(fmt.Sprintf("Found %d placeholders", len(allPlaceholders)))

	return nil
}

func collectSourceFiles() ([]string, error) {
	var files []string
	extensions := []string{".py", ".rs", ".go", ".c", ".cpp", ".cxx", ".cc", ".h", ".hpp", ".hxx", ".hh", ".js", ".ts", ".java", ".kt", ".swift", ".rb", ".php"}

	startDir := "."
	if len(include) > 0 {
		for _, path := range include {
			matches, err := filepath.Glob(path)
			if err != nil {
				return nil, err
			}
			files = append(files, matches...)
		}
		return files, nil
	}

	err := filepath.WalkDir(startDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if shouldExcludeDirPlaceholders(path, exclude) {
				return filepath.SkipDir
			}
			if !recursive && path != startDir {
				return filepath.SkipDir
			}
			if depth > 0 {
				relPath, _ := filepath.Rel(startDir, path)
				if strings.Count(relPath, string(filepath.Separator)) >= depth {
					return filepath.SkipDir
				}
			}
			return nil
		}

		if isValidSourceFile(path, extensions) && !shouldExcludeFile(path, exclude) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func isValidSourceFile(path string, extensions []string) bool {
	ext := filepath.Ext(path)
	for _, validExt := range extensions {
		if ext == validExt {
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

func shouldExcludeDirPlaceholders(path string, exclude []string) bool {
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

func scanFileForPlaceholders(filePath string) ([]Placeholder, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var placeholders []Placeholder
	scanner := bufio.NewScanner(file)
	lineNum := 1

	patterns := []struct {
		regex *regexp.Regexp
		ptype string
	}{
		{regexp.MustCompile(`(?i)#?\s*(TODO|FIXME|HACK|XXX|BUG|NOTE)(\([^)]*\))?\s*:?\s*(.+)`), "comment"},
		{regexp.MustCompile(`(?i)placeholder|temp|temporary|dummy|mock|stub|simple|simplification|basic|minimal|naive|hardcode|hardcoded`), "temporary"},
		{regexp.MustCompile(`\b(localhost|127\.0\.0\.1|0\.0\.0\.0)\b`), "hardcoded_host"},
		{regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`), "ip_address"},
		{regexp.MustCompile(`\b(password|passwd|secret|key|token)\s*[=:]\s*["']([^"']+)["']`), "hardcoded_secret"},
		{regexp.MustCompile(`\btest\w*\s*=\s*true\b`), "test_flag"},
		{regexp.MustCompile(`\bdebug\s*=\s*true\b`), "debug_flag"},
		{regexp.MustCompile(`\b(print|console\.log|fmt\.Print|println!|cout\s*<<)\s*\(`), "debug_print"},
		{regexp.MustCompile(`\b(exit|quit|abort)\s*\(`), "exit_call"},
		{regexp.MustCompile(`\bthrow\s+new\s+Exception\(|panic!\(|unreachable!\(`), "exception"},
		{regexp.MustCompile(`(?i)\b(implement|implementation|implement this|not implemented|unimplemented|not done|incomplete)\b`), "unimplemented"},
		{regexp.MustCompile(`(?i)\b(example|sample|demo|test data|fake data)\b`), "example_data"},
		{regexp.MustCompile(`(?i)\b(quick|dirty|quick and dirty|workaround|kludge|band-aid|bandaid)\b`), "quick_fix"},
	}

	for scanner.Scan() {
		line := scanner.Text()
		
		for _, pattern := range patterns {
			matches := pattern.regex.FindAllStringIndex(line, -1)
			for _, match := range matches {
				placeholder := Placeholder{
					File:    filePath,
					Line:    lineNum,
					Column:  match[0] + 1,
					Content: strings.TrimSpace(line),
					Type:    pattern.ptype,
				}
				placeholders = append(placeholders, placeholder)
			}
		}
		
		lineNum++
	}

	return placeholders, scanner.Err()
}

func displayPlaceholders(placeholders []Placeholder) {
	typeGroups := make(map[string][]Placeholder)
	
	for _, p := range placeholders {
		typeGroups[p.Type] = append(typeGroups[p.Type], p)
	}

	for ptype, items := range typeGroups {
		fmt.Printf("\n\033[1;36m=== %s ===\033[0m\n", strings.ToUpper(ptype))
		
		for _, item := range items {
			fmt.Printf("\033[33m%s:%d:%d\033[0m - %s\n", 
				item.File, item.Line, item.Column, item.Content)
		}
	}
}