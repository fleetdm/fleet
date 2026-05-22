package mysql

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// To run:
//
//	MYSQL_TEST=1 go test -bench=BenchmarkListVulnerabilities -benchtime=10x \
//	    -run='^$' ./server/datastore/mysql/
//
// To compare before/after with benchstat:
//
//	git checkout main
//	MYSQL_TEST=1 go test -bench=. -benchtime=10x -run='^$' \
//	    ./server/datastore/mysql/ -count=5 > /tmp/before.txt
//
//	git checkout <feature-branch>
//	MYSQL_TEST=1 go test -bench=. -benchtime=10x -run='^$' \
//	    ./server/datastore/mysql/ -count=5 > /tmp/after.txt
//
//	benchstat /tmp/before.txt /tmp/after.txt
//
// Tune the dataset size with FLEET_BENCH_SIZE: "smoke" (default, ~5s seed),
// "realistic" (~30s seed), or "large" (~3min seed). EXPLAIN plan differences
// show up at any size; wall-time differences widen with scale.

type benchSize struct {
	numSoftware int
	numCVEs     int
	cvesPerSW   int
	// Fraction of CVEs that also appear in operating_system_vulnerabilities,
	// so the (EXISTS software_cve OR EXISTS operating_system_vulnerabilities)
	// branch in ListVulnerabilities actually has to consider both sides.
	osVulnFraction float64
	// Number of fake operating_system rows the OS vulns are spread across.
	// Doesn't have to be realistic — we just need ids to point at.
	numOperatingSystems int
	// teamSoftwareFraction: fraction of software that also has a team-scoped
	// software_host_counts row (team_id=1, global_stats=0). Same idea for vulns.
	teamSoftwareFraction float64
	teamVulnFraction     float64
}

var benchSizes = map[string]benchSize{
	"smoke": {
		numSoftware: 2_000, numCVEs: 5_000, cvesPerSW: 3,
		osVulnFraction: 0.3, numOperatingSystems: 20,
		teamSoftwareFraction: 0.4, teamVulnFraction: 0.4,
	},
	"realistic": {
		numSoftware: 50_000, numCVEs: 50_000, cvesPerSW: 4,
		osVulnFraction: 0.3, numOperatingSystems: 50,
		teamSoftwareFraction: 0.4, teamVulnFraction: 0.4,
	},
	"large": {
		numSoftware: 200_000, numCVEs: 200_000, cvesPerSW: 5,
		osVulnFraction: 0.3, numOperatingSystems: 100,
		teamSoftwareFraction: 0.4, teamVulnFraction: 0.4,
	},
}

// benchTeamID is the team_id seeded for team-scoped rows. Benchmarks that
// exercise the team path should pass &benchTeamID as the TeamID option.
const benchTeamID uint = 1

func pickBenchSize(tb testing.TB) benchSize {
	tb.Helper()
	name := os.Getenv("FLEET_BENCH_SIZE")
	if name == "" {
		name = "smoke"
	}
	sz, ok := benchSizes[name]
	if !ok {
		tb.Fatalf("unknown FLEET_BENCH_SIZE=%q (want smoke|realistic|large)", name)
	}
	return sz
}

