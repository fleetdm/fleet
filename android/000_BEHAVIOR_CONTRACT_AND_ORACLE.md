# Android Osquery: Behavior Contract and Oracle (Human Review)

## Why this file exists
This PR is large. The goal of this document is to give reviewers a stable behavior contract and a practical oracle, so humans review expected system behavior and safety invariants instead of diff volume.

## System behavior contract

### Scope
This contract covers Android osquery behavior implemented in this PR branch:
- enrollment and re-enrollment behavior
- distributed query read/execute/write loop
- SQL surface and table execution semantics
- osquery table exposure for Android
- security-sensitive behavior for transport and logging

### Actors and interfaces
- Fleet server endpoints:
  - `POST /api/fleet/orbit/enroll`
  - `POST /api/fleet/orbit/config`
  - `POST /api/v1/osquery/distributed/read`
  - `POST /api/v1/osquery/distributed/write`
- Android agent components:
  - `ApiClient`
  - `DistributedCheckinWorker`
  - `OsqueryQueryEngine`
  - `TableRegistry` + registered Android tables

### Contract rules (must remain true)

#### C1. Enrollment identity ownership
If enrollment identity changes (`enroll_secret`, `hardware_uuid`, or `server_url`), stored node key must be cleared before next check-in so host identity is not reused incorrectly.

#### C2. 401 re-enrollment behavior
On `HTTP 401` for Fleet/orbit calls, client clears node key and retries once via re-enrollment path. Non-401 failures must not trigger re-enrollment.

#### C3. URL safety and normalization
`server_url` must be validated before network requests:
- scheme must be `http` or `https`
- non-debug policy requires `https`
- reject user-info, query, fragment, and non-root path
- normalize to canonical origin (`scheme://authority`)

#### C4. Request resiliency
Failure to obtain optional device ID header must not crash requests; requests proceed without `X-Fleet-Device-Id`.

#### C5. Distributed query loop semantics
For each check-in cycle:
1. read queries from Fleet
2. execute each SQL query through osquery engine
3. write result rows back to Fleet
If query execution fails/unsupported, return empty rows for that query to clear retry churn.

#### C6. SQL surface is intentionally narrow
Supported SQL surface:
- `SELECT *` or explicit column list
- single `FROM <table>`
- optional `WHERE` with `AND`-combined terms
- operators: `=` and `LIKE`
Out of scope: joins, aggregates, subqueries, group/order semantics.

#### C7. Table strictness
A table may only emit declared columns; unknown columns are contract violation. Missing columns are filled as empty string.

#### C8. `android_logcat` privacy gate
`android_logcat` must be disabled by default and only active with managed config key `enable_android_logcat_table=true`. When active, output is constrained:
- Fleet agent tags only
- capped row count
- sensitive token redaction patterns
- bounded execution timeout

### Contracted Android table set
Current registered tables:
- `installed_apps`
- `app_permissions`
- `os_version`
- `osquery_info`
- `certificates`
- `device_info`
- `network_interfaces`
- `battery`
- `wifi_networks`
- `system_properties`
- `android_logcat`

## Oracle for reviewers

### Oracle A: enrollment and key lifecycle
Expected outcomes:
- first call without node key enrolls then succeeds
- 401 causes clear-key + re-enroll + retry success
- non-401 does not re-enroll
- identity change forces fresh enrollment
Evidence:
- unit suite: `ApiClientReenrollTest`

### Oracle B: URL and transport hardening
Expected outcomes:
- invalid origin forms rejected pre-request
- HTTPS enforced under non-debug policy
- valid origin normalized and used for requests
Evidence:
- unit suite: `ApiClientReenrollTest` URL validation cases

### Oracle C: distributed query loop
Expected outcomes:
- each check-in does read -> execute -> write
- unsupported/failed queries produce empty rows (clearing behavior)
Evidence:
- runtime logs and manual device validation
- code path: `DistributedCheckinWorker` + `OsqueryQueryEngine`

### Oracle D: SQL contract
Expected outcomes:
- valid limited SQL executes deterministically
- unknown columns/tables fail fast and do not crash worker
Evidence:
- parser and engine behavior in `SqlParser` and `OsqueryQueryEngine`
- manual query validation in Fleet UI

### Oracle E: `android_logcat` data safety
Expected outcomes:
- no rows when feature flag absent/false
- rows only from whitelisted tags
- sensitive value patterns redacted
- execution does not crash agent on read/process failures
Evidence:
- code path: `AndroidLogcatTable`
- manual query verification on managed device

## Reviewer checklist (fast path)
1. Validate C1-C4 against unit test evidence (`ApiClientReenrollTest`).
2. Validate C5-C7 by one end-to-end run: query from Fleet UI and inspect writeback results.
3. Validate C8 by running `android_logcat` once with flag off (expect empty), then on (expect filtered/redacted rows).
4. Approve only if behavior and safety expectations match this contract, even if implementation details change later.
