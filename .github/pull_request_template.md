# Checklist for submitter

If some of the following don't apply, delete the relevant line.

- [ ] Changes file added for user-visible changes (in `changes/` and/or `orbit/changes/`).
- [ ] Documented any API changes (docs/Using-Fleet/REST-API.md or docs/Contributing/API-for-contributors.md)
- [ ] Documented any permissions changes
- [ ] Input data is properly validated, `SELECT *` is avoided, SQL injection is prevented (using placeholders for values in statements)
- [ ] Added support on fleet's osquery simulator `cmd/osquery-perf` for new osquery data ingestion features.
- [ ] Added/updated tests
- [ ] Manual QA for all new/changed functionality
  - For Orbit and Fleet Desktop changes:
    - [ ] Manual QA must be performed in the three main OSs, macOS, Windows and Linux.
    - [ ] Auto-update manual QA, from released version of component to new version (see [tools/tuf/test](../tools/tuf/test/README.md)).
