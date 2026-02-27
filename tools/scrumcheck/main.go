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
	phaseReleaseStoryTODO = iota
	phaseGenericQueries
	phaseMissingSprint
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

var (
	startUIBridgeFn = startUIBridge
	openInBrowserFn = openInBrowser
)

// main is the CLI entrypoint: it parses flags, runs all checks, prepares report
// data, starts the local bridge server, and opens the browser report view.
func main() {
	os.Exit(run())
}

// run executes the full scrumcheck flow and returns the process exit code.
func run() int {
	org := flag.String("org", "fleetdm", "GitHub org")
	limit := flag.Int("limit", 100, "Max project items to scan (no pagination; expected usage is small)")
	staleDays := flag.Int("stale-days", defaultStaleDays, "Flag Awaiting QA items unchanged for this many days")
	bridgeIdleMinutes := flag.Int("bridge-idle-minutes", defaultBridgeIdleMinutes, "Minutes to keep UI bridge alive without activity")
	openReport := flag.Bool("open-report", true, "Open HTML report in browser when finished")
	uiDevDir := flag.String("ui-dev-dir", "", "Serve frontend files from a local dev directory (expects index.html and assets/)")
	var projectNums intListFlag
	var labels stringListFlag
	flag.Var(&projectNums, "project", "Project number(s)")
	flag.Var(&projectNums, "p", "Project number(s) shorthand")
	flag.Var(&labels, "label", "Label filter(s); items must match at least one (supports values with or without leading #)")
	flag.Var(&labels, "l", "Label filter(s) shorthand")
	flag.Parse()

	for _, arg := range flag.Args() {
		// Support bare positional values as convenience input:
		// numeric values are treated as project numbers, everything else as labels.
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
		return 2
	}
	if *staleDays < 1 {
		fmt.Fprintln(os.Stderr, "-stale-days must be >= 1")
		flag.Usage()
		return 2
	}
	if *bridgeIdleMinutes < 1 {
		fmt.Fprintln(os.Stderr, "-bridge-idle-minutes must be >= 1")
		flag.Usage()
		return 2
	}
	if err := setUIRuntimeDir(*uiDevDir); err != nil {
		fmt.Fprintf(os.Stderr, "invalid -ui-dev-dir: %v\n", err)
		flag.Usage()
		return 2
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Printf("GITHUB_TOKEN env var is required")
		return 1
	}

	ctx := context.Background()
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := githubv4.NewClient(oauth2.NewClient(ctx, src))
	tracker := newPhaseTracker([]string{
		"Release stories with TODO",
		"Generic queries scan",
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

	tracker.phaseStart(phaseReleaseStoryTODO)
	start := time.Now()
	releaseStoryTODO := runReleaseStoryTODOChecks(ctx, client, *org, projectNums, *limit, token, labelFilter)
	tracker.phaseDone(phaseReleaseStoryTODO, phaseSummaryKV(
		fmt.Sprintf("issues=%d", len(releaseStoryTODO)),
		shortDuration(time.Since(start)),
	))

	tracker.phaseStart(phaseGenericQueries)
	start = time.Now()
	// Generic query checks are token-authenticated GitHub issue searches with
	// placeholder expansion (<<group>> / <<project>>) and independent reporting.
	genericQueries := runGenericQueryChecks(ctx, token, projectNums, groupLabels)
	tracker.phaseDone(phaseGenericQueries, phaseSummaryKV(
		fmt.Sprintf("queries=%d", len(genericQueries)),
		fmt.Sprintf("issues=%d", countGenericQueryIssues(genericQueries)),
		shortDuration(time.Since(start)),
	))

	tracker.phaseStart(phaseAwaitingQAGate)
	tracker.phaseStart(phaseAwaitingQAStale)
	start = time.Now()
	staleAfter := time.Duration(*staleDays) * 24 * time.Hour
	// Awaiting-QA and stale checks share one scan pass; we split metrics into
	// separate phases for visibility in the tracker and report summary.
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
	// One query path produces both "missing assignee" and "assigned to me"
	// sections; splitAssigneeCounts separates totals for phase reporting.
	missingAssignees := runMissingAssigneeChecks(ctx, client, *org, projectNums, *limit, token)
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
	releaseLabelIssues := runReleaseLabelChecks(ctx, client, *org, projectNums, *limit)
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
	bridge, err := startUIBridgeFn(token, time.Duration(*bridgeIdleMinutes)*time.Minute, tracker.bridgeSignal, policy)
	if err != nil {
		log.Printf("could not start UI bridge: %v", err)
		tracker.phaseFail(phaseUIAssembly, phaseSummaryKV("bridge unavailable", shortDuration(time.Since(start))))
		return 1
	}
	bridgeEnabled, bridgeBaseURL := false, ""
	bridgeSessionToken := ""
	bridgeEnabled = true
	bridgeBaseURL = bridge.baseURL
	bridgeSessionToken = bridge.sessionToken()
	bridge.setTimestampCheckResult(timestampCheck)
	reportData := buildHTMLReportData(
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
		releaseStoryTODO,
		genericQueries,
		unassignedUnreleasedBugs,
		groupLabels,
		timestampCheck,
		bridgeEnabled,
		bridgeBaseURL,
		bridgeSessionToken,
	)
	bridge.setReportData(reportData)
	bridge.setUnassignedUnreleasedResults(reportData.UnassignedUnreleased)
	bridge.setReleaseStoryTODOResults(reportData.ReleaseStoryTODO)
	bridge.setMissingSprintResults(reportData.MissingSprint)
	// These refresh callbacks let individual UI sections re-query data on demand
	// without restarting the process or rebuilding the bridge.
	bridge.setTimestampRefresher(func(ctx context.Context) (TimestampCheckResult, error) {
		refreshCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		return checkUpdatesTimestamp(refreshCtx, time.Now().UTC()), nil
	})
	bridge.setUnreleasedRefresher(func(ctx context.Context) ([]UnassignedUnreleasedProjectReport, error) {
		refreshCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		// Refresh computes only the target check, then reuses the report builder
		// to keep section formatting identical to the full run.
		fresh := runUnassignedUnreleasedBugChecks(refreshCtx, client, *org, projectNums, *limit, token, labelFilter, groupLabels)
		return buildHTMLReportData(
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
			releaseStoryTODO,
			genericQueries,
			fresh,
			groupLabels,
			timestampCheck,
			bridgeEnabled,
			bridgeBaseURL,
			bridgeSessionToken,
		).UnassignedUnreleased, nil
	})
	bridge.setReleaseStoryTODORefresher(func(ctx context.Context) ([]ReleaseStoryTODOProjectReport, error) {
		refreshCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		fresh := runReleaseStoryTODOChecks(refreshCtx, client, *org, projectNums, *limit, token, labelFilter)
		return buildHTMLReportData(
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
			fresh,
			genericQueries,
			unassignedUnreleasedBugs,
			groupLabels,
			timestampCheck,
			bridgeEnabled,
			bridgeBaseURL,
			bridgeSessionToken,
		).ReleaseStoryTODO, nil
	})
	bridge.setMissingSprintRefresher(func(ctx context.Context) ([]MissingSprintProjectReport, map[string]sprintApplyTarget, error) {
		refreshCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		fresh := runMissingSprintChecks(refreshCtx, client, *org, projectNums, *limit, labelFilter)
		report := buildHTMLReportData(
			*org,
			projectNums,
			awaitingByProject,
			staleByProject,
			*staleDays,
			byStatus,
			missingMilestones,
			fresh,
			missingAssignees,
			releaseLabelIssues,
			releaseStoryTODO,
			genericQueries,
			unassignedUnreleasedBugs,
			groupLabels,
			timestampCheck,
			bridgeEnabled,
			bridgeBaseURL,
			bridgeSessionToken,
		).MissingSprint
		// Sprint apply allowlist must be rebuilt from fresh findings so UI actions
		// cannot target stale item IDs.
		refreshedPolicy := buildBridgePolicy(nil, nil, fresh, nil, nil)
		return report, refreshedPolicy.SprintsByItemID, nil
	})
	bridge.setRefreshAllState(func(ctx context.Context) (HTMLReportData, bridgePolicy, error) {
		refreshCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		defer cancel()

		// Recompute every check from current GitHub state so the UI can get a
		// complete synchronized snapshot when a full refresh is requested.
		staleAfter := time.Duration(*staleDays) * 24 * time.Hour
		refAwaitingByProject, refStaleByProject := runAwaitingQACheck(
			refreshCtx,
			client,
			*org,
			*limit,
			projectNums,
			staleAfter,
			labelFilter,
		)
		refDrafting := runDraftingCheck(refreshCtx, client, *org, *limit, labelFilter)
		refByStatus := groupViolationsByStatus(refDrafting)
		refMissingMilestones := runMissingMilestoneChecks(refreshCtx, client, *org, projectNums, *limit, token, labelFilter)
		refMissingSprints := runMissingSprintChecks(refreshCtx, client, *org, projectNums, *limit, labelFilter)
		refMissingAssignees := runMissingAssigneeChecks(refreshCtx, client, *org, projectNums, *limit, token)
		refReleaseIssues := runReleaseLabelChecks(refreshCtx, client, *org, projectNums, *limit)
		refReleaseTODO := runReleaseStoryTODOChecks(refreshCtx, client, *org, projectNums, *limit, token, labelFilter)
		refGenericQueries := runGenericQueryChecks(refreshCtx, token, projectNums, groupLabels)
		refUnreleased := runUnassignedUnreleasedBugChecks(refreshCtx, client, *org, projectNums, *limit, token, labelFilter, groupLabels)
		refTimestamp := checkUpdatesTimestamp(refreshCtx, time.Now().UTC())

		refData := buildHTMLReportData(
			*org,
			projectNums,
			refAwaitingByProject,
			refStaleByProject,
			*staleDays,
			refByStatus,
			refMissingMilestones,
			refMissingSprints,
			refMissingAssignees,
			refReleaseIssues,
			refReleaseTODO,
			refGenericQueries,
			refUnreleased,
			groupLabels,
			refTimestamp,
			bridgeEnabled,
			bridgeBaseURL,
			bridgeSessionToken,
		)
		refPolicy := buildBridgePolicy(refDrafting, refMissingMilestones, refMissingSprints, refMissingAssignees, refReleaseIssues)
		return refData, refPolicy, nil
	})

	tracker.phaseDone(phaseUIAssembly, phaseSummaryKV("report + bridge ready", shortDuration(time.Since(start))))

	tracker.phaseStart(phaseBrowserBridge)
	openTarget := bridge.reportURL()
	tracker.waitingForBrowser(openTarget)
	if *openReport {
		if err := openInBrowserFn(openTarget); err != nil {
			log.Printf("could not auto-open report: %v", err)
			tracker.phaseWarn(phaseBrowserBridge, "browser auto-open failed")
			return 0
		}
		tracker.phaseDone(phaseBrowserBridge, "browser open signal sent")
	} else {
		// Keep bridge live even when auto-open is disabled so callers can open
		// the URL manually and still use interactive actions.
		tracker.phaseWarn(phaseBrowserBridge, "auto-open disabled (-open-report=false)")
	}

	if bridge == nil {
		return 0
	}

	sigCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stopSignals()
	tracker.bridgeListening(bridge.baseURL, time.Duration(*bridgeIdleMinutes)*time.Minute)
	reason := bridge.waitUntilDone(sigCtx)
	tracker.bridgeStopped(reason)
	return 0
}

// runAwaitingQACheck evaluates selected projects for two outcomes:
// checklist violations in Awaiting QA, and stale Awaiting QA items.
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
			// Filtering and status guards are applied first so later checks only run
			// on in-scope Awaiting QA items.
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

// splitAssigneeCounts separates one combined assignee result list into
// missing-assignee and assigned-to-me totals.
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

// runDraftingCheck scans the drafting project for estimation statuses that
// still contain unchecked checklist entries.
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

// printDraftingSummary prints drafting violations grouped by status.
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
