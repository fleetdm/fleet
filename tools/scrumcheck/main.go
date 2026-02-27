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
	// Core CLI flags controlling org scope, scan depth, and UI behavior.
	org := flag.String("org", "fleetdm", "GitHub org")
	limit := flag.Int("limit", 100, "Max project items to scan (no pagination; expected usage is small)")
	staleDays := flag.Int("stale-days", defaultStaleDays, "Flag Awaiting QA items unchanged for this many days")
	bridgeIdleMinutes := flag.Int("bridge-idle-minutes", defaultBridgeIdleMinutes, "Minutes to keep UI bridge alive without activity")
	openReport := flag.Bool("open-report", true, "Open HTML report in browser when finished")
	uiDevDir := flag.String("ui-dev-dir", "", "Serve frontend files from a local dev directory (expects index.html and assets/)")
	// Repeated flags are collected into custom list types.
	var projectNums intListFlag
	var labels stringListFlag
	flag.Var(&projectNums, "project", "Project number(s)")
	flag.Var(&projectNums, "p", "Project number(s) shorthand")
	flag.Var(&labels, "label", "Label filter(s); items must match at least one (supports values with or without leading #)")
	flag.Var(&labels, "l", "Label filter(s) shorthand")
	// Parse known flags first; any leftovers are handled below for convenience.
	flag.Parse()

	for _, arg := range flag.Args() {
		// Support bare positional values as convenience input:
		// numeric values are treated as project numbers, everything else as labels.
		arg = strings.TrimSpace(arg)
		if strings.HasPrefix(arg, "-") {
			continue
		}
		// Try positional project number first.
		n, err := strconv.Atoi(arg)
		if err == nil {
			projectNums = append(projectNums, n)
			continue
		}
		// Non-numeric positional args are treated as labels.
		labels = append(labels, arg)
	}

	// Validate runtime inputs early and return usage errors with exit code 2 so
	// callers can distinguish configuration failures from check failures.
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
	// Validate dev UI dir (if provided) before any network work starts.
	if err := setUIRuntimeDir(*uiDevDir); err != nil {
		fmt.Fprintf(os.Stderr, "invalid -ui-dev-dir: %v\n", err)
		flag.Usage()
		return 2
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		// Missing token is a runtime failure (exit code 1), not usage error.
		log.Printf("GITHUB_TOKEN env var is required")
		return 1
	}

	// Build shared GitHub GraphQL client once and reuse for all checks.
	ctx := context.Background()
	// OAuth2 static token source feeds GraphQL and REST calls consistently.
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := githubv4.NewClient(oauth2.NewClient(ctx, src))
	// Phase tracker drives the terminal "flight console" progress UI.
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

	// Normalize operator inputs once for all downstream checks.
	projectNums = uniqueInts(projectNums)
	labelFilter := compileLabelFilter(labels)
	groupLabels := orderedGroupLabels(labels)
	// Check phases run in a fixed order so tracker output is deterministic and
	// easy to compare between runs.

	tracker.phaseStart(phaseReleaseStoryTODO)
	// Phase 1: release-story TODO audit.
	start := time.Now()
	releaseStoryTODO := runReleaseStoryTODOChecks(ctx, client, *org, projectNums, *limit, token, labelFilter)
	tracker.phaseDone(phaseReleaseStoryTODO, phaseSummaryKV(
		fmt.Sprintf("issues=%d", len(releaseStoryTODO)),
		shortDuration(time.Since(start)),
	))

	tracker.phaseStart(phaseGenericQueries)
	// Phase 2: configured generic query templates (may expand into many queries).
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
	// Phases 6+7 share one scan but publish separate summaries.
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
	// Drafting gate checks project 67 estimation columns for unchecked checklist.
	start = time.Now()
	badDrafting := runDraftingCheck(ctx, client, *org, *limit, labelFilter)
	byStatus := groupViolationsByStatus(badDrafting)
	tracker.phaseDone(phaseDraftingGate, phaseSummaryKV(
		fmt.Sprintf("drafting violations=%d", len(badDrafting)),
		shortDuration(time.Since(start)),
	))

	tracker.phaseStart(phaseMissingMilestones)
	// Missing milestone scan also loads suggestion candidates per repository.
	start = time.Now()
	missingMilestones := runMissingMilestoneChecks(ctx, client, *org, projectNums, *limit, token, labelFilter)
	tracker.phaseDone(phaseMissingMilestones, phaseSummaryKV(
		fmt.Sprintf("issues=%d", len(missingMilestones)),
		shortDuration(time.Since(start)),
	))

	tracker.phaseStart(phaseMissingSprint)
	// Missing sprint scan skips done items and "ready for release" grouping.
	start = time.Now()
	missingSprints := runMissingSprintChecks(ctx, client, *org, projectNums, *limit, labelFilter)
	tracker.phaseDone(phaseMissingSprint, phaseSummaryKV(
		fmt.Sprintf("issues=%d", len(missingSprints)),
		shortDuration(time.Since(start)),
	))

	tracker.phaseStart(phaseMissingAssignee)
	tracker.phaseStart(phaseAssignedToMe)
	// Assignee scan powers two panels (missing + assigned-to-me).
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
	// Release label guard enforces :product/:release policy in selected projects.
	start = time.Now()
	releaseLabelIssues := runReleaseLabelChecks(ctx, client, *org, projectNums, *limit)
	tracker.phaseDone(phaseReleaseLabel, phaseSummaryKV(
		fmt.Sprintf("issues=%d", len(releaseLabelIssues)),
		shortDuration(time.Since(start)),
	))

	tracker.phaseStart(phaseUnassignedUnreleased)
	// Unreleased bug scan is grouped by provided -l labels.
	start = time.Now()
	unassignedUnreleasedBugs := runUnassignedUnreleasedBugChecks(ctx, client, *org, projectNums, *limit, token, labelFilter, groupLabels)
	tracker.phaseDone(phaseUnassignedUnreleased, phaseSummaryKV(
		fmt.Sprintf("issues=%d", len(unassignedUnreleasedBugs)),
		shortDuration(time.Since(start)),
	))

	tracker.phaseStart(phaseTimestampExpiry)
	// Timestamp check validates update metadata freshness window.
	start = time.Now()
	timestampCheck := checkUpdatesTimestamp(ctx, time.Now().UTC())
	if timestampCheck.Error != "" {
		// Unavailable check is warning-only so other checks still complete.
		tracker.phaseWarn(phaseTimestampExpiry, phaseSummaryKV("check unavailable", timestampCheck.Error, shortDuration(time.Since(start))))
	} else if !timestampCheck.OK {
		// Expiry below threshold is a failing condition.
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
	// Build mutation allowlists from findings before exposing bridge actions.
	start = time.Now()
	policy := buildBridgePolicy(badDrafting, missingMilestones, missingSprints, missingAssignees, releaseLabelIssues)
	// Start local loopback bridge used by browser UI and action endpoints.
	bridge, err := startUIBridgeFn(token, time.Duration(*bridgeIdleMinutes)*time.Minute, tracker.bridgeSignal, policy)
	if err != nil {
		log.Printf("could not start UI bridge: %v", err)
		tracker.phaseFail(phaseUIAssembly, phaseSummaryKV("bridge unavailable", shortDuration(time.Since(start))))
		return 1
	}
	bridgeEnabled, bridgeBaseURL := false, ""
	bridgeSessionToken := ""
	// Bridge metadata is injected into report model for frontend API calls.
	bridgeEnabled = true
	bridgeBaseURL = bridge.baseURL
	bridgeSessionToken = bridge.sessionToken()
	// Seed bridge cache with current run snapshot before serving UI.
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
	// Prime bridge handlers with full report and per-check slices.
	bridge.setReportData(reportData)
	bridge.setUnassignedUnreleasedResults(reportData.UnassignedUnreleased)
	bridge.setReleaseStoryTODOResults(reportData.ReleaseStoryTODO)
	bridge.setMissingSprintResults(reportData.MissingSprint)
	// These refresh callbacks let individual UI sections re-query data on demand
	// without restarting the process or rebuilding the bridge.
	bridge.setTimestampRefresher(func(ctx context.Context) (TimestampCheckResult, error) {
		// Lightweight refresh path: only timestamp endpoint.
		refreshCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		return checkUpdatesTimestamp(refreshCtx, time.Now().UTC()), nil
	})
	bridge.setUnreleasedRefresher(func(ctx context.Context) ([]UnassignedUnreleasedProjectReport, error) {
		// Targeted refresh path for unreleased-bug section.
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
		// Targeted refresh path for release-story TODO section.
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
		// Targeted refresh path for missing sprint section + sprint allowlist.
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
		// Full refresh endpoint recomputes every check with a larger timeout.
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

		// Rebuild complete report model and bridge action policy from fresh data.
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

	// UI/report assembly phase is complete once bridge cache + refreshers are set.
	tracker.phaseDone(phaseUIAssembly, phaseSummaryKV("report + bridge ready", shortDuration(time.Since(start))))

	tracker.phaseStart(phaseBrowserBridge)
	// Bridge serves app shell at root.
	openTarget := bridge.reportURL()
	tracker.waitingForBrowser(openTarget)
	if *openReport {
		// Best-effort browser open; failure should not fail scan results.
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
		// Defensive guard; normal flow always has a bridge here.
		return 0
	}

	// Keep process alive while bridge is active so UI actions can mutate state;
	// exit cleanly when interrupted or bridge closes itself.
	sigCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stopSignals()
	// Announce active bridge URL and idle timeout in tracker footer.
	tracker.bridgeListening(bridge.baseURL, time.Duration(*bridgeIdleMinutes)*time.Minute)
	// Block until Ctrl+C or bridge self-shutdown.
	reason := bridge.waitUntilDone(sigCtx)
	// Print final bridge stop reason before exiting.
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
		// Resolve project and fetch current window of items for that project.
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
			// Gate violation: awaiting QA item still includes required unchecked line.
			if hasUncheckedChecklistLine(getBody(it), checkText) {
				badAwaitingQA = append(badAwaitingQA, it)
			}
			// Stale violation: awaiting QA item has not been updated within threshold.
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
		// Always populate both maps so report rendering can show empty states.
		awaitingByProject[projectNum] = badAwaitingQA
		staleByProject[projectNum] = staleAwaiting
	}
	return awaitingByProject, staleByProject
}

// splitAssigneeCounts separates one combined assignee result list into
// missing-assignee and assigned-to-me totals.
func splitAssigneeCounts(items []MissingAssigneeIssue) (missingAssignee int, assignedToMe int) {
	for _, it := range items {
		// Assigned-to-me issues are informational and tracked separately.
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
		// Any remaining unchecked drafting checklist item is a violation.
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
