# Checklist for submitter

If some of the following don't apply, delete the relevant line.

- [ ] Changes file added for user-visible changes in `changes/`, `orbit/changes/` or `ee/fleetd-chrome/changes`.
  See [Changes files](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/guides/committing-changes.md#changes-files) for more information.

- [ ] Input data is properly validated, `SELECT *` is avoided, SQL injection is prevented (using placeholders for values in statements)
- [ ] If paths of existing endpoints are modified without backwards compatibility, checked the frontend/CLI for any necessary changes

## Testing

- [ ] Added/updated automated tests
- [ ] Where appropriate, [automated tests simulate multiple hosts and test for host isolation](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/reference/patterns-backend.md#unit-testing) (updates to one hosts's records do not affect another)

- [ ] QA'd all new/changed functionality manually

For unreleased bug fixes in a release candidate, one of:

- [ ] Confirmed that the fix is not expected to adversely impact load test results
- [ ] Alerted the release DRI if additional load testing is needed

## Database migrations

- [ ] Checked table schema to confirm autoupdate
- [ ] Checked schema for all modified table for columns that will auto-update timestamps during migration.
- [ ] Confirmed that updating the timestamps is acceptable, and will not cause unwanted side effects.
- [ ] Ensured the correct collation is explicitly set for character columns (`COLLATE utf8mb4_unicode_ci`).

## New Fleet configuration settings

- [ ] Setting(s) is/are explicitly excluded from GitOps

If you didn't check the box above, follow this checklist for GitOps-enabled settings:

- [ ] Verified that the setting is exported via `fleetctl generate-gitops`
- [ ] Verified the setting is documented in a separate PR to [the GitOps documentation](https://github.com/fleetdm/fleet/blob/main/docs/Configuration/yaml-files.md#L485)
- [ ] Verified that the setting is cleared on the server if it is not supplied in a YAML file (or that it is documented as being optional)
- [ ] Verified that any relevant UI is disabled when GitOps mode is enabled

## fleetd/orbit/Fleet Desktop

- [ ] Verified compatibility with the latest released version of Fleet (see [Must rule](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/workflows/fleetd-development-and-release-strategy.md))
- [ ] If the change applies to only one platform, confirmed that `runtime.GOOS` is used as needed to isolate changes
- [ ] Verified that fleetd runs on macOS, Linux and Windows
- [ ] Verified auto-update works from the released version of component to the new version (see [tools/tuf/test](../tools/tuf/test/README.md))
