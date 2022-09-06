## Orbit 1.1.0 (Aug 19, 2022)

* Rename `unified_log` table to `macadmins_unified_log` to avoid collision with osquery core. This allows Orbit to support osquery 5.5.0.

## Orbit 1.0.0 (July 14, 2022)

- Update the dropdown in Fleet Desktop to show the number of failing policies along with the status.

- Disable the 'Transparency' menu item in Fleet Desktop until the device is successfully connected to the Fleet server.

- Corrected the macOS logging path for Fleet Desktop to `~/Library/Logs`.

- Added cleanup of osquery extension socket to Orbit at startup.

## Orbit 0.0.13 (Jun 16, 2022)

- Orbit is now a Universal Binary supporting Intel and M1 on macOS machines without Rosetta.

- Updated the Fleet Desktop "Transparency" menu item to use a custom URL if specified (Premium only).

- Added an early check for updates to Orbit (before sub-systems are started) to improve chances of being able to recover from crashes via updates.

- Added log files for Fleet Desktop logs, located at:
  - macOS: `~/Library/Log`
  - Linux: `$XDG_STATE_HOME`, fallback to `$HOME/.local/state`
  - Windows: `%LocalAppData%`

## Orbit 0.0.12 (May 26, 2022)

### This is a security release.

- **Security**: Update go-tuf library to fix [CVE-2022-29173](https://github.com/theupdateframework/go-tuf/security/advisories/GHSA-66x3-6cw3-v5gj). This vulnerability could allow an attacker with network access to perform a rollback attack, forcing Orbit to downgrade to an earlier version. Orbit installations with autoupdate turned on will automatically update, after which the client will no longer be vulnerable.

- Fleet desktop will now notify Premium tier users if policies are failing/passing.

## Orbit 0.0.11 (May 10, 2022)

- Change install path to /opt/orbit. Fixes a permissions issue on platforms with SELinux enabled.
  See [fleetdm/fleet#4176](https://github.com/fleetdm/fleet/issues/4176) for more details.

- Remove support for Orbit to use the legacy osqueryd target on macOS. This has been deprecated since introduction of the app bundle support in Orbit 0.0.8.

## Orbit 0.0.10 (Apr 26, 2022)

- Revert Orbit osquery remote paths to use `v1`.

## Orbit 0.0.9 (Apr 20, 2022)

- Add Fleet Desktop Beta support for Windows.

- Make update interval configurable and increase default from 10s to 15m.

## Orbit 0.0.8 (Mar 25, 2022)

- Fix `orbit shell` command to successfully run when Orbit is already running as daemon.

- Add Fleet Desktop Beta support for macOS.

- Support running osquery as an app bundle on macOS.

- Upgrade [osquery-go](https://github.com/osquery/osquery-go) and [osquery-extension](https://github.com/macadmins/osquery-extension) dependencies.

## Orbit 0.0.7 (Mar 8, 2022)

- Improve reliability of osquery extension connection at startup.

- Fix orbit not detecting updates at startup when they are published while orbit was not running.

- Create and set log paths for "result" and "status" logs when launching osquery.

## Orbit 0.0.6 (Jan 13, 2022)

- Add logging when running as a Windows Service (because Windows discards stdout/stderr).

- Improve flaky startups by adding wait for osquery extension socket.

## Orbit 0.0.5 (Dec 22, 2021)

- Fix handling of enroll secrets to address 0.0.4 enrollment issue.

## Orbit 0.0.4 (Dec 19, 2021)

- Use `certs.pem` if available in root directory to improve TLS compatibility.

- Use UUID as the default host identifier for osquery.

- Add github.com/macadmins/osquery-extension tables.

- Add support for osquery flagfile (loaded automatically if it exists in the Orbit root).

- Fix permissions for building MSI when packaging as root user. Fixes fleetdm/fleet#1424.
