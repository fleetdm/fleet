# iOS MDM Policy Engine -- Phase 2 MVP Scope

## What Already Exists (Phase 1 POC)

### Types (`server/fleet/mdm_policies.go`)
- `MDMPolicyCheckOperator` -- 12 operators (eq, neq, gt, lt, gte, lte, contains, not_contains, version_gte, version_lte, exists, not_exists)
- `MDMPolicyCheckSource` -- DeviceInformation, SecurityInfo, InstalledApplicationList
- `MDMPolicyCheck` struct -- field, operator, expected, source
- `MDMPolicyDefinition` -- checks slice with AND logic
- `DeviceStateEntry` -- value, source, observed_at
- `DeviceStateStore` interface -- UpdateDeviceState, GetDeviceState
- `MDMPolicyResult` -- host_id, policy_id, passes, err, timestamp

### Evaluator (`server/service/mdm_policy_evaluator.go`)
- `EvaluateMDMPolicy()` -- pure function, fully tested
- `compareValues()` -- routes to string, numeric, or version comparison
- `compareNumeric()` -- float parsing with lexicographic fallback
- `compareVersions()` -- semver-style dot-split comparison

### Parsers (`server/mdm/apple/parsers.go`)
- `ParseDeviceInformationResponse()` -- plist to flat map
- `ParseSecurityInfoResponse()` -- plist to flat map
- `ParseInstalledApplicationListResponse()` -- plist to flat map, keyed by bundle ID
- `flattenMap()` -- recursive dot-notation flattener
- `toStringValue()` -- universal plist value to string converter

### In-Memory Store (`server/fleet/mdm_device_state.go`)
- `InMemoryDeviceStateStore` -- sync.RWMutex + nested map
- `UpdateDeviceState()` -- merge-on-write semantics
- `GetDeviceState()` -- returns copy to prevent races

### Architecture Document
- Complete data flow diagram
- Policy check mapping tables (40+ compliance questions mapped to MDM commands)
- BYOD limitations documented
- iOS version fragmentation strategy
- DDM integration plan for Phase 3
- Integration with all existing automations via `policy_membership` table

### Existing Infrastructure to Leverage
- **`policies.type` column already exists** -- values are "dynamic" (default) and "patch". We add "mdm".
- **`PolicySpec` already supports `type`** -- though only "dynamic" and "patch" currently.
- **`CronAppleMDMIPhoneIPadRefetcher` already exists** -- sends DeviceInformation to iOS/iPadOS devices periodically. The response handler (`handleRefetch`) already parses and stores some device info. This is the natural extension point.
- **MDM Commander already has** `DeviceInformation()` and `InstalledApplicationList()` methods. SecurityInfo is missing.
- **`RecordPolicyQueryExecutions()`** accepts `map[uint]*bool` results -- exactly what we need to write MDM policy results into `policy_membership`.
- **`FlippingPoliciesForHost()`** -- existing flip detection feeds all automations.
- **Frontend `IPolicy` interface** already has a `type: string` field.
- **`GitOpsPolicySpec`** embeds `fleet.PolicySpec` which has a `Type` field.

---

## Phase 2 MVP Work Items

### BACKEND

#### B1. Add `PolicyTypeMDM` constant and `MDMCheckDefinition` column
**What:** Add `PolicyTypeMDM = "mdm"` constant. Add `mdm_check_definition` JSON column to `policies` table. Update `PolicyData` struct with `MDMCheckDefinition *json.RawMessage`. Update `PolicySpec` with optional `mdm_checks` field. Validation: when type=mdm, mdm_checks required, query ignored. When type=dynamic/patch, query required, mdm_checks ignored.

**Complexity:** M (1-2 days)
**Dependencies:** None
**Risk:** Low -- additive column, no data migration needed. `type` column already exists.

**Details:**
- Migration: `ALTER TABLE policies ADD COLUMN mdm_check_definition JSON DEFAULT NULL`
- Go types: add `MDMCheckDefinition *json.RawMessage` to `PolicyData` struct
- Add `PolicyTypeMDM = "mdm"` to constants in `policies.go`
- Validation in `PolicyPayload.Verify()` and `ModifyPolicyPayload.Verify()`
- Update `SavePolicy`, `NewGlobalPolicy`, `NewTeamPolicy` to handle the new column
- Update `ApplyPolicySpecs` to handle type=mdm specs

