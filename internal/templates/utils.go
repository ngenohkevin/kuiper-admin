package templates

import (
	"fmt"
	"time"
)

// Utility functions for time formatting
func formatTimeRemaining(t time.Time) string {
	duration := time.Until(t)
	if duration < 0 {
		return "expired"
	}

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60

	if hours > 24 {
		days := hours / 24
		return fmt.Sprintf("%d days", days)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		return fmt.Sprintf("%d minutes", minutes)
	}
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60

	if hours > 24 {
		days := hours / 24
		return fmt.Sprintf("%d days", days)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		return fmt.Sprintf("%d minutes", minutes)
	}
}

func formatDateTimeLocal(t time.Time) string {
	return t.Format("2006-01-02T15:04")
}
