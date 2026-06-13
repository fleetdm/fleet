## Why

MySQL 9.6.0 and 9.7 LTS [removed the `MD5()` and `SHA1()` SQL functions by default](https://dev.mysql.com/doc/relnotes/mysql/9.6/en/news-9-6-0.html#mysqld-9-6-0-security) and [forbid them in generated columns even when the `legacy_hashing` component is installed](https://dev.mysql.com/doc/refman/9.6/en/legacy-hashing-component.html). Fleet (4.81.1) cannot start fresh against these versions because several migrations define STORED generated columns using `md5()` and several runtime queries call `MD5()` directly. MySQL 9.7 is LTS, so customers will want to run it.

## What Changes

- Compute every hash that is currently produced by a SQL `MD5()`/`SHA1()` call **in Go instead**, binding the bytes as query parameters (`UNHEX(?)` or `[]byte`). **The md5 algorithm and the stored hash values are unchanged** — only the place of computation moves (SQL → Go).
- Convert three STORED generated columns to plain `BINARY(16)` columns populated in Go at every write site: `mdm_apple_declarations.token`, `mdm_windows_configuration_profiles.checksum`, `mdm_android_configuration_profiles.checksum`.
- Replace remaining runtime `MD5()` calls with Go-computed values: `mdm_apple_configuration_profiles.checksum`, `policies.checksum` (unique-index hash), and the device-facing DDM ServerToken aggregate.
- Add one forward migration that drops the `GENERATED` expression from the three columns while **preserving stored bytes** (no data rewrite).
- Retcon the ~11 historical migrations that emit SQL `MD5()` so a from-zero `migrate up` parses and runs on MySQL 9.6/9.7.
- Regenerate `schema.sql` and add MySQL 9.7 (and 9.6) to the CI test matrix (currently maxes at 9.5.0, which still has the functions and so never catches this).
- **Explicitly NOT switching to sha256.** Keeping md5 values byte-identical avoids column widening, host-side checksum backfill, profile re-delivery, and DDM device re-sync. 

## Capabilities

### New Capabilities
- `db-hash-computation`: All hashing that feeds database checksum/token columns or unique indexes is computed in Go (not via SQL `MD5()`/`SHA1()`), so Fleet runs on MySQL versions that have removed those functions, while preserving existing stored hash values.

### Modified Capabilities
<!-- None: no existing OpenSpec specs in openspec/specs/. -->

## Impact

- **Code**: `server/datastore/mysql/apple_mdm.go` (declarations token, apple config checksum, DDM ServerToken aggregate), `microsoft_mdm.go` (windows checksum), `android.go` (android checksum), `policies.go` (policies checksum), `scripts.go` (shared md5 helper).
- **Migrations**: 1 new forward migration (`ALTER ... MODIFY COLUMN`) + ~11 retconned historical migrations; regenerated `server/datastore/mysql/schema.sql`.
- **CI**: `.github/workflows/test-go.yaml` MySQL matrix gains 9.7.x (and 9.6.x).
- **Out of scope**: Go `crypto/md5` that is not a SQL function call already works on 9.6 — `script_contents.md5_checksum`, `software.checksum`, `mdm_config_assets.md5_checksum` are untouched.
- **Compatibility**: existing deployments upgrade with zero churn (values preserved); no API or device-protocol behavior changes.
