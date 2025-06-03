package utils

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
)

// Global mutex to ensure only one progress indicator renders at a time
var progressMutex sync.Mutex
var activeProgress *ProgressIndicator

// ProgressIndicator represents a progress indicator with percentage and colored output
type ProgressIndicator struct {
	total         int64
	current       int64
	description   string
	startTime     time.Time
	finished      bool
	mutex         sync.Mutex
	terminalWidth int
	lastLine      string
}

// NewProgressBar creates a new progress indicator
// For backward compatibility, we keep the same function name
func NewProgressBar(total int, description string) *ProgressIndicator {
	// Use a fixed terminal width
	width := 100 // reasonable default width

	return &ProgressIndicator{
		total:         int64(total),
		current:       0,
		description:   description,
		startTime:     time.Now(),
		terminalWidth: width,
	}
}

// Start starts the progress indicator
func (p *ProgressIndicator) Start() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// If there's already an active progress indicator, finish it first
	progressMutex.Lock()
	if activeProgress != nil && activeProgress != p && !activeProgress.finished {
		oldIndicator := activeProgress
		progressMutex.Unlock()
		oldIndicator.Finish()
		progressMutex.Lock()
	}

	// Register this progress indicator as the active one
	activeProgress = p
	progressMutex.Unlock()

	p.startTime = time.Now()
	p.finished = false
	p.current = 0

	// Print initial message in cmake style
	infoColor := color.New(color.Reset) // No color for -i--
	fmt.Printf("%s %s", infoColor.Sprint("-i--"), p.description)
}

// Increment increments the progress indicator
func (p *ProgressIndicator) Increment() {
	newCurrent := atomic.AddInt64(&p.current, 1)

	// Calculate percentage
	percentage := float64(newCurrent) / float64(p.total) * 100.0

	// Create progress display with dots and percentage
	dotsCount := int(newCurrent)
	progressStr := ""

	// Add dots for progress, but limit to reasonable number
	maxDots := 50
	if dotsCount <= maxDots {
		progressStr = fmt.Sprintf(" %s %.2f%% (%d/%d)",
			generateDots(dotsCount),
			percentage,
			newCurrent,
			p.total)
	} else {
		// Show fewer dots but still track progress
		scaledDots := int(float64(dotsCount) / float64(p.total) * float64(maxDots))
		progressStr = fmt.Sprintf(" %s %.2f%% (%d/%d)",
			generateDots(scaledDots),
			percentage,
			newCurrent,
			p.total)
	}

	// Clear previous line and print new one (if not exceeding terminal width)
	if len(p.lastLine) > 0 {
		// Move cursor back and clear the line
		fmt.Printf("\r%s\r", color.New(color.Reset).Sprint(fmt.Sprintf("%*s", len(p.lastLine), "")))
	}

	// Construct full line
	fullLine := fmt.Sprintf("-i-- %s%s", p.description, progressStr)

	// Truncate if too long for terminal
	if len(fullLine) > p.terminalWidth-5 {
		fullLine = fullLine[:p.terminalWidth-8] + "..."
	}

	fmt.Printf("\r%s", fullLine)
	p.lastLine = fullLine
}

// Finish completes the progress indicator
func (p *ProgressIndicator) Finish() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Don't finish twice
	if p.finished {
		return
	}

	// Set current to total
	p.current = p.total
	p.finished = true

	// Deregister this progress indicator if it's the active one
	progressMutex.Lock()
	if activeProgress == p {
		activeProgress = nil
	}
	progressMutex.Unlock()

	// Clear current line and print success message
	if len(p.lastLine) > 0 {
		fmt.Printf("\r%s\r", fmt.Sprintf("%*s", len(p.lastLine), ""))
	}

	// Print success message in cmake style
	successColor := color.New(color.FgGreen)
	elapsed := time.Since(p.startTime)
	fmt.Printf("%s %s completed in %v\n",
		successColor.Sprint("-s--"),
		p.description,
		elapsed.Round(time.Millisecond))
}

// generateDots creates a string of dots for visual progress
func generateDots(count int) string {
	if count <= 0 {
		return ""
	}
	if count > 50 {
		count = 50
	}

	dots := make([]byte, count)
	for i := range dots {
		dots[i] = '.'
	}
	return string(dots)
}
