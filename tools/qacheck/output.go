package main

import (
	"fmt"
	"strings"
)

func groupViolationsByStatus(items []DraftingCheckViolation) map[string][]DraftingCheckViolation {
	out := make(map[string][]DraftingCheckViolation)
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item.Status))
		out[key] = append(out[key], item)
	}
	return out
}

func printDraftingStatusSection(status string, items []DraftingCheckViolation) {
	if len(items) == 0 {
		return
	}

	emoji := "📝"
	msg := fmt.Sprintf("These items are in %q but still have checklist items not checked.", status)
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ready to estimate":
		emoji = "🧩"
		msg = `These items are in "Ready to estimate" but still have checklist items not checked.`
	case "estimated":
		emoji = "📏"
		msg = `These items are in "Estimated" but still have checklist items not checked.`
	}

	fmt.Printf("%s %s\n\n", emoji, msg)
	for _, v := range items {
		it := v.Item
		fmt.Printf("❌ #%d – %s\n   %s\n", getNumber(it), getTitle(it), getURL(it))
		for _, line := range v.Unchecked {
			fmt.Printf("   - [ ] %s\n", line)
		}
		fmt.Println()
	}
}
