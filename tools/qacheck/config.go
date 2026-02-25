package main

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	awaitingQAColumn = "✔️Awaiting QA"
	checkText        = "Engineer: Added comment to user story confirming successful completion of test plan."

	// Drafting board (Project 67) check:
	draftingProjectNum   = 67
	draftingStatusNeedle = "Ready to estimate,Estimated"
	reportDirName        = "qacheck-report"
	reportFileName       = "index.html"

	defaultStaleDays         = 21
	defaultBridgeIdleMinutes = 15

	updatesTimestampURL = "https://updates.fleetdm.com/timestamp.json"
	minTimestampDays    = 5
)

var draftingChecklistIgnorePrefixes = []string{
	"Once shipped, requester has been notified",
	"Once shipped, dogfooding issue has been filed",
	"Review of all files under server/mdm/microsoft",
	"Review of any files named microsoft_mdm.go",
	"Review of windows_mdm_profiles.go",
	"All Microsoft MDM related endpoints not defined in these files",
}

type intListFlag []int

func (f *intListFlag) String() string {
	if f == nil || len(*f) == 0 {
		return ""
	}
	out := make([]string, 0, len(*f))
	for _, n := range *f {
		out = append(out, strconv.Itoa(n))
	}
	return strings.Join(out, ",")
}

func (f *intListFlag) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return fmt.Errorf("invalid project number %q", part)
		}
		*f = append(*f, n)
	}
	return nil
}
