## Orbit 1.17.0 (Sep 28, 2023)

* Updated the image and the overall layout of the migration dialog

* Added a mechanism to retry a Fleet Desktop token when the Fleet server response indicates it has expired or is invalid.

* Upgraded Go version to 1.21.1

## Orbit 1.16.0 (Sep 6, 2023)

* Updated the default TUF update roots with the newest metadata in the server. (#13381)

* Updated bundled-in CA certificates. (#13446)

* Removed a listener for the OS. Kill signal since golang can't capture it. (#12861)

* Allow clients to report errors back to the server during the MDM migration flow. (#13189)

* Use OrbitNodeKey for windows mdm enrollment authentication instead of HostUUID (#12847)

* Implemented script execution on the fleetd agent (disabled by default). (#9583)

* Improved the MDM migration dialogs:
  * Adjusted the copy and images. (#13158)
  * Made sure that all dialogs take over the screen. (#13512)
  * Ensure migration dialog doesn't open automatically if it was opened manually. (#13505)

* Fixed theme detection and icon coloring issues for Fleet Desktop on Windows. (#13457)

## Orbit 1.2.0 - Orbit 1.15.0 (Oct 4, 2022 - Aug 17, 2023)

* Fixed an issue preventing Nudge from reading the configuration file delivered by Fleet on some installations. This only affects you if Nudge was enabled and configured on a host using Orbit v1.8.0.

* Added `pmset` table extension to Fleet for CIS check 2.9.1.

* Fixed a bug in Fleet Desktop causing it to spam servers without licenses for policies.

* Added support to enhance the DEP migration flow in macOS for MDM.

* Added `firmware_eficheck_integrity_check` table for macOS CIS 5.9.

* Fixed an issue where Orbit service on Windows was not creating the `secret-orbit-node-key.txt` with a restricted ACL.

* Added periodical restart of the `softwareupdated` service to work around a macOS bug.

* Set `--database_path` in the shell `osqueryd` invocation to retrieve UUID and other fields.

* Updated MDM migration flow to include checking the output of `profiles show -type enrollment`.

* Ensured MDM migration modal is not shown if the host is already enrolled into Fleet.

* Embedded Augeas lenses into Orbit on Unix platforms.

* Added a new table to support the CIS audit process.

* Added `sudo_info` table to Orbit for CIS checks 5.4 and 5.5 on macOS.

* Fixed an issue affecting macOS devices with MDM enabled that prevented Orbit from restarting if Nudge was still open.

* Added support to query Windows MDM enrollment status and enforce MDM commands through the `mdm_bridge` virtual table.

* Dumped pprof data into a `profiles` directory in the Orbit root directory on Unix systems when receiving a SIGUSR1.

* Added `launchctl bootstrap` retries in Orbit `pkg` installer to fix MDM deployments.

* Allowed `fleetd` to get an enroll secret and Fleet URL configuration from a macOS configuration profile.

* Added version information and icons to Orbit and Fleet Desktop binaries.

* Implemented a table to hold `user_login_settings` options extension via Orbit.

* Removed automatic functionality to call `launchctl kickstart -k softwareupdated`.

* Fixed a panic in `fleetd` that might occur when concurrent requests are made to the server.

* Fixed an issue where Orbit lost communication with Fleet server when the certificate used for insecure mode was deleted.

* Added `dscl` table to Orbit for CIS check 5.6 on macOS.

* Fixed an issue that prevented Orbit shell from running when the `osqueryd` instance attempted to register the same named pipe name.

* Ensured Orbit now installs properly on Windows Server 2012 and 2016 with legacy Orbit or Osquery previously installed.

* Fixed an Orbit bug causing repeated restarts when Fleet agent options were configured with `command_line_flags: {}`.

* Fixed an update bug where the Orbit symlink was not present.

* Adjusted the dialog shown during MDM migration to close when the "contact IT" button is pressed.

* Added support for mTLS to `fleetd`.

* Added `authdb` table for macOS CIS check 5.7.

* Fixed a crash that occurred when updates were disabled under certain conditions.

* Implemented a table to hold `csrutil_info` extension via Orbit.

* Fixed a bug that set a wrong Fleet URL in Windows installers.

* Added `sntp_request` table implementation to query NTP servers.

* Stopped rendering errors as tooltips in Fleet Desktop. Errors are now found in the logs.

* Retrieved UUID by reading the SMBIOS interface when WMI call fails on Windows.

* Implemented autoupdate and deploy extensions via Orbit.

* Implemented a table to hold `nvram_info` and `pwd_policy` options extension via Orbit.

* Improved the logic to read enroll secrets from macOS configuration profiles.

* Implemented `icloud_private_relay` table to get iCloud Private Relay status.

* Ensured Orbit kills any pre-existing Fleet Desktop processes at startup.

* Added support for `fleetd` to renew the MDM enrollment profile on pending devices.

* Fixed an issue in Windows where the Fleet service was getting killed if the start took longer than 30 seconds.

* Updated `fleetctl` to generate installer flags that are compatible with MySQL 8 & S3.

* Ensured Fleet Desktop app on Windows removes the tray icon when it exits.

* Added functionality to rotate device tokens every one hour.

* Waited until the device is fully unenrolled from the previous MDM to close the migration dialog.

* Ensured Orbit restarts and switches channels when needed, even if the new channel is already installed.

* Added a new flag, `--use-system-configuration`, for Orbit to read configuration values from the system.

* Added `software_update` table implementation to check whether Apple software needs updating.

* Updated Windows MSI installer to use custom actions to remove Orbit files.

* Allowed configuring osquery startup flags from Fleet, with important notes for existing deployments:

This feature requires Orbit to communicate with Fleet. Orbit uses osquery's enroll secret to authenticate and enroll to Fleet.

On environments where an enroll secret has been revoked, Orbit hosts that were deployed with such secret will fail to enroll to Fleet.

This is not a regression, all existing features should work as expected, but we recommend to fix this issue given that we will be adding more features to Orbit that will use the new communication channel.

1. To determine which hosts need to be fixed, run the following query: `SELECT * FROM orbit_info WHERE enrolled = false`.
Hosts not running Orbit will fail to execute such query because the table doesn't exist, those can be ignored.
2. Generate Orbit packages with the new enroll secret.
3. Deploy Orbit packages to the hosts returned in (1).

* Ensured Orbit re-enrolls when encountering a 401/unauthenticated error when communicating with Fleet server endpoints.

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
