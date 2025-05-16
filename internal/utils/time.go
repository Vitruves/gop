package utils

import (
	"fmt"
	"time"
)

// FormatDuration formats a duration in a human-readable format
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.2f ms", float64(d.Milliseconds()))
	} else if d < time.Minute {
		return fmt.Sprintf("%.2f sec", d.Seconds())
	} else if d < time.Hour {
		minutes := d.Minutes()
		seconds := d.Seconds() - float64(int(minutes))*60
		return fmt.Sprintf("%.0f min %.0f sec", minutes, seconds)
	} else {
		hours := d.Hours()
		minutes := d.Minutes() - float64(int(hours))*60
		return fmt.Sprintf("%.0f hr %.0f min", hours, minutes)
	}
}
