package cmd

import (
	"fmt"
	"runtime"
	"time"

	"github.com/spf13/cobra"
)

var (
	language  string
	include   []string
	exclude   []string
	recursive bool
	depth     int
	jobs      int
	verbose   bool
)

var rootCmd = &cobra.Command{
	Use:   "gop",
	Short: "A tool to provide utilities to help code with AI",
	Long: `gop is a CLI tool that provides various utilities to help with AI-assisted coding.
It can concatenate code files, create function registries, find placeholders, and generate statistics.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&language, "language", "l", "", "Programming language (python,rust,go,c,cpp)")
	rootCmd.PersistentFlags().StringArrayVarP(&include, "include", "i", []string{}, "Include directories or files (supports wildcards)")
	rootCmd.PersistentFlags().StringArrayVarP(&exclude, "exclude", "e", []string{}, "Exclude directories or files")
	rootCmd.PersistentFlags().BoolVarP(&recursive, "recursive", "R", false, "Recursively process all directories")
	rootCmd.PersistentFlags().IntVarP(&depth, "depth", "d", -1, "Maximum depth for recursive processing")
	rootCmd.PersistentFlags().IntVarP(&jobs, "jobs", "j", runtime.NumCPU(), "Number of CPU cores to use")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	rootCmd.AddCommand(concatenateCmd)
	rootCmd.AddCommand(functionRegistryCmd)
	rootCmd.AddCommand(placeholdersCmd)
	rootCmd.AddCommand(statsCmd)
}

func logInfo(msg string) {
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