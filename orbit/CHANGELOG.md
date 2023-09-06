## Orbit 1.16.0 (Sep 6 17, 2023)

## Orbit 1.2.0 - Orbit 1.15.0 (Oct 4, 2022 - Aug 17, 2023)

* Fixed an issue preventing Nudge to read the configuration file delivered by Fleet on some installations. This only affects you if Nudge was enabled and configured on a host using Orbit v1.8.0

* Add `pmset` table extension to fleed for CIS check 2.9.1.

- Fixed a bug in Fleet Desktop causing it to spam servers without licenses for policies.

* MDM: added support to enhance the DEP migration flow in macOS.

* Add `firmware_eficheck_integrity_check` table for macOS CIS 5.9.

* Orbit service on windows is not creating the secret-orbit-node-key.txt with a restricted ACL to allow only privileged users to access its content

* Added periodical restart of the `softwareupdated` service to work around a macOS bug where it sometimes hangs and prevents software updates.

* Set `--database_path` in the shell osqueryd invocation to retrieve UUID and other fields.

- Updated MDM migration flow to include checking the output of `profiles show -type enrollment`
  as a pre-condition for `profiles renew -type enrollment` to mitigate issues where caching or other
  unexpected delays in Apple DEP profile assignment could cause the wrong profile to be renewed.

* Ensure MDM migration modal is not shown, and enrollment commands are not run if the host is already enrolled into Fleet

- Embed augeas lenses into orbit on Unix platforms so that the `augeas`
  table works without further configuration

* New table was added to support CIS audit process

* Fix theme detection and icon coloring issues for Fleet Desktop on Windows.

* Add `sudo_info` table to Orbit for CIS checks 5.4 and 5.5 on macOS.

* Fixed an issue affecting macOS devices with MDM enabled that prevented Orbit for restarting if Nudge was still open.

* Adding support to query Windows MDM enrollment status and to enforce MDM commands through the mdm_bridge virtual table

- On Unix systems, dump pprof data into a `profiles` directory in the orbit root dir
  when receiving a SIGUSR1. This is to assist debugging for memory leaks

- Added `launchctl bootstrap` retries in Orbit `pkg` installer to fix MDM deployments of Orbit (when pushed with `InstallEnterpriseApplication`).

* Allow `fleetd` to get an enroll secret and Fleet URL configuration from a configuration profile on macOS.

* Added version information and icon on orbit and fleet-desktop binaries

- Implement table to hold user_login_settings options extension via Orbit

* Removed automatic functionality to call `launchctl kickstart -k softwareupdated` periodically, which was causing issues on some macOS devices.
  The `--disable-kickstart-softwareupdated` flag is kept for backwards compatibility but it doesn't have any effect.

* Fix a panic in `fleetd` that might occurr when concurrent requests are made to the server.

* Orbit lost communication with Fleet server 
when the certificate used for insecure mode gets deleted.  

* Add `dscl` table to Orbit for CIS check 5.6 on macOS.

* Fixed an issue that prevented orbit shell to run when the osqueryd instance ran through orbit shell attempted to register the same named pipe name used by the osqueryd instance launched by orbit service

* Orbit now installs propery on Windows Server 2012 and 2016 environments with legacy Orbit or Osquery previously installed

- Fixed Orbit bug that caused it to restart repeatedly when Fleet agent options are configured with `command_line_flags: {}`.

* An update bug where orbit symlink was not present is now fixed

* Adjusted the dialog shown during MDM migration to close when the button to contact IT is pressed.

* Add support for mTLS to fleetd.

* Add a `--enable-scripts` flag to `fleetctl package` to build a package capable of script execution
* Allow script execution to be enabled by providing a configuration profile with `PayloadType` equal to `com.fleetdm.fleetd.config` and a key `ScriptsEnabled` set to `true`.

* Add `authdb` table for macOS CIS check 5.7.

* Fixed a crash that happened when updates where disabled and certain conditions (Nudge configuration set or host elegible for MDM migration) were met.

- Implement table to hold csrutil_info extension via Orbit

* Fixed a bug that set a wrong Fleet URL in Windows installers.

* Add table implementation `sntp_request` to query NTP servers.

* Stop rendering errors as tooltips in Fleet Desktop. Errors can now be found in the Fleet Desktop logs.

* When WMI call fails on Windows, UUID can now be retrieved by reading the SMBIOS interface.

- Implement autoupdate and deploy extensions via Orbit

- Implement table to hold nvram_info extension via Orbit

- Implement table to hold pwd_policy options extension via Orbit

* Improve the logic to read enroll secrets from macOS configuration profiles to be compatible with different MDM providers.

- Implement `icloud_private_relay` table to get iCloud Private Relay status.

* Orbit now kills any pre-existing fleet desktop processes at startup.
* Orbit now handles SIGTERM on unix.

* Added support to `fleetd` to run the necessary command to renew the MDM enrollment profile on the devices that are pending automatic enrollment into Fleet MDM.

* When running on Windows, Fleet service was getting killed by the OS when
service start takes longer than 30 secs due to missing calls to the 
Service Control Manager (SCM) APIs.

* Replace the black and white Fleet desktop icons with a single colorful icon on Windows.

- update fleetctl to generate installer flags that use a larger default file carving block size compatible with MySQL 8 & S3

* Fleet-desktop app on windows now removes the tray icon when it exits

- Added functionality to rotate device tokens every one hour

* Wait until the device is fully unenrolled from the previous MDM to close the migration dialog.

* Orbit now restarts and switches channels when needed,
even if the new channel is already installed

* Ensure migration dialog is not opened automatically if it was opened manually in the last 15 minutes

* Added a new flag, `--use-system-configuration` to make orbit read configuration values from the system. Currently this is only supported in macOS via configuration profiles.

* Add table implementation `software_update` to check whether Apple software needs updating.

* Windows MSI installer now uses custom actions to remove Orbit files

* Orbit allows configuring osquery startup flags from Fleet, see [#7377](https://github.com/fleetdm/fleet/issues/7377).
Important note for existing deployments that use Orbit: 
This feature requires Orbit to communicate with Fleet. Orbit uses osquery's enroll secret to authenticate and enroll to Fleet.
On environments where an enroll secret has been revoked, Orbit hosts that were deployed with such secret will fail to enroll to Fleet.
This is not a regression, all existing features should work as expected, but we recommend to fix this issue given that we will be adding
more features to Orbit that will use the new communication channel.
1. To determine which hosts need to be fixed, run the following query: `SELECT * FROM orbit_info WHERE enrolled = false`.
Hosts not running Orbit will fail to execute such query because the table doesn't exist, those can be ignored.
2. Generate Orbit packages with the new enroll secret.
3. Deploy Orbit packages to the hosts returned in (1).

* Orbit now re-enroll when encountering a 401/unauthenticated error when communicating with orbit endpoints on Fleet server

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
