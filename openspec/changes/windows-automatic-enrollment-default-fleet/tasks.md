# Tasks: Default fleet for Windows automatic enrollment

## 1. Storage and types

- [ ] 1.1 DB migration (`make migration`): create `windows_automatic_enrollment_config` (single row, `team_id` nullable FK to `teams(id)` ON DELETE SET NULL) and add nullable `hardware_serial` to `mdm_windows_enrollments`; migration test per convention
- [ ] 1.2 Datastore methods: get/set Windows automatic enrollment default fleet (hydrating fleet name via join), store serial on unlinked enrollment, look up unlinked automatic enrollment by serial; regenerate `server/mock/datastore_mock.go`; run `go test ./server/service/` for uninitialized-mock crashes
- [ ] 1.3 Add `WindowsAutomaticEnrollment` (with `DefaultFleet` string) to `fleet.MDM` app config types following `AppleBusinessManager` optjson conventions

## 2. Config API, validation, activity

- [ ] 2.1 `ModifyAppConfig`: validate `mdm.windows_automatic_enrollment.default_fleet` (existing fleet name, Premium license, empty string clears, omitted key no-op), resolve name to id, persist to the new table
- [ ] 2.2 Hydrate the setting into `GET /config` responses (Premium only)
- [ ] 2.3 New activity type `edited_windows_automatic_enrollment_default_fleet` (`fleet_id`, `fleet_name`) in `server/fleet/activities.go`, emitted only on change; update `docs/Contributing/reference/audit-logs.md` generation if needed (doc already drafted in #49594)
- [ ] 2.4 Authorization: global admin and gitops role only (policy.rego if new authz subject is introduced; prefer riding app config authz)
- [ ] 2.5 Clear the setting when the referenced fleet is deleted (EE delete-team path plus FK), matching ABM `CleanRemovedTeam` behavior
- [ ] 2.6 Service and integration tests: set/clear/no-op/invalid name/Free tier/activity emitted-once; team deletion clears setting

## 3. Enrollment-time assignment

- [ ] 3.1 `tryLinkUnlinkedEnrollmentFromDevDetail`: persist non-placeholder serial on the enrollment row when host lookup is NotFound
- [ ] 3.2 Shared assignment helper invoked from `LinkWindowsHostMDMEnrollment`: automatic enrollment (valid UPN) + default fleet configured + host has no fleet + host new in this enrollment cycle (row created at/after the enrollment row; reverse-link path knows creation directly) → transfer host with full manual-transfer side effects (`BulkSetPendingMDMHostProfiles`, encryption key cleanup); no transferred_hosts activity
- [ ] 3.3 Reverse link in the Windows orbit enrollment path: after host create/match, look up unlinked automatic enrollment by serial on the primary DB, link, and assign default fleet before returning (so `SetupExperienceInit` sees the new fleet)
- [ ] 3.4 Tests: new host → default fleet; existing host keeps fleet; host parked in No team stays in No team on re-enroll; deleted-then-re-enrolled host → default fleet; programmatic enrollment untouched; no default → No team; changing default affects only later enrollments; placeholder serial falls back to DevDetail/osquery link paths
- [ ] 3.5 ESP ordering test (integration-mdm): default fleet with setup experience items + empty No team → items enqueued for default fleet, no release deadlock

## 4. GitOps

- [ ] 4.1 `pkg/spec` + `cmd/fleetctl/fleetctl/gitops.go`: parse `org_settings.mdm.windows_automatic_enrollment.default_fleet` (object form), validate against declared plus existing fleets, apply after team creation using the ABM/VPP deferred mechanism, honor dry-run
- [ ] 4.2 `fleetctl generate-gitops`: export the current value
- [ ] 4.3 fleetctl tests: apply with same-run fleet, unknown fleet failure, dry run, omitted key no-op, export round-trip

## 5. Frontend

- [ ] 5.1 `frontend/interfaces/mdm.ts` + `frontend/services/entities/`: add `windows_automatic_enrollment` to config types and PATCH payload
- [ ] 5.2 WindowsMdmPage: "Automatic enrollment" section with Default fleet dropdown (Unassigned default, helper text + Learn more per Figma), disabled + tooltip when `windows_entra_tenant_ids` is empty, `GitOpsModeTooltipWrapper` read-only, Premium gating
- [ ] 5.3 WindowsMdmPage: replace End user experience radios with "Turn on MDM programmatically" toggle (inverted `enable_turn_on_windows_mdm_manually`), tooltip + Learn more link per Figma
- [ ] 5.4 WindowsMdmPage: "Migration" heading above the auto-migration checkbox
- [ ] 5.5 Activity feed rendering for `edited_windows_automatic_enrollment_default_fleet` ("edited the default fleet for Windows automatic enrollment hosts to <fleet>.") and dashboard type filter label "Edited automatic enrollment default fleet: Windows"
- [ ] 5.6 Component tests (`WindowsMdmPage.tests.tsx`) for the new section states; run frontend-reviewer expectations (`yarn lint`, `yarn test`)

## 6. Docs and follow-ups

- [ ] 6.1 Fix reference docs on `docs-v4.91.0`: REST API doc (name not id-string; remove team-level `windows_automatic_enrollment` placements), YAML doc (object, not list); done ahead of implementation on a branch off `docs-v4.91.0`. Guide PR #48658 key names (`windows_automatic_enrollment.default_fleet`) still need fixing in that PR
- [ ] 6.2 Changes file in `changes/`
- [ ] 6.3 Confirm with product (melpike): all automatic enrollments vs Autopilot-OOBE-only
- [ ] 6.4 Fill in the issue's engineering checklist (test plan finalize, DB schema note, risk level)
