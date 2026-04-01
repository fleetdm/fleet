# Fix: VPP token assignment fails when app_store_apps are defined (#40785)

## Summary

When a GitOps run includes a `volume_purchasing_program` config that references a team that doesn't exist yet, the code temporarily removes the entire VPP config from the global AppConfig — clearing ALL VPP token-to-team assignments on the server. However, the code only deferred `app_store_apps` for the missing teams, not for existing teams that also lost their VPP assignments. Those existing teams then failed with "No available VPP Token" when their `app_store_apps` were applied.

The fix widens the deferral scope to match the clearing scope: when VPP assignments are temporarily cleared, `app_store_apps` are now deferred for all teams in the VPP config, not just the missing ones.

## Root Cause Analysis

### The bug scenario

Given a VPP config like:

```yaml
volume_purchasing_program:
  - location: Company Name
    teams:
      - workstations          # exists
      - workstations-canary   # exists
      - new-team              # exists
      - new-team-canary       # does NOT exist yet
```

With `app_store_apps` defined in the `new-team` team YAML.

1. `checkVPPTeamAssignments` finds `missingVPPTeams = ["new-team-canary"]` (the only team not in the DB)
2. VPP config is removed from the global config (`mdmMap["volume_purchasing_program"] = nil`) — this is necessary because the server would reject assigning a token to a non-existent team
3. In `DoGitOps`, `nil` becomes `[]any{}` (line 2057 in `client.go`), which when sent to the server **clears ALL VPP token assignments for ALL teams**
4. `app_store_apps` are stripped only for teams in `missingVPPTeams` — so only `new-team-canary` gets deferred
5. `new-team` still has its `app_store_apps` and they're applied immediately, but its VPP assignment was just cleared → **"No available VPP Token"**

### Why `missingVPPTeams` was used originally

The feature was introduced in commit `27b6174543` (PR #28624, for issue #26114) to solve a narrow problem: *brand new teams that don't exist yet can't have VPP apps applied because they don't have a VPP token yet*.

Using `missingVPPTeams` was logically correct for that specific case — only the new (missing) teams need their apps deferred, because only they lack a VPP token. The original author was solving a narrowly-scoped problem and the test only covered the single-new-team case.

What was overlooked: the mechanism for deferring VPP token assignment is removing `volume_purchasing_program` entirely from the global config. This gets converted to an empty array, which when sent to the server clears ALL VPP token assignments for ALL teams — not just the missing ones. The blast radius of the VPP clearing is global, but the blast radius of the app deferral was only the missing teams.

### The server-side clearing mechanism

In `server/service/appconfig.go` (line 961-988), when `volume_purchasing_program` is defined (even as empty):

1. ALL VPP tokens not present in the config have their team assignments reset to nil
2. Then new assignments from the config are applied

When the config is `[]` (empty), step 1 clears everything and step 2 does nothing.

## Fix

### `cmd/fleetctl/fleetctl/gitops.go`

**Line 524:** Changed `missingVPPTeams` → `vppTeams` in the loop that decides which teams' `app_store_apps` to defer. This ensures that when VPP assignments are temporarily cleared (because of missing teams), ALL teams in the VPP config have their `app_store_apps` deferred, not just the missing ones. Added `break` to avoid duplicate entries.

**Line 22:** Updated `ReapplyingTeamForVPPAppsMsg` to not say "new teams" since it now applies to existing teams too.

### `cmd/fleetctl/integrationtest/gitops/software_test.go`

Added `TestGitOpsExistingTeamVPPAppsWithMissingTeam` covering the exact reproduction scenario: an existing team with `app_store_apps` alongside a new (missing) team, both in the VPP config.

## Verification

- All 9 VPP-related tests pass (`TestGitOpsTeamVPPAndApp`, `TestGitOpsExistingTeamVPPAppsWithMissingTeam`, 7 `TestGitOpsVPP` subtests)
- `go vet`: no issues
- `golangci-lint`: 0 issues

### Manual QA steps

1. Set up Fleet with a VPP token and an existing team ("team-A") assigned to it
2. Create GitOps config with `volume_purchasing_program` including "team-A" AND a new team "team-B"
3. Add `app_store_apps` to both team YAML files
4. Run `fleetctl gitops -f default.yml -f team-a.yml -f team-b.yml`
5. Verify: both teams get created/updated, VPP assigned, and `app_store_apps` applied without errors
6. Verify: the workaround path (removing `app_store_apps`, applying, then adding them back) also still works
