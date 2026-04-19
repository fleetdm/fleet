# Tasks

## 1. Database migration
- [ ] 1.1 Add awaiting_configuration TINYINT NOT NULL DEFAULT 0 and awaiting_configuration_at DATETIME(6)
      NULL to mdm_windows_enrollments
- [ ] 1.2 Update server/datastore/mysql/schema.sql
- [ ] 1.3 Migration test covering default values on existing rows

## 2. Config struct and rename
- [ ] 2.1 Rename MacOSSetup.RequireAllSoftware to RequireAllSoftwareMacOS
- [ ] 2.2 Add RequireAllSoftwareWindows *bool to the setup experience struct
- [ ] 2.3 Add renameto tag so require_all_software deserializes to the macOS field
- [ ] 2.4 Support both macos_setup and setup_experience YAML parent keys for one release

## 3. REST API and GitOps
- [ ] 3.1 Extend MDMAppleSetupPayload with RequireAllSoftwareWindows *bool
- [ ] 3.2 Wire the EE handler in ee/server/service/mdm.go to persist it
- [ ] 3.3 Team config apply path in ee/server/service/teams.go handles the new field
- [ ] 3.4 Validate that Windows MDM is enabled before accepting the field
- [ ] 3.5 Update cmd/fleetctl/fleetctl/testdata/ golden files
- [ ] 3.6 Document the field in docs/REST API/rest-api.md

## 4. Detection at enrollment
- [ ] 4.1 Classify the enrollment source in storeWindowsMDMEnrolledDevice using the binary security token type
- [ ] 4.2 Set awaiting_configuration=1 and awaiting_configuration_at=NOW() when source is Autopilot (JWT or
      WSTEP) and NotInOobe=false
- [ ] 4.3 Set awaiting_configuration=0 otherwise
- [ ] 4.4 Verify ON DUPLICATE KEY UPDATE correctly updates the columns on re-enrollment
- [ ] 4.5 Integration test: fleetd programmatic enroll leaves awaiting_configuration=0
- [ ] 4.6 Integration test: simulated Autopilot JWT enroll with NotInOobe=false sets awaiting_configuration=1
- [ ] 4.7 Integration test: Autopilot re-enroll post-OOBE resets to 0

## 5. Frontend config plumbing
- [ ] 5.1 Add field to frontend/interfaces/config.ts and frontend/interfaces/team.ts
- [ ] 5.2 Add field to frontend/__mocks__/configMock.ts
- [ ] 5.3 Add mdmAPI.updateRequireAllSoftwareWindows in frontend/services/entities/mdm.ts

## 6. DMClient and EnrollmentStatusTracking SyncML helpers
- [ ] 6.1 Add DMClient CSP path constants for ExpectedPolicies, TimeoutUntilSyncFailure, CustomErrorText, and
      BlockInStatusPage in server/mdm/microsoft/syncml/syncml.go
- [ ] 6.2 Add EnrollmentStatusTracking CSP helpers for software tracking entries
- [ ] 6.3 Add SCEP certificate ESP node helpers
- [ ] 6.4 Unit tests for each helper with golden SyncML XML assertions

## 7. Single-host profile listing
- [ ] 7.1 Add hostUUID string parameter to ListMDMWindowsProfilesToInstall mirroring the macOS helper
- [ ] 7.2 Update existing callers to pass empty string
- [ ] 7.3 Test for single-host filtering

## 8. Initial ESP command generation
- [ ] 8.1 On getManagementResponse with awaiting_configuration=1 and host_uuid set, query setup software for
      the host's team
- [ ] 8.2 If no items and orbit-enrolled less than 10 minutes ago, do nothing
- [ ] 8.3 If no items and at least 10 minutes, proceed to release path
- [ ] 8.4 If items exist, fetch applicable profiles and current setup_experience_status_results
- [ ] 8.5 Build initial SyncML command: profile top-most nodes, SCEP nodes, software tracking entries
- [ ] 8.6 Set TimeoutUntilSyncFailure to 3 hours
- [ ] 8.7 Enqueue as stored command; transition row to awaiting_configuration=2
- [ ] 8.8 Integration test: first checkin after orbit registers produces the expected SyncML

## 9. Ongoing status sync
- [ ] 9.1 On getManagementResponse with awaiting_configuration=2, fetch current setup_experience_status_results
- [ ] 9.2 Build EnrollmentStatusTracking entries reflecting current state of each item
- [ ] 9.3 Return inline in SyncML response; do not enqueue
- [ ] 9.4 Integration test: three consecutive checkins return consistent status with no duplicate stored commands

## 10. Frontend checkbox
- [ ] 10.1 Add checkbox block for platform windows in InstallSoftwareForm.tsx at the existing macOS block
- [ ] 10.2 Add requireAllSoftwareWindows state and handler
- [ ] 10.3 Pass savedRequireAllSoftwareWindows prop from parent
- [ ] 10.4 Extend shouldUpdateRequireAll to trigger for Windows
- [ ] 10.5 Wire onClickSave to call updateRequireAllSoftwareWindows
- [ ] 10.6 Update macOS checkbox label and tooltip to match the new Windows copy
- [ ] 10.7 Verify canceled_setup_experience appears in the activity filter dropdown and feed
- [ ] 10.8 Test coverage for the new checkbox

## 11. Timeout path
- [ ] 11.1 Compute age = now - awaiting_configuration_at in getManagementResponse state 2
- [ ] 11.2 If age > 3h, call CancelPendingSetupExperienceSteps
- [ ] 11.3 If require_all_software_windows=true, enqueue final ESP with BlockInStatusPage=true and
      CustomErrorText
- [ ] 11.4 If require_all_software_windows=false, enqueue final ESP marking remaining items failed but
      allowing the user through
- [ ] 11.5 Emit canceled_setup_experience activity using display_name with fallback to name for the first
      failed software item
- [ ] 11.6 Set awaiting_configuration=0

## 12. Blocking failure path
- [ ] 12.1 If require_all_software_windows=true and any profile or software is in failure state, cancel
      remaining, emit activity, enqueue blocking ESP, set awaiting_configuration=0

## 13. Release path
- [ ] 13.1 If all items terminal, enqueue final ESP; include CustomErrorText if any errors exist; set
      awaiting_configuration=0

## 14. macOS activity text update
- [ ] 14.1 Add "End user was asked to restart" to macOS canceled_setup_experience activity text

## 15. Docs and QA
- [ ] 15.1 Update docs/Using Fleet/Windows & Linux setup experience.md with SHIFT+F10 breakglass and remote
      fleet script instructions
- [ ] 15.2 Load test: verify no additional steady-state load from the new columns and checkin paths
- [ ] 15.3 Execute all test plan items from issue #38785
