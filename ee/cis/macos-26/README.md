# macOS 26 Tahoe — CIS benchmark

Fleet policies for the **CIS Apple macOS 26 Tahoe Benchmark, v1.1.0**.

## Status

**Generation complete.** All automated recommendations across
§1–§6 of the CIS Apple macOS 26 Tahoe Benchmark v1.1.0 are
covered. §7 (Supplemental) is skipped per Fleet convention.
Manual-only recommendations are documented in **Limitations**.

### v1.1.0 update

Updated from v1.0.0 to v1.1.0 (CIS "Jun 1, 2026" release). Changes:

- **2.3.5 Device Management** — new informational sub-section only (no recommendation), so no policy.
- **2.7.1** rescoped to the *current user* only and moved to Level 1; the query now checks the current console user's Dock hot corners.
- **3.4** retitled "…Retained for 30 Days"; requirement relaxed to `expire-after:` ≥ 30 days (no size requirement).
- **3.5** — only the CIS *remediation* changed (`chmod 700`), but its *audit* still checks 440, so Fleet's query is unchanged (it accepts 440/400).
- **5.1.6** — CIS added `2>/dev/null`; `find_cmd` already handles that, so the query is unchanged.
- **5.1.7** — query now excludes the non-accessible `/Library/AppStore` folder.
- **5.3.1** — CIS split disk encryption into internal (5.3.1) and external (5.3.2) and moved the internal APFS check to **Automated**; a new 5.3.1 policy was added (previously Manual/limitation). 5.3.2 (external) and 5.3.3 (FAT32/ExFAT) remain in **Limitations**.
- **5.6** — audit now detects a lingering root *secure token*; the existing `AuthenticationAuthority` query already covers it, so only the resolution and pass script changed (`fdesetup remove -user root`).

## Sections covered

| Section | Title | Status |
|---------|-------|--------|
| 1 | Install Updates, Patches and Additional Security Software | complete (6/6 automated) |
| 2 | System Settings | complete (all automated — §2.1–§2.18) |
| 3 | Logging and Auditing | complete (5/5 automated) |
| 4 | Network Configurations | complete (3/3 automated) |
| 5 | System Access, Authentication and Authorization | complete (20/20 automated — incl. 5.3.1 added in v1.1.0) |
| 6 | Applications | complete (7/7 automated) |
| 7 | Supplemental | skipped (per convention) |

## Limitations

Manual-assessment recommendations cannot be automated as Fleet
policies. They are listed here for reference so auditors know where to
perform out-of-band checks.

- **2.3.3.11** Ensure Computer Name Does Not Contain PII or Protected
  Organizational Information (Level 2). Requires inspection of the
  hostname against organizational naming policy — not a mechanically
  checkable condition.
