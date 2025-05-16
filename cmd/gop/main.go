package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/vitruves/gop/internal/analyzer"
)

var (
	// Common flags
	inputFile           string
	directory           string
	depth               int
	outputFile          string
	languages           []string
	excludes            []string
	jobs                int
	verbose             bool
	shortOutput         bool

	// Todo command flags
	maxContext          int
	groupByType         bool
	filter              []string

	// Coherence command flags
	todoFilter          string
	checkHeaders        bool
	checkFiles          bool
	showDiscrepancies   bool
	nonImplemented      bool
	notDeclared         bool
	similarityThreshold float64
	minLineCount        int

	// Monitoring options
	monitorEnabled      bool
	monitorFile         string
	monitorComment      string

	// Additional options for existing commands
	format              string
	complexityThreshold int
	apiFile             string
	includeHeaders      bool
	addLineNumbers      bool
	removeComments      bool
	types               []string
	shortRegistry       bool
	namesOnly           bool
	showRelations       bool
	showStats           bool
	iaOutput            bool

	// Docs command flags
	includeCode         bool

	// Profile command flags
	executable          string
	args                []string
	profileType         string
	duration            int

	// Refactor command flags
	pattern             string
	replacement         string
	regexMode           bool
	wholeWord           bool
	caseSensitive       bool
	dryRun              bool
	backup              bool
)

// Custom help templates with colors
const (
	customHelpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

	customUsageTemplate = `{{bold "Usage:"}}{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

{{bold "Aliases:"}}{{range .Aliases}}
  {{.}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{bold "Flags:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{bold "Global Flags:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

{{bold "Available Commands:"}}{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding | cyan}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{bold "Flags:"}}{{range .LocalFlags.SortedFlags}}
  {{yellow (rpad .Shorthand 4)}}{{if .Shorthand}}, {{end}}{{yellow (rpad .Name 20)}} {{.Usage}}{{if .DefValue}} (default {{italic .DefValue}}){{end}}{{end}}{{end}}{{if .HasAvailableInheritedFlags}}

{{bold "Global Flags:"}}{{range .InheritedFlags.SortedFlags}}
  {{yellow (rpad .Shorthand 4)}}{{if .Shorthand}}, {{end}}{{yellow (rpad .Name 20)}} {{.Usage}}{{if .DefValue}} (default {{italic .DefValue}}){{end}}{{end}}{{end}}{{if .HasHelpSubCommands}}

{{bold "Additional help topics:"}}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

{{bold "Use "}}{{.CommandPath}} [command] --help{{bold "" for more information about a command."}}{{end}}
`
)

// Custom template functions for colors and formatting
var templateFuncs = template.FuncMap{
	"cyan":   color.CyanString,
	"yellow": color.YellowString,
	"green":  color.GreenString,
	"red":    color.RedString,
	"bold":   color.New(color.Bold).SprintFunc(),
	"italic": color.New(color.Italic).SprintFunc(),
	"rpad":   rpad,
}

// rpad adds padding to the right of a string
func rpad(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(template, s)
}

