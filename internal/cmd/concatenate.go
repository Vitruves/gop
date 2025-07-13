package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vitruves/gop/internal/concatenate"
)

var (
	removeTests     bool
	removeComments  bool
	addLineNumbers  bool
	addHeaders      bool
	outputFile      string
)

var concatenateCmd = &cobra.Command{
	Use:   "concatenate",
	Short: "Concatenate all code matching language extension in current directory",
	Long:  `Concatenate code files based on language extension with various filtering and formatting options.`,
	RunE:  runConcatenate,
}

func init() {
	concatenateCmd.Flags().BoolVar(&removeTests, "remove-tests", false, "Remove test files and test code")
	concatenateCmd.Flags().BoolVar(&removeComments, "remove-comments", false, "Remove comments from code")
	concatenateCmd.Flags().BoolVar(&addLineNumbers, "add-line-numbers", false, "Add line numbers to each line")
	concatenateCmd.Flags().BoolVar(&addHeaders, "add-headers", false, "Add file headers to separate scripts")
	concatenateCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (if not specified, output to console)")
}

func runConcatenate(cmd *cobra.Command, args []string) error {
	config := concatenate.Config{
		Language:       language,
		Include:        include,
		Exclude:        exclude,
		Recursive:      recursive,
		Depth:          depth,
		Jobs:           jobs,
		Verbose:        verbose,
		RemoveTests:    removeTests,
		RemoveComments: removeComments,
		AddLineNumbers: addLineNumbers,
		AddHeaders:     addHeaders,
		OutputFile:     outputFile,
	}

	return concatenate.Run(config)
}