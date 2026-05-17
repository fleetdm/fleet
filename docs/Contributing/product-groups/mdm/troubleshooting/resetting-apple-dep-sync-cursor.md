# DEP/ADE device sync in Fleet

## How it works

Fleet syncs ABM-assigned devices via [NanoDEP](https://github.com/fleetdm/fleet/tree/main/server/mdm/nanodep),
driven by a cursor persisted per ABM token.

- **No cursor** → Fleet calls Apple's [`fetch-devices`](https://developer.apple.com/documentation/devicemanagement/fetch-devices) (full list).
- **Cursor present** → Fleet calls [`sync-devices`](https://developer.apple.com/documentation/devicemanagement/sync-devices) (deltas only).

Fleet pages at **200 devices per request** (`DEPSyncLimit` in
[`server/mdm/apple/apple_mdm.go`](https://github.com/fleetdm/fleet/blob/main/server/mdm/apple/apple_mdm.go))
and loops until Apple reports no more pages. Sync cadence is set by
`mdm.apple_dep_sync_periodicity` (default 1 minute). Apple expires cursors
after 7 days; Fleet recovers automatically by falling back to a full fetch.

## Resetting the cursor

A "reset" is just clearing the stored cursor — the next sync then uses
`fetch-devices` and rebuilds Fleet's view from scratch.

This is fine to do even in larger deployments.

```sql
UPDATE nano_dep_names
SET    syncer_cursor = NULL
WHERE  name = '<your-abm-token-name>';
```

Storage backend: [`server/mdm/nanodep/storage/mysql`](https://github.com/fleetdm/fleet/tree/main/server/mdm/nanodep/storage/mysql).

A reset does **not** re-enroll devices, change ABM assignments, or re-push
DEP profiles — it only re-baselines Fleet's assignment list.

## Notes

- We've seen cases of where devices hit the threshold for moving into a cooldown/throttled state, Apple stops returning them on the sync cursor requests.

## Reference

- Apple ADE / DEP API:
  [`fetch-devices`](https://developer.apple.com/documentation/devicemanagement/fetch-devices),
  [`sync-devices`](https://developer.apple.com/documentation/devicemanagement/sync-devices)
- Fleet ADE integration: [`server/mdm/apple/apple_mdm.go`](https://github.com/fleetdm/fleet/blob/main/server/mdm/apple/apple_mdm.go)
- DEP sync tests: [`server/service/integration_mdm_dep_test.go`](https://github.com/fleetdm/fleet/blob/main/server/service/integration_mdm_dep_test.go)
- NanoDEP syncer: [`server/mdm/nanodep/sync`](https://github.com/fleetdm/fleet/tree/main/server/mdm/nanodep/sync)