func main() {
	// Root command
	rootCmd := &cobra.Command{
		Use:   "gop",
		Short: "Go Project - A tool for C/C++ code analysis and management",
		Long: color.GreenString(`Go Project (gop)`) + ` is a comprehensive tool for analyzing and managing C/C++ codebases.
It provides functionality for concatenating files, creating registries of code elements,
finding TODOs, and checking coherence between headers and implementations.`,
	}

	// Custom help display with colors and justification
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// Print the header with color
		titleStyle := color.New(color.Bold, color.FgGreen).SprintFunc()
		headerStyle := color.New(color.Bold, color.FgWhite).SprintFunc()
		cmdStyle := color.New(color.FgCyan).SprintFunc()
		subCmdStyle := color.New(color.FgYellow).SprintFunc()
		flagStyle := color.New(color.FgMagenta).SprintFunc()
		
		fmt.Println(titleStyle("Go Project (gop) v1.0"))
		fmt.Println(headerStyle("A comprehensive tool for C/C++ code analysis and management"))
		fmt.Println()
		
		// Print usage
		fmt.Println(headerStyle("Usage:"))
		fmt.Printf("  %s [command]\n\n", cmdStyle(cmd.CommandPath()))
		
		// Print available commands
		fmt.Println(headerStyle("Available Commands:"))
		
		// Find the longest command name for proper justification
		maxLen := 0
		for _, subcmd := range cmd.Commands() {
			if subcmd.IsAvailableCommand() || subcmd.Name() == "help" {
				if len(subcmd.Name()) > maxLen {
					maxLen = len(subcmd.Name())
				}
			}
		}
		
		// Group commands by category for better organization
		analysisCommands := []*cobra.Command{}
		utilityCommands := []*cobra.Command{}
		otherCommands := []*cobra.Command{}
		
		// Categorize commands
		for _, subcmd := range cmd.Commands() {
			if !subcmd.IsAvailableCommand() && subcmd.Name() != "help" {
				continue
			}
			
			switch subcmd.Name() {
			case "registry", "call-graph", "memory-safety", "undefined-behavior", "complexity", "api-usage", "include-graph", "coherence", "duplicate":
				analysisCommands = append(analysisCommands, subcmd)
			case "concat", "todo", "help", "completion":
				utilityCommands = append(utilityCommands, subcmd)
			default:
				otherCommands = append(otherCommands, subcmd)
			}
		}
		
		// Print analysis commands
		if len(analysisCommands) > 0 {
			fmt.Println(headerStyle("  Analysis Commands:"))
			for _, subcmd := range analysisCommands {
				fmt.Printf("    %s  %s\n", 
					subCmdStyle(fmt.Sprintf("%-*s", maxLen+2, subcmd.Name())),
					subcmd.Short)
			}
			fmt.Println()
		}
		
		// Print utility commands
		if len(utilityCommands) > 0 {
			fmt.Println(headerStyle("  Utility Commands:"))
			for _, subcmd := range utilityCommands {
				fmt.Printf("    %s  %s\n", 
					subCmdStyle(fmt.Sprintf("%-*s", maxLen+2, subcmd.Name())),
					subcmd.Short)
			}
			fmt.Println()
		}
		
		// Print other commands
		if len(otherCommands) > 0 {
			fmt.Println(headerStyle("  Other Commands:"))
			for _, subcmd := range otherCommands {
				fmt.Printf("    %s  %s\n", 
					subCmdStyle(fmt.Sprintf("%-*s", maxLen+2, subcmd.Name())),
					subcmd.Short)
			}
			fmt.Println()
		}
		
		// Print flags if available
		if cmd.HasAvailableLocalFlags() {
			fmt.Println(headerStyle("Flags:"))
			flags := cmd.LocalFlags()
			flags.VisitAll(func(flag *pflag.Flag) {
				if flag.Hidden {
					return
				}
				
				name := ""
				if flag.Shorthand != "" && flag.ShorthandDeprecated == "" {
					name = fmt.Sprintf("-%s, --%s", flag.Shorthand, flag.Name)
				} else {
					name = fmt.Sprintf("    --%s", flag.Name)
				}
				
				fmt.Printf("  %s  %s\n", flagStyle(fmt.Sprintf("%-20s", name)), flag.Usage)
			})
			fmt.Println()
		}
		
		// Print global flags
		if cmd.HasAvailableInheritedFlags() {
			fmt.Println(headerStyle("Global Flags:"))
			cmd.InheritedFlags().VisitAll(func(flag *pflag.Flag) {
				if flag.Hidden {
					return
				}
				
				name := ""
				if flag.Shorthand != "" && flag.ShorthandDeprecated == "" {
					name = fmt.Sprintf("-%s, --%s", flag.Shorthand, flag.Name)
				} else {
					name = fmt.Sprintf("    --%s", flag.Name)
				}
				
				fmt.Printf("  %s  %s\n", flagStyle(fmt.Sprintf("%-20s", name)), flag.Usage)
			})
		}
		
		// Print additional help text
		fmt.Println()
		fmt.Println(headerStyle("Use"), cmdStyle("gop [command] --help"), headerStyle("for more information about a command."))
	})

	// Add template functions for custom templates if needed
	cobra.AddTemplateFunc("yellow", func(text string) string {
		return color.YellowString(text)
	})

	cobra.AddTemplateFunc("cyan", func(text string) string {
		return color.CyanString(text)
	})

	cobra.AddTemplateFunc("green", func(text string) string {
		return color.GreenString(text)
	})

	// Configure common flags
	rootCmd.PersistentFlags().StringVarP(&inputFile, "input-file", "i", "", "Path to input file containing list of files to process")
	rootCmd.PersistentFlags().StringVarP(&directory, "directory", "d", ".", "Root directory to analyze")
	rootCmd.PersistentFlags().IntVarP(&depth, "depth", "", -1, "Maximum depth for directory traversal (-1 for unlimited)")
	rootCmd.PersistentFlags().StringVarP(&outputFile, "output-file", "o", "", "Path to output file for results")
	rootCmd.PersistentFlags().StringSliceVarP(&languages, "languages", "l", []string{"c", "cpp", "h", "hpp"}, "Languages to analyze (e.g., 'c', 'cpp')")
	rootCmd.PersistentFlags().StringSliceVarP(&excludes, "excludes", "e", []string{}, "Directories or files to exclude")
	rootCmd.PersistentFlags().IntVarP(&jobs, "jobs", "j", runtime.NumCPU(), "Number of concurrent jobs for processing") 
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Concat command
	concatCmd := &cobra.Command{
		Use:   "concat",
		Short: "Concatenate source files into a single text file",
		Long: `Concatenate all source files into a single text file.
Organizes content as header then source, header then source, etc.

This command is useful for:
- Creating a single file from multiple source files for code review
- Preparing code for documentation or sharing
- Generating a consolidated view of a codebase

Examples:
  # Concatenate all C files in the current directory
  gop concat -l c

  # Concatenate all C++ files in a specific directory with line numbers
  gop concat -d /path/to/project -l cpp -L

  # Concatenate files with headers and remove comments
  gop concat -H -R`,
		Run: func(cmd *cobra.Command, args []string) {
			// No default output file - will output to console if not specified
			options := analyzer.ConcatOptions{
				InputFile:      inputFile,
				Directory:      directory,
				Depth:          depth,
				OutputFile:     outputFile,
				Languages:      languages,
				Excludes:       excludes,
				Jobs:           jobs,
				IncludeHeaders: includeHeaders,
				AddLineNumbers: addLineNumbers,
				RemoveComments: removeComments,
				Short:          shortOutput,
				Verbose:        verbose,
			}
			analyzer.ConcatenateFiles(options)
		},
	}

	concatCmd.Flags().BoolVarP(&includeHeaders, "include-headers", "H", false, "Include file headers in output")
	concatCmd.Flags().BoolVarP(&addLineNumbers, "add-line-numbers", "L", false, "Add line numbers to output")
	concatCmd.Flags().BoolVarP(&removeComments, "remove-comments", "R", false, "Remove comments from output")
	concatCmd.Flags().BoolVar(&shortOutput, "short-concat", false, "Use short output format")

	// Registry command
	registryCmd := &cobra.Command{
		Use:   "registry",
		Short: "Create a registry of code elements",
		Long: `Create a text file that summarizes all definitions of constants, methods, and other elements in files.

This command catalogs all code elements in your codebase, making it easier to:
- Understand the structure and organization of a project
- Find specific functions, methods, classes, or constants
- Generate documentation or API references
- Analyze code dependencies and relationships

Examples:
  # Create a registry of all code elements in the current directory
  gop registry

  # Create a condensed registry of only functions and methods
  gop registry -t functions -t methods -s

  # Create a registry with statistics and relationships
  gop registry --stats -r

  # Create an AI-optimized registry for further processing
  gop registry --ia`,
		Run: func(cmd *cobra.Command, args []string) {
			// No default output file - will output to console if not specified
			options := analyzer.RegistryOptions{
				InputFile:  inputFile,
				Directory:  directory,
				Depth:      depth,
				OutputFile: outputFile,
				Languages:  languages,
				Excludes:   excludes,
				Jobs:       jobs,
				Types:      types,
				Short:      shortRegistry || shortOutput,
				Relations:  showRelations,
				Stats:      showStats,
				IAOutput:   iaOutput,
				Verbose:    verbose,
			}
			analyzer.CreateRegistry(options)
		},
	}

	// Define flags for registry command
	registryCmd.Flags().StringSliceVar(&types, "types", []string{"all"}, "Types to include in output (e.g., function,method,constant,all)")
	registryCmd.Flags().BoolVarP(&shortRegistry, "short-registry", "s", false, "Use short output format")
	registryCmd.Flags().BoolVarP(&showRelations, "show-relations", "r", false, "Display relations between files and methods")
	registryCmd.Flags().BoolVar(&showStats, "show-stats", false, "Show statistics (counts of methods, etc.)")
	registryCmd.Flags().BoolVar(&iaOutput, "ai-output", false, "Output in a format optimized for AI processing")

	// Todo command
	todoCmd := &cobra.Command{
		Use:   "todo",
		Short: "List all TODO items in the codebase",
Long: `List all methods marked as TODO in files.
By default, searches for "TODO", "placeholders", "simplification", "heuristic", "todo", "simple", etc.

This command helps you track and manage technical debt by:
- Finding all TODO comments across your codebase
- Categorizing them by type (TODO, FIXME, HACK, etc.)
- Providing context around each TODO item
- Generating a comprehensive report for task planning

The command uses multiple parallel jobs to quickly process large codebases.

Examples:
  # Find all TODOs in the current directory
  gop todo

  # Find TODOs in a specific file
  gop todo -i /path/to/file.c

  # Find only FIXME items
  gop todo --filter=FIXME

  # Find placeholder and simplification items
  gop todo --filter=PLACEHOLDER,SIMPLIFY

  # Find multiple specific types of items
  gop todo --filter=TODO,FIXME,HACK,PLACEHOLDER,SIMPLIFY,HEURISTIC

  # Find TODOs in a specific directory with verbose output
  gop todo -d /path/to/project -v`,
		Run: func(cmd *cobra.Command, args []string) {
			// Parse filter string if provided
			var filters []string
			if todoFilter != "" {
				filters = strings.Split(todoFilter, ",")
				// Trim spaces from each filter
				for i, filter := range filters {
					filters[i] = strings.TrimSpace(filter)
				}
			}

			// No default output file - will output to console if not specified
			options := analyzer.TodoOptions{
				InputFile:  inputFile,
				Directory:  directory,
				Depth:      depth,
				OutputFile: outputFile,
				Languages:  languages,
				Excludes:   excludes,
				Jobs:       jobs,
				Filter:     filters,
				Short:      shortOutput,
				Verbose:    verbose,
			}
			analyzer.FindTodos(options)
		},
	}

	// Add filter flag for todo command
	todoCmd.Flags().StringVar(&todoFilter, "filter", "", "Filter TODOs by type (comma-separated list, e.g., 'TODO,FIXME,HACK,PLACEHOLDER,SIMPLIFY,HEURISTIC,NOTE,BUG,OPTIMIZE,WORKAROUND')")
	todoCmd.Flags().BoolVar(&shortOutput, "short-todo", false, "Use short output format")

	// Coherence command
	coherenceCmd := &cobra.Command{
		Use:   "coherence",
		Args:  cobra.NoArgs,
		Short: "Check coherence between headers and implementations",
		Long: `Check that what is declared in headers is implemented in sources and vice versa.

This command helps maintain code quality by:
- Identifying declarations in headers that lack implementations
- Finding implementations that aren't properly declared in headers
- Detecting potential naming discrepancies using similarity matching
- Generating comprehensive reports of code inconsistencies

The command uses parallel processing to efficiently analyze large codebases and can be configured to focus on specific types of discrepancies.

Examples:
  # Check coherence in the current directory
  gop coherence

  # Check only for non-implemented declarations
  gop coherence --non-implemented

  # Check only for non-declared implementations
  gop coherence --not-declared

  # Enable similarity detection with a threshold
  gop coherence --similarity-threshold 0.8

  # Generate AI-optimized output for further processing
  gop coherence --ia`,
		Run: func(cmd *cobra.Command, args []string) {
			// No default output file - will output to console if not specified
			if !checkHeaders && !checkFiles {
				checkHeaders = true
				checkFiles = true
			}
			options := analyzer.CoherenceOptions{
				InputFile:           inputFile,
				Directory:           directory,
				Depth:               depth,
				OutputFile:          outputFile,
				Languages:           languages,
				Excludes:            excludes,
				Jobs:                jobs,
				CheckHeaders:        checkHeaders,
				CheckFiles:          checkFiles,
				ShowDiscrepancies:   showDiscrepancies,
				NonImplemented:      nonImplemented,
				NotDeclared:         notDeclared,
				IAOutput:            iaOutput,
				SimilarityThreshold: similarityThreshold,
				Short:               shortOutput,
				Verbose:             verbose,
			}
			analyzer.CheckCoherence(options)
		},
	}
	// Define all flags with Flags(), not PersistentFlags()
	coherenceCmd.Flags().BoolVar(&checkHeaders, "headers-to-file", true, "Check declarations in headers are implemented in source files")
	coherenceCmd.Flags().BoolVar(&checkFiles, "file-to-headers", true, "Check implementations in source files are declared in headers")
	coherenceCmd.Flags().BoolVar(&showDiscrepancies, "discrepancies", true, "Show discrepancies between headers and implementations")
	coherenceCmd.Flags().BoolVar(&nonImplemented, "non-implemented", false, "Show only non-implemented declarations")
	coherenceCmd.Flags().BoolVar(&notDeclared, "not-declared", false, "Show only non-declared implementations")
	coherenceCmd.Flags().BoolVar(&iaOutput, "ia", false, "Output in a format optimized for AI processing")
	coherenceCmd.Flags().Float64Var(&similarityThreshold, "similarity-threshold", 0.0, "Threshold for function similarity detection (0.0-1.0, 0 disables)")
	coherenceCmd.Flags().BoolVar(&shortOutput, "short-coherence", false, "Use short output format")

	// Duplicate command
	duplicateCmd := &cobra.Command{
		Use:   "duplicate",
		Args:  cobra.NoArgs,
		Short: "Find duplicate code in the codebase",
		Long: `Find duplicate or similar code blocks in the codebase.

This command helps maintain code quality by:
- Identifying duplicate or similar code blocks
- Suggesting potential refactoring opportunities
- Providing similarity scores for code comparison

The command uses text similarity algorithms to detect code duplication
and can be configured with different thresholds and minimum block sizes.

Examples:
  # Find duplicates in the current directory
  gop duplicate

  # Find duplicates with a custom similarity threshold
  gop duplicate --threshold 0.9

  # Find duplicates with a minimum block size
  gop duplicate --min-lines 10`,
		Run: func(cmd *cobra.Command, args []string) {
			// No default output file - will output to console if not specified
			options := analyzer.DuplicateOptions{
				InputFile:           inputFile,
				Directory:           directory,
				Depth:               depth,
				OutputFile:          outputFile,
				Languages:           languages,
				Excludes:            excludes,
				Jobs:                jobs,
				SimilarityThreshold: similarityThreshold,
				MinLineCount:        minLineCount,
				Short:               shortOutput,
				NamesOnly:           namesOnly,
				Verbose:             verbose,
				// Monitoring options
				Monitor:        monitorEnabled,
				MonitorFile:    monitorFile,
				MonitorComment: monitorComment,
			}
			analyzer.FindDuplicates(options)
		},
	}
	
	// Define flags for duplicate command
	duplicateCmd.Flags().Float64Var(&similarityThreshold, "threshold", 0.8, "Similarity threshold for duplicate detection (0.0-1.0)")
	duplicateCmd.Flags().IntVar(&minLineCount, "min-lines", 5, "Minimum number of lines for a code block to be considered")
	duplicateCmd.Flags().BoolVar(&shortOutput, "short-duplicate", false, "Use short output format")
	duplicateCmd.Flags().BoolVar(&namesOnly, "names-only", false, "Show only method/function names in output")
	// Monitoring flags
	duplicateCmd.Flags().BoolVar(&monitorEnabled, "monitor", false, "Enable monitoring of duplication over time")
	duplicateCmd.Flags().StringVar(&monitorFile, "monitor-file", "duplication_history.json", "Path to the monitoring history file")
	duplicateCmd.Flags().StringVar(&monitorComment, "monitor-comment", "", "Optional comment for this monitoring run")

	// Add version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version number of gop",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(color.GreenString("gop version 1.0.0"))
		},
	})
	
	// Call Graph command
	callGraphCmd := &cobra.Command{
		Use:   "call-graph",
		Short: "Generate a call graph for C/C++ code",
		Long: `Generate a static call graph for C/C++ code to visualize function call relationships.

Examples:
  # Generate a call graph for the current directory
  gop call-graph

  # Generate a call graph in DOT format
  gop call-graph --format=dot -o call_graph.dot

  # Generate a call graph with verbose output
  gop call-graph -v`,
		Run: func(cmd *cobra.Command, args []string) {
			// No default output file - will output to console if not specified
			options := analyzer.CallGraphOptions{
				InputFile:  inputFile,
				Directory:  directory,
				Depth:      depth,
				OutputFile: outputFile,
				Languages:  languages,
				Excludes:   excludes,
				Jobs:       jobs,
				Format:     "md",
				Short:      shortOutput,
				Verbose:    verbose,
			}
			analyzer.GenerateCallGraph(options)
		},
	}
	
	// Define flags for call-graph command
	callGraphCmd.Flags().StringVar(&format, "format", "md", "Output format (md, dot, json)")
	callGraphCmd.Flags().BoolVar(&shortOutput, "short", false, "Use short output format")

	// Memory Safety command
	memorySafetyCmd := &cobra.Command{
		Use:   "memory-safety",
		Short: "Check for memory safety issues in C/C++ code",
		Long: `Analyze C/C++ code for potential memory safety issues like buffer overflows,
memory leaks, null pointer dereferences, and use-after-free bugs.

Examples:
  # Check for memory safety issues in the current directory
  gop memory-safety

  # Check with verbose output
  gop memory-safety -v`,
		Run: func(cmd *cobra.Command, args []string) {
			// No default output file - will output to console if not specified
			options := analyzer.MemorySafetyOptions{
				InputFile:  inputFile,
				Directory:  directory,
				Depth:      depth,
				OutputFile: outputFile,
				Languages:  languages,
				Excludes:   excludes,
				Jobs:       jobs,
				Short:      shortOutput,
				Verbose:    verbose,
			}
			analyzer.AnalyzeMemorySafety(options)
		},
	}
	
	// Define flags for memory-safety command
	memorySafetyCmd.Flags().BoolVar(&shortOutput, "short", false, "Use short output format")

	// Undefined Behavior command
	undefinedBehaviorCmd := &cobra.Command{
		Use:   "undefined-behavior",
		Short: "Detect undefined behavior in C/C++ code",
		Long: `Analyze C/C++ code for potential undefined behavior like signed integer overflow,
null pointer dereference, division by zero, and uninitialized variables.

Examples:
  # Check for undefined behavior in the current directory
  gop undefined-behavior

  # Check with verbose output
  gop undefined-behavior -v`,
		Run: func(cmd *cobra.Command, args []string) {
			// No default output file - will output to console if not specified
			options := analyzer.UndefinedBehaviorOptions{
				InputFile:           inputFile,
				Directory:           directory,
				Depth:               depth,
				OutputFile:          outputFile,
				Languages:           languages,
				Excludes:            excludes,
				Jobs:                jobs,
				CheckSignedOverflow: true,
				CheckNullDereference: true,
				CheckDivByZero:      true,
				CheckUninitVar:      true,
				CheckOutOfBounds:    true,
				CheckShiftOperations: true,
				Short:               shortOutput,
				Verbose:             verbose,
			}
			analyzer.AnalyzeUndefinedBehavior(options)
		},
	}
	
	// Define flags for undefined-behavior command
	undefinedBehaviorCmd.Flags().BoolVar(&shortOutput, "short", false, "Use short output format")

	// Complexity command
	complexityCmd := &cobra.Command{
		Use:   "complexity",
		Short: "Analyze code complexity in C/C++ code",
		Long: `Analyze C/C++ code for cyclomatic complexity and identify functions
that may be too complex and need refactoring.

Examples:
  # Analyze code complexity in the current directory
  gop complexity

  # Analyze with a specific threshold
  gop complexity --threshold 15`,
		Run: func(cmd *cobra.Command, args []string) {
			// No default output file - will output to console if not specified
			options := analyzer.ComplexityOptions{
				InputFile:  inputFile,
				Directory:  directory,
				Depth:      depth,
				OutputFile: outputFile,
				Languages:  languages,
				Excludes:   excludes,
				Jobs:       jobs,
				Threshold:  complexityThreshold,
				Short:      shortOutput,
				Verbose:    verbose,
			}
			analyzer.AnalyzeComplexity(options)
		},
	}
	
	// Define flags for complexity command
	complexityCmd.Flags().IntVar(&complexityThreshold, "threshold", 10, "Complexity threshold for flagging functions")
	complexityCmd.Flags().BoolVar(&shortOutput, "short", false, "Use short output format")

	// API Usage command
	apiUsageCmd := &cobra.Command{
		Use:   "api-usage",
		Short: "Analyze API usage in C/C++ code",
		Long: `Analyze C/C++ code for API usage patterns and identify potential issues
like deprecated API usage or misuse of functions.

Examples:
  # Analyze API usage in the current directory
  gop api-usage

  # Analyze with a specific API definition file
  gop api-usage --api-file api_defs.json`,
		Run: func(cmd *cobra.Command, args []string) {
			// No default output file - will output to console if not specified
			options := analyzer.APIUsageOptions{
				InputFile:         inputFile,
				Directory:         directory,
				Depth:             depth,
				OutputFile:        outputFile,
				Languages:         languages,
				Excludes:          excludes,
				Jobs:              jobs,
				APIDefinitionFile: apiFile,
				CheckDeprecated:   true,
				Short:             shortOutput,
				Verbose:           verbose,
			}
			analyzer.AnalyzeAPIUsage(options)
		},
	}
	
	// Define flags for api-usage command
	apiUsageCmd.Flags().StringVar(&apiFile, "api-file", "", "Path to API definition file")
	apiUsageCmd.Flags().BoolVar(&shortOutput, "short", false, "Use short output format")

	// Include Graph command
	includeGraphCmd := &cobra.Command{
		Use:   "include-graph",
		Short: "Generate an include dependency graph for C/C++ code",
		Long: `Generate a graph of include dependencies for C/C++ code to visualize
header file relationships and identify potential circular dependencies.

Examples:
  # Generate an include graph for the current directory
  gop include-graph

  # Generate an include graph in DOT format
  gop include-graph --format=dot -o include_graph.dot`,
		Run: func(cmd *cobra.Command, args []string) {
			// No default output file - will output to console if not specified
			options := analyzer.IncludeGraphOptions{
				InputFile:  inputFile,
				Directory:  directory,
				Depth:      depth,
				OutputFile: outputFile,
				Languages:  languages,
				Excludes:   excludes,
				Jobs:       jobs,
				Format:     format,
				Short:      shortOutput,
				Verbose:    verbose,
			}
			analyzer.GenerateIncludeGraph(options)
		},
	}
	
	// Define flags for include-graph command
	includeGraphCmd.Flags().StringVar(&format, "format", "md", "Output format (md, dot, json)")
	includeGraphCmd.Flags().BoolVar(&shortOutput, "short", false, "Use short output format")

	// Metrics command
	metricsCmd := &cobra.Command{
		Use:   "metrics",
		Short: "Calculate code metrics for C/C++ code",
		Long: `Calculate various code metrics for C/C++ code, including:
- Lines of code (total, code, comments, blank)
- Cyclomatic complexity distribution
- Function/method count and average size
- Class/struct count and average size
- Comment-to-code ratio

Examples:
  # Calculate metrics for the current directory
  gop metrics

  # Calculate metrics with output to a file
  gop metrics -o metrics.md`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create options for metrics command
			options := analyzer.MetricsOptions{
				InputFile:  inputFile,
				Directory:  directory,
				Depth:      depth,
				OutputFile: outputFile,
				Languages:  languages,
				Excludes:   excludes,
				Jobs:       jobs,
				Short:      shortOutput,
				Verbose:    verbose,
			}
			analyzer.CalculateMetrics(options)
		},
	}

	// Define flags for metrics command
	metricsCmd.Flags().BoolVar(&shortOutput, "short", false, "Use short output format")

	// Docs command
	docsCmd := &cobra.Command{
		Use:   "docs",
		Short: "Generate documentation from code comments",
		Long: `Extract documentation from code comments and generate Markdown documentation.
Identifies functions, classes, structs, and enums with documentation comments.

Examples:
  # Generate documentation for the current directory
  gop docs

  # Generate documentation with code snippets
  gop docs --include-code`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create options for docs command
			options := analyzer.DocsOptions{
				InputFile:   inputFile,
				Directory:   directory,
				Depth:       depth,
				OutputFile:  outputFile,
				Languages:   languages,
				Excludes:    excludes,
				Jobs:        jobs,
				IncludeCode: includeCode,
				Short:       shortOutput,
				Verbose:     verbose,
			}
			analyzer.GenerateDocs(options)
		},
	}

	// Define flags for docs command
	docsCmd.Flags().BoolVar(&includeCode, "include-code", false, "Include code snippets in documentation")
	docsCmd.Flags().BoolVar(&shortOutput, "short", false, "Use short output format")

	// Profile command
	profileCmd := &cobra.Command{
		Use:   "profile",
		Short: "Profile the performance of an executable",
		Long: `Profile the performance of an executable using platform-specific profiling tools.
Supports CPU, memory, and time profiling.

Examples:
  # Profile the time performance of an executable
  gop profile --executable ./myapp --type time

  # Profile CPU usage for 30 seconds
  gop profile --executable ./myapp --type cpu --duration 30`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create options for profile command
			options := analyzer.ProfileOptions{
				Executable:  executable,
				Args:        args,
				OutputFile:  outputFile,
				Format:      format,
				ProfileType: profileType,
				Duration:    duration,
				Short:       shortOutput,
				Verbose:     verbose,
			}
			analyzer.RunProfiler(options)
		},
	}

	// Define flags for profile command
	profileCmd.Flags().StringVar(&executable, "executable", "", "Path to the executable to profile")
	profileCmd.Flags().StringSliceVar(&args, "args", []string{}, "Arguments to pass to the executable")
	profileCmd.Flags().StringVar(&profileType, "type", "time", "Type of profiling (cpu, memory, time)")
	profileCmd.Flags().IntVar(&duration, "duration", 10, "Duration of profiling in seconds (for cpu and memory)")
	profileCmd.Flags().StringVar(&format, "format", "md", "Output format (md, txt)")
	profileCmd.Flags().BoolVar(&shortOutput, "short", false, "Use short output format")
	profileCmd.MarkFlagRequired("executable")

	// Refactor command
	refactorCmd := &cobra.Command{
		Use:   "refactor",
		Short: "Refactor code by replacing patterns",
		Long: `Refactor code by replacing patterns with specified replacements.
Supports literal or regex pattern matching with various options.

Examples:
  # Replace a function name across all files
  gop refactor --pattern oldFunction --replacement newFunction --whole-word

  # Use regex for more complex replacements
  gop refactor --pattern "foo\\(([^)]*)\\)" --replacement "bar($1)" --regex`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create options for refactor command
			options := analyzer.RefactorOptions{
				InputFile:     inputFile,
				Directory:     directory,
				Depth:         depth,
				OutputFile:    outputFile,
				Languages:     languages,
				Excludes:      excludes,
				Pattern:       pattern,
				Replacement:   replacement,
				RegexMode:     regexMode,
				WholeWord:     wholeWord,
				CaseSensitive: caseSensitive,
				Jobs:          jobs,
				DryRun:        dryRun,
				Backup:        backup,
				Verbose:       verbose,
			}
			analyzer.RunRefactor(options)
		},
	}

	// Define flags for refactor command
	refactorCmd.Flags().StringVar(&pattern, "pattern", "", "Pattern to search for")
	refactorCmd.Flags().StringVar(&replacement, "replacement", "", "Replacement for the pattern")
	refactorCmd.Flags().BoolVar(&regexMode, "regex", false, "Use regex for pattern matching")
	refactorCmd.Flags().BoolVar(&wholeWord, "whole-word", false, "Match whole words only")
	refactorCmd.Flags().BoolVar(&caseSensitive, "case-sensitive", false, "Use case-sensitive matching")
	refactorCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Perform a dry run (no actual changes)")
	refactorCmd.Flags().BoolVar(&backup, "backup", true, "Create backup files before making changes")
	refactorCmd.MarkFlagRequired("pattern")
	refactorCmd.MarkFlagRequired("replacement")

	// Add commands to root command
	rootCmd.AddCommand(concatCmd)
	rootCmd.AddCommand(todoCmd)
	rootCmd.AddCommand(coherenceCmd)
	rootCmd.AddCommand(duplicateCmd)
	rootCmd.AddCommand(callGraphCmd)
	rootCmd.AddCommand(includeGraphCmd)
	rootCmd.AddCommand(metricsCmd)
	rootCmd.AddCommand(docsCmd)
	rootCmd.AddCommand(profileCmd)
	rootCmd.AddCommand(refactorCmd)
	rootCmd.AddCommand(coherenceCmd)
	rootCmd.AddCommand(duplicateCmd)
	rootCmd.AddCommand(callGraphCmd)
	rootCmd.AddCommand(memorySafetyCmd)
	rootCmd.AddCommand(undefinedBehaviorCmd)
	rootCmd.AddCommand(complexityCmd)
	rootCmd.AddCommand(apiUsageCmd)
	rootCmd.AddCommand(includeGraphCmd)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(color.RedString("Error:"), err)
		os.Exit(1)
	}
}
