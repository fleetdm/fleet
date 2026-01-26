package messages

import "fmt"

// LoadingMessage returns a human-friendly loading message based on a simple key.
// Keys: "issues", "project", "estimated", "sprint", "milestone".
func LoadingMessage(key string, projectID int, hint string) string {
	switch key {
	case "issues":
		return "Fetching Issues..."
	case "project":
		return fmt.Sprintf("Fetching Project Items (ID: %d)...", projectID)
	case "estimated":
		return fmt.Sprintf("Fetching Estimated Tickets (Project: %d)...", projectID)
	case "sprint":
		return fmt.Sprintf("Fetching Sprint Items (Project: %d)...", projectID)
	case "milestone":
		return fmt.Sprintf("Fetching Milestone Issues (%s)...", hint)
	default:
		return "Fetching..."
	}
}

// LimitExceeded returns the common banner message about items not shown due to limit.
func LimitExceeded(missing, limit, total int) string {
	return fmt.Sprintf("⚠ %d items not shown (limit=%d, total=%d). Increase --limit to include all issues.", missing, limit, total)
}