#### B2. `device_mdm_state` MySQL Table + Store Implementation
**What:** Create `device_mdm_state` table for persistent device state. Implement `MySQLDeviceStateStore` satisfying the existing `DeviceStateStore` interface.

**Complexity:** M (1-2 days)
**Dependencies:** None
**Risk:** Low -- new table, no existing data affected.

**Details:**
```sql
CREATE TABLE device_mdm_state (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    host_uuid  VARCHAR(255) NOT NULL,
    field_key  VARCHAR(255) NOT NULL,
    value      TEXT NOT NULL,
    source     VARCHAR(50) NOT NULL DEFAULT 'mdm_poll',
    observed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY idx_host_field (host_uuid, field_key),
    KEY idx_host_uuid (host_uuid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```
- `UpdateDeviceState` uses `INSERT ... ON DUPLICATE KEY UPDATE`
- `GetDeviceState` uses `SELECT WHERE host_uuid = ?`
- Wire into service layer replacing in-memory store

#### B3. SecurityInfo Command in MDM Commander
**What:** Add `SecurityInfo()` method to `MDMAppleCommander`. Fleet already has DeviceInformation and InstalledApplicationList but lacks SecurityInfo -- needed for passcode, encryption, and enrollment checks.

**Complexity:** S (half day)
**Dependencies:** None
**Risk:** Low -- follows exact same pattern as `DeviceInformation()`.

**Details:**
- Add `func (svc *MDMAppleCommander) SecurityInfo(ctx, hostUUIDs, cmdUUID)` following DeviceInformation pattern
- XML plist template with `RequestType: SecurityInfo`
- No query items needed (SecurityInfo returns all fields)

#### B4. MDM Polling Cron Job -- Extend Existing Refetcher
**What:** Extend `CronAppleMDMIPhoneIPadRefetcher` to also send SecurityInfo and InstalledApplicationList commands alongside existing DeviceInformation. Parse all three responses and write results to `device_mdm_state` via `MySQLDeviceStateStore`.

**Complexity:** L (3-5 days)
**Dependencies:** B2, B3
**Risk:** Medium -- this is the integration point. Must handle:
  - Command ordering (three commands per device per poll cycle)
  - APNs push batching (don't overwhelm push notifications)
  - Response parsing in `handleRefetch()` extension
  - Staleness tracking (observed_at timestamps)
  - Error handling for devices that don't respond

**Details:**
- The existing refetcher already sends DeviceInformation via `RefetchBaseCommandUUIDPrefix`. Extend it to also enqueue SecurityInfo and InstalledApplicationList.
- In the MDM command result handler, detect response type from command UUID prefix and route to appropriate parser.
- After parsing, call `deviceStateStore.UpdateDeviceState(hostUUID, parsedEntries)`.
- Configurable poll interval (default 1 hour, matching existing refetcher cadence).
- Add metric/logging for poll cycle duration and per-device response rates.

#### B5. Policy Evaluation Trigger -- On Device State Update
**What:** After device state is updated (B4), evaluate all MDM policies for that host and write results to `policy_membership` via `RecordPolicyQueryExecutions`.

**Complexity:** M (1-2 days)
**Dependencies:** B2, B4, B1
**Risk:** Medium -- must correctly bridge MDM results into the existing policy pipeline.

**Details:**
- After `UpdateDeviceState()`, load all MDM policies applicable to the host's team.
- For each policy, call `EvaluateMDMPolicy()` (already exists and tested).
- Convert results to `map[uint]*bool` format expected by `RecordPolicyQueryExecutions`.
- Call `RecordPolicyQueryExecutions()` -- this triggers `FlippingPoliciesForHost()`, which feeds all existing automations (webhooks, scripts, software install, VPP, conditional access, calendar).
- Handle the "not applicable" case: when a check field doesn't exist in device state (e.g., BYOD device can't report all apps), result should be nil (no pass/fail) rather than false.

