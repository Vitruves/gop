package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
)

// ProfileOptions contains options for the profile command
type ProfileOptions struct {
	// Input options
	Executable string   // Path to the executable to profile
	Args       []string // Arguments to pass to the executable
	
	// Output options
	OutputFile string // Path to output file for results
	Format     string // Output format (md, txt)
	
	// Profiling options
	ProfileType string // Type of profiling (cpu, memory, time)
	Duration    int    // Duration of profiling in seconds (for cpu and memory)
	
	// Display options
	Short   bool // Whether to use short output format
	Verbose bool // Whether to enable verbose output
}

// ProfileResult contains the results of profiling
type ProfileResult struct {
	ExecutablePath string            // Path to the profiled executable
	Args           []string          // Arguments passed to the executable
	ProfileType    string            // Type of profiling performed
	Duration       time.Duration     // Duration of profiling
	ExitCode       int               // Exit code of the profiled process
	Output         string            // Raw output from the profiler
	Metrics        map[string]string // Key metrics extracted from the profiler output
	HotspotFiles   []string          // Files with performance hotspots
	HotspotLines   []int             // Lines with performance hotspots
}

// RunProfiler runs a performance profiler on the given executable
func RunProfiler(options ProfileOptions) {
	startTime := time.Now()

	// Validate executable path
	if _, err := os.Stat(options.Executable); os.IsNotExist(err) {
		fmt.Println(color.RedString("Error: Executable not found:"), options.Executable)
		return
	}

	// Determine profiler to use based on OS
	var profilerCmd *exec.Cmd
	var profilerOutput []byte
	var err error
	var profileResult ProfileResult

	profileResult = ProfileResult{
		ExecutablePath: options.Executable,
		Args:           options.Args,
		ProfileType:    options.ProfileType,
		Metrics:        make(map[string]string),
	}

	// Set up profiler based on OS and profile type
	switch runtime.GOOS {
	case "darwin":
		// macOS - use Instruments or time
		switch options.ProfileType {
		case "cpu":
			fmt.Println(color.CyanString("Running CPU profiling with Instruments..."))
			tempFile := filepath.Join(os.TempDir(), "gop_profile.trace")
			
			// Build command with all arguments
			args := []string{
				"-l", "Instruments", 
				"-t", "Time Profiler", 
				"-D", tempFile,
			}
			args = append(args, options.Executable)
			args = append(args, options.Args...)
			
			profilerCmd = exec.Command("xcrun", args...)
			
			// Set timeout if duration is specified
			if options.Duration > 0 {
				go func() {
					time.Sleep(time.Duration(options.Duration) * time.Second)
					profilerCmd.Process.Signal(os.Interrupt)
				}()
			}
			
		case "memory":
			fmt.Println(color.CyanString("Running memory profiling with Instruments..."))
			tempFile := filepath.Join(os.TempDir(), "gop_profile.trace")
			
			// Build command with all arguments
			args := []string{
				"-l", "Instruments", 
				"-t", "Allocations", 
				"-D", tempFile,
			}
			args = append(args, options.Executable)
			args = append(args, options.Args...)
			
			profilerCmd = exec.Command("xcrun", args...)
			
			// Set timeout if duration is specified
			if options.Duration > 0 {
				go func() {
					time.Sleep(time.Duration(options.Duration) * time.Second)
					profilerCmd.Process.Signal(os.Interrupt)
				}()
			}
			
		case "time":
			fmt.Println(color.CyanString("Running time profiling..."))
			
			// Build command with all arguments
			args := []string{options.Executable}
			args = append(args, options.Args...)
			
			profilerCmd = exec.Command("time", args...)
		}
		
	case "linux":
		// Linux - use perf or time
		switch options.ProfileType {
		case "cpu":
			fmt.Println(color.CyanString("Running CPU profiling with perf..."))
			
			// Build command with all arguments
			args := []string{"record", "-g", "-o", "perf.data"}
			args = append(args, options.Executable)
			args = append(args, options.Args...)
			
			profilerCmd = exec.Command("perf", args...)
			
			// Set timeout if duration is specified
			if options.Duration > 0 {
				go func() {
					time.Sleep(time.Duration(options.Duration) * time.Second)
					profilerCmd.Process.Signal(os.Interrupt)
				}()
			}
			
		case "memory":
			fmt.Println(color.CyanString("Running memory profiling with valgrind..."))
			
			// Build command with all arguments
			args := []string{"--tool=massif", "--massif-out-file=massif.out"}
			args = append(args, options.Executable)
			args = append(args, options.Args...)
			
			profilerCmd = exec.Command("valgrind", args...)
			
		case "time":
			fmt.Println(color.CyanString("Running time profiling..."))
			
			// Build command with all arguments
			args := []string{options.Executable}
			args = append(args, options.Args...)
			
			profilerCmd = exec.Command("time", args...)
		}
		
	default:
		// Windows or other - use built-in time measurement
		fmt.Println(color.CyanString("Running basic time profiling..."))
		
		// Build command with all arguments
		args := []string{options.Executable}
		args = append(args, options.Args...)
		
		profilerCmd = exec.Command("cmd", append([]string{"/C", "time"}, args...)...)
	}

	// Run the profiler
	if profilerCmd != nil {
		profilerOutput, err = profilerCmd.CombinedOutput()
		if err != nil {
			// Check if it's just a non-zero exit code from the profiled program
			if exitErr, ok := err.(*exec.ExitError); ok {
				profileResult.ExitCode = exitErr.ExitCode()
			} else {
				fmt.Println(color.RedString("Error running profiler:"), err)
				return
			}
		}
		
		profileResult.Output = string(profilerOutput)
		profileResult.Duration = time.Since(startTime)
		
		// Process profiler output to extract metrics
		processProfilerOutput(&profileResult, options)
	} else {
		fmt.Println(color.RedString("Error: Unsupported profiling configuration"))
		return
	}

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

	// Write output
	writeProfileOutput(writer, profileResult, options)
}

