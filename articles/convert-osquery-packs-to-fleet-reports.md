# Convert osquery packs to Fleet reports

Many osquery packs shared by the community use the JSON pack format from 2017. Fleet still supports packs, but reports are the simpler way to schedule queries and to import and export them. This guide shows how to use an AI assistant to convert a pack into report YAML, then check the result and apply it with GitOps.

## Prerequisites

- An osquery pack file (`.conf` or `.json`)
- A Fleet GitOps repository, or [`fleetctl`](https://fleetdm.com/docs/using-fleet/fleetctl-cli) logged in to your Fleet instance
- An AI assistant, like Claude or ChatGPT

## Understand how pack fields map to reports

Each entry under `queries` in a pack becomes one report. Use this mapping to check the AI's output:

| osquery pack field | Fleet report field |
| --- | --- |
| Key under `queries` | `name` |
| `query` | `query` |
| `description` | `description` |
| `interval` | `interval` |
| `platform` (on the query or the pack) | `platform` (`darwin`, `linux`, or `windows`, comma-separated) |
| `version` | `min_osquery_version` |
| `snapshot: true` | `logging: snapshot` |
| `removed: false` | `logging: differential_ignore_removals` |
| Neither `snapshot` nor `removed` set | `logging: differential` |

Pack results always go to your osquery log destination, so set `automations_enabled: true` on each report to keep that behavior.

> **Note:** Reports have no equivalent of a pack's `discovery` queries or `shard` setting. To target a subset of hosts, create a [label](https://fleetdm.com/guides/managing-labels-in-fleet) and add it to the report with `labels_include_any`.

## Convert the pack with AI

1. Open your AI assistant and paste in the prompt below, followed by the contents of your pack file.
2. Copy the YAML it returns into a new file in your GitOps repository, like `lib/incident-response.reports.yml`.

```
Convert this osquery pack into a YAML list of Fleet reports, using the format at
https://fleetdm.com/docs/configuration/yaml-files#reports.

For each entry under "queries":
- name: the pack name, a dash, then the query key (for example, "incident-response - launchd")
- query, description, interval: copy as-is
- platform: use the query's platform, or the pack's if the query has none. Map "posix" to "darwin,linux" and leave platform out for "all".
- min_osquery_version: copy from "version", if present
- logging: "snapshot" if snapshot is true, "differential_ignore_removals" if removed is false, otherwise "differential"
- automations_enabled: true

List any discovery queries, shard settings, or other fields you couldn't convert after the YAML.

Here is the pack:
```

For example, this pack entry:

```json
{
  "platform": "darwin",
  "queries": {
    "launchd": {
      "query": "SELECT * FROM launchd;",
      "interval": 3600,
      "description": "Retrieves all the daemons that will run in the start of the target OSX system.",
      "removed": false
    }
  }
}
```

Becomes this report:

```yaml
- name: incident-response - launchd
  description: Retrieves all the daemons that will run in the start of the target OSX system.
  query: SELECT * FROM launchd;
  platform: darwin
  interval: 3600
  logging: differential_ignore_removals
  automations_enabled: true
```

> **Warning:** AI assistants can drop, rename, or invent fields. Compare the output to the mapping table above before you apply it, and check that the number of reports matches the number of queries in the pack.

## Apply the reports

Reference the new file from `default.yml` (for all hosts) or from a fleet's file in `fleets/`:

```yaml
reports:
  - path: ../lib/incident-response.reports.yml
```

Then preview the change before you apply it:

```bash
fleetctl gitops -f default.yml --dry-run
```

If the dry run succeeds, merge your change so your GitOps workflow applies it, or run the same command without `--dry-run`.

## Verify

1. In the Fleet UI, go to **Reports**.
2. Confirm each converted report appears with the expected interval and platform.
3. After the first interval passes, open a report to confirm hosts are returning results.

## Further reading

- [Reports YAML reference](https://fleetdm.com/docs/configuration/yaml-files#reports)
- [Why does Fleet support query packs?](https://fleetdm.com/handbook/company/why-this-way#why-does-fleet-support-query-packs)

<meta name="articleTitle" value="Convert osquery packs to Fleet reports">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="publishedOn" value="2026-11-20">
<meta name="category" value="guides">
<meta name="description" value="Use AI to convert osquery packs into Fleet report YAML, then apply the reports with GitOps.">