// seedVulnPerfData populates a Fleet schema with enough data to make the
// query planner exercise the indexed paths. It bypasses Fleet's ingestion
// code in favor of direct multi-row INSERTs so seeding stays under a minute
// even at "large" size.
func seedVulnPerfData(tb testing.TB, ds *Datastore, sz benchSize) {
	tb.Helper()
	ctx := context.Background()
	w := ds.writer(ctx)

	// FK checks slow batch insert significantly; safe to disable for seeding.
	if _, err := w.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS=0"); err != nil {
		tb.Fatalf("disable FK: %v", err)
	}
	defer func() {
		if _, err := w.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS=1"); err != nil {
			tb.Logf("re-enable FK: %v", err)
		}
	}()

	r := rand.New(rand.NewPCG(1, 2)) // nolint:gosec,G404 // benchmark seed, not security-sensitive
	start := time.Now()

	// software_titles — one per software for simplicity
	batchInsert(tb, ds,
		"INSERT INTO software_titles (id, name, source) VALUES ",
		sz.numSoftware, 3, func(i int) []any {
			return []any{i + 1, fmt.Sprintf("title-%d", i+1), "programs"}
		})

	// software — title_id maps 1:1. checksum is a unique binary(16) so we
	// derive it from the row index deterministically.
	batchInsert(tb, ds,
		"INSERT INTO software (id, name, version, source, title_id, checksum) VALUES ",
		sz.numSoftware, 6, func(i int) []any {
			checksum := make([]byte, 16)
			for j := range 8 {
				checksum[j] = byte(uint(i+1) >> (8 * j))
			}
			return []any{
				i + 1,
				fmt.Sprintf("software-%d", i+1),
				fmt.Sprintf("1.%d.%d", i%10, i%100),
				"programs",
				i + 1,
				checksum,
			}
		})

	// cve_meta — distribute cvss_score and a CISA known-exploit bit
	batchInsert(tb, ds,
		"INSERT INTO cve_meta (cve, cvss_score, epss_probability, cisa_known_exploit, published, description) VALUES ",
		sz.numCVEs, 6, func(i int) []any {
			return []any{
				fmt.Sprintf("CVE-2024-%07d", i),
				r.Float64() * 10,
				r.Float64(),
				r.IntN(20) == 0, // ~5% are CISA known exploits
				time.Now().Add(-time.Duration(r.IntN(365*24)) * time.Hour),
				fmt.Sprintf("Description for CVE-2024-%07d", i),
			}
		})

	// software_cve — every software gets [0, cvesPerSW] random CVEs
	type swcve struct {
		swID int
		cve  string
	}
	pairs := make([]swcve, 0, sz.numSoftware*sz.cvesPerSW)
	seen := make(map[swcve]struct{})
	for i := 0; i < sz.numSoftware; i++ {
		n := r.IntN(sz.cvesPerSW + 1)
		for range n {
			p := swcve{swID: i + 1, cve: fmt.Sprintf("CVE-2024-%07d", r.IntN(sz.numCVEs))}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			pairs = append(pairs, p)
		}
	}
	batchInsert(tb, ds,
		"INSERT INTO software_cve (software_id, cve, source) VALUES ",
		len(pairs), 3, func(i int) []any {
			return []any{pairs[i].swID, pairs[i].cve, 1}
		})

	// operating_system_vulnerabilities — a fraction of CVEs also appear as
	// OS vulns so the OR-EXISTS branch in ListVulnerabilities is actually
	// exercised (in prod about half of vuln catalog entries are OS-side).
	// Unique key is (operating_system_id, cve), so we spread CVEs across a
	// small pool of OS ids.
	if sz.osVulnFraction > 0 && sz.numOperatingSystems > 0 {
		osPairs := make([]struct {
			osID int
			cve  string
		}, 0, int(float64(sz.numCVEs)*sz.osVulnFraction))
		osSeen := make(map[[2]int]struct{})
		want := int(float64(sz.numCVEs) * sz.osVulnFraction)
		for len(osPairs) < want {
			cveIdx := r.IntN(sz.numCVEs)
			osID := 1 + r.IntN(sz.numOperatingSystems)
			key := [2]int{osID, cveIdx}
			if _, dup := osSeen[key]; dup {
				continue
			}
			osSeen[key] = struct{}{}
			osPairs = append(osPairs, struct {
				osID int
				cve  string
			}{osID, fmt.Sprintf("CVE-2024-%07d", cveIdx)})
		}
		batchInsert(tb, ds,
			"INSERT INTO operating_system_vulnerabilities (operating_system_id, cve, source) VALUES ",
			len(osPairs), 3, func(i int) []any {
				return []any{osPairs[i].osID, osPairs[i].cve, 1}
			})
	}

	// software_host_counts — global row per software, plus a team-scoped row
	// for a fraction of software so team_id-filtered benchmarks have data.
	batchInsert(tb, ds,
		"INSERT INTO software_host_counts (software_id, hosts_count, team_id, global_stats) VALUES ",
		sz.numSoftware, 4, func(i int) []any {
			return []any{i + 1, 1 + r.IntN(500), 0, 1}
		})
	numTeamSW := int(float64(sz.numSoftware) * sz.teamSoftwareFraction)
	if numTeamSW > 0 {
		batchInsert(tb, ds,
			"INSERT INTO software_host_counts (software_id, hosts_count, team_id, global_stats) VALUES ",
			numTeamSW, 4, func(i int) []any {
				return []any{i + 1, 1 + r.IntN(200), benchTeamID, 0}
			})
	}

	// vulnerability_host_counts — global row per CVE, plus team-scoped rows
	// for a fraction of CVEs.
	batchInsert(tb, ds,
		"INSERT INTO vulnerability_host_counts (cve, team_id, host_count, global_stats) VALUES ",
		sz.numCVEs, 4, func(i int) []any {
			return []any{fmt.Sprintf("CVE-2024-%07d", i), 0, 1 + r.IntN(500), 1}
		})
	numTeamVuln := int(float64(sz.numCVEs) * sz.teamVulnFraction)
	if numTeamVuln > 0 {
		batchInsert(tb, ds,
			"INSERT INTO vulnerability_host_counts (cve, team_id, host_count, global_stats) VALUES ",
			numTeamVuln, 4, func(i int) []any {
				return []any{fmt.Sprintf("CVE-2024-%07d", i), benchTeamID, 1 + r.IntN(200), 0}
			})
	}

	tb.Logf("seeded %d software (%d team-scoped), %d cves (%d team-scoped, %d also OS-vuln), %d software_cve rows in %s",
		sz.numSoftware, numTeamSW, sz.numCVEs, numTeamVuln,
		int(float64(sz.numCVEs)*sz.osVulnFraction), len(pairs),
		time.Since(start).Round(time.Millisecond))
}

