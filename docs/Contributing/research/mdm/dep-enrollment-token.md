# Automatic enrollment (ADE) token

Fleet puts a token in the `url` of every automatic enrollment profile it defines with Apple, and devices present it to `/api/mdm/apple/enroll` to download their enrollment profile. This doc covers what that token is, where it flows, the Apple constraints that shape how it can change, and how rotation works.

## What the token is

- Fleet generates it (a UUID) the first time it creates the default automatic enrollment profile, in `createDefaultAutomaticProfile` (`server/mdm/apple/apple_mdm.go`).
- It's stored in `mdm_apple_enrollment_profiles.token`, on the single row with `type = 'automatic'`. The column is `utf8mb4_bin`, so lookups are case-sensitive.
- It's shared by every fleet and every AB token. `RegisterProfileWithAppleDEPServer` always reads it from the automatic row and builds `{server_url}/api/mdm/apple/enroll?token=<token>` with `EnrollURL`.
- It isn't the AB server token. The AB token (`abm_tokens`) is Apple-issued OAuth material that authenticates Fleet to Apple's DEP API and is renewed yearly. The enrollment token authenticates devices to Fleet.

## Where it flows

- **Fleets without end user authentication:** the token is in the DEP profile's `url`. Setup Assistant calls it directly with the device's signed `deviceinfo`.
- **Fleets with end user authentication:** the DEP profile's `url` and `configuration_web_url` both point at `{server_url}/mdm/sso`, so the profile never contains the token. After the end user signs in, the SSO callback returns a one-time token instead (see [end user authentication](../../authentication/mdm-end-user-authentication.md)). It's bound to the IdP account and the device's serial number and UDID, works once, and expires after 1 hour.
- **Redemption:** `/api/mdm/apple/enroll` accepts the current token, the previous token during a rotation grace period, or an unused one-time token. The device's serial number must be assigned to Fleet in AB.

## Apple constraints

- Every DEP profile needs a `url`. Apple has asked MDM vendors not to put per-device values in it, so the token in fleets without end user authentication has to be shared.
- A device gets its profile when it's activated and keeps it until it's erased and activated again. Apple: changes to an assigned profile have "no effect until the device is wiped and activated again." A device that was activated but hasn't enrolled keeps the old URL no matter what Fleet does.
- Apple only learns a new URL when Fleet re-defines the profile, and devices only move to the new profile when Fleet re-assigns them.

## Approaches considered

| Approach | Decision | Why |
|---|---|---|
| Rotate the shared token, with a grace period | Chosen | Reuses the existing job that re-defines and re-assigns every fleet's profile. Devices activated before the rotation work until the grace period ends. |
| One-time tokens after end user authentication | Chosen | Profiles for fleets with end user authentication never contain the shared token, so the SSO callback doesn't need to hand it out. |
| Per-device tokens through per-device DEP profiles | Rejected | One profile per host that Apple never lets Fleet delete, a define and an assign call per host on every change, undocumented rate limits, and it still doesn't help devices that already fetched their profile. |
| Per-fleet tokens | Deferred | Fleet already defines one profile per fleet per AB token, so it's possible later. Not needed now. |
| Automatic periodic rotation | Rejected | See below. |

## Rotation

`POST /api/v1/fleet/enrollment_profiles/automatic/rotate_token` (contributor API, Fleet Premium, global admins only). See [API for contributors](../../reference/api-for-contributors.md#rotate-automatic-enrollment-token).

- Fleet moves the current token to `previous_token`, sets `previous_token_expires_at` to now plus the grace period (default 24 hours, up to 720), and generates a new token. A grace period of `0` sets both previous columns to `NULL`, so the old token stops working immediately.
- Rotating again during a grace period replaces `previous_token` with the token from just before the second rotation. The earlier one stops working right away.
- Fleet then queues the `update_all_profiles` worker job. For each fleet and "No team", it clears the stored profile UUIDs, re-defines the profile with the new `url`, and re-assigns the fleet's DEP devices.
- A fleet with no DEP devices gets the new URL the next time it defines a profile: when a device is first assigned to it, or when it becomes an AB default fleet.
- Rotation updates `mdm_apple_enrollment_profiles.updated_at`, which changes the timestamp `GET /api/v1/fleet/enrollment_profiles/automatic/default` returns. For AB default fleets, the newer profile modification time also resets the DEP sync cursor, so the next sync re-scans devices.
- Token values are never returned by the API or logged by the rotation code. No activity is recorded.

## Recovering a device stuck on an old URL

A device that was activated before a rotation, but hadn't enrolled when the grace period ended, fails at the Remote Management step. To get a fresh profile:

- **Mac:** in Setup Assistant, open Terminal with Control-Option-Command-T and run `profiles renew -type enrollment` (no `sudo` needed). If that doesn't work, erase the Mac and activate it again.
- **iPhone and iPad:** restore the device from a Mac in recovery or DFU mode (Finder or Apple Configurator), then activate it again.
- **Apple TV:** Apple TV can't be forced back through activation, so an Apple TV in this state can't be recovered by the admin. That's one reason rotation is manual and the default grace period is a day.

## Why the token isn't rotated automatically

Every rotation risks stranding devices that were activated but hadn't enrolled, and recovery is manual and device-specific (see above). Rotating on a schedule would strand devices without any admin action to explain it. Other MDM vendors keep a static enrollment URL for enrollments without end user authentication for the same reason. Admins who want enrollment gated on identity should turn on end user authentication, which uses one-time tokens.
