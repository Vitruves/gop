package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetFilesToProcess returns a list of files to process based on input options
func GetFilesToProcess(inputFile, directory string, depth int, languages []string, excludes []string) ([]string, error) {
	var files []string
	
	// If input file is specified, use only that file
	if inputFile != "" {
		// Check if file exists
		if _, err := os.Stat(inputFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("input file does not exist: %s", inputFile)
		}
		
		// Check if file has a valid extension
		if !hasValidExtension(inputFile, languages) {
			return nil, fmt.Errorf("input file has invalid extension: %s", inputFile)
		}
		
		return []string{inputFile}, nil
	}
	
	// Otherwise, find all files in the directory with valid extensions
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Check if path should be excluded
		for _, exclude := range excludes {
			// Handle both exact matches and pattern matches
			if matched, _ := filepath.Match(exclude, filepath.Base(path)); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			
			// Check if path contains the exclude pattern
			if strings.Contains(path, exclude) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		
		// Skip directories
		if info.IsDir() {
			// Skip if we've reached the maximum depth
			if depth >= 0 {
				// Calculate current depth
				relPath, err := filepath.Rel(directory, path)
				if err != nil {
					return err
				}
				
				currentDepth := len(strings.Split(relPath, string(os.PathSeparator)))
				if currentDepth > depth {
					return filepath.SkipDir
				}
			}
			return nil
		}
		
		// Check if file has a valid extension
		if hasValidExtension(path, languages) {
			files = append(files, path)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %s", err)
	}
	
	return files, nil
}

// hasValidExtension checks if a file has a valid extension based on the languages
func hasValidExtension(path string, languages []string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	
	// If no languages specified, accept all files
	if len(languages) == 0 {
		return true
	}
	
	// Check for "all" language first as a shortcut
	for _, lang := range languages {
		if strings.ToLower(lang) == "all" {
			return true // Accept all files when "all" is specified
		}
	}
	
	// Map of file extensions by language category
	extensionMap := map[string][]string{
		"c":       {".c", ".h"},
		"cpp":     {".cpp", ".cxx", ".cc", ".hpp", ".hxx", ".h"},
		"go":      {".go"},
		"js":      {".js", ".jsx", ".ts", ".tsx", ".json"},
		"python":  {".py", ".pyw", ".pyx", ".pyi"},
		"java":    {".java", ".class", ".jar"},
		"ruby":    {".rb", ".erb"},
		"php":     {".php", ".phtml", ".php3", ".php4", ".php5"},
		"csharp":  {".cs"},
		"swift":   {".swift"},
		"rust":    {".rs"},
		"html":    {".html", ".htm", ".xhtml"},
		"css":     {".css", ".scss", ".sass", ".less"},
		"xml":     {".xml", ".svg", ".xsl"},
		"shell":   {".sh", ".bash", ".zsh", ".fish"},
		"sql":     {".sql"},
		"markdown": {".md", ".markdown"},
		"text":    {".txt", ".text"},
		"doc":     {".doc", ".docx", ".odt", ".rtf", ".tex", ".pdf"},
		"config":  {".yaml", ".yml", ".toml", ".ini", ".conf", ".config", ".properties", ".env"},
	}
	
	// Aliases for languages
	aliases := map[string]string{
		"golang":     "go",
		"javascript": "js",
		"typescript": "js",
		"py":         "python",
		"rb":         "ruby",
		"c++":        "cpp",
		"cs":         "csharp",
		"rs":         "rust",
		"bash":       "shell",
		"sh":         "shell",
		"yml":        "config",
		"yaml":       "config",
		"json":       "js",
		"md":         "markdown",
		"docs":       "doc",
		"documents":  "doc",
		"txt":        "text",
	}
	
	// Check if the file extension matches any of the requested languages
	for _, lang := range languages {
		lang = strings.ToLower(lang)
		
		// Check for aliases
		if alias, ok := aliases[lang]; ok {
			lang = alias
		}
		
		// Check if the extension is in the language's list
		if extensions, ok := extensionMap[lang]; ok {
			for _, validExt := range extensions {
				if ext == validExt {
					return true
				}
			}
		}
	}
	
	return false
}

// RemoveDuplicates removes duplicate strings from a slice
func RemoveDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	
	return list
}
