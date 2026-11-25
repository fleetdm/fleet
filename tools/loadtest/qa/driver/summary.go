package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"text/tabwriter"
	"time"
)

// summarize prints, per action, the runs in [from, to): verdict counts, success rate and client-side p50/p95/max.
// It also lists trips and pauses. It reads event-log files, not Fleet.
func summarize(w io.Writer, paths []string, from, to time.Time) error {
	type stats struct {
		verdicts map[string]int
		took     []int64
	}
	by := map[string]*stats{}
	var notes []string
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for sc.Scan() {
			var e event
			if json.Unmarshal(sc.Bytes(), &e) != nil {
				continue
			}
			t, err := time.Parse(eventTime, e.Start)
			if err != nil || t.Before(from) || !t.Before(to) {
				continue
			}
			switch e.Kind {
			case "action":
				s := by[e.ID]
				if s == nil {
					s = &stats{verdicts: map[string]int{}}
					by[e.ID] = s
				}
				s.verdicts[e.Verdict]++
				if e.Verdict != "skipped" {
					s.took = append(s.took, e.TookMS)
				}
			case "trip", "pause", "resume", "setup":
				notes = append(notes, fmt.Sprintf("%s %s %s %s", e.Start, e.Kind, e.ID, e.Error))
			}
		}
		f.Close()
		if err := sc.Err(); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
	}

	fmt.Fprintf(w, "Actions from %s to %s (UTC)\n\n", from.UTC().Format("2006-01-02 15:04"), to.UTC().Format("2006-01-02 15:04"))
	tw := tabwriter.NewWriter(w, 2, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tRUNS\tOK\tSLOW\tBROKEN\tSKIPPED\tSUCCESS\tP50\tP95\tMAX")
	ids := make([]string, 0, len(by))
	for id := range by {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		s, ok := by[id]
		if !ok {
			continue
		}
		ran := s.verdicts["ok"] + s.verdicts["slow"] + s.verdicts["broken"]
		success := "-"
		if ran > 0 {
			success = fmt.Sprintf("%.1f%%", 100*float64(s.verdicts["ok"]+s.verdicts["slow"])/float64(ran))
		}
		slices.Sort(s.took)
		fmt.Fprintf(tw, "%s\t%d\t%d\t%d\t%d\t%d\t%s\t%s\t%s\t%s\n", id, ran, s.verdicts["ok"], s.verdicts["slow"],
			s.verdicts["broken"], s.verdicts["skipped"], success, pct(s.took, 0.5), pct(s.took, 0.95), pct(s.took, 1))
	}
	_ = tw.Flush()
	if len(notes) > 0 {
		fmt.Fprintln(w, "\nTrips, pauses and setup failures:")
		for _, n := range notes {
			fmt.Fprintln(w, " ", n)
		}
	}
	return nil
}

// pct returns the q-quantile of sorted millisecond values, nearest rank.
func pct(sorted []int64, q float64) string {
	if len(sorted) == 0 {
		return "-"
	}
	i := int(q*float64(len(sorted))+0.5) - 1
	i = max(0, min(i, len(sorted)-1))
	return (time.Duration(sorted[i]) * time.Millisecond).String()
}
