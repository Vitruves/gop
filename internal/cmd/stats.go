package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
)

type FileStats struct {
	File         string
	Language     string
	Lines        int
	CodeLines    int
	CommentLines int
	BlankLines   int
	Functions    int
	Classes      int
	Imports      int
	Size         int64
	Complexity   int
}

type CodebaseStats struct {
	TotalFiles        int
	TotalLines        int
	TotalCodeLines    int
	TotalCommentLines int
	TotalBlankLines   int
	TotalFunctions    int
	TotalClasses      int
	TotalImports      int
	TotalSize         int64
	LanguageStats     map[string]LanguageStats
	FileStats         []FileStats
}

type LanguageStats struct {
	Files        int
	Lines        int
	CodeLines    int
	CommentLines int
	Functions    int
	Classes      int
}

var (
	statsOutputFile string
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Generate comprehensive codebase statistics",
	Long:  `Generate detailed statistics about your codebase including file counts, line counts, function counts, and complexity metrics.`,
	RunE:  runStats,
}

func init() {
	statsCmd.Flags().StringVarP(&statsOutputFile, "output", "o", "", "Output file (.txt)")
}

func runStats(cmd *cobra.Command, args []string) error {
	if verbose {
		logInfo("Starting codebase analysis")
	}

	files, err := collectAllFiles()
	if err != nil {
		logError(fmt.Sprintf("Failed to collect files: %v", err))
		return err
	}

	if len(files) == 0 {
		logWarning("No files found")
		return nil
	}

	if verbose {
		logInfo(fmt.Sprintf("Analyzing %d files", len(files)))
	}

	stats := &CodebaseStats{
		LanguageStats: make(map[string]LanguageStats),
		FileStats:     make([]FileStats, 0, len(files)),
	}

	bar := progressbar.NewOptions(len(files),
		progressbar.OptionSetDescription("Analyzing files"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionClearOnFinish(),
	)

	sem := semaphore.NewWeighted(int64(jobs))
	var mu sync.Mutex
	var wg sync.WaitGroup

	results := make([]FileStats, len(files))

	for i, file := range files {
		wg.Add(1)
		go func(idx int, filePath string) {
			defer wg.Done()
			sem.Acquire(context.Background(), 1)
			defer sem.Release(1)

			fileStats, err := analyzeFile(filePath)
			if err != nil {
				logError(fmt.Sprintf("Error analyzing %s: %v", filePath, err))
				return
			}

			mu.Lock()
			results[idx] = fileStats
			bar.Add(1)
			mu.Unlock()
		}(i, file)
	}

	wg.Wait()
	bar.Finish()

	for _, fileStats := range results {
		if fileStats.File != "" {
			stats.FileStats = append(stats.FileStats, fileStats)
			updateStats(stats, fileStats)
		}
	}

	stats.TotalFiles = len(stats.FileStats)

	err = displayStats(stats)
	if err != nil {
		logError(fmt.Sprintf("Failed to display stats: %v", err))
		return err
	}

	logSuccess("Codebase analysis completed")
	return nil
}

func collectAllFiles() ([]string, error) {
	var files []string

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
			if shouldExcludeDirStats(path, exclude) {
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

		if !shouldExcludeFileStats(path, exclude) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func analyzeFile(filePath string) (FileStats, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return FileStats{}, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return FileStats{}, err
	}

	stats := FileStats{
		File:     filePath,
		Language: detectLanguage(filePath),
		Size:     fileInfo.Size(),
	}

	scanner := bufio.NewScanner(file)

	functionRegexes := []*regexp.Regexp{
		regexp.MustCompile(`^\s*(def|async def)\s+\w+`),                                    // Python
		regexp.MustCompile(`^\s*(pub\s+)?fn\s+\w+`),                                        // Rust
		regexp.MustCompile(`^\s*func\s+\w+`),                                               // Go
		regexp.MustCompile(`^\s*(\w+\s+)*\w+\s+\w+\s*\(.*\)\s*[{;]`),                       // C/C++
		regexp.MustCompile(`^\s*(public|private|protected)?\s*(static\s+)?\w+\s+\w+\s*\(`), // Java/C#
	}

	classRegexes := []*regexp.Regexp{
		regexp.MustCompile(`^\s*class\s+\w+`),           // Python, C++, Java, C#
		regexp.MustCompile(`^\s*(pub\s+)?struct\s+\w+`), // Rust
		regexp.MustCompile(`^\s*type\s+\w+\s+struct`),   // Go
	}

	importRegexes := []*regexp.Regexp{
		regexp.MustCompile(`^\s*(import|from\s+\w+\s+import)`), // Python
		regexp.MustCompile(`^\s*use\s+`),                       // Rust
		regexp.MustCompile(`^\s*import\s+`),                    // Go, Java
		regexp.MustCompile(`^\s*#include\s+`),                  // C/C++
		regexp.MustCompile(`^\s*using\s+`),                     // C#
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		stats.Lines++

		if trimmed == "" {
			stats.BlankLines++
		} else if isCommentLine(trimmed, stats.Language) {
			stats.CommentLines++
		} else {
			stats.CodeLines++
		}

		for _, regex := range functionRegexes {
			if regex.MatchString(line) {
				stats.Functions++
				break
			}
		}

		for _, regex := range classRegexes {
			if regex.MatchString(line) {
				stats.Classes++
				break
			}
		}

		for _, regex := range importRegexes {
			if regex.MatchString(line) {
				stats.Imports++
				break
			}
		}
	}

	return stats, scanner.Err()
}

func detectLanguage(filePath string) string {
	ext := filepath.Ext(filePath)

	languageMap := map[string]string{
		".py":    "Python",
		".rs":    "Rust",
		".go":    "Go",
		".c":     "C",
		".h":     "C",
		".cpp":   "C++",
		".cxx":   "C++",
		".cc":    "C++",
		".hpp":   "C++",
		".hxx":   "C++",
		".hh":    "C++",
		".js":    "JavaScript",
		".ts":    "TypeScript",
		".java":  "Java",
		".kt":    "Kotlin",
		".swift": "Swift",
		".rb":    "Ruby",
		".php":   "PHP",
		".cs":    "C#",
		".sh":    "Shell",
		".ps1":   "PowerShell",
		".sql":   "SQL",
		".xml":   "XML",
		".html":  "HTML",
		".css":   "CSS",
		".json":  "JSON",
		".yaml":  "YAML",
		".yml":   "YAML",
		".toml":  "TOML",
		".md":    "Markdown",
		".txt":   "Text",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return "Unknown"
}

func isCommentLine(line, language string) bool {
	switch language {
	case "Python", "Ruby", "Shell":
		return strings.HasPrefix(line, "#")
	case "Rust", "Go", "C", "C++", "JavaScript", "TypeScript", "Java", "Kotlin", "Swift", "C#":
		return strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*")
	case "SQL":
		return strings.HasPrefix(line, "--")
	case "HTML", "XML":
		return strings.HasPrefix(line, "<!--")
	default:
		return strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//")
	}
}

func updateStats(stats *CodebaseStats, fileStats FileStats) {
	stats.TotalLines += fileStats.Lines
	stats.TotalCodeLines += fileStats.CodeLines
	stats.TotalCommentLines += fileStats.CommentLines
	stats.TotalBlankLines += fileStats.BlankLines
	stats.TotalFunctions += fileStats.Functions
	stats.TotalClasses += fileStats.Classes
	stats.TotalImports += fileStats.Imports
	stats.TotalSize += fileStats.Size

	langStats := stats.LanguageStats[fileStats.Language]
	langStats.Files++
	langStats.Lines += fileStats.Lines
	langStats.CodeLines += fileStats.CodeLines
	langStats.CommentLines += fileStats.CommentLines
	langStats.Functions += fileStats.Functions
	langStats.Classes += fileStats.Classes
	stats.LanguageStats[fileStats.Language] = langStats
}

func displayStats(stats *CodebaseStats) error {
	output := formatStats(stats)

	if statsOutputFile != "" {
		return os.WriteFile(statsOutputFile, []byte(output), 0644)
	} else {
		fmt.Print(output)
		return nil
	}
}

func formatStats(stats *CodebaseStats) string {
	var sb strings.Builder

	sb.WriteString("# Codebase Statistics\n\n")

	sb.WriteString("## Overall Summary\n")
	sb.WriteString(fmt.Sprintf("- **Total Files**: %d\n", stats.TotalFiles))
	sb.WriteString(fmt.Sprintf("- **Total Lines**: %d\n", stats.TotalLines))
	sb.WriteString(fmt.Sprintf("- **Code Lines**: %d (%.1f%%)\n", stats.TotalCodeLines, percentage(stats.TotalCodeLines, stats.TotalLines)))
	sb.WriteString(fmt.Sprintf("- **Comment Lines**: %d (%.1f%%)\n", stats.TotalCommentLines, percentage(stats.TotalCommentLines, stats.TotalLines)))
	sb.WriteString(fmt.Sprintf("- **Blank Lines**: %d (%.1f%%)\n", stats.TotalBlankLines, percentage(stats.TotalBlankLines, stats.TotalLines)))
	sb.WriteString(fmt.Sprintf("- **Total Functions**: %d\n", stats.TotalFunctions))
	sb.WriteString(fmt.Sprintf("- **Total Classes**: %d\n", stats.TotalClasses))
	sb.WriteString(fmt.Sprintf("- **Total Imports**: %d\n", stats.TotalImports))
	sb.WriteString(fmt.Sprintf("- **Total Size**: %.2f MB\n", float64(stats.TotalSize)/(1024*1024)))
	sb.WriteString("\n")

	sb.WriteString("## Language Breakdown\n")

	type langStat struct {
		lang  string
		stats LanguageStats
	}

	var langStats []langStat
	for lang, stat := range stats.LanguageStats {
		langStats = append(langStats, langStat{lang, stat})
	}

	sort.Slice(langStats, func(i, j int) bool {
		return langStats[i].stats.Lines > langStats[j].stats.Lines
	})

	for _, ls := range langStats {
		sb.WriteString(fmt.Sprintf("### %s\n", ls.lang))
		sb.WriteString(fmt.Sprintf("- Files: %d\n", ls.stats.Files))
		sb.WriteString(fmt.Sprintf("- Lines: %d (%.1f%%)\n", ls.stats.Lines, percentage(ls.stats.Lines, stats.TotalLines)))
		sb.WriteString(fmt.Sprintf("- Code Lines: %d\n", ls.stats.CodeLines))
		sb.WriteString(fmt.Sprintf("- Comment Lines: %d\n", ls.stats.CommentLines))
		sb.WriteString(fmt.Sprintf("- Functions: %d\n", ls.stats.Functions))
		sb.WriteString(fmt.Sprintf("- Classes: %d\n", ls.stats.Classes))
		sb.WriteString("\n")
	}

	sb.WriteString("## Top Files by Size\n")

	sort.Slice(stats.FileStats, func(i, j int) bool {
		return stats.FileStats[i].Lines > stats.FileStats[j].Lines
	})

	maxFiles := 10
	if len(stats.FileStats) < maxFiles {
		maxFiles = len(stats.FileStats)
	}

	for i := 0; i < maxFiles; i++ {
		fs := stats.FileStats[i]
		sb.WriteString(fmt.Sprintf("1. **%s** (%s) - %d lines, %d functions\n",
			fs.File, fs.Language, fs.Lines, fs.Functions))
	}

	return sb.String()
}

func percentage(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}

func shouldExcludeDirStats(path string, exclude []string) bool {
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

func shouldExcludeFileStats(path string, exclude []string) bool {
	for _, excludePattern := range exclude {
		if matched, _ := filepath.Match(excludePattern, path); matched {
			return true
		}
	}
	return false
}
