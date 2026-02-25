package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

func main() {
	org := flag.String("org", "fleetdm", "GitHub org")
	limit := flag.Int("limit", 100, "Max project items to scan (no pagination; expected usage is small)")
	openReport := flag.Bool("open-report", true, "Open HTML report in browser when finished")
	var projectNums intListFlag
	flag.Var(&projectNums, "project", "Project number(s)")
	flag.Var(&projectNums, "p", "Project number(s) shorthand")
	flag.Parse()

	for _, arg := range flag.Args() {
		n, err := strconv.Atoi(arg)
		if err != nil {
			log.Fatalf("unexpected argument %q: only project numbers are allowed after -p", arg)
		}
		projectNums = append(projectNums, n)
	}

	if len(projectNums) == 0 {
		log.Fatal("at least one project is required")
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN env var is required")
	}

	ctx := context.Background()
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := githubv4.NewClient(oauth2.NewClient(ctx, src))

	projectNums = uniqueInts(projectNums)

	awaitingByProject := runAwaitingQACheck(ctx, client, *org, *limit, projectNums)
	badDrafting := runDraftingCheck(ctx, client, *org, *limit)
	byStatus := groupViolationsByStatus(badDrafting)
	printDraftingSummary(byStatus, len(badDrafting))

	reportPath, err := writeHTMLReport(buildHTMLReportData(*org, projectNums, awaitingByProject, byStatus))
	if err != nil {
		log.Printf("could not write HTML report: %v", err)
		return
	}

	reportURL := fileURLFromPath(reportPath)
	fmt.Printf("📄 HTML report: %s\n", reportPath)
	fmt.Printf("🔗 Open report: %s\n", reportURL)
	fmt.Printf("%s\n", reportURL)
	fmt.Printf("🔗 \x1b]8;;%s\x1b\\Click here to open the report\x1b]8;;\x1b\\\n", reportURL)
	if *openReport {
		if err := openInBrowser(reportPath); err != nil {
			log.Printf("could not auto-open report: %v", err)
			fmt.Printf("Run this manually: open %q\n", reportPath)
		}
	}
}

func runAwaitingQACheck(
	ctx context.Context,
	client *githubv4.Client,
	org string,
	limit int,
	projectNums []int,
) map[int][]Item {
	awaitingByProject := make(map[int][]Item)
	for _, projectNum := range projectNums {
		projectID := fetchProjectID(ctx, client, org, projectNum)
		items := fetchItems(ctx, client, projectID, limit)

		var badAwaitingQA []Item
		for _, it := range items {
			if !inAwaitingQA(it) {
				continue
			}
			if hasUncheckedChecklistLine(getBody(it), checkText) {
				badAwaitingQA = append(badAwaitingQA, it)
			}
		}
		awaitingByProject[projectNum] = badAwaitingQA

		fmt.Printf(
			"\nFound %d items in project %d (%q) with UNCHECKED test-plan confirmation:\n\n",
			len(badAwaitingQA),
			projectNum,
			awaitingQAColumn,
		)
		for _, it := range badAwaitingQA {
			fmt.Printf("❌ #%d – %s\n   %s\n\n", getNumber(it), getTitle(it), getURL(it))
		}
	}
	return awaitingByProject
}

func runDraftingCheck(
	ctx context.Context,
	client *githubv4.Client,
	org string,
	limit int,
) []DraftingCheckViolation {
	draftingProjectID := fetchProjectID(ctx, client, org, draftingProjectNum)
	draftingItems := fetchItems(ctx, client, draftingProjectID, limit)

	needles := strings.Split(draftingStatusNeedle, ",")
	var badDrafting []DraftingCheckViolation
	for _, it := range draftingItems {
		status, ok := matchedStatus(it, needles)
		if !ok {
			continue
		}
		unchecked := uncheckedChecklistItems(getBody(it))
		if len(unchecked) > 0 {
			badDrafting = append(badDrafting, DraftingCheckViolation{
				Item:      it,
				Unchecked: unchecked,
				Status:    status,
			})
		}
	}
	return badDrafting
}

func printDraftingSummary(byStatus map[string][]DraftingCheckViolation, total int) {
	fmt.Printf("\n🧭 Drafting checklist audit (project %d)\n", draftingProjectNum)
	fmt.Printf("Found %d items in estimation columns with unchecked checklist items.\n\n", total)

	printDraftingStatusSection("Ready to estimate", byStatus["ready to estimate"])
	printDraftingStatusSection("Estimated", byStatus["estimated"])

	for status, items := range byStatus {
		if status == "ready to estimate" || status == "estimated" {
			continue
		}
		printDraftingStatusSection(status, items)
	}
}
