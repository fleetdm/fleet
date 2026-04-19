# Design: Windows setup experience, cancel if software fails

## Technical approach
- Reuse setup_experience_status_results and the canceled_setup_experience activity; no parallel Windows tables
- Store awaiting_configuration on mdm_windows_enrollments directly, not a sibling table, since the SyncML
  checkin already joins that row
- Tri-state awaiting_configuration: 0 not awaiting, 1 ESP not yet issued, 2 ESP issued and in progress
- State 1 to 2 transition enqueues a single SyncML command so the outcome is stored and retried
- State 2 checkins return ESP status inline in the SyncML response, not stored, so the 3 hour window does not
  explode the commands table
- require_all_software splits into require_all_software_macos and require_all_software_windows; legacy name
  remains as a YAML/JSON alias via the existing renameto struct tag
- ListMDMWindowsProfilesToInstall gets a hostUUID parameter mirroring the macOS helper
