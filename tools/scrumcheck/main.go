package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

const (
	phaseMissingSprint = iota
	phaseMissingMilestones
	phaseReleaseLabel
	phaseAwaitingQAStale
	phaseAwaitingQAGate
	phaseDraftingGate
	phaseMissingAssignee
	phaseAssignedToMe
	phaseUnassignedUnreleased
	phaseTimestampExpiry
	phaseUIAssembly
	phaseBrowserBridge
)

func main() {
	org := flag.String("org", "fleetdm", "GitHub org")
	limit := flag.Int("limit", 100, "Max project items to scan (no pagination; expected usage is small)")
	staleDays := flag.Int("stale-days", defaultStaleDays, "Flag Awaiting QA items unchanged for this many days")
	bridgeIdleMinutes := flag.Int("bridge-idle-minutes", defaultBridgeIdleMinutes, "Minutes to keep UI bridge alive without activity")
	openReport := flag.Bool("open-report", true, "Open HTML report in browser when finished")
	var projectNums intListFlag
	var labels stringListFlag
	flag.Var(&projectNums, "project", "Project number(s)")
	flag.Var(&projectNums, "p", "Project number(s) shorthand")
	flag.Var(&labels, "label", "Label filter(s); items must match at least one (supports values with or without leading #)")
	flag.Var(&labels, "l", "Label filter(s) shorthand")
	flag.Parse()

	for _, arg := range flag.Args() {
		arg = strings.TrimSpace(arg)
		if strings.HasPrefix(arg, "-") {
			continue
		}
		n, err := strconv.Atoi(arg)
		if err == nil {
			projectNums = append(projectNums, n)
			continue
		}
		labels = append(labels, arg)
	}

	if len(projectNums) == 0 {
		fmt.Fprintln(os.Stderr, "at least one project is required")
		flag.Usage()
		os.Exit(2)
	}
	if *staleDays < 1 {
		fmt.Fprintln(os.Stderr, "-stale-days must be >= 1")
		flag.Usage()
		os.Exit(2)
	}
	if *bridgeIdleMinutes < 1 {
		fmt.Fprintln(os.Stderr, "-bridge-idle-minutes must be >= 1")
		flag.Usage()
		os.Exit(2)
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN env var is required")
	}

	ctx := context.Background()
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := githubv4.NewClient(oauth2.NewClient(ctx, src))
	tracker := newPhaseTracker([]string{
		"Missing sprint check",
		"Missing milestones check",
		"Release label guard",
		"Awaiting QA stale watchdog",
		"Awaiting QA gate",
		"Drafting estimation gate",
		"Missing assignee",
		"Assigned to me",
		"Unassigned unreleased bugs",
		"Updates timestamp expiry",
		"Assembling UI and report",
		"Opening browser bridge",
	})

	projectNums = uniqueInts(projectNums)
	labelFilter := compileLabelFilter(labels)
	groupLabels := orderedGroupLabels(labels)

	tracker.phaseStart(phaseAwaitingQAGate)
	tracker.phaseStart(phaseAwaitingQAStale)
	start := time.Now()
	staleAfter := time.Duration(*staleDays) * 24 * time.Hour
	awaitingByProject, staleByProject := runAwaitingQACheck(ctx, client, *org, *limit, projectNums, staleAfter, labelFilter)
	awaitingElapsed := shortDuration(time.Since(start))
	tracker.phaseDone(phaseAwaitingQAGate, phaseSummaryKV(
		fmt.Sprintf("awaiting violations=%d", countAwaitingViolations(awaitingByProject)),
		awaitingElapsed,
	))
	tracker.phaseDone(phaseAwaitingQAStale, phaseSummaryKV(
		fmt.Sprintf("stale=%d", countStaleViolations(staleByProject)),
		awaitingElapsed,
	))

	tracker.phaseStart(phaseDraftingGate)
	start = time.Now()
	badDrafting := runDraftingCheck(ctx, client, *org, *limit, labelFilter)
	byStatus := groupViolationsByStatus(badDrafting)
	tracker.phaseDone(phaseDraftingGate, phaseSummaryKV(
		fmt.Sprintf("drafting violations=%d", len(badDrafting)),
		shortDuration(time.Since(start)),
	))

	tracker.phaseStart(phaseMissingMilestones)
	start = time.Now()
	missingMilestones := runMissingMilestoneChecks(ctx, client, *org, projectNums, *limit, token, labelFilter)
	tracker.phaseDone(phaseMissingMilestones, phaseSummaryKV(
		fmt.Sprintf("issues=%d", len(missingMilestones)),
		shortDuration(time.Since(start)),
	))

	tracker.phaseStart(phaseMissingSprint)
	start = time.Now()
	missingSprints := runMissingSprintChecks(ctx, client, *org, projectNums, *limit, labelFilter)
	tracker.phaseDone(phaseMissingSprint, phaseSummaryKV(
		fmt.Sprintf("issues=%d", len(missingSprints)),
		shortDuration(time.Since(start)),
	))

	tracker.phaseStart(phaseMissingAssignee)
	tracker.phaseStart(phaseAssignedToMe)
	start = time.Now()
	missingAssignees := runMissingAssigneeChecks(ctx, client, *org, projectNums, *limit, token, labelFilter)
	missingAssigneeCount, assignedToMeCount := splitAssigneeCounts(missingAssignees)
	assigneeElapsed := shortDuration(time.Since(start))
	tracker.phaseDone(phaseMissingAssignee, phaseSummaryKV(
		fmt.Sprintf("issues=%d", missingAssigneeCount),
		assigneeElapsed,
	))
	tracker.phaseDone(phaseAssignedToMe, phaseSummaryKV(
		fmt.Sprintf("issues=%d", assignedToMeCount),
		assigneeElapsed,
	))

	tracker.phaseStart(phaseReleaseLabel)
	start = time.Now()
	releaseLabelIssues := runReleaseLabelChecks(ctx, client, *org, projectNums, *limit, labelFilter)
	tracker.phaseDone(phaseReleaseLabel, phaseSummaryKV(
		fmt.Sprintf("issues=%d", len(releaseLabelIssues)),
		shortDuration(time.Since(start)),
	))

	tracker.phaseStart(phaseUnassignedUnreleased)
	start = time.Now()
	unassignedUnreleasedBugs := runUnassignedUnreleasedBugChecks(ctx, client, *org, projectNums, *limit, token, labelFilter, groupLabels)
	tracker.phaseDone(phaseUnassignedUnreleased, phaseSummaryKV(
		fmt.Sprintf("issues=%d", len(unassignedUnreleasedBugs)),
		shortDuration(time.Since(start)),
	))

	tracker.phaseStart(phaseTimestampExpiry)
	start = time.Now()
	timestampCheck := checkUpdatesTimestamp(ctx, time.Now().UTC())
	if timestampCheck.Error != "" {
		tracker.phaseWarn(phaseTimestampExpiry, phaseSummaryKV("check unavailable", timestampCheck.Error, shortDuration(time.Since(start))))
	} else if !timestampCheck.OK {
		daysLeft := int(timestampCheck.DurationLeft.Hours() / 24)
		tracker.phaseFail(phaseTimestampExpiry, phaseSummaryKV(
			fmt.Sprintf("expires in %d days (min %d)", daysLeft, timestampCheck.MinDays),
			shortDuration(time.Since(start)),
		))
	} else {
		daysLeft := int(timestampCheck.DurationLeft.Hours() / 24)
		tracker.phaseDone(phaseTimestampExpiry, phaseSummaryKV(
			fmt.Sprintf("expires in %d days", daysLeft),
			shortDuration(time.Since(start)),
		))
	}

	tracker.phaseStart(phaseUIAssembly)
	start = time.Now()
	policy := buildBridgePolicy(badDrafting, missingMilestones, missingSprints, missingAssignees, releaseLabelIssues)
	bridge, err := startUIBridge(token, time.Duration(*bridgeIdleMinutes)*time.Minute, tracker.bridgeSignal, policy)
	if err != nil {
		log.Printf("could not start UI bridge: %v", err)
	}
	bridgeEnabled, bridgeBaseURL := false, ""
	if bridge != nil {
		bridgeEnabled = true
		bridgeBaseURL = bridge.baseURL
	}
	reportPath, err := writeHTMLReport(
		buildHTMLReportData(
			*org,
			projectNums,
			awaitingByProject,
			staleByProject,
			*staleDays,
			byStatus,
			missingMilestones,
			missingSprints,
			missingAssignees,
			releaseLabelIssues,
			unassignedUnreleasedBugs,
			groupLabels,
			timestampCheck,
			bridgeEnabled,
			bridgeBaseURL,
		),
	)
	if err != nil {
		log.Printf("could not write HTML report: %v", err)
		return
	}
	tracker.phaseDone(phaseUIAssembly, phaseSummaryKV("report + bridge ready", shortDuration(time.Since(start))))

	tracker.phaseStart(phaseBrowserBridge)
	tracker.waitingForBrowser(reportPath)
	if *openReport {
		openTarget := reportPath
		if bridge != nil {
			bridge.setReportPath(reportPath)
			openTarget = bridge.reportURL()
		}
		if err := openInBrowser(openTarget); err != nil {
			log.Printf("could not auto-open report: %v", err)
			tracker.phaseWarn(phaseBrowserBridge, "browser auto-open failed")
			return
		}
		tracker.phaseDone(phaseBrowserBridge, "browser open signal sent")
	} else {
		tracker.phaseWarn(phaseBrowserBridge, "auto-open disabled (-open-report=false)")
	}

	if bridge == nil {
		return
	}

	sigCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stopSignals()
	tracker.bridgeListening(bridge.baseURL, time.Duration(*bridgeIdleMinutes)*time.Minute)
	reason := bridge.waitUntilDone(sigCtx)
	tracker.bridgeStopped(reason)
}