// processProfilerOutput processes the raw profiler output to extract metrics
func processProfilerOutput(result *ProfileResult, options ProfileOptions) {
	// Process output based on profiler type and OS
	switch runtime.GOOS {
	case "darwin":
		switch options.ProfileType {
		case "cpu", "memory":
			// For Instruments, we can't easily parse the output directly
			// We'll just provide basic information
			result.Metrics["profiler"] = "Instruments"
			result.Metrics["duration"] = result.Duration.String()
			
			// Extract some basic metrics from the output
			if strings.Contains(result.Output, "CPU usage") {
				cpuUsageRegex := regexp.MustCompile(`CPU usage: (\d+\.\d+)%`)
				if matches := cpuUsageRegex.FindStringSubmatch(result.Output); len(matches) > 1 {
					result.Metrics["cpu_usage"] = matches[1] + "%"
				}
			}
			
		case "time":
			// Extract time information
			realTimeRegex := regexp.MustCompile(`real\s+(\d+m\d+\.\d+s)`)
			userTimeRegex := regexp.MustCompile(`user\s+(\d+m\d+\.\d+s)`)
			sysTimeRegex := regexp.MustCompile(`sys\s+(\d+m\d+\.\d+s)`)
			
			if matches := realTimeRegex.FindStringSubmatch(result.Output); len(matches) > 1 {
				result.Metrics["real_time"] = matches[1]
			}
			
			if matches := userTimeRegex.FindStringSubmatch(result.Output); len(matches) > 1 {
				result.Metrics["user_time"] = matches[1]
			}
			
			if matches := sysTimeRegex.FindStringSubmatch(result.Output); len(matches) > 1 {
				result.Metrics["sys_time"] = matches[1]
			}
		}
		
	case "linux":
		switch options.ProfileType {
		case "cpu":
			// For perf, we need to run perf report to get the hotspots
			reportCmd := exec.Command("perf", "report", "-i", "perf.data", "--stdio")
			reportOutput, err := reportCmd.CombinedOutput()
			if err == nil {
				result.Output += "\n\n--- Perf Report ---\n" + string(reportOutput)
				
				// Extract hotspot information
				hotspotRegex := regexp.MustCompile(`\s+(\d+\.\d+)%.*\s+(\S+)\s+(\S+)`)
				lines := strings.Split(string(reportOutput), "\n")
				
				for _, line := range lines {
					if matches := hotspotRegex.FindStringSubmatch(line); len(matches) > 3 {
						result.Metrics["hotspot_"+matches[3]] = matches[1] + "%"
						
						// Try to extract file and line information
						if strings.Contains(matches[2], ":") {
							parts := strings.Split(matches[2], ":")
							if len(parts) > 1 {
								result.HotspotFiles = append(result.HotspotFiles, parts[0])
								// Try to parse line number
								if lineNum, err := fmt.Sscanf(parts[1], "%d", new(int)); err == nil && lineNum > 0 {
									result.HotspotLines = append(result.HotspotLines, lineNum)
								}
							}
						}
					}
				}
			}
			
		case "memory":
			// For valgrind massif, we need to run ms_print to get the memory profile
			reportCmd := exec.Command("ms_print", "massif.out")
			reportOutput, err := reportCmd.CombinedOutput()
			if err == nil {
				result.Output += "\n\n--- Memory Profile ---\n" + string(reportOutput)
				
				// Extract peak memory usage
				peakMemoryRegex := regexp.MustCompile(`Peak heap usage: (\d+,\d+) bytes`)
				if matches := peakMemoryRegex.FindStringSubmatch(string(reportOutput)); len(matches) > 1 {
					result.Metrics["peak_memory"] = matches[1] + " bytes"
				}
			}
			
		case "time":
			// Extract time information (similar to macOS)
			realTimeRegex := regexp.MustCompile(`real\s+(\d+m\d+\.\d+s)`)
			userTimeRegex := regexp.MustCompile(`user\s+(\d+m\d+\.\d+s)`)
			sysTimeRegex := regexp.MustCompile(`sys\s+(\d+m\d+\.\d+s)`)
			
			if matches := realTimeRegex.FindStringSubmatch(result.Output); len(matches) > 1 {
				result.Metrics["real_time"] = matches[1]
			}
			
			if matches := userTimeRegex.FindStringSubmatch(result.Output); len(matches) > 1 {
				result.Metrics["user_time"] = matches[1]
			}
			
			if matches := sysTimeRegex.FindStringSubmatch(result.Output); len(matches) > 1 {
				result.Metrics["sys_time"] = matches[1]
			}
		}
	}
}

