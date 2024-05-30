# Checklist for submitter

If some of the following don't apply, delete the relevant line.

<!-- Note that API documentation changes are now addressed by the product design team. -->

- [ ] Changes file added for user-visible changes in `changes/`, `orbit/changes/` or `ee/fleetd-chrome/changes`.
  See [Changes files](https://fleetdm.com/docs/contributing/committing-changes#changes-files) for more information.
- [ ] Input data is properly validated, `SELECT *` is avoided, SQL injection is prevented (using placeholders for values in statements)
- [ ] Added support on fleet's osquery simulator `cmd/osquery-perf` for new osquery data ingestion features.
- [ ] Added/updated tests
- [ ] If database migrations are included, checked table schema to confirm autoupdate
- For database migrations:
  - [ ] Checked schema for all modified table for columns that will auto-update timestamps during migration.
  - [ ] Confirmed that updating the timestamps is acceptable, and will not cause unwanted side effects.
  - [ ] Ensured the correct collation is explicitly set for character columns (`COLLATE utf8mb4_unicode_ci`).
- [ ] Manual QA for all new/changed functionality
  - For Orbit and Fleet Desktop changes:
    - [ ] Manual QA must be performed in the three main OSs, macOS, Windows and Linux.
    - [ ] Auto-update manual QA, from released version of component to new version (see [tools/tuf/test](../tools/tuf/test/README.md)).
