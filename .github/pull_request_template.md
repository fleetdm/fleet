# Checklist for submitter

If some of the following don't apply, delete the relevant line.

<!-- Note that API documentation changes are now addressed by the product design team. -->

- [ ] Changes file added for user-visible changes in `changes/`, `orbit/changes/` or `ee/fleetd-chrome/changes`.
  See [Changes files](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/guides/committing-changes.md#changes-files) for more information.
- [ ] Input data is properly validated, `SELECT *` is avoided, SQL injection is prevented (using placeholders for values in statements)
- [ ] If paths of existing endpoints are modified without backwards compatibility, checked the frontend/CLI for any necessary changes
- [ ] If database migrations are included, checked table schema to confirm autoupdate
- For new Fleet configuration settings
  - [ ] Verified that the setting can be managed via GitOps, or confirmed that the setting is explicitly being excluded from GitOps.  If managing via Gitops:
    - [ ] Verified that the setting is exported via `fleetctl generate-gitops`
    - [ ] Added the setting to [the GitOps documentation](https://github.com/fleetdm/fleet/blob/main/docs/Configuration/yaml-files.md#L485)
    - [ ] Verified that the setting is cleared on the server if it is not supplied in a YAML file (or that it is documented as being optional)
    - [ ] Verified that any relevant UI is disabled when GitOps mode is enabled
- For database migrations:
  - [ ] Checked schema for all modified table for columns that will auto-update timestamps during migration.
  - [ ] Confirmed that updating the timestamps is acceptable, and will not cause unwanted side effects.
  - [ ] Ensured the correct collation is explicitly set for character columns (`COLLATE utf8mb4_unicode_ci`).
- [ ] Added/updated automated tests
  - [ ] Where appropriate, automated tests simulate multiple hosts and test for host isolation (updates to one hosts's records do not affect another.)
- [ ] Manual QA for all new/changed functionality
- For Orbit and Fleet Desktop changes:
   - [ ] Make sure fleetd is compatible with the latest released version of Fleet (see [Must rule](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/workflows/fleetd-development-and-release-strategy.md)).
   - [ ] Orbit runs on macOS, Linux and Windows. Check if the orbit feature/bugfix should only apply to one platform (`runtime.GOOS`).
   - [ ] Manual QA must be performed in the three main OSs, macOS, Windows and Linux.
   - [ ] Auto-update manual QA, from released version of component to new version (see [tools/tuf/test](../tools/tuf/test/README.md)).
- [ ] For unreleased bug fixes in a release candidate, confirmed that the fix is not expected to adversely impact load test results or alerted the release DRI if additional load testing is needed.
