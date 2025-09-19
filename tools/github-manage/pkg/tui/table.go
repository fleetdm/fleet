package tui

import (
	"fmt"
	"strings"

	"fleetdm/gm/pkg/ghapi"
)

func truncTitle(title string) string {
	if len(title) > 50 {
		return title[:47] + "..."
	}
	return title
}

// prioritize labels 'story', 'bug', ':release', ':product', '#g-mdm', '#g-orchestration',
// '#g-software'
// show a maximum of 30 characters, if longer, truncate and add '...+n' where n is the number of additional labels
func truncLables(labels []ghapi.Label) string {
	if len(labels) == 0 {
		return fmt.Sprintf("%-35s", "- No Labels -")
	}
	priorityLabels := map[string]bool{"story": true, "bug": true, ":release": true, ":product": true, "#g-mdm": true, "#g-orchestration": true, "#g-software": true}
	var labelNames []string
	for _, label := range labels {
		labelNames = append(labelNames, label.Name)
	}
	displayLabels := []string{}
	secondaryLabels := []string{}
	for _, label := range labelNames {
		if priorityLabels[label] {
			displayLabels = append(displayLabels, label)
		} else {
			secondaryLabels = append(secondaryLabels, label)
		}
	}
	if len(secondaryLabels) > 0 {
		displayLabels = append(displayLabels, secondaryLabels...)
	}
	countDisplay := 0
	sumCharacters := 0
	for _, label := range displayLabels {
		if (sumCharacters + len(label)) > 30 {
			break
		}
		countDisplay += 1
		sumCharacters += len(label) + 2
	}
	extraLabels := 0
	if countDisplay < len(displayLabels) {
		extraLabels = len(displayLabels) - countDisplay
	}
	extraString := ""
	if extraLabels > 0 {
		extraString = fmt.Sprintf("...+%d", extraLabels)
	}
	return fmt.Sprintf("%-35s", fmt.Sprintf("%s%s", strings.Join(displayLabels[:countDisplay], ", "), extraString))
}

func truncEstimate(estimate int) string {
	estString := fmt.Sprintf("%d", estimate)
	if estimate == 0 {
		estString = "-"
	}
	return fmt.Sprintf("%-9s", estString)
}

func truncType(typename string) string {
	if typename == "" {
		typename = "-"
	}
	// Return the typename with spaces filling out to 10 characters
	if len(typename) < 10 {
		return fmt.Sprintf("%-10s", typename)
	}
	return strings.TrimSpace(typename[:10])
}