#### B6. API Endpoints -- MDM Policy CRUD
**What:** Add REST endpoints for creating, reading, updating, and deleting MDM policies. These reuse the existing policy CRUD infrastructure but validate the MDM-specific `mdm_check_definition` JSON.

**Complexity:** M (1-2 days)
**Dependencies:** B1
**Risk:** Low -- extends existing endpoints rather than creating new ones.

**Details:**
- `POST /api/v1/fleet/policies` -- existing endpoint, extended to accept `type: "mdm"` with `mdm_checks` field
- `GET /api/v1/fleet/policies/{id}` -- existing endpoint, returns `mdm_check_definition` when type=mdm
- `PATCH /api/v1/fleet/policies/{id}` -- existing endpoint, allows modifying checks
- `DELETE /api/v1/fleet/policies/{id}` -- no changes needed
- `GET /api/v1/fleet/policies` -- list endpoint, optionally filter by type
- Validation: when type=mdm, validate each check's field against known DeviceState field catalog, validate operator is valid for field type
- Return `mdm_check_definition` as structured JSON in responses (not raw SQL query)

#### B7. MDM Policy Field Catalog Endpoint
**What:** API endpoint returning all available MDM fields, their types, valid operators, and descriptions. Used by both UI (for the structured builder) and GitOps (for validation).

**Complexity:** S (half day)
**Dependencies:** None
**Risk:** Low -- read-only, static data.

**Details:**
- `GET /api/v1/fleet/mdm/policy_fields` returns structured JSON:
  ```json
  [
    {
      "field": "DeviceInformation.OSVersion",
      "display_name": "OS Version",
      "type": "version",
      "valid_operators": ["version_gte", "version_lte", "eq", "neq"],
      "source": "DeviceInformation",
      "description": "The operating system version (e.g., 18.4.1)"
    }
  ]
  ```
- Static catalog defined in Go, not database-backed
- Approximately 30-40 fields from the architecture document mapping tables

---

### FRONTEND

#### F1. Policy Type Discriminator in Policy List and Navigation
**What:** Update the policies list page to display MDM policies alongside existing dynamic/patch policies. Add type indicator (icon/badge) and filter capability. When clicking an MDM policy, route to the MDM-specific editor instead of SQL editor.

**Complexity:** M (1-2 days)
**Dependencies:** B6 (API must return type field -- already does)
**Risk:** Low -- `IPolicy.type` field already exists in the TypeScript interface.

**Details:**
- `ManagePoliciesPage.tsx` -- add type filter dropdown (All / Query-based / MDM)
- `PoliciesTableConfig.tsx` -- add type column or indicator icon
- Route MDM policies to a new editor component instead of `QueryEditor.tsx`
- Update `IPolicyFormData.type` to include "mdm" option

#### F2. MDM Policy Creation Form -- Structured Check Builder
**What:** New form component for creating MDM policies. Instead of a SQL editor, presents a structured builder where users add checks by selecting field, operator, and expected value from dropdowns.

**Complexity:** L (3-5 days)
**Dependencies:** B7 (field catalog API), F1
**Risk:** Medium -- new UI paradigm. Must be intuitive for IT admins who think in compliance terms, not code.

**Details:**
- New component: `MDMPolicyForm.tsx` (parallel to existing `PolicyForm.tsx`)
- Field selector: dropdown populated from B7 catalog endpoint, grouped by category (Security, Device Info, Applications)
- Operator selector: filtered to valid operators for selected field type
- Expected value input: contextual (version picker for version fields, boolean toggle for bool fields, text input for strings)
- "Add check" button to add multiple AND conditions
- Preview panel showing human-readable summary: "OS version >= 17.0 AND Passcode is present AND com.slack.Slack is installed"
- Reuse existing policy metadata fields (name, description, resolution, critical, team, platform)
- Platform locked to "ios,ipados" for MDM policies

#### F3. Template Library -- Pre-built iOS Compliance Checks
**What:** Library of 15-20 pre-built MDM policy templates users can select from. Each template has a name, description, and pre-configured checks that can be customized.

