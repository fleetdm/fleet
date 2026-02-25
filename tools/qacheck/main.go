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
	phaseQAGate = iota
	phaseDrafting
	phaseMilestones
	phaseTimestamp
	phaseReport
	phaseBrowser
)

func main() {
	org := flag.String("org", "fleetdm", "GitHub org")
	limit := flag.Int("limit", 100, "Max project items to scan (no pagination; expected usage is small)")
	staleDays := flag.Int("stale-days", defaultStaleDays, "Flag Awaiting QA items unchanged for this many days")
	bridgeIdleMinutes := flag.Int("bridge-idle-minutes", defaultBridgeIdleMinutes, "Minutes to keep UI bridge alive without activity")
	openReport := flag.Bool("open-report", true, "Open HTML report in browser when finished")
	var projectNums intListFlag
	flag.Var(&projectNums, "project", "Project number(s)")
	flag.Var(&projectNums, "p", "Project number(s) shorthand")
	flag.Parse()

	for _, arg := range flag.Args() {
		n, err := strconv.Atoi(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unexpected argument %q: only project numbers are allowed after -p\n\n", arg)
			flag.Usage()
			os.Exit(2)
		}
		projectNums = append(projectNums, n)
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
		"QA checklist gate scan",
		"Drafting estimation gate scan",
		"Missing milestone audit",
		"Updates timestamp expiry check",
		"Rendering report deck",
		"Opening browser bridge",
	})

	projectNums = uniqueInts(projectNums)

	tracker.phaseStart(phaseQAGate)
	start := time.Now()
	staleAfter := time.Duration(*staleDays) * 24 * time.Hour
	awaitingByProject, staleByProject := runAwaitingQACheck(ctx, client, *org, *limit, projectNums, staleAfter)
	tracker.phaseDone(phaseQAGate,
		phaseSummaryKV(
			fmt.Sprintf("awaiting violations=%d", countAwaitingViolations(awaitingByProject)),
			fmt.Sprintf("stale=%d", countStaleViolations(staleByProject)),
			shortDuration(time.Since(start)),
		),
	)

	tracker.phaseStart(phaseDrafting)
	start = time.Now()
	badDrafting := runDraftingCheck(ctx, client, *org, *limit)
	byStatus := groupViolationsByStatus(badDrafting)
	tracker.phaseDone(phaseDrafting, phaseSummaryKV(
		fmt.Sprintf("drafting violations=%d", len(badDrafting)),
		shortDuration(time.Since(start)),
	))

	tracker.phaseStart(phaseMilestones)
	start = time.Now()
	missingMilestones := runMissingMilestoneChecks(ctx, client, *org, projectNums, *limit, token)
	tracker.phaseDone(phaseMilestones, phaseSummaryKV(
		fmt.Sprintf("missing milestones=%d", len(missingMilestones)),
		shortDuration(time.Since(start)),
	))

	tracker.phaseStart(phaseTimestamp)
	start = time.Now()
	timestampCheck := checkUpdatesTimestamp(ctx, time.Now().UTC())
	if timestampCheck.Error != "" {
		tracker.phaseWarn(phaseTimestamp, phaseSummaryKV("timestamp check unavailable", timestampCheck.Error, shortDuration(time.Since(start))))
	} else if !timestampCheck.OK {
		daysLeft := int(timestampCheck.DurationLeft.Hours() / 24)
		tracker.phaseFail(phaseTimestamp, phaseSummaryKV(
			fmt.Sprintf("expires in %d days (min %d)", daysLeft, timestampCheck.MinDays),
			shortDuration(time.Since(start)),
		))
	} else {
		daysLeft := int(timestampCheck.DurationLeft.Hours() / 24)
		tracker.phaseDone(phaseTimestamp, phaseSummaryKV(
			fmt.Sprintf("expires in %d days", daysLeft),
			shortDuration(time.Since(start)),
		))
	}

	tracker.phaseStart(phaseReport)
	start = time.Now()
	bridge, err := startUIBridge(token, time.Duration(*bridgeIdleMinutes)*time.Minute, tracker.bridgeSignal)
	if err != nil {
		log.Printf("could not start UI bridge: %v", err)
	}
	bridgeBaseURL, bridgeSession := "", ""
	if bridge != nil {
		bridgeBaseURL = bridge.baseURL
		bridgeSession = bridge.session
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
			timestampCheck,
			bridgeBaseURL,
			bridgeSession,
		),
	)
	if err != nil {
		log.Printf("could not write HTML report: %v", err)
		return
	}
	tracker.phaseDone(phaseReport, phaseSummaryKV("report generated", shortDuration(time.Since(start))))

	tracker.phaseStart(phaseBrowser)
	tracker.waitingForBrowser(reportPath)
	if *openReport {
		if err := openInBrowser(reportPath); err != nil {
			log.Printf("could not auto-open report: %v", err)
			tracker.phaseWarn(phaseBrowser, "browser auto-open failed")
			return
		}
		tracker.phaseDone(phaseBrowser, "browser open signal sent")
	} else {
		tracker.phaseWarn(phaseBrowser, "auto-open disabled (-open-report=false)")
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
