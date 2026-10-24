# Seeded-scale database snapshots: full-scale data without full-scale traffic

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering` · **Effort:** M · **Track:** local scale testing

## Problem

Most scaling bugs are "the query got slow when the table got big": index depth, optimizer plan flips at cardinality thresholds, buffer-pool miss rates. None of that needs 400k *active* hosts to reproduce — it needs 400k hosts' worth of *rows*. Data volume is cheap to generate; request volume is expensive to simulate. Today every engineer tests against a dev DB with a handful of hosts, so this entire bug class ships undetected.

## Impact

Enabler for the whole local-testing track: the proportionate rig (17), the slope test (18), and the at-scale conditions of satisfaction in issues 08, 09, 11, 12 all run against these snapshots.

## Proposed fix

Build in `tools/loadtest/` (alongside the existing metrics baselines):

- **Seeder:** enroll N hosts via `cmd/osquery-perf` in waves (it already generates realistic software inventory, users, policies from its platform templates) until the DB reaches target shape; include MDM-enrolled profiles/commands history, labels, policy results, and representative activity-table volume (per issue 11's inventory). Direct-SQL generation acceptable where osquery-perf is too slow, but prefer the real enrollment path so rows have realistic shape and cardinality.
- **Snapshot/restore scripts:** `seed.sh` (hours, run rarely), `snapshot.sh` / `restore.sh` (minutes, run per test) — mysqldump or binary datadir copy; store compressed snapshots out of the repo (S3/artifact storage) with versioned names tied to the migration level, plus a CI-friendly fetch.
- **Standard sizes:** 20k and 100k snapshots as the working set (the two points the issue 18 slope test needs), 400k as the periodic deep-check size.
- **Re-activation supported:** seeded hosts keep usable node keys (`-node_key_file`, `-only_already_enrolled`) so any subset of the seeded population can be driven live by osquery-perf later.
- Docs: a README covering the two-command workflow (`restore 100k`, point server at it) on a laptop.

## Condition of satisfaction

- An engineer can go from clean checkout to a running server against a 100k-host DB in under 15 minutes (snapshot download time aside).
- Snapshots versioned against migrations with a documented refresh procedure (restore → `fleet prepare db` → re-snapshot).
- At least one real finding filed by running an existing feature against the 100k snapshot (dogfood the tool).
