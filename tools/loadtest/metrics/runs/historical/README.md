# Historical runs (spreadsheet backfill)

These runs were backfilled from the manual-logging era of the
"Fleet release load testing results" Google Sheet (releases 4.63.0 → 4.85.0,
Feb 2025 – May 2026), one JSON per sheet row, so the whole history is usable by
the tooling in this directory (`compare-metrics.sh` and the metrics dashboard).
From 4.83 onward, runs collected with
`collect-metrics.sh` live in `../baseline`, `../migration`, and `../mdm`; sheet
rows duplicating those runs were not backfilled.

## Reading these files

- The sheet recorded human-observed 1-hour averages, so `metadata.interval` is
  `"1h"` and per-hour rate normalization is a no-op.
- `metadata.source` is `"spreadsheet-backfill"`, `metadata.category` records the
  run type (used by the dashboard's run selector), and `metadata.sheet_row`
  points back to the source row.
- **JSON `null` means the sheet tracked the metric but the cell was blank or
  unparseable for that row.** Fields the sheet never tracked (ALB metrics,
  percentiles, coverage, Performance Insights) are omitted entirely.
- `backfill.raw` preserves the original cell text for every parsed or dropped
  value — nothing from the sheet was discarded, even where parsing was lossy.
- `<url>`, `<tok>`, and `Token-N` placeholders mark where links and token-like
  strings were redacted at extraction time (these files live in a public repo).
  The unredacted text remains in the source sheet row (`metadata.sheet_row`).

## Parsing rules

Sheet cells were free text; they were converted with these rules:

| Cell style | Example | Result |
|---|---|---|
| Approximate | `~75%`, `~2,700`, `1.3K` | `75`, `2700`, `1300` |
| Leading range | `19-27%` | Average = midpoint (23), Maximum = 27 |
| Average keyword | `5-17 avg ~9` | Average = 9, Maximum = 17 |
| Peak/spike/max | `avg 1 spikes to 55`, `16% at peak, avg ~10%` | Average = 1/10, Maximum = 55/16 |
| Shift during run | `50% > 23%` | Average = mean of observations |
| IOPS with utilization | `2635 (~13%)` | `write_iops.Average` = 2635, `iops_utilization` = 13 |
| Utilization only | `11.57%` in the IOPS column | `iops_utilization` only — never converted to ops/s |
| Errors prose | `No new errors` | `0`; any other prose → `null` |
| Deadlocks prose | `~0` → `0` | rate descriptions (`0-8.4 per minute`) → `null` (unit differs from the collector's `Sum`) |

## Rows not backfilled

Sheet rows with no parseable performance metrics (pure ticket pointers):

| Sheet row | Environment | Reason |
|---|---|---|
| 14 | 4660-mdm | Profile-timing UX test; results in ticket #21338 |
| 21 | 469-mdmloadtest | DDM profile test; results in ticket #27979 |
| 22 | 469-mdmbatchtest | Batch profile resend test; results in ticket #25549 |
| 31 | 4720mdm (second row) | Results in ticket #30409 |
| 37 | 4740mdm (continuation row) | Host-count change note only |
| 48 | 4770vbt | High-software-count experiment, no recorded metrics |

## Caveats

- Averages derived from ranges are midpoints of human-observed ranges — treat
  them as indicative, not precise. The exact original text is in `backfill.raw`.
- Some `baseline`-categorized runs used nonstandard configurations (e.g.
  `4640loadtest-gabe` at 20K hosts, the `orch` variants); their absolute values
  aren't comparable to standard 100K-host baselines. Deselect them in the
  dashboard's run list when they distort a trend, and check `backfill.notes`.
- Two `4750orchl` runs happened the same day; the second file carries a `-2`
  suffix and a +1h `collected_at` so the runs keep distinct identities.
