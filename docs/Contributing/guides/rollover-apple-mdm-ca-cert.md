# Rolling over the Apple MDM CA certificate

The Apple MDM CA certificate (`ca_cert` in `mdm_config_assets`) is the self-signed root that signs every SCEP client certificate issued during Apple MDM enrollment. If it expires, enrolled hosts can no longer validate the chain on profile installs and re-enrollment.

The `tools/mdm/assets` MDM Asset Manager has a `rollover-ca-cert` subcommand that re-signs the certificate in place, reusing the existing CA private key so previously-issued client certificates remain valid.

## When to use it

Run this when the Apple MDM CA cert is approaching its `NotAfter` (or has already expired). The operation re-signs only — no key rotation. If the CA private key has been compromised, this is **not** the tool you want; that scenario requires re-enrolling every Apple host. This tool should rarely, if ever, be needed - it was added to address specific usecases around migrations where customers bring their own keys. If in doubt please ask Engineering for clarity.

## Prerequisites

- All Fleet server containers for the target deployment must be stopped before running the tool to ensure a clean cutover to the new certificate.
- Access to the MySQL primary the Fleet servers use.
- The server's private key (the same value passed via `FLEET_SERVER_PRIVATE_KEY`) — used to decrypt `mdm_config_assets`.

## Procedure

1. Stop every Fleet server container that talks to this database.
2. Run the tool:

   ```sh
   ./mdm-assets rollover-ca-cert \
     -key "$FLEET_SERVER_PRIVATE_KEY" \
     -db-user fleet \
     -db-password "$DB_PASSWORD" \
     -db-address mysql.internal:3306 \
     -db-name fleet \
     -extend-years 5
   ```

   `-extend-years` defaults to `5`. The tool prints the previous and new `NotAfter`, the new serial, and the cert common name on success.

3. Start the Fleet server containers back up. On boot they read `ca_cert` from `mdm_config_assets` and the SCEP depot and verifier uses the renewed cert from then on.

## What the tool does

- Loads `ca_cert` + `ca_key` from `mdm_config_assets`.
- Reserves a fresh serial via `INSERT INTO identity_serials () VALUES ();` and uses `LAST_INSERT_ID()` as the new CA serial. This guarantees the new CA cert serial is greater than every historical client cert serial and that no future client cert can ever be issued with the same value.
- Builds a new self-signed cert that reuses the existing Subject, SubjectKeyId, KeyUsage, CA flags, and public key. Only the serial and `NotAfter` (now + `extend-years`) change.
- Calls `ReplaceMDMConfigAssets`, which soft-deletes the previous `ca_cert` row (sets `deletion_uuid` + `deleted_at`) and inserts the new cert in one transaction. The `ca_key` row is untouched.

## Expected side effects

- **Fleet root CA profile resend.** The "Fleet root certificate authority (CA)" mobileconfig (`com.fleetdm.caroot`) is generated from the current `ca_cert` every time `ReconcileAppleProfiles` runs. After the rollover the profile contents change, the profile checksum changes, and the cron tick re-pushes the profile to every Apple-MDM-enrolled host. Expect a wave of `InstallProfile` MDM commands on the next reconcile cycle. Similar to uploading a new static profile
- **`identity_serials` jumps by one.** The reserved serial is permanently consumed. The next SCEP-issued client certificate will get the value after that.
- **Existing SCEP client certs keep validating.** Subject, SubjectKeyId, and the signing key are unchanged, so client certs already issued by the previous CA chain to the renewed CA identically. Nothing on the device side needs to change.
- **Soft-deleted CA cert row remains in the table.** The previous `ca_cert` row stays in `mdm_config_assets` with `deletion_uuid` set, so the rollover is auditable. It will not be returned by `GetAllMDMConfigAssetsByName`.
- **Unaffected.** APNS, ABM, VPP, Android, and Windows MDM (WSTEP) do not use `ca_cert` / `ca_key` and are not touched by this operation.
- **Previously-escrowed Filevault keys still decrypt.** Filevault keys are escrowed using a PKCS7 CMS envelope with a recipient field specific to the certificate of the fleet server. The decryption process(only invoked during the hourly cleanups_then_aggregation cron run or on-demand when Filevault keys are viewed) loads the prior certificate for proper decryption(even though, cryptographically speaking, the new certificate could be used to decrypt it but would require modifications to the PKCS7 library).

## Verification

After restarting the Fleet servers:

```sh
./mdm-assets export -key "$FLEET_SERVER_PRIVATE_KEY" -name ca_cert -dir /tmp/cacheck
openssl x509 -in /tmp/cacheck/ca_cert.crt -noout -dates -serial -subject
```

Confirm the `notAfter` date matches the new value and the `serial` matches what the tool printed. On a sample enrolled host, expect the "Fleet root certificate authority (CA)" profile to redeliver on the next profile reconcile.

## Rollback

There is no built-in undo. If you need to revert immediately:

1. Recover the prior `ca_cert` PEM from the soft-deleted `mdm_config_assets` row (`SELECT value FROM mdm_config_assets WHERE name = 'ca_cert' AND deletion_uuid <> '' ORDER BY id DESC LIMIT 1;`) and decrypt it with `FLEET_SERVER_PRIVATE_KEY`.
2. Stop the servers again.
3. Re-import it: `./mdm-assets import -key … -name ca_cert -value "$(cat old.pem)"`.
4. Restart the servers.

The bumped `identity_serials` value cannot be returned to the pool, but that has no functional impact — the next client cert simply skips ahead.