**Complexity:** M (1-2 days)
**Dependencies:** F2
**Risk:** Low -- static data, straightforward UI.

**Details:**
Templates (examples):
1. Passcode Required -- `SecurityInfo.PasscodePresent eq true`
2. Passcode Compliant -- `SecurityInfo.PasscodeCompliant eq true`
3. Minimum OS Version (iOS 17+) -- `DeviceInformation.OSVersion version_gte 17.0`
4. Minimum OS Version (iOS 18+) -- `DeviceInformation.OSVersion version_gte 18.0`
5. Device Supervised -- `DeviceInformation.IsSupervised eq true`
6. Encryption Capable -- `SecurityInfo.HardwareEncryptionCaps gt 0`
7. Find My Enabled -- `DeviceInformation.IsDeviceLocatorServiceEnabled eq true`
8. iCloud Backup Enabled -- `DeviceInformation.IsCloudBackupEnabled eq true`
9. Storage Not Critical (>1GB) -- `DeviceInformation.AvailableDeviceCapacity gt 1`
10. Required App: Slack -- `InstalledApplicationList.com.slack.Slack.Identifier exists`
11. Required App: Microsoft Teams -- `InstalledApplicationList.com.microsoft.skype.teams.Identifier exists`
12. No TestFlight Apps -- combination check
13. DEP Enrolled -- `SecurityInfo.ManagementStatus.EnrolledViaDEP eq true`
14. Not Roaming -- `DeviceInformation.IsRoaming eq false`
15. Activation Lock Enabled -- `DeviceInformation.IsActivationLockEnabled eq true`

- Template selector modal when creating new MDM policy
- "Start from template" vs "Start from scratch" option
- Templates populate the check builder form, user can modify before saving

#### F4. Policy Results Display with Freshness Timestamps
**What:** Update policy results page to show MDM-specific information: when data was last collected (observed_at), data source (MDM poll), and freshness indicator.

**Complexity:** M (1-2 days)
**Dependencies:** B6
**Risk:** Low -- extends existing `PolicyResults` component.

**Details:**
- `PolicyResults.tsx` -- add "Last checked" timestamp for MDM policies
- Show staleness warning if data is older than 2x poll interval
- For MDM policies, show "Evaluated via MDM" indicator instead of "Evaluated via osquery"
- Pass/fail counts work identically (same `policy_membership` table)

#### F5. Three-State Results (Pass / Fail / Not Applicable)
**What:** MDM policies can have a "not applicable" state (e.g., BYOD device cannot report all installed apps). Display this third state in the UI.

**Complexity:** S (half day)
**Dependencies:** F4
**Risk:** Low -- additive UI change. The `policy_membership` table already supports null results.

**Details:**
- Add "N/A" or "Not Applicable" indicator in host policy response
- Gray styling for N/A (distinct from green pass and red fail)
- Tooltip explaining why: "This check is not applicable to user-enrolled (BYOD) devices"
- Existing `PolicyStatusResponse = "pass" | "fail" | ""` -- the empty string already maps to N/A

---

### GITOPS

#### G1. MDM Policy Definition in GitOps YAML Schema
**What:** Extend `GitOpsPolicySpec` to support MDM policies. When `type: mdm`, the `query` field is ignored and `mdm_checks` is required.

**Complexity:** M (1-2 days)
**Dependencies:** B1
**Risk:** Low -- `GitOpsPolicySpec` already embeds `PolicySpec` which has `Type`.

**Details:**
YAML format:
```yaml
policies:
  - name: iOS Passcode Required
    type: mdm
    platform: ios,ipados
    description: Ensures all iOS devices have a passcode set
    resolution: Go to Settings > Face ID & Passcode and set a passcode
    critical: true
    mdm_checks:
      - field: SecurityInfo.PasscodePresent
        operator: eq
        expected: "true"
      - field: DeviceInformation.OSVersion
        operator: version_gte
        expected: "17.0"
```
- Extend `GitOpsPolicySpec` with `MDMChecks []MDMPolicyCheck` field
- Extend `parsePolicies()` in `pkg/spec/gitops.go` to validate MDM checks
- Validation: field must be in known catalog, operator must be valid for field
- Error on `type: mdm` with `query` set, or `type: dynamic` with `mdm_checks` set

