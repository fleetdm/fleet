# Rotating the static Apple MDM SCEP challenge

Apple hosts get their MDM identity certificate from Fleet's SCEP server, and every request has to carry a challenge. Since Fleet 4.94, most enrollments use a dynamic challenge: it's tied to one device, it expires, and it can only be used once. Fleet also keeps one static challenge, the `scep_challenge` row in `mdm_config_assets`. Fleet's SCEP server accepts it from any device for as long as `mdm.apple_scep_static_challenge_enabled` is `true`, which is the default.

Anyone who has the static challenge can get a Fleet-signed identity certificate. If it leaks, for example because a manual enrollment profile ends up somewhere it shouldn't, rotate it. Fleet has no API for this, so you do it with the `tools/mdm/assets` MDM Asset Manager.

## How the static challenge gets into the database

- On startup, `fleet serve` checks for a `scep_challenge` asset. If none exists, it inserts `FLEET_MDM_APPLE_SCEP_CHALLENGE` or, when that's unset, a random UUID (`cmd/fleet/serve.go`).
- Once a row exists, Fleet ignores `FLEET_MDM_APPLE_SCEP_CHALLENGE` and logs a warning. **You can't rotate the challenge by changing the env var.**
- The value is encrypted with AES-GCM using `FLEET_SERVER_PRIVATE_KEY` (its first 32 bytes) and stored as raw bytes in the `value` column, which is a `longblob`.

## Where the static challenge is used

These paths only use it while `mdm.apple_scep_static_challenge_enabled` is `true`:

- **Manual enrollment profile.** The profile from `GET /api/latest/fleet/enrollment_profiles/manual` contains the static challenge in plaintext, since it isn't tied to a device.
- **SCEP server.** `AppleMDMChallengeMiddleware` (`server/mdm/scep/challenge/apple_mdm.go`) checks the static challenge first and falls back to the dynamic one. A certificate issued with the static challenge has no binding extension (`1.3.6.1.4.1.63991.1.2`).
- **Legacy silent migration profiles.** If the `SCEP` payload `Challenge` in `FLEET_SILENT_MIGRATION_ENROLLMENT_PROFILE` is the literal static challenge (instead of `$FLEET_VAR_SILENT_MIGRATION_SCEP_CHALLENGE`), the renewal cron uses the current static challenge to find it and swap in a per-host challenge.

If no customer needs the static challenge anymore, turning it off with `mdm.apple_scep_static_challenge_enabled: false` is better than rotating it. Note that Fleet then also rejects `Authenticate` from certificates that don't have a binding extension.

## Prerequisites

- Access to the MySQL primary the Fleet servers use.
- The server's private key, the same value as `FLEET_SERVER_PRIVATE_KEY`.
- The tool, run from the repo (`go run ./tools/mdm/assets`) or built (`go build -o mdm-assets ./tools/mdm/assets`).

You don't need to stop the Fleet servers. Each read from `cached_mysql` compares the asset's `md5_checksum` against the database, so every server picks up the new value on its next SCEP request.

## Procedure

1. Write down the current challenge so you can roll back (see [Verification](#verification)).

2. Import a new random value:

   ```sh
   ./mdm-assets import \
     -key "$FLEET_SERVER_PRIVATE_KEY" \
     -db-user fleet \
     -db-password "$DB_PASSWORD" \
     -db-address mysql.internal:3306 \
     -db-name fleet \
     -name scep_challenge \
     -value "$(openssl rand -hex 32)"
   ```

   `import` calls `ReplaceMDMConfigAssets`, which soft-deletes the current `scep_challenge` row (setting `deletion_uuid` and `deleted_at`) and inserts the new encrypted value, all in one transaction.

3. If the customer distributes the manual enrollment profile, they need to download it again and replace every copy. Old copies stop working right away.

4. If `FLEET_SILENT_MIGRATION_ENROLLMENT_PROFILE` has the old challenge as a literal, change its `Challenge` to `$FLEET_VAR_SILENT_MIGRATION_SCEP_CHALLENGE`. Otherwise SCEP renewals for migrated hosts stop, and the cron logs `not sending SCEP renewals to hosts from migration`. On startup, Fleet logs `FLEET_SILENT_MIGRATION_ENROLLMENT_PROFILE can't be used for SCEP renewals` when the profile doesn't match.

## Expected side effects

- **Enrolled hosts aren't affected.** The challenge only matters when a certificate is issued. Existing certificates stay valid, and SCEP renewals use per-host dynamic challenges.
- **In-flight enrollments that use the old static challenge fail** at the SCEP step and have to start again with a new profile.
- **Dynamic challenges aren't affected.** ADE, OTA, account-driven user enrollment, and renewals never read the static challenge.
- **The old row stays in the table** as a soft-deleted row, for auditing. `GetAllMDMConfigAssetsByName` doesn't return it.

## Verification

Make sure there's exactly one active row and the previous one is soft-deleted:

```sql
SELECT id, deletion_uuid <> '' AS deleted, deleted_at, created_at
FROM mdm_config_assets
WHERE name = 'scep_challenge'
ORDER BY id;
```

Decrypt the active value. `decrypt` takes hex or base64 and strips a leading `0x`. Use `HEX(value)`: `TO_BASE64()` wraps its output across lines, and `mysql -N` prints those newlines as literal `\n`, which the tool can't decode.

```sh
./mdm-assets decrypt \
  -key "$FLEET_SERVER_PRIVATE_KEY" \
  -db-user fleet -db-password "$DB_PASSWORD" -db-address mysql.internal:3306 -db-name fleet \
  -value "$(mysql -N -e "SELECT HEX(value) FROM mdm_config_assets WHERE name = 'scep_challenge' AND deletion_uuid = ''" fleet)"
# Decrypted value: <the new challenge>
```

`export` goes through the same datastore read path the server uses, so it confirms the server can decrypt the new value:

```sh
./mdm-assets export -key "$FLEET_SERVER_PRIVATE_KEY" -name scep_challenge -dir /tmp/scepcheck
cat /tmp/scepcheck/scep_challenge
```

If you see `cipher: message authentication failed`, the value was encrypted with a different key than the one you passed. Make sure `-key` was `FLEET_SERVER_PRIVATE_KEY` during the import too.

## Rollback

1. Decrypt the previous value from the soft-deleted row:

   ```sh
   ./mdm-assets decrypt -key "$FLEET_SERVER_PRIVATE_KEY" <db flags> \
     -value "$(mysql -N -e "SELECT HEX(value) FROM mdm_config_assets WHERE name = 'scep_challenge' AND deletion_uuid <> '' ORDER BY id DESC LIMIT 1" fleet)"
   ```

2. Import it again: `./mdm-assets import -key … <db flags> -name scep_challenge -value '<previous value>'`. This adds another row, and the rotated value is soft-deleted in turn.
