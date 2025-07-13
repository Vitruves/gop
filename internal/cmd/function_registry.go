package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vitruves/gop/internal/registry"
)

var (
	registryOutputFile      string
	registryByScript        bool
	registryOnlyHeaderFiles bool
	registryAddRelations    bool
	registryOnlyDeadCode    bool
)

var functionRegistryCmd = &cobra.Command{
	Use:   "function-registry",
	Short: "Create a registry of all functions in codebase",
	Long: `Create a comprehensive registry of all functions in the codebase with detailed information
including usage, availability (private/public), call relationships, and more.`,
	RunE: runFunctionRegistry,
}

func init() {
	functionRegistryCmd.Flags().StringVarP(&registryOutputFile, "output", "o", "", "Output file (.md, .txt, .yaml, .json, or .csv)")
	functionRegistryCmd.Flags().BoolVar(&registryByScript, "by-script", false, "Group functions by script/file")
	functionRegistryCmd.Flags().BoolVar(&registryOnlyHeaderFiles, "only-header-files", false, "For C/C++: only analyze header files")
	functionRegistryCmd.Flags().BoolVar(&registryAddRelations, "add-relations", false, "Analyze function call relationships")
	functionRegistryCmd.Flags().BoolVar(&registryOnlyDeadCode, "only-dead-code", false, "Show only unused/dead functions")
}

func runFunctionRegistry(cmd *cobra.Command, args []string) error {
	config := registry.Config{
		Language:        language,
		Include:         include,
		Exclude:         exclude,
		Recursive:       recursive,
		Depth:           depth,
		Jobs:            jobs,
		Verbose:         verbose,
		OutputFile:      registryOutputFile,
		ByScript:        registryByScript,
		OnlyHeaderFiles: registryOnlyHeaderFiles,
		AddRelations:    registryAddRelations,
		OnlyDeadCode:    registryOnlyDeadCode,
	}

	return registry.Run(config)
}