// batchInsert issues multi-row INSERTs of `cols` placeholders per row in
// chunks small enough to stay under MySQL's default max_allowed_packet.
func batchInsert(
	tb testing.TB,
	ds *Datastore,
	prefix string,
	n, cols int,
	row func(i int) []any,
) {
	tb.Helper()
	ctx := context.Background()
	const rowsPerStmt = 500
	placeholder := "(" + strings.Repeat("?,", cols-1) + "?)"
	for start := 0; start < n; start += rowsPerStmt {
		end := min(start+rowsPerStmt, n)
		parts := make([]string, 0, end-start)
		args := make([]any, 0, (end-start)*cols)
		for i := start; i < end; i++ {
			parts = append(parts, placeholder)
			args = append(args, row(i)...)
		}
		stmt := prefix + strings.Join(parts, ",")
		if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
			tb.Fatalf("batch insert (rows %d-%d): %v", start, end, err)
		}
	}
}

func BenchmarkListVulnerabilities(b *testing.B) {
	ds := CreateMySQLDS(b)
	sz := pickBenchSize(b)
	seedVulnPerfData(b, ds, sz)

	cases := []struct {
		name string
		opt  fleet.VulnListOptions
	}{
		{
			name: "cvss_score_page0_per20",
			opt: fleet.VulnListOptions{
				IsEE: true,
				ListOptions: fleet.ListOptions{
					OrderKey:        "cvss_score",
					OrderDirection:  fleet.OrderDescending,
					PerPage:         20,
					IncludeMetadata: true,
				},
			},
		},
		{
			name: "cvss_score_page50_per20",
			opt: fleet.VulnListOptions{
				IsEE: true,
				ListOptions: fleet.ListOptions{
					OrderKey:        "cvss_score",
					OrderDirection:  fleet.OrderDescending,
					PerPage:         20,
					Page:            50,
					IncludeMetadata: true,
				},
			},
		},
		{
			name: "exploit_filter_page0",
			opt: fleet.VulnListOptions{
				IsEE:         true,
				KnownExploit: true,
				ListOptions: fleet.ListOptions{
					OrderKey:        "cvss_score",
					OrderDirection:  fleet.OrderDescending,
					PerPage:         20,
					IncludeMetadata: true,
				},
			},
		},
		{
			name: "created_at_page0_legacy",
			opt: fleet.VulnListOptions{
				IsEE: true,
				ListOptions: fleet.ListOptions{
					OrderKey:        "created_at",
					OrderDirection:  fleet.OrderDescending,
					PerPage:         20,
					IncludeMetadata: true,
				},
			},
		},
		{
			name: "team_cvss_score_page0",
			opt: fleet.VulnListOptions{
				IsEE:   true,
				TeamID: new(benchTeamID),
				ListOptions: fleet.ListOptions{
					OrderKey:        "cvss_score",
					OrderDirection:  fleet.OrderDescending,
					PerPage:         20,
					IncludeMetadata: true,
				},
			},
		},
		{
			name: "team_exploit_page0",
			opt: fleet.VulnListOptions{
				IsEE:         true,
				TeamID:       new(benchTeamID),
				KnownExploit: true,
				ListOptions: fleet.ListOptions{
					OrderKey:        "cvss_score",
					OrderDirection:  fleet.OrderDescending,
					PerPage:         20,
					IncludeMetadata: true,
				},
			},
		},
	}

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := ds.ListVulnerabilities(context.Background(), c.opt)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCountVulnerabilities(b *testing.B) {
	ds := CreateMySQLDS(b)
	sz := pickBenchSize(b)
	seedVulnPerfData(b, ds, sz)

	cases := []struct {
		name string
		opt  fleet.VulnListOptions
	}{
		{"global_no_filter", fleet.VulnListOptions{IsEE: true}},
		{"global_exploit", fleet.VulnListOptions{IsEE: true, KnownExploit: true}},
		{"match_query", fleet.VulnListOptions{
			IsEE:        true,
			ListOptions: fleet.ListOptions{MatchQuery: "CVE-2024-0001"},
		}},
		{"team_no_filter", fleet.VulnListOptions{IsEE: true, TeamID: new(benchTeamID)}},
		{"team_exploit", fleet.VulnListOptions{IsEE: true, TeamID: new(benchTeamID), KnownExploit: true}},
	}

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := ds.CountVulnerabilities(context.Background(), c.opt)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkListSoftwareVersions(b *testing.B) {
	ds := CreateMySQLDS(b)
	sz := pickBenchSize(b)
	seedVulnPerfData(b, ds, sz)

	cases := []struct {
		name string
		opt  fleet.SoftwareListOptions
	}{
		{
			name: "no_filter_page0",
			opt: fleet.SoftwareListOptions{
				WithHostCounts:   true,
				IncludeCVEScores: true,
				ListOptions: fleet.ListOptions{
					OrderKey:        "hosts_count",
					OrderDirection:  fleet.OrderDescending,
					PerPage:         20,
					IncludeMetadata: true,
				},
			},
		},
		{
			name: "vulnerable_page0",
			opt: fleet.SoftwareListOptions{
				VulnerableOnly:   true,
				WithHostCounts:   true,
				IncludeCVEScores: true,
				ListOptions: fleet.ListOptions{
					OrderKey:        "hosts_count",
					OrderDirection:  fleet.OrderDescending,
					PerPage:         20,
					IncludeMetadata: true,
				},
			},
		},
		{
			name: "vulnerable_exploit_cvss_page0",
			opt: fleet.SoftwareListOptions{
				VulnerableOnly:   true,
				KnownExploit:     true,
				MinimumCVSS:      3,
				WithHostCounts:   true,
				IncludeCVEScores: true,
				ListOptions: fleet.ListOptions{
					OrderKey:        "hosts_count",
					OrderDirection:  fleet.OrderDescending,
					PerPage:         20,
					IncludeMetadata: true,
				},
			},
		},
		{
			name: "vulnerable_narrow_query_page0",
			opt: fleet.SoftwareListOptions{
				VulnerableOnly:   true,
				WithHostCounts:   true,
				IncludeCVEScores: true,
				ListOptions: fleet.ListOptions{
					MatchQuery:      "software-1",
					OrderKey:        "hosts_count",
					OrderDirection:  fleet.OrderDescending,
					PerPage:         20,
					IncludeMetadata: true,
				},
			},
		},
		{
			// Broad match (1-char LIKE) — matches almost every software row;
			// useful for spotting plan flips driven by predicate selectivity.
			name: "vulnerable_broad_query_page0",
			opt: fleet.SoftwareListOptions{
				VulnerableOnly:   true,
				WithHostCounts:   true,
				IncludeCVEScores: true,
				ListOptions: fleet.ListOptions{
					MatchQuery:      "s",
					OrderKey:        "hosts_count",
					OrderDirection:  fleet.OrderDescending,
					PerPage:         20,
					IncludeMetadata: true,
				},
			},
		},
		{
			name: "team_vulnerable_page0",
			opt: fleet.SoftwareListOptions{
				TeamID:           new(benchTeamID),
				VulnerableOnly:   true,
				WithHostCounts:   true,
				IncludeCVEScores: true,
				ListOptions: fleet.ListOptions{
					OrderKey:        "hosts_count",
					OrderDirection:  fleet.OrderDescending,
					PerPage:         20,
					IncludeMetadata: true,
				},
			},
		},
		{
			// Matches the customer's full Q1 pattern: team + exploit + cvss
			// floor + vulnerable + search.
			name: "team_full_customer_pattern_page0",
			opt: fleet.SoftwareListOptions{
				TeamID:           new(benchTeamID),
				VulnerableOnly:   true,
				KnownExploit:     true,
				MinimumCVSS:      3,
				WithHostCounts:   true,
				IncludeCVEScores: true,
				ListOptions: fleet.ListOptions{
					MatchQuery:      "s",
					OrderKey:        "hosts_count",
					OrderDirection:  fleet.OrderDescending,
					PerPage:         20,
					IncludeMetadata: true,
				},
			},
		},
	}

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := ds.ListSoftware(context.Background(), c.opt)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCountSoftware(b *testing.B) {
	ds := CreateMySQLDS(b)
	sz := pickBenchSize(b)
	seedVulnPerfData(b, ds, sz)

	cases := []struct {
		name string
		opt  fleet.SoftwareListOptions
	}{
		{"no_filter", fleet.SoftwareListOptions{}},
		{"vulnerable", fleet.SoftwareListOptions{VulnerableOnly: true}},
		{"vulnerable_exploit", fleet.SoftwareListOptions{
			VulnerableOnly: true, KnownExploit: true, IncludeCVEScores: true,
		}},
		{"vulnerable_cvss", fleet.SoftwareListOptions{
			VulnerableOnly: true, MinimumCVSS: 3, IncludeCVEScores: true,
		}},
		{"team_vulnerable", fleet.SoftwareListOptions{
			TeamID: new(benchTeamID), VulnerableOnly: true,
		}},
		{"team_vulnerable_exploit", fleet.SoftwareListOptions{
			TeamID: new(benchTeamID), VulnerableOnly: true,
			KnownExploit: true, IncludeCVEScores: true,
		}},
	}

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := ds.CountSoftware(context.Background(), c.opt)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
