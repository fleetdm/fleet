# Activity driver

Runs a load-test instance's admin workload all day: the action specs in `../scenarios`, each as a Poisson arrival
stream at its rate, shaped by a business-hours curve, and watches the instance with the check specs. It writes one
JSON line per action and per check reading to stdout (and `-events`), so server metrics can be sliced by what was
running.

```sh
go build -o driver ./tools/loadtest/qa/driver

# What the schedule would do over a day, without calling Fleet:
./driver -scenarios tools/loadtest/qa/scenarios -profile ~/private/profile.yaml -dry_run 24h

# Each spec once, in order, then exit (checks read one window each):
FLEET_DRIVER_TOKEN=... ./driver -fleet_url https://fleet.example.com -scenarios tools/loadtest/qa/scenarios -once

# The workload, until interrupted:
FLEET_DRIVER_TOKEN=... ./driver -fleet_url https://fleet.example.com -scenarios tools/loadtest/qa/scenarios \
  -profile ~/private/profile.yaml -events events.jsonl -stop_file STOP -on_trip 'notify.sh'

# Per action: runs, verdicts, success rate, p50/p95/max over the last day:
./driver -summary events.jsonl -since 24h
```

The token is an admin API token, read from `$FLEET_DRIVER_TOKEN` (`-token_env` to change the variable).

## Rates and the profile

Each spec has `rate: {start: <guess>, profile: <key>, share: <0-1>}`. With `-profile`, the driver uses the
profile's rate for the key, times its `growth`, times the spec's `share`; without one, or for a key the profile
doesn't have, it uses the start guess. The profile holds a real customer's measured activity, so it lives outside
the repository and its numbers never go in events or logs (the `-dry_run` table shows them, so keep that output
private).

```yaml
growth: 1.3                       # target = measured × growth
rates:
  activity.live_query: 100/day    # keys name Fleet activity types
curve:                            # optional; overrides the defaults below field by field
  zones: [America/New_York, America/Chicago, America/Los_Angeles]
  open: 9
  close: 17
  off_hours: 0.05                 # weekday activity outside business hours, relative to business hours
  weekend: 0.1
  busy_hours: [10, 14]            # weekday local hours in the first zone
  busy_x: 3                       # a busy hour runs at 3× the weekday hourly average
```

