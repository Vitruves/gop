package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
)

// DuplicationMetrics contains metrics about code duplication
type DuplicationMetrics struct {
	Timestamp       time.Time `json:"timestamp"`
	TotalFiles      int       `json:"total_files"`
	TotalBlocks     int       `json:"total_blocks"`
	DuplicatePairs  int       `json:"duplicate_pairs"`
	DuplicationRate float64   `json:"duplication_rate"`
	Comment         string    `json:"comment,omitempty"`
	Directory       string    `json:"directory"`
	Threshold       float64   `json:"threshold"`
	MinLineCount    int       `json:"min_line_count"`
}

// MonitoringHistory contains the history of duplication metrics
type MonitoringHistory struct {
	Metrics []DuplicationMetrics `json:"metrics"`
}

// SaveDuplicationMetrics saves duplication metrics to a monitoring file
func SaveDuplicationMetrics(metrics DuplicationMetrics, monitorFile string) error {
	// Create or load existing history
	var history MonitoringHistory
	
	// Check if file exists
	if _, err := os.Stat(monitorFile); err == nil {
		// File exists, read it
		data, err := os.ReadFile(monitorFile)
		if err != nil {
			return fmt.Errorf("error reading monitoring file: %v", err)
		}
		
		// Parse JSON
		if err := json.Unmarshal(data, &history); err != nil {
			// If file is corrupted, start with empty history
			history = MonitoringHistory{
				Metrics: []DuplicationMetrics{},
			}
		}
	} else {
		// File doesn't exist, create new history
		history = MonitoringHistory{
			Metrics: []DuplicationMetrics{},
		}
	}
	
	// Add new metrics
	history.Metrics = append(history.Metrics, metrics)
	
	// Write to file
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling monitoring data: %v", err)
	}
	
	if err := os.WriteFile(monitorFile, data, 0644); err != nil {
		return fmt.Errorf("error writing monitoring file: %v", err)
	}
	
	return nil
}

// PrintDuplicationTrend prints the trend of duplication metrics
func PrintDuplicationTrend(monitorFile string) error {
	// Check if file exists
	if _, err := os.Stat(monitorFile); err != nil {
		return fmt.Errorf("monitoring file not found: %v", err)
	}
	
	// Read file
	data, err := os.ReadFile(monitorFile)
	if err != nil {
		return fmt.Errorf("error reading monitoring file: %v", err)
	}
	
	// Parse JSON
	var history MonitoringHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return fmt.Errorf("error parsing monitoring file: %v", err)
	}
	
	// Need at least 2 data points for a trend
	if len(history.Metrics) < 2 {
		fmt.Println(color.YellowString("Not enough data points to show a trend. Run with --monitor flag multiple times."))
		return nil
	}
	
	// Print trend
	fmt.Println(color.CyanString("\nDuplication Trend:"))
	fmt.Println("┌────────────────────┬───────────┬────────────┬─────────────────┐")
	fmt.Println("│ Date               │ Files     │ Duplicates │ Duplication Rate │")
	fmt.Println("├────────────────────┼───────────┼────────────┼─────────────────┤")
	
	// Get the last 5 entries or all if less than 5
	startIdx := 0
	if len(history.Metrics) > 5 {
		startIdx = len(history.Metrics) - 5
	}
	
	for i := startIdx; i < len(history.Metrics); i++ {
		m := history.Metrics[i]
		dateStr := m.Timestamp.Format("2006-01-02 15:04")
		
		// Determine color based on duplication rate trend
		var rateColor *color.Color
		if i > startIdx {
			prevRate := history.Metrics[i-1].DuplicationRate
			if m.DuplicationRate > prevRate {
				rateColor = color.New(color.FgRed)
			} else if m.DuplicationRate < prevRate {
				rateColor = color.New(color.FgGreen)
			} else {
				rateColor = color.New(color.FgYellow)
			}
		} else {
			rateColor = color.New(color.FgWhite)
		}
		
		fmt.Printf("│ %-18s │ %-9d │ %-10d │ %s │\n",
			dateStr,
			m.TotalFiles,
			m.DuplicatePairs,
			rateColor.Sprintf("%14.2f%%", m.DuplicationRate*100),
		)
	}
	
	fmt.Println("└────────────────────┴───────────┴────────────┴─────────────────┘")
	
	// Calculate overall trend
	first := history.Metrics[0]
	last := history.Metrics[len(history.Metrics)-1]
	
	if last.DuplicationRate > first.DuplicationRate {
		fmt.Printf("Overall trend: %s (%.2f%% → %.2f%%)\n",
			color.RedString("↑ Increasing"),
			first.DuplicationRate*100,
			last.DuplicationRate*100,
		)
	} else if last.DuplicationRate < first.DuplicationRate {
		fmt.Printf("Overall trend: %s (%.2f%% → %.2f%%)\n",
			color.GreenString("↓ Decreasing"),
			first.DuplicationRate*100,
			last.DuplicationRate*100,
		)
	} else {
		fmt.Printf("Overall trend: %s (%.2f%%)\n",
			color.YellowString("→ Stable"),
			last.DuplicationRate*100,
		)
	}
	
	return nil
}
