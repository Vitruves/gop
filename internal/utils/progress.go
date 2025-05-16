package utils

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Global mutex to ensure only one progress indicator renders at a time
var progressMutex sync.Mutex
var activeProgress *ProgressIndicator

// ProgressIndicator represents a simple progress indicator that prints dots
type ProgressIndicator struct {
	total       int64
	current     int64
	description string
	startTime   time.Time
	finished    bool
	mutex       sync.Mutex
}

// NewProgressBar creates a new progress indicator
// For backward compatibility, we keep the same function name
func NewProgressBar(total int, description string) *ProgressIndicator {
	return &ProgressIndicator{
		total:       int64(total),
		current:     0,
		description: description,
		startTime:   time.Now(),
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

	// Print the description followed by a colon and a space
	fmt.Printf("%s: ", p.description)
}

// Increment increments the progress indicator
func (p *ProgressIndicator) Increment() {
	atomic.AddInt64(&p.current, 1)

	// Print a visible character for each processed item
	// Use a more visible character than a dot
	fmt.Print(".")
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

	// Print a space to separate from the next output (but no newline)
	fmt.Print(" ")
}