Curves: `business_hours` (above, normalized so a week averages the spec's rate), `flat`, and `nightly`
(01:00–05:00 in the first zone).

## Verbs

| Verb | What it does | `with` |
|---|---|---|
| `live_query` | An ad-hoc live query through the campaign API and the results websocket, waiting until every online target answers | `target: hosts\|all`, `hosts: 1-50`, `timeout: 60s` |
| `users.login` | Logs one of the driver's password users in, loads `/me`, logs out sometimes; with `fail`, a wrong password expecting 401 | `logout: 0.3`, `fail: false` |
| `hosts.idp` | Sets a host's IdP user (`PUT /hosts/{id}/device_mapping`, source `idp`; Premium) | `users: 5000` (distinct emails) |
| `labels.membership` | Adds a host to, or removes one from, one of the driver's manual labels through `POST/DELETE /hosts/{id}/labels`, keeping each label near its size | `route: api`, `labels: 10`, `hosts: 200`, `band: 50`, `prefix` |
| `fleets.agent_options` | Changes one fleet's agent options through `POST /fleets/{id}/agent_options` (one option alternates between two values), the route that writes `edited_agent_options` for a fleet | `option: distributed_tls_max_attempts`, `values: 3,4` |
| `software.uninstall` | Uninstalls a Fleet-managed package from an online host where it's installed | `tries: 10` (hosts to look at) |
| `profiles.resend` | Resends one delivered configuration profile to an online host | `platform: darwin`, `tries: 10` |
| `hosts.transfer` | Moves a few hosts between the listed fleets, so they pick up the other fleet's config | `hosts: 1-20`, `fleets: [names]` |
| `gitops.apply` | A CI run on a GitOps repo: a small real change, then the repo's apply command (see below) | `repo`, `command`, `pause_file`, `busy_file`, `timeout: 45m`, `scripts: 3`, `profiles`, `profiles_glob`, `agent_options`, `agent_option`, `agent_option_values: 3,4`, `global_agent_options_file`, `labels`, `labels_file`, `label_hosts: 200`, `label_change: 2` |

Verdicts per action: `ok`, `slow` (over `check.request.took`, or a late outcome such as a live query that some
online targets never answered), `broken` (an error), `skipped` (paused, the previous run still going, or nothing
to act on). An arrival that can't run is logged as skipped and not made up later, so a pause never turns into a
burst.

Verbs that first look up what to act on (`software.uninstall` reads hosts' software until it finds an installed
package, `profiles.resend` reads hosts' profiles, `fleets.agent_options` reads the fleet) report the action's own
request in `took_ms` and judge it against the budget; the lookups go in `detail` (`lookups`, `lookup_ms`,
`lookup_max_ms`, and `total_ms` for the whole run).

## GitOps runs

`gitops.apply` works on an operator's GitOps repo (`repo`, usually from an env var, because a real repo holds
private config) and runs `command` in it (for example `fleetctl gitops -f default.yml -f fleets/a.yml ...`). Before
each run it changes a few files with markers the next run replaces, so the repo never drifts:

- `scripts` script files (`scripts_glob`, default `lib/*/scripts/*`) get a trailing `# fleet-driver edit <time>`.
- `profiles` macOS profiles, one per directory (`profiles_glob`), get a `<!-- fleet-driver edit <time> -->`
  comment: the checksum changes, so Fleet re-delivers the profile to the fleet's Macs.
- `agent_options` fleet files (`fleets_glob`) alternate one agent option (`agent_option`) between two values, and
  so does `global_agent_options_file` (e.g. `default.yml`) on every run. Only the global change writes an
  `edited_agent_options` activity; a GitOps fleet apply changes a fleet's agent options without one.
- `labels` manual labels in `labels_file` (include it from `default.yml` with `labels: [{path: ...}]`) of about
  `label_hosts` hosts each (UUIDs), with `label_change` hosts swapped on each label every run. This is the GitOps
  route to `edited_label`; give the repo's other manual labels no `hosts` key so GitOps leaves their membership alone.

`pause_file` skips a run while it exists; `busy_file` exists while a run is going. A non-zero exit is `broken`, and
only a gateway error (502/504) counts toward the broken-actions trip.

## Objects the driver owns

It creates them on first use and reuses them after:

- `-users` password users, `<user_prefix>NNN@example.com` (default `driver-user-`), with mixed global roles. The
  passwords are kept only in `-state_dir` (default `~/.fleet-driver/<fleet host>/users.json`, mode 0600) and never
  logged; a pool user whose password the state file doesn't have is deleted and created again.
- Manual labels `<prefix>NN` (default `driver-api-label-`). If GitOps owns the instance's global labels, list them
  in `default.yml` without a `hosts` key, so GitOps keeps them and leaves their membership to the driver.

To clean up, delete the users and labels with those prefixes.

## Checks

`kind: check` specs (in `../scenarios/checks`) watch a metric all the time: every `every` (default: the window), the
driver reads the last `window` and compares it with `budget` (`<= 2s`, `< 60%`, `== 0`). `breach` windows over budget
in a row hold a breach: with `trip: true` it trips the workload (below), otherwise it's a warning in the log. A metric
without a budget is only recorded. Every reading is a `kind: check` event with the value and the window, and checks keep
reading while the workload is paused, so recovery is on record too.

```yaml
kind: check
id: C-HOST-01
title: Host check-ins stay fast
metrics:
  - {metric: host.distributed_write.p99, window: 5m, budget: "<= 2s", breach: 3}
  - {metric: checkin.errors, window: 5m}
```

FORMAT.md's single-metric shape (`metric`, `window`, `budget`, `breach` at the top) works too; invariant checks
(`compare`) and relative budgets (trailing baselines) aren't run yet.

| Metric | Unit | Source |
|---|---|---|
| `healthz.p99` | seconds | the driver's own `/healthz` readings, one a minute |
| `hosts.online_drop` | percent | online hosts below their highest count in the window (`/host_summary`) |
| `fleet.memory.max`, `fleet.cpu.max` | percent | ECS service metrics, highest minute |
| `fleet.tasks_missing` | count | the Fleet service's desired minus running tasks |
| `fleet.task_stops` | count | Fleet tasks stopped in the window, with their reasons |
| `alb.5xx_pct` | percent | the device load balancer's 5xx (targets' and its own) per request |
| `writer.cpu`, `reader.cpu` | percent | Aurora CPU by role, window average |
| `host.distributed_write.p99`, `osquery.config.p99`, `orbit.config.p99` | seconds | Fleet's request log (`took`), through Logs Insights |
| `checkin.errors` | count | the busiest minute's `context canceled` errors in Fleet's log |

Every metric but the first two reads AWS, so it needs `-lt fleet-<workspace>` (the load-test terraform names the ECS
cluster, the Aurora cluster and the log group after it) and credentials from the default chain that can read ECS,
CloudWatch metrics and Logs Insights; without `-lt` those metrics are dropped at start. The log-based metrics cost a
few cents a day in Logs Insights scans at 10k hosts.

## Pausing and trips

While the `-stop_file` exists, every arrival is skipped. The driver creates the file on a trip and runs `-on_trip`
once (with `DRIVER_TRIP_REASON` in its environment); it resumes only when someone removes the file. It trips on
Fleet's own symptoms:

- `/healthz` failing or taking 2 s or more 3 minutes in a row;
- 10 of the last 20 actions getting a 5xx or no response;
- a check metric with `trip: true` whose breach holds.

Database load alone doesn't trip it. Watch that separately: a busy reader during a vulnerabilities run is normal at
a large population.