- **2.3.4.2** Ensure Time Machine Volumes Are Encrypted (Level 1,
  Automated). The query detects an unencrypted backup destination,
  but remediation is GUI-only (drive must be re-added with "Encrypt
  Backup" checked). No shippable script/profile.
- **2.5.2.2** Ensure Listen for (Siri) Is Disabled (Level 1,
  Manual). Per CIS, Hey Siri cannot be disabled via profile or
  plist — only through the GUI — so the recommendation was
  explicitly moved to Manual in this benchmark. Disabling Siri
  entirely (policy 2.5.2.1) is the proxy control.
- **2.6.1.3** Audit Location Services Access (Level 2, Manual).
  Requires per-application review of which apps hold location
  permission — policy-driven, not mechanical.
- **2.6.2.1** Audit Full Disk Access for Applications (Level 2,
  Manual). Requires per-application review of the Full Disk Access
  list against organizational policy.
- **2.6.3.5** Ensure Share iCloud Analytics Is Disabled (Level 1,
  Manual). Setting is per-user and only appears when the user is
  signed into a personal Apple Account — there is no profile key
  or systemwide plist.
- **2.6.7** Audit Lockdown Mode (Level 2, Manual). Lockdown Mode
  is per-user (`.GlobalPreferences.plist` key `LDMGlobalEnabled`)
  and CIS does not prescribe a required value — organizations
  must decide per user/device.
- **2.1.1.1, 2.1.1.2, 2.1.1.4, 2.1.1.5, 2.1.1.6, 2.1.2** All
  iCloud / Apple Account audits in §2.1 are Manual — require
  per-user review of iCloud Passwords & Keychain, iCloud Drive,
  security keys, Freeform sync, Find My Mac, App Store password
  settings against organizational policy.
- **2.4.1** Audit Menu Bar and Control Center Icons (Level 2,
  Manual). Per-user review of menu bar configuration.
- **2.7.2** Audit iPhone Mirroring (Level 2, Manual).
  Organization-defined allow/deny decision.
- **2.8.1** Audit Universal Control Settings (Level 2, Manual).
  Organization-defined decision.
- **2.10.1.1** Ensure the OS Is Not Active When Resuming from
  Standby (Intel) (Level 2, Manual). The audit requires the
  tester to pick between different remediation paths depending
  on whether the Mac is Intel vs Apple Silicon.
- **2.12.2** Audit Touch ID (Level 1, Manual). Per-user
  verification of enrollment and use against organizational
  policy.
- **2.14.1** Audit Game Center Settings (Level 2, Manual).
- **2.15.1** Audit Notification Settings (Level 2, Manual).
- **2.16.1** Audit Wallet & Apple Pay Settings (Level 2, Manual).
- **2.17.1** Audit Internet Accounts for Authorized Use (Level 2,
  Manual).
- **3.6** Audit Software Inventory (Level 2, Manual). Requires
  per-organization review of installed software against an
  approved inventory — not mechanically checkable.
- **5.2.3** Ensure Complex Password Must Contain Alphabetic
  Characters (Level 2, Manual). CIS explicitly left as Manual —
  Fleet does not ship an automated policy.
- **5.2.4** Ensure Complex Password Must Contain Numeric
  Character (Level 2, Manual).
- **5.2.5** Ensure Complex Password Must Contain Special
  Character (Level 2, Manual).
- **5.2.6** Ensure Complex Password Must Contain Uppercase and
  Lowercase Characters (Level 2, Manual).
- **5.3.2** Ensure all APFS and HFS+ external user storage volumes
  are encrypted (Level 1, Automated). The fleetd `apfs_volumes`
  table does not expose an internal/external indicator, so
  "external" volumes cannot be reliably identified as a policy
  query. Internal APFS volumes are covered by the automated 5.3.1
  policy. (In v1.0.0 the single APFS/CoreStorage check was Manual;
  v1.1.0 split it into internal 5.3.1 (now Automated — shipped) and
  external 5.3.2 (this limitation).)
- **5.3.3** Audit Connected FAT32 and ExFAT Drives (Level 2,
  Manual). CIS ships this as a Manual audit — an organizational
  review of connected removable drives, not a mechanically
  checkable condition.
- **6.1.1** Audit Show All Filename Extensions (Level 2, Manual).
  Per-user Finder preference.
- **6.2.1** Ensure Protect Mail Activity in Mail Is Enabled
  (Level 2, Manual). Per-user Mail preference.
- **6.3.2** Audit History and Remove History Items (Level 2,
  Manual). Organization-defined retention window.
- **6.3.5** Audit Hide IP Address in Safari Setting (Level 2,
  Manual). Organization-defined; also requires FDA to read
  per-user Safari preferences.
- **6.3.8** Audit AutoFill (Level 2, Manual). Organization-defined.
- **6.3.9** Audit Pop-up Windows (Level 1, Manual). Per-user
  Safari setting with organization-defined allow-list.
- **6.5.1** Audit Passwords (Level 1, Manual). Requires in-app
  review of the macOS Passwords app.

### Section 1 notes

- **1.1** depends on the fleetd-specific `software_update` osquery
  table. Hosts running upstream osquery without fleetd will be unable
  to evaluate this policy.
- **1.6** (software update deferment) also passes when no deferment
  profile is installed — the query checks for a managed value
  exceeding 30 days and treats absence as compliant. The
  `1.6.mobileconfig` artifact sets `enforcedSoftwareUpdateDelay=30`
  and `forceDelayedSoftwareUpdates=true` to satisfy Apple's
  requirement that both keys be present in the same profile.

### Section 2.2 notes

- **2.2.1** (Firewall) and **2.2.2** (Stealth Mode) — both use the
  osquery `alf` table (Query pattern #1) which reflects live
  firewall state, so scripts toggling via
  `/usr/libexec/ApplicationFirewall/socketfilterfw` are the primary
  test mechanism.
- The CIS benchmark explicitly requires `EnableFirewall` and
  `EnableStealthMode` to be in the **same** configuration profile
  ("If it is set in its own configuration profile, it will fail").
  We ship a single combined `2.2.1-and-2.2.2.mobileconfig` covering
  both keys — first use of the `{id1}-and-{id2}` naming convention
  in this benchmark.

### Section 2.3.1 notes

- **2.3.1.1** (AirDrop) and **2.3.1.2** (AirPlay Receiver) are
  profile-only by CIS's own specification — the benchmark explicitly
  notes that these settings can only be enabled or disabled via
  configuration profile. No test scripts are shippable; the runner
  validates by pushing the `.mobileconfig` and re-evaluating the
  `managed_policies` query.

### Section 2.3.2 notes

- **2.3.2.1** (Set Time and Date Automatically) — the PDF's audit
  uses `systemsetup -getusingnetworktime`, for which osquery has no
  direct equivalent. The query checks that `/private/etc/ntp.conf`
  exists with a non-empty body, which is what
  `systemsetup -setusingnetworktime on` writes. Worth revisiting if
  Apple changes how `systemsetup` persists the setting.
- **2.3.2.2** (Time Service enabled) — CIS states that if `timed`
  is disabled, the system should be treated as compromised and
  reinstalled. The `_fail.sh` script still disables the service for
  test purposes; the `_pass.sh` script restores it via `launchctl
  enable` + `bootstrap`.

### Section 2.5 notes

- All 5 Automated checks (§2.5.1.1–2.5.1.4 + §2.5.2.1) are
  profile-only `managed_policies` checks on
  `com.apple.applicationaccess`. No scripts ship.
- **2.5.1.1** and **2.5.1.4** require two keys each (respectively
  `allowExternalIntelligenceIntegrations` +
  `allowExternalIntelligenceIntegrationsSignIn`, and
  `allowNotesTranscription` + `allowNotesTranscriptionSummary`). The
  query verifies both keys are managed-false, following the new
  Query pattern #2 combined with AND semantics.
- **2.5.2.1** (Siri) replaces the deprecated
  `com.apple.ironwood.support` payload that earlier benchmark
  versions used. Current key is `allowAssistant=false` on
  `com.apple.applicationaccess`.

### Section 2.3.4 notes

- Both checks are *conditional on Time Machine being configured*.
  CIS states explicitly that if Time Machine is disabled, the audit
  passes by default. Our queries use Query pattern #4
  (absence-passes) — they return 1 row when the TimeMachine plist is
  absent or doesn't contain the offending value.
- **2.3.4.1** (Backup Automatically) — plist-based setting with a
  companion profile (`com.apple.MCX.TimeMachine` /
  `AutoBackup=true`). Terminal remediation was removed in macOS 15
  Sequoia (plist now protected), so no scripts ship. Profile-only
  for deliberate enforcement.
- **2.3.4.2** (Volumes Encrypted) — GUI-only remediation per CIS.
  No shippable script or profile key; the query detects
  non-encrypted destinations when Time Machine is configured, and
  default-passes on unconfigured hosts. Flagged in **Limitations**
  below too — enforcement must happen out of band.

### Section 2.6 notes

- **2.6.1.1** (Location Services) and **2.6.1.2** (menu bar icon)
  both use local-state queries (`location_services` table and
  `plist` table on `/Library/Preferences/com.apple.locationmenu.plist`)
  with pass/fail shell scripts. No MDM profile keys — the PDF only
  provides Terminal and Graphical remediation paths.
- **2.6.3.1–2.6.3.4** (Analytics & Improvements) — all four are
  profile-only on different `PayloadType`s
  (`com.apple.SubmitDiagInfo`, `com.apple.assistant.support`,
  `com.apple.Accessibility`, `com.apple.applicationaccess`). One
  profile per policy. CIS 2.6.3.5 is Manual (see Limitations).
- **2.6.3.2** (Improve Siri & Dictation) — the key name is literally
  `Siri Data Sharing Opt-In Status` with spaces, set to integer 2.
  The query uses `CAST(value AS INTEGER) = 2`.
- **2.6.4** (Limit Ad Tracking) — profile-only
  (`allowApplePersonalizedAdvertising=false` on
  `com.apple.applicationaccess`). CIS says "profile must be
  installed for this recommendation" to be compliant.
- **2.6.5** (Gatekeeper) — local-state query on the `gatekeeper`
  osquery table (matches the PDF's `spctl --status` audit). CIS
  notes the `spctl` binary method was removed in macOS 15 Sequoia,
  so only a profile remediation ships. Both `EnableAssessment=true`
  and `AllowIdentifiedDevelopers=true` are combined into the single
  `2.6.5.mobileconfig` per CIS's same-profile requirement.
  Gatekeeper is on by default, so the runner may record a
  pre-delivery pass note — not disqualifying.
- **2.6.6** (FileVault) — combines two checks: the
  `com.apple.MCX`/`dontAllowFDEDisable=true` managed policy and
  `disk_encryption.filevault_status='on'`. Enabling FileVault still
  requires on-device user interaction (no scriptable path), so the
  artifact is profile-only; the runner's pre-delivery query will
  fail on hosts without FileVault configured.
- **2.6.7** (Lockdown Mode) is Manual — see Limitations.
- **2.6.8** (admin password for system-wide preferences) — query
  uses the fleetd `authdb` table (flagged `(Fleetd Required)`) and
  checks all eight `system.preferences.*` rights for
  `shared=false`, `group=admin`, `authenticate-user=true`,
  `session-owner=false`. The pass script reimplements the CIS
  remediation script; the fail script only flips
  `system.preferences` `shared=true` (single-right regression is
  enough to break the query).

### Section 5 notes

- **5.1.1** (Home folders) — absence-passes query on `/Users/*`
  with mode in {700, 701, 710, 711}. Excludes `/Users/Shared/`.
  Pass script sets 700; fail script loosens the console user's
  home to 755.
- **5.1.2** (SIP) uses fleetd-independent `sip_config` table.
  5.1.3 AMFI uses fleetd `nvram_info`. 5.1.4 SSV uses fleetd
  `csrutil_info`. All three are one-liner state checks.
  Neither 5.1.2 nor 5.1.3 nor 5.1.4 ships test scripts —
  disabling SIP requires a reboot into Recovery, and the state
  is expected to be enabled by default.
- **5.1.5** uses the `apps` table JOINed with `file` on path, and
  bitwise-tests the "other" permission triad for the world-write
  bit. Fail script creates a stub world-writable `.app` bundle.
- **5.1.6** and **5.1.7** scan `/System/Volumes/Data/System` and
  `/Library` for world-writable directories. 5.1.6 uses the
  fleetd `find_cmd` table (faster than walking the `file` table);
  5.1.7 uses the core `file` table with sticky-bit filter and
  `extended_attributes.com.apple.rootless` exclusion. As of v1.1.0
  the 5.1.7 query also excludes the non-accessible `/Library/AppStore`
  folder per the updated CIS audit.
- **5.3.1** (internal APFS volumes encrypted) — added in v1.1.0
  (Automated; it was Manual in v1.0.0). Query uses the fleetd
  `apfs_volumes` table, requiring `filevault=1` on every volume
  that has no reserved role (VM/Update/Recovery/Preboot/xART/
  Hardware). The table does not distinguish internal from external
  disks, so it evaluates all non-role volumes; external volumes
  (CIS 5.3.2) and FAT32/ExFAT (5.3.3) are documented in
  Limitations. Encryption cannot be applied by a script, so no
  test scripts ship (MANUAL).
- **5.2.1–5.2.2, 5.2.7–5.2.8** all use the fleetd `pwd_policy`
  or `password_policy` table. Scripts use `pwpolicy
  -setglobalpolicy` despite the CIS note that the command is
  deprecated — it is still the only terminal-scriptable path.
- **5.4, 5.5, 5.11** each drop a file into `/etc/sudoers.d/` via
  `tee`. macOS ignores sudoers.d filenames containing `.`, so
  scripts use `CIS_5_4_sudoconfiguration` (no extension).
  Each query reads the fleetd `sudo_info` table which parses
  `sudo -V` output.
- **5.6** uses the fleetd `dscl` table to verify the root
  account has no `AuthenticationAuthority` (i.e. is disabled).
- **5.7** uses the fleetd `authdb` table with JSON extraction of
  the rule string; rule must contain `authenticate-session-owner`.
- **5.8** (Login banner) — requires the banner file to exist at
  `/Library/Security/PolicyBanner.{txt,rtf}` with mode 0644,
  root:wheel ownership. Pass script creates a .txt banner;
  fail script deletes it.
- **5.9** (Guest Home Folder) — absence-passes on
  `/Users/Guest`. Pairs with 2.13.1 (Guest Account disabled)
  and 2.13.2 (Guest SMB access disabled).
- **5.10** counts XProtect's two LaunchDaemon plists in the
  `launchd` table. Expects both to be registered.
- **5.11** uses `sudo_info` to confirm the "Log when a command
  is allowed by sudoers" field is true. Defaults to disabled in
  macOS 15 Sequoia and later.

### Section 6 notes

- **All 7 automated §6 recommendations are profile-only** —
  every query checks `managed_policies` on either
  `com.apple.Safari` or `com.apple.Terminal`. Single-key
  profiles each, except 6.3.4 which carries three keys
  (`BlockStoragePolicy=2`, `WebKitPreferences.storageBlockingPolicy=1`,
  `WebKitStorageBlockingPolicy=1`) in the same payload.
- **6.3.1 scope note:** Safari-managed keys typically deliver at
  user scope rather than system scope. The query omits a
  `username = ''` filter so any delivered scope satisfies it.

### Section 4 notes

- **4.1** (Bonjour advertising) — profile-only on
  `com.apple.mDNSResponder`/`NoMulticastAdvertisements=true`. The
  PDF also provides a local `defaults write` Terminal Method, but
  because mDNSResponder re-reads its config from managed sources
  on launch, the managed_policies path is the durable one.
- **4.2** (HTTP server) — absence-passes query on
  `processes.path = '/usr/sbin/httpd'`. Default is not running;
  fail script loads the LaunchDaemon and starts Apache.
- **4.3** (NFS server) — compound absence-passes: no
  `/sbin/nfsd` process AND `/etc/exports` does not exist. Pass
  script disables the LaunchDaemon and removes `/etc/exports`;
  fail script creates the file and starts nfsd.

### Section 3 notes

- **3.1** uses osquery's `launchd` joined with `processes` to verify
  `com.apple.auditd` is both loaded (plist registered) and running
  (live process whose cmdline matches the plist's `program_arguments`).
  Simply loading the LaunchDaemon is not enough — the daemon must
  have actually spawned. `launchctl load -w` flips both.
- **3.2** reads `/etc/security/audit_control` via the `file_lines`
  osquery table with substring LIKE checks. Two alternative
  flag-sets are accepted: explicit `aa,ad,-ex,-fm,-fr,-fw,lo` OR
  `-all` substituting for the failed-event flags. Scripts use
  the explicit form.
- **3.3** parses `/etc/asl/com.apple.install` with `regex_match`
  to extract `ttl=N` and compare to 365, AND verifies `all_max=`
  is absent. Both conditions must hold. The scripts use `awk` to
  target only the install.log file line (leaving other ASL rules
  untouched).
- **3.4** parses the `expire-after:` line in
  `/etc/security/audit_control` with `regex_match` and requires a
  day value ≥ 30. As of v1.1.0 CIS retains logs for 30 days with no
  maximum, so the query no longer requires a size clause (a size
  clause such as `OR 5G` is still accepted).
- **3.5** verifies root:wheel ownership and mode 440 (or 400)
  on three scopes: the `/etc/security/audit_control` file itself,
  the `dir:` target inside it, and the default `/var/audit`.
  Accepts either 440 or 400 since Apple's default and CIS's
  remediation have varied. Note: v1.1.0 changed only the CIS
  *remediation* to `chmod 700`, which contradicts its own audit
  (still `-r--r-----`/440); the Fleet query follows the audit and
  is unchanged.

### Section 2.1 notes

- **2.1.1.3** (iCloud Desktop & Documents sync) is the only
  automated check in §2.1 — profile-only on
  `com.apple.applicationaccess`/`allowCloudDesktopAndDocuments`.
  Every other §2.1 recommendation is Manual (see Limitations).

### Section 2.7 notes

- **2.7.1** (Screen Saver Hot Corners) — as of v1.1.0 CIS rescoped
  this to the *current user* only (Level 1). The query reads only
  the current console user's
  `/Users/<current user>/Library/Preferences/com.apple.dock.plist`
  (via `logged_in_users` with `tty = 'console'`), which requires
  FDA (flagged `(FDA Required)`). Any hot corner set to 6 (Disable
  Screen Saver) for that user fails. Because it depends on a
  console user, a non-root user must be logged in when the policy
  is evaluated (see the console-user caveat in
  `ee/cis/CIS-BENCHMARKS.md`); the `_pass`/`_fail` scripts toggle
  corners for the console user.

### Section 2.9 notes

- **2.9.1** (Help Apple Improve Search) is profile-only on
  `com.apple.assistant.support` with the spaced key name
  `Search Queries Data Sharing Status`. Integer value 2 means
  "off/disabled" per Apple's opt-in-status convention.

### Section 2.10 notes

- **2.10.1.2** (Apple Silicon sleep ≤15 min) — query uses the
  fleetd `pmset` table with JSON extraction, branching on
  `Battery Power` first, falling back to `AC Power`. Default-
  passes on non-Apple-Silicon hosts via `system_info.cpu_type`
  check. Also automatically satisfied when the 2.11.1 screen
  saver profile is enforced, per CIS's own note.
- **2.10.2** (Power Nap) and **2.10.3** (Wake for Network
  Access) — both use the fleetd `pmset` table; require pass on
  both AC Power and Battery Power. 2.10.2 is Intel-specific; on
  Apple Silicon, `pmset -a powernap` may be ignored but the
  query still returns the current setting regardless.
- **2.10.1.1** (OS Not Active When Resuming from Standby, Intel)
  is Manual — see Limitations.

### Section 2.12 notes

- **2.12.1** (no password hints on local accounts) — query uses
  the fleetd `user_login_settings` table which enumerates local
  users and reports `password_hint_enabled` per account. Pass
  script removes the `hint` attribute from all local users via
  `dscl`; fail script sets a test hint on the console user.

### Section 2.13 notes

- **2.13.1** (Guest Account) accepts either the local
  `com.apple.loginwindow.GuestEnabled=false` plist value OR the
  managed `com.apple.MCX` profile with both `DisableGuestAccount`
  and `EnableGuestAccount` set. Scripts exercise the local plist
  path.
- **2.13.2** (SMB guest access) — query reads
  `/Library/Preferences/SystemConfiguration/com.apple.smb.server.plist`
  for `AllowGuestAccess`, using absence-passes pattern (default
  is disabled). Scripts use `sysadminctl -smbGuestAccess on/off`.
- **2.13.3** (Automatic Login) accepts either the local
  `autoLoginUser` key being absent OR the managed
  `com.apple.login.mcx.DisableAutoLoginClient=true` profile.
  Scripts exercise the local plist path (`defaults delete` /
  `defaults write`).

### Section 2.18 notes

- **2.18.1** (On-Device Dictation) is profile-only on
  `com.apple.applicationaccess`/`forceOnDeviceOnlyDictation`.

### Section 2.11 notes

- **2.11.1** (screen saver idle ≤15 min) and **2.11.2** (require
  password on wake) are profile-only on `com.apple.screensaver`.
  2.11.1 uses the absence-passes / numeric threshold pattern
  (query pattern #4 combined with pattern #2): the setting must
  be managed with a value between 1 and 900 inclusive.
- **2.11.2** is a two-key profile: `askForPassword=true` AND
  `askForPasswordDelay ≤ 5`. Both keys live in the same
  `com.apple.screensaver` payload dict. The PDF notes the terminal
  command-line method "does not work as expected" on modern macOS,
  so a profile is required.
- **2.11.3**, **2.11.4**, **2.11.5** all read the local
  `/Library/Preferences/com.apple.loginwindow.plist` via the `plist`
  osquery table — world-readable, no FDA needed. Scripts
  (`defaults write`) are the primary test mechanism.
- **2.11.3** (custom login message) — query passes on any
  non-empty `LoginwindowText`. CIS leaves the actual text to the
  organization.

### Section 2.3.3 notes

- **2.3.3.3** (Printer Sharing) — the PDF's audit uses `cupsctl`, but
  osquery has no native CUPS-settings table. The query uses
  `listening_ports` to detect CUPS listening on a non-loopback
  interface (which happens when sharing is enabled). Heuristic but
  reliable for the common case.
- **2.3.3.7** (Internet Sharing) — the PDF accepts either a local
  `defaults` setting or a managed profile as compliant. The query
  checks the local `com.apple.nat` plist only. Both a test script
  (local `defaults write`) and a profile (`forceInternetSharingOff`
  via `com.apple.MCX`) are provided; scripts take priority in the
  runner.
- **2.3.3.8**, **2.3.3.9** (Content Caching, Media Sharing) —
  profile-only tests. `managed_policies` queries; CIS 2.3.3.9
  explicitly states the profile method is the only supported path.
- **2.3.3.10** (Bluetooth Sharing) — per-user ByHost setting. Scripts
  iterate `/Users/*` and run `defaults -currentHost write` per
  console user. Query uses the `preferences` table's negation
  pattern. Hosts without login users at test time may fail to
  exercise the setting.

## Org-decision policies

Where CIS leaves the choice to the organization, Fleet provides both
enable and disable profile variants.

(empty — populated per section)

## Optional policies

Recommendations that CIS includes but does not require at a given
level (e.g. password complexity components) ship here for teams that
want them.

(empty — populated per section)