#### G2. GitOps Apply + Dry-Run Support for MDM Policies
**What:** Extend `doGitOpsPolicies()` in `server/service/client.go` to handle MDM policy specs. The dry-run path validates check definitions. The apply path creates/updates MDM policies.

**Complexity:** M (1-2 days)
**Dependencies:** G1, B6
**Risk:** Low -- follows existing pattern exactly. The `ApplyPolicySpecs` already handles type dispatch.

**Details:**
- In `doGitOpsPolicies()`, MDM policy specs flow through the same `ApplyPolicySpecs` endpoint
- Dry-run: validate field names against catalog, validate operators, return validation errors
- Apply: create/update policies with `mdm_check_definition` JSON column
- Deletion: MDM policies not in spec get deleted (same as existing policy sync behavior)
- `fleetctl gitops --dry-run` shows "Would create MDM policy: iOS Passcode Required"

---

### TESTING

#### T1. Backend Integration Tests
**What:** Integration tests for the full MDM policy pipeline: create policy via API, trigger poll, parse response, evaluate, check policy_membership results.

**Complexity:** L (3-5 days)
**Dependencies:** B1-B6 all complete
**Risk:** Medium -- requires MDM test infrastructure (mock APNs, test devices).

**Details:**
- Test MDM policy CRUD via API
- Test device state store MySQL operations
- Test evaluation trigger writes correct results to policy_membership
- Test flip detection fires automations for MDM policy pass->fail transitions
- Test BYOD handling (not_applicable results)
- Test field catalog endpoint
- Existing test infrastructure: `server/service/integration_mdm_test.go` patterns

#### T2. Frontend Tests
**What:** Component tests for MDM policy form, template library, and results display.

**Complexity:** M (1-2 days)
**Dependencies:** F1-F5 complete
**Risk:** Low -- follows existing test patterns.

**Details:**
- MDMPolicyForm renders correctly, validates input
- Template selection populates form
- Type filter works in policy list
- Three-state display renders correctly
- Test files: `MDMPolicyForm.tests.tsx`, etc.

#### T3. GitOps Integration Tests
**What:** Test `fleetctl gitops` apply and dry-run with MDM policies in YAML.

**Complexity:** M (1-2 days)
**Dependencies:** G1, G2 complete
**Risk:** Low -- extends existing gitops test suite.

**Details:**
- Test YAML parsing with mdm_checks
- Test dry-run validation catches invalid fields/operators
- Test apply creates MDM policies correctly
- Test sync deletes removed MDM policies
- Extend existing tests in `server/service/integration_enterprise_test.go`

---

## Effort Estimates

### Backend: 10.5-15.5 days (2-3 engineer-weeks)

| Item | Estimate | Days |
|------|----------|------|
| B1. PolicyTypeMDM + mdm_check_definition column | M | 1-2 |
| B2. device_mdm_state table + MySQL store | M | 1-2 |
| B3. SecurityInfo command | S | 0.5 |
| B4. MDM polling cron extension | L | 3-5 |
| B5. Policy evaluation trigger | M | 1-2 |
| B6. API endpoints | M | 1-2 |
| B7. Field catalog endpoint | S | 0.5 |
| T1. Backend integration tests | L | 3-5 |
| **Backend Total** | | **10.5-18.5** |

### Frontend: 7.5-12.5 days (1.5-2.5 engineer-weeks)

| Item | Estimate | Days |
|------|----------|------|
| F1. Policy type discriminator | M | 1-2 |
| F2. MDM policy creation form | L | 3-5 |
| F3. Template library | M | 1-2 |
| F4. Results display with freshness | M | 1-2 |
| F5. Three-state results | S | 0.5 |
| T2. Frontend tests | M | 1-2 |
| **Frontend Total** | | **7.5-13.5** |

### GitOps: 3-5 days (0.5-1 engineer-week)

| Item | Estimate | Days |
|------|----------|------|
| G1. YAML schema extension | M | 1-2 |
| G2. Apply + dry-run support | M | 1-2 |
| T3. GitOps tests | M | 1-2 |
| **GitOps Total** | | **3-6** |

