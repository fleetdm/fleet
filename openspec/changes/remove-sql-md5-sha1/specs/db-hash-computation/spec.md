## ADDED Requirements

### Requirement: No SQL hash functions in schema or queries

Fleet SHALL NOT use the MySQL `MD5()` or `SHA1()` SQL functions in any table definition, generated column, migration, or runtime query. All hashes that feed database checksum/token columns or unique indexes SHALL be computed in Go and bound as query parameters.

#### Scenario: Fresh install on MySQL 9.6/9.7

- **WHEN** Fleet runs `migrate up` from an empty database against MySQL 9.6.x or 9.7.x LTS
- **THEN** all migrations succeed and no error referencing `MD5` or `SHA1` is raised

#### Scenario: Runtime queries on MySQL 9.6/9.7

- **WHEN** Fleet creates, updates, or batch-applies configuration profiles, declarations, and policies against MySQL 9.6.x or 9.7.x LTS
- **THEN** all operations succeed without invoking a removed SQL hash function

#### Scenario: Schema contains no SQL hash functions

- **WHEN** `server/datastore/mysql/schema.sql` is inspected after this change
- **THEN** it contains no `md5(` or `sha1(` expression in any column definition or index

### Requirement: Hash values are byte-identical to the previous MD5 output

Computing a hash in Go SHALL produce the exact same bytes that MySQL's `MD5()` produced for the same input, so that existing stored values continue to compare equal and no data is recomputed on upgrade.

#### Scenario: Go md5 matches SQL md5 for the same input

- **WHEN** a checksum or token is computed in Go for a given input value
- **THEN** the result equals `SELECT UNHEX(MD5(<same input>))` evaluated on MySQL 8.0 for that input

#### Scenario: Policies checksum preserves the unique index

- **WHEN** a policy is created or updated and its `checksum` is computed in Go from `team_id` and `name`
- **THEN** the value matches the previous `UNHEX(MD5(CONCAT_WS(CHAR(0), COALESCE(team_id,''), name)))` and the `idx_policies_checksum` uniqueness constraint behaves unchanged

#### Scenario: Declaration token includes secrets timestamp

- **WHEN** a declaration's `token` is computed in Go from `raw_json` and a non-NULL `secrets_updated_at`
- **THEN** the value matches the previous `UNHEX(MD5(CONCAT(raw_json, IFNULL(secrets_updated_at, ''))))`, using MySQL's `DATETIME(6)` string format for the timestamp

### Requirement: Checksum/token columns are populated at every write site

Columns that were previously STORED generated columns (`mdm_apple_declarations.token`, `mdm_windows_configuration_profiles.checksum`, `mdm_android_configuration_profiles.checksum`) SHALL be populated with a correct hash on every insert and update path, including single-item and batch/GitOps paths.

#### Scenario: Single-item write populates checksum

- **WHEN** a profile or declaration is created or updated through a single-item datastore method
- **THEN** the corresponding checksum/token column is set to the Go-computed hash of its source content

#### Scenario: Batch/GitOps write populates checksum

- **WHEN** profiles or declarations are applied through a batch/GitOps path
- **THEN** every affected row's checksum/token column is set to the Go-computed hash, with no row left NULL or stale

#### Scenario: Declaration token recomputed when secrets change

- **WHEN** a declaration's `secrets_updated_at` is updated independently of `raw_json`
- **THEN** the `token` column is recomputed in Go to reflect the new timestamp

### Requirement: Existing deployments upgrade without profile churn

The migration that converts generated columns to plain columns SHALL preserve existing stored hash bytes, so that no host is flagged for unnecessary profile re-delivery and no device receives a spurious DDM re-sync.

#### Scenario: Generated-to-plain conversion preserves data

- **WHEN** the forward migration runs `ALTER TABLE ... MODIFY COLUMN` to drop the `GENERATED` expression
- **THEN** the previously stored hash bytes remain unchanged in each row

#### Scenario: No re-delivery after upgrade

- **WHEN** the desired-state reconcile query runs after the migration with pre-existing host and profile checksums
- **THEN** profiles whose content did not change are not flagged as needing delivery (host checksum still equals desired checksum)

### Requirement: CI exercises MySQL 9.7

The Go test matrix SHALL include a MySQL version that has removed `MD5()`/`SHA1()` (9.7 LTS, and 9.6) so regressions are caught.

#### Scenario: Test matrix includes 9.7

- **WHEN** the Go test workflow matrix is inspected
- **THEN** it includes a `mysql:9.7.x` image (and `mysql:9.6.x`)
