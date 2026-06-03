## 1. Shared helper & ground-truth tests (do first)

- [ ] 1.1 Promote/centralize the md5 helper (currently `md5ChecksumBytes`, `server/datastore/mysql/scripts.go:692`) into a shared datastore helper; add a `[]byte`-returning variant if useful for direct binding.
- [ ] 1.2 Add a `MYSQL_TEST=1` ground-truth test asserting the Go helper equals `SELECT UNHEX(MD5(?))` on MySQL 8.0 for representative inputs (binary blobs, UTF-8 text, empty string).

## 2. Convert generated columns to plain BINARY(16), populate in Go

- [ ] 2.1 `mdm_apple_declarations.token` (= md5 of `raw_json` + MySQL `DATETIME(6)` of `secrets_updated_at`, `''` when NULL; mirror `fleet.EffectiveDDMToken`): populate in `NewMDMAppleDeclaration` (`apple_mdm.go:5548`), `SetOrUpdateMDMAppleDeclaration` (`:5578`), `insertOrUpdateDeclarations` (`:5403`, batch), `batchSetMDMAppleDeclarations` (`:5232`).
- [ ] 2.2 Recompute `mdm_apple_declarations.token` wherever `secrets_updated_at` is updated independently of `raw_json` (secret-variable resolution path ~`apple_mdm.go:6026`); audit for any other site.
- [ ] 2.3 `mdm_windows_configuration_profiles.checksum` (= md5 of raw `syncml` bytes): populate in `NewMDMWindowsConfigProfile` (`microsoft_mdm.go:2978`), `SetOrUpdateMDMWindowsConfigProfile` (`:3084`), `batchSetMDMWindowsProfilesDB` (`:3135`, batch).
- [ ] 2.4 `mdm_android_configuration_profiles.checksum` (= md5 of the exact bytes inserted as `raw_json` — do NOT re-serialize JSON): populate in `NewMDMAndroidConfigProfile` (`android.go:554`), `batchSetMDMAndroidProfiles` (`:1530`, batch).

## 3. Replace runtime SQL MD5() with Go-computed values

- [ ] 3.1 `mdm_apple_configuration_profiles.checksum`: bind the already-computed `cp.Checksum` (`apple_mdm.go:135`) instead of `UNHEX(MD5(mobileconfig))` at `apple_mdm.go:196`, `2482` (`batchSetMDMAppleProfilesDB`), `4127` (`upsertAppleMDMConfigProfiles`); compute md5 per profile in the batch paths.
- [ ] 3.2 `policies.checksum`: replace `policiesChecksumComputedColumn()` (`policies.go:294`) usages at `:118, :397, :1280, :1594, :1751` with a Go md5 of `team_id` (decimal, `''` if NULL) + `CHAR(0)` + `name`, bound as `UNHEX(?)`. Covers `CreatePolicy`, `UpdatePolicy`, `NewTeamPolicy`, `ApplyPolicySpecs` (batch).
- [ ] 3.3 DDM ServerToken aggregate (`MDMAppleDDMDeclarationsToken`, `apple_mdm.go:5840`): SELECT raw rows (HEX(token) + `variables_updated_at`) and compute `md5(concat(count, group_concat(...)))` in Go, replicating order (`uploaded_at DESC, declaration_uuid ASC`, separator `''`) and the `COUNT(0)` prefix.

## 4. Migrations & schema

- [ ] 4.1 New forward migration (`make migration name=DropMd5GeneratedColumns`): `ALTER TABLE ... MODIFY COLUMN ... BINARY(16)` to drop the `GENERATED` expression on the three columns while preserving stored bytes; include a migration test.
- [ ] 4.2 Retcon generated-column migrations to plain-column adds: `20241230000000_AddSecretsUpdatedAt.go`, `20250318165922_AddChecksumAndSecretsToWindowsProfiles.go`, `20260528213326_AddChecksumToAndroidProfiles.go`.
- [ ] 4.3 Retcon `MD5()` data-backfill migrations (drop the `MD5()` statement / convert to Go backfill; no-op on empty tables): `20230408084104`, `20230711144622`, `20231212094238`, `20240131083822`, `20240221112844`, `20240725152735`, `20250410104321`, `20251015103505`.
- [ ] 4.4 Regenerate `server/datastore/mysql/schema.sql`; confirm no `md5(`/`sha1(` remains.

## 5. CI

- [ ] 5.1 Add `mysql:9.7.0` (and a `mysql:9.6.x`) to the MySQL matrix in `.github/workflows/test-go.yaml:94`.

## 6. Verification

- [ ] 6.1 Per-case equality tests: Go value == old `SELECT UNHEX(MD5(...))` for policies checksum, declaration token (incl. non-NULL `secrets_updated_at`), windows/android/apple checksums, and the DDM aggregate.
- [ ] 6.2 Coverage tests: insert via every single-item and batch path; assert checksum/token columns are non-NULL and correct.
- [ ] 6.3 No-churn test: pre-seed host + desired checksums (old md5), run the forward migration, run the reconcile/desired-state query, assert unchanged profiles are NOT flagged for delivery.
- [ ] 6.4 Fresh-install test: `migrate up` from zero against MySQL 9.6 and 9.7 succeeds; `migration_test.go` schema check passes.
- [ ] 6.5 Run `go test ./server/service/` (mock init), MDM integration suites with `MYSQL_TEST=1 REDIS_TEST=1`, and `make lint-go-incremental`.