func runAwaitingQACheck(
	ctx context.Context,
	client *githubv4.Client,
	org string,
	limit int,
	projectNums []int,
	staleAfter time.Duration,
	labelFilter map[string]struct{},
) (map[int][]Item, map[int][]StaleAwaitingViolation) {
	awaitingByProject := make(map[int][]Item)
	staleByProject := make(map[int][]StaleAwaitingViolation)
	now := time.Now().UTC()
	for _, projectNum := range projectNums {
		projectID := fetchProjectID(ctx, client, org, projectNum)
		items := fetchItems(ctx, client, projectID, limit)

		var badAwaitingQA []Item
		var staleAwaiting []StaleAwaitingViolation
		for _, it := range items {
			if !matchesLabelFilter(it, labelFilter) {
				continue
			}
			if !inAwaitingQA(it) {
				continue
			}
			if hasUncheckedChecklistLine(getBody(it), checkText) {
				badAwaitingQA = append(badAwaitingQA, it)
			}
			if isStaleAwaitingQA(it, now, staleAfter) {
				lastUpdated := it.UpdatedAt.Time.UTC()
				staleAwaiting = append(staleAwaiting, StaleAwaitingViolation{
					Item:        it,
					StaleDays:   int(now.Sub(lastUpdated).Hours() / 24),
					LastUpdated: lastUpdated,
					ProjectNum:  projectNum,
				})
			}
		}
		awaitingByProject[projectNum] = badAwaitingQA
		staleByProject[projectNum] = staleAwaiting
	}
	return awaitingByProject, staleByProject
}

func splitAssigneeCounts(items []MissingAssigneeIssue) (missingAssignee int, assignedToMe int) {
	for _, it := range items {
		if it.AssignedToMe {
			assignedToMe++
			continue
		}
		missingAssignee++
	}
	return missingAssignee, assignedToMe
}

func runDraftingCheck(
	ctx context.Context,
	client *githubv4.Client,
	org string,
	limit int,
	labelFilter map[string]struct{},
) []DraftingCheckViolation {
	draftingProjectID := fetchProjectID(ctx, client, org, draftingProjectNum)
	draftingItems := fetchItems(ctx, client, draftingProjectID, limit)

	needles := strings.Split(draftingStatusNeedle, ",")
	var badDrafting []DraftingCheckViolation
	for _, it := range draftingItems {
		if !matchesLabelFilter(it, labelFilter) {
			continue
		}
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
