package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
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

func printStaleAwaitingSummary(staleByProject map[int][]StaleAwaitingViolation, staleDays int) {
	total := 0
	for _, items := range staleByProject {
		total += len(items)
	}

	fmt.Printf("\n⏳ Awaiting QA stale watchdog (threshold: %d days)\n", staleDays)
	fmt.Printf("Found %d stale items in %q.\n\n", total, awaitingQAColumn)

	projects := make([]int, 0, len(staleByProject))
	for projectNum := range staleByProject {
		projects = append(projects, projectNum)
	}
	sort.Ints(projects)

	for _, projectNum := range projects {
		items := staleByProject[projectNum]
		if len(items) == 0 {
			continue
		}
		fmt.Printf("🗂️ Project %d has %d stale Awaiting QA items:\n\n", projectNum, len(items))
		for _, v := range items {
			it := v.Item
			fmt.Printf(
				"⌛ #%d – %s\n   %s\n   Last updated: %s (%d days ago)\n\n",
				getNumber(it),
				getTitle(it),
				getURL(it),
				v.LastUpdated.Format("2006-01-02"),
				v.StaleDays,
			)
		}
	}
}

func printTimestampCheckSummary(result TimestampCheckResult) {
	fmt.Printf("\n🕒 Updates timestamp check (%s)\n", result.URL)
	if result.Error != "" {
		fmt.Printf("⚠️ Could not validate timestamp expiry: %s\n\n", result.Error)
		return
	}

	daysLeft := int(result.DurationLeft.Hours() / 24)
	expires := result.ExpiresAt.Format(time.RFC3339)
	if result.OK {
		fmt.Printf(
			"✅ expires at %s (%d days left, minimum %d days)\n\n",
			expires,
			daysLeft,
			result.MinDays,
		)
		return
	}

	fmt.Printf(
		"❌ expires at %s (%d days left, minimum %d days)\n\n",
		expires,
		daysLeft,
		result.MinDays,
	)
}

func printMissingMilestoneSummary(items []MissingMilestoneIssue) {
	byProject := make(map[int][]MissingMilestoneIssue)
	for _, it := range items {
		byProject[it.ProjectNum] = append(byProject[it.ProjectNum], it)
	}

	fmt.Printf("\n🎯 Missing milestone audit (selected projects)\n")
	fmt.Printf("Found %d issue(s) without a milestone.\n\n", len(items))

	projects := make([]int, 0, len(byProject))
	for p := range byProject {
		projects = append(projects, p)
	}
	sort.Ints(projects)

	for _, projectNum := range projects {
		fmt.Printf("🗂️ Project %d:\n\n", projectNum)
		for _, v := range byProject[projectNum] {
			it := v.Item
			fmt.Printf("❗ #%d – %s\n   %s\n", getNumber(it), getTitle(it), getURL(it))
			if len(v.SuggestedMilestones) == 0 {
				fmt.Printf("   Suggested milestones: (none found)\n\n")
				continue
			}
			fmt.Printf("   Suggested milestones:\n")
			for _, m := range v.SuggestedMilestones {
				fmt.Printf("   - %s\n", m.Title)
			}
			fmt.Println()
		}
	}
}