### TOTAL: 21-37 days = 4-7.5 engineer-weeks

**Realistic estimate with integration overhead: 6 engineer-weeks** (buffer for cross-layer integration issues, code review cycles, and design iteration on the MDM policy builder UI).

---

## Parallelization Plan

### Week 1-2: Foundation (Backend + Frontend start together)

**Backend Engineer:**
- B1: PolicyTypeMDM constant + migration (day 1)
- B2: device_mdm_state table + MySQL store (days 2-3)
- B3: SecurityInfo command (day 3)
- B7: Field catalog endpoint (day 4)

**Frontend Engineer:**
- F1: Policy type discriminator (days 1-2) -- uses existing type field, no API dependency
- F2: MDM policy creation form (days 3-6) -- can start with mocked field catalog data

**GitOps Engineer (or Backend Engineer in parallel):**
- G1: YAML schema extension (days 1-2)

### Week 3-4: Core Pipeline + UI Completion

**Backend Engineer:**
- B4: MDM polling cron extension (days 1-4) -- depends on B2, B3
- B5: Policy evaluation trigger (day 5) -- depends on B4
- B6: API endpoints (days 6-7) -- depends on B1

**Frontend Engineer:**
- F2: Complete MDM policy form (days 1-2, finishing up)
- F3: Template library (days 3-4)
- F4: Results display + freshness (days 5-6)
- F5: Three-state results (day 7)

### Week 5-6: Integration + Testing

**Backend Engineer:**
- T1: Backend integration tests (days 1-5)
- G2: GitOps apply/dry-run (days 3-4) -- can overlap with tests

**Frontend Engineer:**
- T2: Frontend tests (days 1-2)
- Integration testing with real backend (days 3-5)

**GitOps:**
- T3: GitOps integration tests (days 1-2)

### Parallel Execution Summary

Three work streams can run simultaneously:
1. **Backend** -- B1/B2/B3/B7 have no cross-dependencies, then B4/B5/B6 chain
2. **Frontend** -- F1 can start day 1, F2 can use mocked data, F3/F4/F5 chain from F2
3. **GitOps** -- G1 can start day 1 (only needs B1 types), G2 needs B6

With 2 engineers (1 backend, 1 frontend), the critical path is approximately 5-6 weeks.
With 3 engineers (dedicated GitOps person), the critical path stays at 5-6 weeks but GitOps is done sooner.

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| APNs rate limiting during polling | Medium | High | Batch commands, stagger polling across fleet, honor Apple retry-after headers |
| iOS devices not responding to poll commands | Medium | Medium | Staleness tracking with observed_at; UI shows "last checked" with warning for stale data |
| MDM policy builder UX complexity | Medium | High | Start with templates (F3) so users rarely need to build from scratch; iterate on builder UX based on feedback |
| Field catalog completeness | Low | Medium | Start with 30-40 most common fields; extensible catalog allows adding more without migration |
| BYOD devices returning partial data | Low | Medium | Three-state results (F5) handle this gracefully; documented in architecture doc |
| Performance of evaluation on large fleets | Low | Medium | Evaluation is per-host, per-poll-cycle; same as osquery policy execution. device_mdm_state indexed by host_uuid |
| GitOps backward compatibility | Low | Low | mdm_checks is optional field; existing YAML files work unchanged |

---

## Out of Scope for Phase 2 MVP

1. **DDM status report integration** -- Phase 3
2. **ProfileList, CertificateList, Restrictions commands** -- Phase 4 (only DeviceInformation, SecurityInfo, InstalledApplicationList in MVP)
3. **BYOD-aware policy evaluation** -- policies evaluate against available data, but no special BYOD enrollment type filtering
4. **Real-time compliance** (push-based) -- MVP uses polling only
5. **Custom MDM command support** -- users cannot add arbitrary MDM queries
6. **Per-check failure drill-down** -- F4 shows overall pass/fail; individual check breakdown is Phase 3 UI enhancement
7. **MDM policy analytics/trends** -- existing policy stats aggregation covers basic counts
