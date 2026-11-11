# Fleet 4.95.0 | Okta conditional access for Windows, built-in MCP server, AI tools in inventory, and more...

<!--
<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/TODO" allowfullscreen></iframe>
</div>
-->

Fleet 4.95.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.95.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Okta conditional access for Windows](#okta-conditional-access-for-windows)
- [Google conditional access for iPhone, iPad, and Android](#google-conditional-access-for-iphone-ipad-and-android)
- [Multiple versions of the same App Store app](#multiple-versions-of-the-same-app-store-app)
- [Built-in MCP server](#built-in-mcp-server)
- [AI tools in software inventory](#ai-tools-in-software-inventory)
- [Patch deadline for macOS apps](#patch-deadline-for-macos-apps)
- [Multiple VPP tokens in one fleet](#multiple-vpp-tokens-in-one-fleet)
- [Deploy certificates from EJBCA to Apple hosts](#deploy-certificates-from-ejbca-to-apple-hosts)
- [Deploy certificates from Google CAS to Linux hosts](#deploy-certificates-from-google-cas-to-linux-hosts)

### Okta conditional access for Windows

IT admins can now use Okta conditional access on Windows hosts, not only macOS. End users on hosts that fail a conditional access policy are blocked from signing in to Okta apps until they fix the issue.

Copy the Windows profile and the Windows browser profile from the Okta conditional access page in **Settings**, and upload both as custom OS settings. The first profile installs a certificate from Fleet. The second tells Chrome and Edge to present it to Okta, so end users don't see a certificate prompt. Firefox asks the end user to pick the certificate. Windows 11 Pro, Enterprise, and Education are supported. You can also get both profiles with the [API](https://fleetdm.com/docs/rest-api/rest-api#get-okta-windows-configuration-profiles).

Learn how in the [Okta conditional access guide](https://fleetdm.com/guides/okta-conditional-access-integration#windows).

GitHub issue: [#53284](https://github.com/fleetdm/fleet/issues/53284)

### Google conditional access for iPhone, iPad, and Android

IT admins whose organization uses Google Workspace can now block sign-in from iPhones, iPads, and Android devices that aren't company-owned and enrolled to Fleet. Fleet installs Chrome with a bookmark on each company-owned device. When the end user opens it, Fleet confirms the device is enrolled and company-owned, and the device is marked as managed in Google. Google's Context-Aware Access then blocks devices that aren't.

Fleet saves each host's Google device ID, shows it on **Host details**, and refuses a bookmark link that's opened on a different device. You can also set and remove a host's Google device ID with the [API](https://fleetdm.com/docs/rest-api/rest-api#update-hosts-google-device-id). Built-in support for Google, without the bookmark, is [coming soon](https://github.com/fleetdm/fleet/issues/56195).

Learn how in the [Google conditional access guide](https://fleetdm.com/guides/google-conditional-access-integration).

GitHub issue: [#54888](https://github.com/fleetdm/fleet/issues/54888)

### Multiple versions of the same App Store app

IT admins can now add more than one version of the same App Store app to a fleet for iOS, iPadOS, and Android. Each version has its own managed configuration, automatic update window, self-service setting, categories, and labels. For example, scope one Zoom version with a VPN or SSO configuration to your "Production" label, and another to your "Test" label. If a host is in the scope of more than one version, Fleet installs the version that was added first.

Learn more in the [App Store apps guide](https://fleetdm.com/guides/install-app-store-apps#multiple-app-instances).

GitHub issue: [#47904](https://github.com/fleetdm/fleet/issues/47904)

### Built-in MCP server

Fleet Premium now includes a built-in MCP server, so AI tools like Claude Code, Claude Desktop, and Cursor can look up hosts, run live reports, check policies, and investigate CVEs in Fleet. There's nothing extra to deploy. Connect your AI tool to `https://<your-fleet-url>/api/v1/fleet/mcp` with an API-only user's token.

The AI tool only gets the tools that the API-only user's role and API endpoints allow, so it can't do more than the same token can through the REST API. The MCP server is on by default. Turn it off in **Settings > Organization settings > Advanced options**, or with `server_settings.mcp_disabled` in [GitOps](https://fleetdm.com/docs/configuration/yaml-files#server-settings).

Learn how in the [Fleet MCP server guide](https://fleetdm.com/guides/fleet-mcp). If you already run the standalone MCP server, the guide also covers how to move to the built-in one.

GitHub issue: [#44448](https://github.com/fleetdm/fleet/issues/44448)

### AI tools in software inventory

Fleet Premium now shows the AI tools on your hosts, without a custom report. On the **Software** page, **Host details**, and **My device**, open **Filters** and turn on **AI tools**. Fleet adds three software types for AI tools it didn't collect before: **MCP server**, **AI skill**, and **AI CLI tool**. AI apps and extensions that Fleet already collects, like the ChatGPT app or the GitHub Copilot VS Code extension, keep their type and are marked as AI tools. In the API, use the `ai_tool` filter on [list software](https://fleetdm.com/docs/rest-api/rest-api#list-software).

See what Fleet collects for each kind of AI tool in the [software inventory reference](https://fleetdm.com/guides/software-inventory-reference#ai-tools).

GitHub issue: [#51599](https://github.com/fleetdm/fleet/issues/51599)

### Patch deadline for macOS apps

IT admins can now patch macOS Fleet-maintained apps on a schedule that matches their organization's patching rules (SLAs). Choose **Force patch: Deadline** on a patch policy, then set the deadline's days and time for each fleet in **Software > Library > Settings**. Deadlines use each host's local time. The default is Tuesday at 17:00.

If the app is closed when the policy fails, Fleet patches it right away. If it's open, Fleet Desktop notifies the end user 1 hour before the deadline and again 5 minutes before, then closes and updates the app. Apps waiting on the same deadline show up together in one notification. Requires Fleet Desktop 1.5.0 or later.

Learn more in the [patching end user experience guide](https://fleetdm.com/guides/patching-end-user-experience#force-patch-deadline), or set `patch_deadline_settings` in GitOps.

GitHub issue: [#39176](https://github.com/fleetdm/fleet/issues/39176)

### Multiple VPP tokens in one fleet

IT admins can now assign more than one Volume Purchasing Program (VPP) token to a fleet. This helps international organizations that keep all their iPhones and iPads in one fleet. For example, assign both a US and a Japan token to the fleet, and add Japan-only App Store apps from the Japan token. A token assigned to "All fleets" is now available alongside fleet-specific tokens.

When you add an App Store app, choose which token it comes from. Each app installs with the token it was added from. In the API, pass `vpp_token_id` to [add App Store app](https://fleetdm.com/docs/rest-api/rest-api#add-app-store-app). In GitOps, use [`location`](https://fleetdm.com/docs/configuration/yaml-files#multiple-vpp-tokens) to pick the token.

Learn more in the [Apple MDM setup guide](https://fleetdm.com/guides/apple-mdm-setup#volume-purchasing-program-vpp).

GitHub issue: [#53835](https://github.com/fleetdm/fleet/issues/53835)

### Deploy certificates from EJBCA to Apple hosts

IT admins who use [EJBCA](https://www.ejbca.org/) as their certificate authority (CA) can now deploy certificates to macOS, iOS, and iPadOS hosts, so end users can connect to corporate Wi-Fi or VPN. Add EJBCA in **Settings > Integrations > Certificate authorities**, with the API, or in [GitOps](https://fleetdm.com/docs/configuration/yaml-files#ejbca). Then use the `$FLEET_VAR_EJBCA_DATA_<name>` and `$FLEET_VAR_EJBCA_PASSWORD_<name>` variables in a PKCS#12 configuration profile.

Learn how in the [certificates guide](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#ejbca).

GitHub issue: [#30986](https://github.com/fleetdm/fleet/issues/30986)

### Deploy certificates from Google CAS to Linux hosts

IT admins who use Google Cloud's [Certificate Authority Service (CAS)](https://cloud.google.com/certificate-authority-service) can now deploy certificates to Linux hosts. Connect Fleet to Google CAS, then deploy a script-only package that requests a certificate with Fleet's [request certificate API](https://fleetdm.com/docs/rest-api/rest-api#request-certificate). A policy runs the same script to renew the certificate before it expires.

Learn how in the [certificates guide](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#google-cas).

GitHub issue: [#52623](https://github.com/fleetdm/fleet/issues/52623)

## Changes

- Removed fleetd's built-in Nudge integration. Fleet no longer sends Nudge settings to fleetd for hosts on macOS 13 and older, which are no longer supported. Hosts on macOS 14 and later continue to get OS update enforcement via DDM. To keep using Nudge, deploy it with the Nudge Fleet-maintained app. Learn how in the [Nudge guide](https://fleetdm.com/guides/enforce-os-updates-on-macos-with-nudge).

### IT Admins
- Added Okta conditional access for Windows hosts (Fleet Premium): a new `$FLEET_VAR_CONDITIONAL_ACCESS_SCEP_URL` variable for Windows SCEP profiles, a `GET /api/v1/fleet/conditional_access/idp/windows/profiles` endpoint, and a new Okta conditional access page in Settings that shows the macOS and Windows profiles.
- Added `PUT` and `DELETE /api/v1/fleet/hosts/:id/google_device_id` endpoints (Fleet Premium) to bind an iOS, iPadOS, or Android host to its device ID in Google and to remove that binding. Once set, a host's Google device ID can only be removed with `DELETE`, and each ID can only be bound to one host.
- Added a `google_device_id` field to the host details response for iOS, iPadOS, and Android hosts. The Google device ID is shown on the host details page once set.
- Added multiple versions of the same App Store app in a fleet for iOS, iPadOS, and Android (Fleet Premium). Each version has its own configuration, automatic update window, self-service setting, categories, and labels.
- Added a built-in MCP server to Fleet Premium at `/api/v1/fleet/mcp`, on by default and turned off with `server_settings.mcp_disabled`. AI harnesses connect with an API-only user's token and only get the tools that user's API endpoints allow.
- Added Streamable HTTP transport to the standalone `fleet-mcp` server.
- Removed the Render blueprint and Render deployment instructions for the standalone `fleet-mcp` server. In Fleet Premium, use the MCP server built into Fleet instead.
- Added AI tools to software inventory from fleetd's `ai_tools` table (Fleet Premium). MCP servers, AI CLI tools, and AI skills are new software types. AI apps and extensions Fleet already reports under another type are marked as AI tools instead.
- Added an "AI tools" filter and the new types to the software filters on the **Software** page, host details, and **My device**, plus an `ai_tool` filter and field on the software, host software, and device software API endpoints (Fleet Premium).
- Added a "Deadline" option for macOS Fleet-maintained app patch policies. When the app is open, Fleet Desktop tells the end user 1 hour before the deadline and again 5 minutes before, then closes and updates the app. Requires Fleet Desktop 1.5.0.
- Added patch deadline settings (days and time) per fleet and for "Unassigned" in Software > Library > Settings, the API, and GitOps. Deadlines use each host's local time. The default is Tuesday at 17:00.
- Fleet now collects the host's timezone on macOS, Linux, and Windows and shows it on the host details page.
- Added support for assigning multiple VPP tokens to a fleet. A VPP token on "All fleets" is now available alongside fleet-specific tokens. When adding an App Store app, you choose the VPP token, and each app installs with the token it was added from.
- Added `vpp_token_id` to the list and add App Store app API endpoints. It's required when a fleet has more than one VPP token.
- Added `location` to App Store apps in GitOps to pick the VPP token. Apps without `location` come from the fleet's first-added VPP token, and apps already added keep their token.
- Deleting a VPP token now removes its apps from every fleet.
- Added EJBCA as a certificate authority type (Fleet Premium). Add it in **Settings > Integrations > Certificate authorities**, with the API, or with GitOps (`org_settings.certificate_authorities.ejbca`), and use the `$FLEET_VAR_EJBCA_DATA_<name>` and `$FLEET_VAR_EJBCA_PASSWORD_<name>` variables in the PKCS#12 payload of a macOS, iOS, or iPadOS configuration profile to deliver a certificate to each host.
- A configuration profile that can't get a certificate from an EJBCA or DigiCert certificate authority, because the certificate authority is temporarily unavailable, is now retried up to 3 times before it is marked as failed.
- Added Google Cloud Certificate Authority Service (CAS) as a certificate authority type, so Linux hosts can get certificates from it through the request certificate API.
- Added support for `$FLEET_VAR_HOST_UUID` and the `$FLEET_VAR_HOST_END_USER_IDP_*` variables in a DigiCert certificate authority's common name, seat ID, and UPN. Certificates that use an IdP variable are reissued when the host's IdP values change.
- Editing a DigiCert certificate authority's common name, seat ID, UPN, or profile ID, in the UI, API, or GitOps, now reissues certificates to every host that has a certificate from that certificate authority.
- Added the `$FLEET_VAR_HOST_END_USER_IDP_EXTERNAL_ID` variable, which resolves to the host end user's SCIM `externalId`, for configuration profiles, declarations, scripts, managed app configurations, certificate templates, and host name templates. It's also supported in DigiCert certificate authority common name, seat ID, and user principal name fields.
- Added the end user's IdP external ID (SCIM `externalId`) to the User card on the host details page.
- Allowed `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in the common name (CN) of Windows SCEP profiles, so certificates from certificate authorities that drop the organizational unit (OU), like Okta, are verified and renewed.
- Added a target option to Add software > App store > Android so Google Play apps can be scoped to hosts with specific labels when they are added.
- Changed Android apps and configuration changes to be sent to hosts in a cron job, to correctly update hosts after label changes, fleet deletion, and GitOps runs, and to stop unnecessary resends.
- Added an "After setup" page for Android end users, shown after selecting Enroll, so the enroll button can't be selected twice.
- Added `macos_enable_bootstrap_package_during_manual_enrollment` and `macos_enable_bootstrap_package_during_migration` setup experience settings to install a configured bootstrap package on a manual enrollment and MDM migration respectively.
- Deprecated the `FLEET_ALLOW_BOOTSTRAP_PACKAGE_DURING_MIGRATION` environment variable in favor of the `macos_enable_bootstrap_package_during_migration` setting. If the variable is set when upgrading, the setting is turned on for "Unassigned" and every fleet.
- Combined assets into the configuration profiles list in Controls > OS settings. Assets are now added from the "Advanced" button next to "Add profile".
- Added API endpoints that use AI to generate a configuration profile, or its name and description, from plain-language instructions, checking the profile the way Fleet does when it is added.
- Added profile validation, with a lenient mode for uploaded profiles and a strict mode for AI-generated profiles. The `.mobileconfig` upload parser is now slightly stricter.
- Added a warning when a Windows configuration profile is deleted in the UI, the API, or `fleetctl gitops`, explaining that some Windows settings may not be cleanly removed from hosts.
- Added the Fleet-maintained app slug, App Store ID, or Google Play ID to the software title details page in Fleet Premium.
- Renamed "Add package", "Edit package", and "Delete package" to "Add version", "Edit version", and "Delete version" for custom packages on the software details page, matching App Store app versions.
- The "Update details" modal now explains that a macOS app installed in multiple places may keep showing an update, since updating only replaces the copy in the Applications folder.
- Added Parrot OS as a recognized Linux platform. Hosts running Parrot OS now enroll with `platform=parrot`, support `.deb` software installs, and have CVEs matched against the underlying Debian OVAL feed (Parrot 4 → Debian 10, 5 → Debian 11, 6 → Debian 12). Unknown future Parrot versions fall through to an unsupported platform string so vulnerability scanning is skipped rather than served stale data from an aging Debian feed.
- Added Omarchy (4 and later) to the Linux distributions Fleet escrows LUKS disk encryption keys for, and fixed the escrowed-key verification query so it finds the LUKS device when the root filesystem is a btrfs subvolume that lsblk lists only under "mountpoints".
- Fleet now collects uptime and last restarted time from ChromeOS hosts running a fleetd Chrome extension version that includes the `uptime` table.
- Windows Autopilot devices with a placeholder hardware serial (for example "Default string") now appear in Fleet as pending hosts, and every Autopilot device is matched to its pending host by its Autopilot device ID when fleetd enrolls, so devices that share a serial each stay a single host. Requires fleetd v1.64.0.
- Added a banner that tells global users when enrolled hosts are more than 20% over their license count, with a link to a guide for adding hosts (Fleet Premium).
- Added Azure Blob Storage support (via new `FLEET_CLOUD_STORAGE_AZURE_FILES_*` environment variables) as an alternative to S3 for storing software installers, bootstrap packages, software title icons, and org logos.
- Added the experimental `websocket.osquery_config_enabled` server setting: hosts connected over the agent WebSocket transport are notified when their osquery config changes (agent options, reports, fleet changes), and fetch it every 10 minutes (`websocket.osquery_config_poll_interval`) as a fallback instead of every minute.
- Rewrote the `fleet-gitops` agent skill that `fleetctl new` scaffolds into GitOps repositories and made it available to every major coding agent: the skill now lives at `.agents/skills/fleet-gitops/` (read by Codex, Cursor, GitHub Copilot, Gemini CLI, and Kilo Code) with a Claude Code entry under `.claude/skills/`, instructions moved to `AGENTS.md` (imported by `CLAUDE.md` and `GEMINI.md`), and the skill ships per-platform references plus an offline validator (`scripts/validate.py`) that checks paths, label references, osquery tables, Fleet-maintained app slugs, and profile structure before a dry run.
- `fleetctl config set` now accepts an explicit `--tls-skip-verify=false` and `--rootca ""`, so a context can turn certificate verification back on or clear its custom root CA without editing the config file by hand.
- Improved the MDM command details modal to show the configuration profile sent in an `InstallProfile` command instead of base64-encoded data, and to keep long lines in the request payload from wrapping below their indentation.
- Apple MDM commands in the MDM command modal are now indented with spaces instead of tabs, so the request is easier to read.
- Added the number of configuration profiles generated with AI, per platform, to anonymous usage statistics.
- Added LM Studio as a macOS Fleet-maintained app.

### Security Engineers
- Windows BitLocker recovery keys are now encrypted with the server private key instead of the Windows MDM (WSTEP) certificate, so replacing the certificate no longer makes escrowed keys unrecoverable. Keys escrowed before this release are re-encrypted automatically when they can still be read, and a host whose stored key can't be read shows "Enforcing" and is asked to escrow a new one.
- Escrowing Windows BitLocker keys now requires `server.private_key`, so deployments that turned on Windows disk encryption without one should set it when upgrading.
- Hardened Windows MDM device authentication. Windows MDM devices that enrolled before Fleet 4.81.0, have not connected to Fleet MDM since, and have not checked in to Fleet for more than 180 days, or whose host was removed from Fleet, can't reconnect to MDM until fleetd checks in again. Devices that can't check in must be re-enrolled in MDM.
- Fixed users with the Technician role being able to run ad hoc scripts (`script_contents`) through the API and `fleetctl run-script`. Technicians can now only run saved scripts.
- Fixed a bug where sessions for other users were incorrectly marked as accessed/active when listing or getting sessions via the API. This could cause the `status` or `last_activity_at` for a user to be incorrect and result in sessions being valid indefinitely if sessions are listed more frequently than the configured `session_duration` (default: 5d).
- Single sign-on now only accepts a relative path for `relay_url` when initiating a login, as the REST API documents. Absolute URLs and protocol-relative values are rejected with a 400 instead of being used as the post-login destination.
- Fixed the initial setup endpoint so it can no longer be reopened after setup by a lagging read replica or emptied users table, and concurrent setup requests can no longer create multiple admins or overwrite app config and enroll secrets.
- Fleet now checks the Apple account provisioning token URL, client ID, and client secret with your identity provider when they change, and rejects settings the IdP doesn't accept.
- ACME endpoints now reject requests whose embedded JWK is not an EC public key before parsing the JWS.
- Updated `go-jose` to v4 for Apple Platform SSO.
- Google Workspace and Google Calendar integrations now perform their OAuth token exchange through Fleet's outbound HTTP client, so the same outbound network settings apply to them as to other integrations.
- Fixed false positive vulnerabilities on RHEL-based hosts for patched RPM packages that have an epoch (e.g. `mod_ssl`, `tomcat`, `openssl`), which were always reported as older than Red Hat's fixed version. Fleet now collects the RPM package epoch and uses it in the comparison.
- Improved vulnerability detection for Android apps: CVEs filed only for other platforms are no longer reported, CVEs that apply only on Android are now detected, and Chrome for Android now reports its vulnerabilities (previously matched to the wrong product). Android OS vulnerabilities continue to be reported on the operating system, not on the "Android System" app.
- Fixed desktop VLC being reported vulnerable to CVE-2026-26227 and CVE-2026-26228, which affect only VLC for Android.
- Fixed the `Brotli` Python package and the `brotli` Homebrew package being matched to the CPE for the Go port of brotli (`andybalholm:brotli`) instead of Google's, which caused missed vulnerabilities.
- Amazon Linux 2022 (the AL2023 Tech Preview) is no longer supported for vulnerability scanning; Amazon has retired its package repository. AL2023 hosts are unaffected.
- Updated `golang.org/x/net` to v0.62.0.

### Bug fixes and improvements
- Database table migrations that are missing are now applied even when newer migrations were already applied (for example, a migration shipped in a patch release after later-numbered migrations). Out-of-order application only covers table migrations with a version of `20260101000000` or later; older gaps are skipped with a log message, as before.
- Reduced the size of the `fleet` binary and `fleetdm/fleet` Docker image by no longer embedding unused images and videos, or a second copy of the ones the UI uses.
- Sped up applying and replacing large sets of Windows configuration profiles on many hosts, and reduced the database lock contention with MDM check-ins while it runs.
- Improved performance of the policy automation runs page.
- Fleet's dashboard now loads its host counts with one request instead of two.
- Fleet now logs how many hosts the upcoming-activities maintenance cron unblocked whenever a run unblocks at least one, along with the run's cap and whether it was reached.
- Resending an Apple configuration profile, manually or after an IdP user or certificate authority change, now gives it a fresh set of verification retries.
- Users with the GitOps role can now list Apple Business (AB) tokens, so `fleetctl gitops` no longer warns that it's "skipping setup assistant ABM coverage check" and can warn about setup assistants that won't take effect.
- `fleetctl gitops` on Fleet Free now returns an error for custom EST and Smallstep certificate authorities, which require Fleet Premium, instead of ignoring them.
- Windows SCEP profiles that quote a value containing a Fleet variable with `&quot;` are now rejected at upload, instead of failing on the host. Use a literal `"` instead.
- Updated the error messages for a missing or repeated `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in SCEP profiles to say it goes in the OU, or in some use cases the CN.
- Updated the error shown when an Android configuration profile includes a software setting (like `playStoreMode`): Fleet Premium users are pointed to Play Store app management in Fleet, and Fleet Free users are told Android software management is Premium only.
- Updated Host > Controls and My device to show Android certificate errors as reported by the Fleet Android app, which now reports them in plain language.
- Updated the install status tooltip for Android Play Store apps to clarify that counts come from setup experience and don't include apps installed via self service.
- Added the certificate authority's name to the activity recorded when a Hydrant certificate authority is added through the REST API.
- In GitOps mode, "Edit appearance" on any software title and "Edit software" on an Apple VPP (App Store) title now open their modals in a disabled state so admins can see the applied settings. Save remains disabled with the GitOps tooltip.
- Replaced the remaining "team" wording in the UI with "fleet" and removed the "team" search terms from the command palette.
- Updated GitOps `install_software` policy errors to say "fleet" instead of "team" when the referenced software package, App Store app, or Fleet-maintained app isn't found.
- Updated the error shown when deleting a custom variable that's still in use to say "fleet" instead of "team", and "Unassigned" instead of "No team".
- Added missing external link icons so it's clearer when a link or button takes you outside of Fleet.
- Added an ending period to the host details Software > Library tab subheader to match the Inventory tab.
- Fixed Windows hosts not getting certificates from custom SCEP certificate authorities that select the CA by name, such as EJBCA.
- Fixed Fleet losing track of a certificate, so it was never renewed, after moving an Apple host to a fleet with an identical certificate profile.
- Fixed configuration profiles and certificates that use `$FLEET_VAR_HOST_END_USER_IDP_USERNAME` or `$FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART` not being resent when a host's IdP username was set to a value that doesn't match a SCIM user.
- Fixed the certificate authorities batch API (`POST /api/v1/fleet/spec/certificate_authorities`) deleting a certificate authority when a request changed only the letter case of its name. The certificate authority is now renamed, and two certificate authorities of the same type whose names differ only in case are rejected as duplicates.
- Fixed GitOps recording an edited activity and calling the external service for unchanged custom SCEP, Hydrant, custom EST, and Smallstep certificate authorities on every run.
- Fixed GitOps not trimming surrounding whitespace from certificate authority names and URLs, DigiCert profile IDs, and Smallstep challenge URLs and usernames before validating and storing them.
- Fixed the certificate authority update API accepting the masked value ("********") as a DigiCert API token, Hydrant client secret, custom EST password, custom SCEP challenge, or Smallstep password, which replaced the stored secret with the mask.
- Fixed a bug where labels set when adding an Android app were ignored, so the app was made available to every Android host in the fleet.
- Fixed a bug where adding, deleting, or setting Android apps via GitOps, and resending their configuration, could change the default policy every Android device enrolls with, or send the change to an empty policy, for hosts that had not yet moved to their own policy.
- Fixed a bug where omitting an Android VPP app's `configuration` key in a GitOps file did not clear the stored configuration.
- Fixed Android certificate delivery calling Google before a newly enrolled host's policy existed, which failed with a 404, and stopped logging "policy request not found" errors on status reports with no pending profiles.
- Fixed an Android wipe command staying pending forever when its host was deleted from Fleet before the wipe was acknowledged.
- Fixed an Android host's disk space disappearing from host vitals when a status report carries no storage measurement. Fleet now keeps the host's last measurement until the device reports a new one, and hosts that have never reported a measurement still show "Not supported".
- Hid Unenroll, Lock, and Clear passcode from the Host details Actions menu while an Unenroll (or Wipe) is pending on a personally-owned (BYOD) Android host. `GET /api/v1/fleet/hosts/{id}` now reports `pending_action: "unenroll"` for such hosts, and a second Unenroll request returns 409 instead of sending another work-profile wipe.
- Fixed the QR code on the `/enroll` page not updating when switching between Personal (BYOD) and Company-owned, so scanning it on a mobile device now opens the page with the selected option.
- Fixed the host details API reporting wipe, lock, and clear passcode as allowed for hosts enrolled with a Managed Apple Account (account-driven user enrollment).
- Fixed `POST /api/v1/fleet/hosts/:id/clear_passcode` accepting requests for iOS/iPadOS hosts whose MDM is turned off, which queued a command that never reached the device and logged a `cleared_passcode` activity. It now returns a 400 error.
- Fixed App Store (VPP) apps being listed and added, including via GitOps, while Apple MDM is turned off.
- Fixed fleetd installing only after all configuration profiles on manually enrolled Macs. Fleet now sends the fleetd install first, as it already does for automatic (ADE) enrollment.
- Fixed a macOS setup experience script step staying "running" (so the host was never released) when saving the script result partly failed.
- Fixed macOS hosts that had already completed an Apple Business MDM migration skipping setup experience on a later Automated Device Enrollment after Apple Business reported a new migration deadline for them, and improved logging of enrollment flows.
- Fixed refused Apple automatic enrollments showing a username and password prompt with raw error text in Setup Assistant.
- Fixed an issue with correctly resetting AB assignments when not listed.
- Fixed the host's **MDM status** modal showing an Apple Business error for a host whose Apple Business (AB) token was deleted.
- Fixed Recovery Lock passwords stuck at "Enforcing" for macOS hosts that had a password change in progress when Fleet was upgraded from 4.92.
- Fixed Windows hosts showing the managed local account as not created after their UUID changed. On upgrade, affected hosts are asked to create the account again, which resets its password and escrows it under the host's current UUID.
- Fixed Windows hosts without BitLocker (for example, Windows Server with the BitLocker feature turned off) that stopped returning vitals, policy, label, and live report results when their fleet enforced disk encryption.
- Fixed the Windows MDM custom command API timing out (502) when a command targets thousands of hosts, and the command result details taking over a minute to load when a command was sent to thousands of hosts.
- Fixed SSO login getting stuck on "Redirecting to Fleet..." and the Windows MDM enrollment terms of use page failing to continue when `FLEET_SERVER_ENABLE_CSP` is enabled.
- Fixed the ChatGPT Desktop macOS Fleet-maintained app's patch policy, automatic install policy, and "app is open" check not recognizing the current ChatGPT app. OpenAI rebuilt ChatGPT on its Codex app, which reports the `com.openai.codex` bundle identifier instead of `com.openai.chat`. These queries now match both identifiers and ignore the discontinued Codex app, which also uses `com.openai.codex`.
- Fixed Fleet-maintained apps on macOS not relaunching an app after an update when its bundle identifier contains a character other than a letter, digit, dot, dash, or underscore. Pale Moon, whose bundle identifier contains a space, was quit during an update and never reopened.
- Fixed Fleet-maintained apps whose macOS and Windows names differ (e.g. "R for macOS" and "R for Windows") changing their name with the sort direction on the **Add software > Fleet-maintained** page. These apps now show their name without the platform suffix (e.g. "R").
- Fixed Fleet-maintained app endpoints returning a 500 instead of a clear error message when Fleet couldn't fetch the app's manifest.
- Fixed GitOps listing a Fleet-maintained app or package as deleted on every run when it was attached to an existing software title by name, such as a macOS title without a bundle identifier.
- Fixed the `software_installer_id` and `app_store_app_version_id` filters allowing IDs that don't belong to the software title.
- Fixed software title endpoints returning package information from the Unassigned fleet when `fleet_id` is not provided.
- Fixed two confusing validation messages for `.ps1` packages: `setup_experience_platform` on a `.ps1` now reports that the field is unsupported for that package type instead of showing an empty allowed list, and uploading a `.ps1` with a `pwsh` shebang now returns the PowerShell shebang error instead of the shell/Python interpreter list.
- Fixed `fleetctl gitops` and the batch APIs accepting configuration profiles, scripts, reports, and labels whose names differ only by letter case, accents, or trailing spaces, and then saving only one of them. These are now rejected as duplicates.
- Fixed `fleetctl gitops` and the label spec API (`POST /api/latest/fleet/spec/labels`) overwriting a built-in label, such as "All Hosts", when a label's name matched the built-in name except for accents or trailing spaces (for example "Àll Hosts" or "All Hosts "). These labels are now rejected as conflicting with a built-in label.
- Fixed the `POST /api/v1/fleet/configuration_profiles/batch` and `POST /api/v1/fleet/mdm/profiles/batch` endpoints deleting every configuration profile when the request body had no profile list. They now return a validation error.
- Fixed `fleetctl gitops` leaving a fleet with no bootstrap package when uploading its replacement failed. The existing package is now replaced only once the new one is saved.
- Fixed `fleetctl gitops` rejecting macOS disk encryption for a fleet when Apple MDM is off. It's now accepted, the same as the disk encryption API.
- Fixed saving one fleet webhook (host status, failing policies, or host activities) turning off the fleet's other webhooks. This also applies to "Unassigned".
- Fixed `GET /api/v1/fleet/hosts` with `populate_software` holding every host's software in memory until the whole page was sent, which could run Fleet servers out of memory on large pages.
- Fixed `/healthz` waiting for a database connection behind request traffic when the MySQL connection pool was full, which caused load balancers to replace healthy Fleet servers under load. The health check now uses its own connection to the primary (and to the read replica, if configured) and fails within 3 seconds if the database does not answer. Each Fleet server opens one or two more MySQL connections as a result.
- Fixed the vulnerabilities cron rescanning the software table on every unused-software cleanup batch, which made `cron_sync_host_software` take hours at large host counts.
- Fixed chart data cleanup repeatedly failing with MySQL lock wait timeouts on deployments where thousands of vulnerabilities change in the same hour.
- Fixed closed live reports holding a Redis connection open for up to an hour.
- Fixed `POST /api/latest/fleet/hosts/transfer` with an unknown fleet and `GET /api/latest/fleet/scripts/batch/:batch_execution_id/host_results` with an unknown batch returning 500 instead of 404.
- Fixed the "transferred hosts" activity listing a host once per time it appeared in the transfer request, and, for transfers by filter made by users on more than one fleet, pairing host IDs with the wrong host names.
- Fixed `GET /api/v1/fleet/labels/:id/hosts` and the hosts CSV report filtered by a label returning no `orbit_version` and `fleet_desktop_version`. The Hosts page **Agent** column now shows the same version and tooltip with or without a label filter.
- Fixed sorting a label's host list by the Agent column, which returned a 422 "invalid order_key" error.
- Fixed an issue with the "New hosts" filter not excluding pending hosts, and not showing newly enrolled AB hosts.
- Fixed `fleetctl` printing "unknown msg type error" instead of the server's error message when a live report fails.
- Fixed `fleetctl report --labels` without `--hosts` also running the live report on every host with an empty hardware serial.
- Fixed `fleetctl package --type msi --arch arm64` ignoring `--outfile`.
- Fixed an issue where `fleetctl` on Windows required `rootca` or `tls-skip-verify` in the context. `fleetctl` now verifies the Fleet server's certificate against the Windows certificate store by default, the same as on macOS and Linux.
- Fixed the UI showing error states instead of redirecting to the login page when a user's session expired.
- Fixed an error page on the live report page when a browser extension that modifies page text (e.g. text masking extensions) was active while results streamed in.
- Fixed the conditional access bypass toggle, bypass Save, and Okta Delete staying interactive in GitOps mode even though their underlying fields are managed by GitOps YAML.
- Fixed self-service categories that could still be added, edited, and deleted from the UI while GitOps mode was enabled. The Add category, Edit, and Delete buttons are now disabled with a GitOps tooltip on hover.
- Disabled the bootstrap package uploader in GitOps mode, and fixed the delete button jumping when hovered in GitOps mode.
- Fixed the dashboard activity feed showing the fleet name instead of the self-service category name.
- Fixed the APNs certificate expiration banner staying visible after renewing the certificate in **Settings > Integrations > MDM > Apple** until the page was reloaded.
- Fixed the Account provisioning settings showing the same error in two toasts when a field is rejected on save.
- Fixed Settings forms (Okta ACS URL, Fleet web address, SMTP sender address, ticket destination URL/email) showing a validation error for a valid URL or email with a leading or trailing space, which Save accepts after trimming.
- Fixed the command palette's "Manage software automations" option sometimes opening the Software page without the Manage automations modal.
- Fixed the Apps row in the notify before patching activity details modal cutting off the list of apps. The row now wraps so that every app can be read.
- Fixed a bug where wide organization logos appeared too small in macOS patch notifications.
- Fixed an issue where the mobile Self-service UI was rendered on desktop when the browser window was shrunk. The mobile UI now only renders for actual iOS/iPadOS devices; a narrow desktop viewport shows the standard unsupported-screen overlay instead.
- Fixed the JSON and XML editors (software Edit configuration and custom OS settings) requesting a missing `worker-json.js`/`worker-xml.js` file and throwing an uncaught error when opened.
- Fixed a two-finger horizontal swipe over the script, SQL, and YAML editors navigating the browser back to the previous page.
- Fixed disabled bulk-action buttons on the Hosts page, such as **Run script**, still receiving keyboard focus and opening with Enter.
- Fixed tooltips in modals opened from the dashboard activity feed, such as the performance impact tooltip in the report modal, appearing away from the text they describe.
- Fixed the policy automations table overflowing the edit policy page and the manage automations modal at narrow widths or with long fleet names.
- Fixed the Policies page search field not spanning the full width at narrower screen sizes, and the Automations column showing the browser's default focus outline instead of Fleet's when navigating with the keyboard.
- Fixed uneven line wrapping in the disk encryption tooltips for macOS and Windows hosts on the host details and My device pages.
- Fixed a bug where long host names overlapped neighboring columns in the hosts table on the Hosts page.
- Fixed the page behind a modal shifting sideways when the modal opened on a page with a visible scrollbar.
- Fixed selected-hosts table on **Select targets** rendering a 2px overhang past the "No matching hosts" dropdown that covers it.
- Fixed the certificates page's empty state to match the configuration profiles and variables pages.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.95.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Luke Heath">
<meta name="authorGitHubUsername" value="lukeheath">
<meta name="publishedOn" value="2026-11-16">
<meta name="articleTitle" value="Fleet 4.95.0 | Okta conditional access for Windows, built-in MCP server, AI tools in inventory, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.95.0-1600x900@2x.png">
