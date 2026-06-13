## Context

MySQL 9.6.0 and 9.7 LTS removed the `MD5()`/`SHA1()` SQL functions by default and ban them in generated columns even with the `legacy_hashing` component. Fleet relies on SQL `MD5()` in two ways:

1. **STORED generated columns** computed by MySQL: `mdm_apple_declarations.token`, `mdm_windows_configuration_profiles.checksum`, `mdm_android_configuration_profiles.checksum`.
2. **Inline `MD5()` in runtime queries / migrations**: `mdm_apple_configuration_profiles.checksum` (`UNHEX(MD5(mobileconfig))`), `policies.checksum` (`policiesChecksumComputedColumn()`, a UNIQUE index), and the device-facing DDM ServerToken aggregate (`MDMAppleDDMDeclarationsToken`).

These hashes are non-cryptographic — they are used for change detection, deduplication, and uniqueness. Go's `crypto/md5` runs in Fleet's process and is unaffected by the MySQL change. The codebase already computes some of these in Go: `apple_mdm.go:135` (`md5.Sum(cp.Mobileconfig)`), `md5ChecksumBytes` (`scripts.go:692`), and `fleet.EffectiveDDMToken` (`fleet/apple_mdm.go:920`, which already reproduces MySQL's `DATETIME(6)` formatting).

Per-host checksum columns (`host_mdm_apple_profiles.checksum`, `host_mdm_windows_profiles.checksum`, `host_mdm_android_profiles.checksum`, `host_mdm_apple_declarations.token`) record which profile version a host currently has, and reconcile compares them against the desired profile's checksum (`hmap.checksum != ds.checksum`). The declarations token is additionally sent to Apple devices as the DDM ServerToken.

## Goals / Non-Goals

**Goals:**
- Remove every SQL `MD5()`/`SHA1()` call (schema, migrations, runtime queries) so fresh installs and runtime work on MySQL 9.6/9.7 LTS.
- Keep md5 hash **values byte-identical** so existing deployments upgrade with zero churn.
- Catch regressions in CI on MySQL 9.7.

**Non-Goals:**
- Switching to sha256 (explicitly rejected by the user — would force column widening, host-side backfill, profile re-delivery, and DDM re-sync).
- Touching Go `crypto/md5` usages that are not SQL function calls (`script_contents.md5_checksum`, `software.checksum`, `mdm_config_assets.md5_checksum`) — they already work on 9.6.
- Changing any API response or device protocol behavior.

## Decisions

**1. Compute in Go, keep md5, preserve values.** Replace each SQL `MD5(x)` with a Go-computed hash bound as `UNHEX(?)` (hex string from `md5ChecksumBytes`) or a raw `[]byte`. Alternative considered: install the MySQL `legacy_hashing` component — rejected because it does not restore generated-column support and forces an operational dependency on every Fleet MySQL deployment. Alternative considered: sha256 — rejected (see Non-Goals).

**2. Generated columns become plain `BINARY(16)`.** A STORED generated column auto-fills itself; a plain column does not, so every write site must compute and bind the value. Coverage was enumerated up front (single-item + batch/GitOps), since a missed site would leave a column NULL or stale. All MDM batch paths funnel through `BatchSetMDMProfiles` (`mdm.go:594`).

**3. Centralize the helper.** Promote `md5ChecksumBytes` to a shared datastore helper so there is one implementation to test against the SQL ground truth.

**4. One forward migration, no data rewrite.** `ALTER TABLE ... MODIFY COLUMN <col> BINARY(16)` drops the `GENERATED` expression while MySQL keeps the existing stored bytes (precedent: `20240601174138_UpdateMobileConfigColumnType.go`). Because values are unchanged, host-vs-desired comparisons stay consistent and nothing re-delivers.

**5. Retcon historical migrations for from-zero correctness.** A fresh install replays every migration; any emitting SQL `MD5()` fails to resolve on 9.6 even against empty tables. Generated-column adds (`20241230000000`, `20250318165922`, `20260528213326`) become plain-column adds; data-backfill `MD5()` UPDATEs (`20230408084104`, `20230711144622`, `20231212094238`, `20240131083822`, `20240221112844`, `20240725152735`, `20250410104321`, `20251015103505`) drop the `MD5()` statement or convert to a Go backfill (a no-op on the empty tables a fresh install presents). Each migration's net schema effect stays identical to today, and existing instances never re-run them.

**6. Rollout: single release (default) vs. two-release shadow validation (Option B).** The default is to ship the full change in one release, relying on the MySQL-8.0 ground-truth tests (Decision 1 / Risks) to guarantee byte-identical values. If the team wants production-grade validation against real customer data before the irreversible cutover, use the phased alternative below.

> ### Option B — two-release shadow validation
>
> The "old" column is a generated column whose expression is `unhex(md5(...))`, which is exactly what 9.6/9.7 reject — so a release that **keeps** it **cannot run on 9.6/9.7**. Therefore Release 1 is validation-only on currently-supported MySQL (≤9.5), and 9.6/9.7 support lands in Release 2. This **delays 9.6/9.7 support by one release** — the main cost to weigh against 9.7 LTS demand.
>
> **Release 1 (additive, runs on ≤9.5):** add Go computation at every write site, writing to a **new plain shadow column** alongside the existing generated column; keep the generated column / SQL `MD5()` as the source of truth; compare Go-vs-SQL on existing reconcile reads (or a periodic job) and log mismatches via `WarnContext` plus a metric/counter. Bake across deployments until the mismatch counter stays at zero. This also surfaces any **missed write site** (shadow goes stale → logged).
>
> **Release 2 (subtractive cutover, adds 9.6/9.7):** make the Go column authoritative, drop the generated columns / SQL `MD5()`, retcon historical migrations, regenerate `schema.sql`, add 9.7/9.6 to CI, remove the shadow comparison.
>
> **Apply selectively.** Shadow only the cases where byte-matching is genuinely uncertain — **Android `CAST(json AS CHAR)`, `policies.checksum` (`CONCAT_WS`), and the DDM aggregate**. The raw-blob cases (`mobileconfig`, `syncml`) are byte-trivial and need no shadow. Note the limits: `policies.checksum` backs a UNIQUE index, so a non-indexed shadow column validates the *value* but not index behavior; the DDM ServerToken is computed per-query and not stored, so it is compared in-query rather than via a column. Cost: two migrations per shadowed column (add then drop) and extra write/storage overhead during Release 1.
>
> **Recommendation:** ship the single-release version unless the ground-truth tests surface something fragile, in which case fall back to Option B for the uncertain cases above.

## Risks / Trade-offs

- **Byte-identical reproduction of MySQL md5 in Go** → Mitigation: add ground-truth tests that compare the Go value against `SELECT UNHEX(MD5(<same input>))` on MySQL 8.0 (which still has the function). Hardest cases: `policies.checksum` (`CONCAT_WS(CHAR(0), COALESCE(team_id,''), name)` — `team_id` as decimal, NUL separator, NULL→`''`) and the DDM aggregate (`GROUP_CONCAT` order `uploaded_at DESC, declaration_uuid ASC` + separator `''` + `COUNT(0)` prefix). Easy cases: mobileconfig/syncml (raw blob bytes). Android: hash the exact bytes inserted, not a Go re-serialization of JSON. For real-data validation beyond synthetic fixtures, see the two-release shadow option (Decision 6 / Option B).
- **Missed write site leaves a column NULL/stale** → Mitigation: coverage tests that exercise each single-item and batch path and assert the column is populated; the enumerated site list in tasks.md.
- **Declaration token also depends on `secrets_updated_at`** → Mitigation: recompute the token wherever `secrets_updated_at` is updated independently of `raw_json` (secret-variable resolution paths), not only on content change.
- **Retcon rewrites applied history** → Mitigation: existing instances do not re-run migrations; only the from-zero path is affected, and net schema effect is preserved. Verified by the migration schema-equality test (`migration_test.go`).
- **MySQL-upgraded-before-Fleet edge case**: if an operator upgrades MySQL to 9.6 before upgrading Fleet, the still-generated columns reference `md5()`. Recommended/documented upgrade order is Fleet-then-MySQL; the forward `ALTER` removes the reference.

## Migration Plan

1. Land Go changes computing all hashes in Go (write sites + DDM aggregate + policies + shared helper).
2. Add the forward migration converting the three generated columns to plain `BINARY(16)`.
3. Retcon historical migrations; regenerate `schema.sql`.
4. Add MySQL 9.7/9.6 to the CI matrix.
5. **Rollback**: revert the Go and CI changes. The forward migration is value-preserving and leaves plain columns that the prior code can still read/compare; reverting code that expects generated columns would require the columns to be re-generated, so prefer roll-forward fixes over down-migration.

## Open Questions

- Confirm whether any non-MDM path updates `secrets_updated_at` on declarations outside the catalogued upserts (audit `apple_mdm.go` around the secret-resolution flow at ~`:6026`).
- Decide exact CI image tags (`mysql:9.7.0` confirmed for LTS; whether to also pin a `9.6.x`).