// writeProfileOutput writes profiling output to the given writer
func writeProfileOutput(writer *bufio.Writer, result ProfileResult, options ProfileOptions) {
	// Write output based on format
	switch options.Format {
	case "md":
		// Write markdown output
		fmt.Fprintf(writer, "# Performance Profile\n\n")
		fmt.Fprintf(writer, "*Generated on %s*\n\n", time.Now().Format("2006-01-02 15:04:05"))
		
		// Write summary
		fmt.Fprintf(writer, "## Summary\n\n")
		fmt.Fprintf(writer, "| Metric | Value |\n")
		fmt.Fprintf(writer, "|--------|-------|\n")
		fmt.Fprintf(writer, "| Executable | `%s` |\n", result.ExecutablePath)
		fmt.Fprintf(writer, "| Arguments | `%s` |\n", strings.Join(result.Args, " "))
		fmt.Fprintf(writer, "| Profile Type | %s |\n", result.ProfileType)
		fmt.Fprintf(writer, "| Duration | %s |\n", result.Duration.Round(time.Millisecond))
		fmt.Fprintf(writer, "| Exit Code | %d |\n", result.ExitCode)
		
		// Write metrics
		if len(result.Metrics) > 0 {
			fmt.Fprintf(writer, "\n## Metrics\n\n")
			fmt.Fprintf(writer, "| Metric | Value |\n")
			fmt.Fprintf(writer, "|--------|-------|\n")
			
			for metric, value := range result.Metrics {
				fmt.Fprintf(writer, "| %s | %s |\n", formatMetricName(metric), value)
			}
		}
		
		// Write hotspots
		if len(result.HotspotFiles) > 0 {
			fmt.Fprintf(writer, "\n## Hotspots\n\n")
			fmt.Fprintf(writer, "| File | Line | Metric |\n")
			fmt.Fprintf(writer, "|------|------|--------|\n")
			
			for i, file := range result.HotspotFiles {
				line := ""
				if i < len(result.HotspotLines) {
					line = fmt.Sprintf("%d", result.HotspotLines[i])
				}
				
				metric := ""
				metricKey := fmt.Sprintf("hotspot_%s:%s", file, line)
				if value, ok := result.Metrics[metricKey]; ok {
					metric = value
				}
				
				fmt.Fprintf(writer, "| `%s` | %s | %s |\n", file, line, metric)
			}
		}
		
		// Write raw output if verbose
		if options.Verbose {
			fmt.Fprintf(writer, "\n## Raw Output\n\n")
			fmt.Fprintf(writer, "```\n%s\n```\n", result.Output)
		}
		
	default:
		// Write plain text output
		fmt.Fprintf(writer, "Performance Profile\n")
		fmt.Fprintf(writer, "Generated on %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
		
		// Write summary
		fmt.Fprintf(writer, "Summary:\n")
		fmt.Fprintf(writer, "  Executable: %s\n", result.ExecutablePath)
		fmt.Fprintf(writer, "  Arguments: %s\n", strings.Join(result.Args, " "))
		fmt.Fprintf(writer, "  Profile Type: %s\n", result.ProfileType)
		fmt.Fprintf(writer, "  Duration: %s\n", result.Duration.Round(time.Millisecond))
		fmt.Fprintf(writer, "  Exit Code: %d\n\n", result.ExitCode)
		
		// Write metrics
		if len(result.Metrics) > 0 {
			fmt.Fprintf(writer, "Metrics:\n")
			
			for metric, value := range result.Metrics {
				fmt.Fprintf(writer, "  %s: %s\n", formatMetricName(metric), value)
			}
			
			fmt.Fprintf(writer, "\n")
		}
		
		// Write hotspots
		if len(result.HotspotFiles) > 0 {
			fmt.Fprintf(writer, "Hotspots:\n")
			
			for i, file := range result.HotspotFiles {
				line := ""
				if i < len(result.HotspotLines) {
					line = fmt.Sprintf("%d", result.HotspotLines[i])
				}
				
				metric := ""
				metricKey := fmt.Sprintf("hotspot_%s:%s", file, line)
				if value, ok := result.Metrics[metricKey]; ok {
					metric = value
				}
				
				fmt.Fprintf(writer, "  %s:%s - %s\n", file, line, metric)
			}
			
			fmt.Fprintf(writer, "\n")
		}
		
		// Write raw output if verbose
		if options.Verbose {
			fmt.Fprintf(writer, "Raw Output:\n\n")
			fmt.Fprintf(writer, "%s\n", result.Output)
		}
	}
}

// formatMetricName formats a metric name for display
func formatMetricName(name string) string {
	// Replace underscores with spaces
	name = strings.ReplaceAll(name, "_", " ")
	
	// Capitalize first letter of each word
	words := strings.Split(name, " ")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	
	return strings.Join(words, " ")
}
