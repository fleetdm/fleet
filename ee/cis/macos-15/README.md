# macOS 15 Sequoia benchmark

Fleet's policies have been written against v2.1.0 of the benchmark. You can refer to the [CIS website](https://www.cisecurity.org/cis-benchmarks) for full details about this version.

For requirements and usage details, see the [CIS Benchmarks](https://fleetdm.com/docs/using-fleet/cis-benchmarks) documentation.

### Limitations

The following CIS benchmarks cannot be checked with a policy in Fleet:
1. 2.1.2 Audit App Store Password Settings
2. 2.3.3.11 Ensure Computer Name Does Not Contain PII or Protected Organizational Information
3. 2.4.1 Audit Menu Bar and Control Center Icons
4. 2.6.7 Audit Lockdown Mode
5. 2.12.2 Audit Touch ID
6. 2.16.1 Audit Wallet & Apple Pay Settings
7. 2.15.1 Audit Notification Settings
8. 3.6 Audit Software Inventory
9. 6.1.1 Audit Show All Filename Extensions
10. 6.2.1 Ensure Protect Mail Activity in Mail Is Enabled
11. 2.6.3.5 Ensure Share iCloud Analytics Is Disabled
12. 5.3.2 Ensure all APFS and HFS+ external user storage volumes are encrypted — the fleetd `apfs_volumes` table does not expose an internal/external indicator, so "external" volumes cannot be reliably identified as a policy query. Internal APFS volumes are covered by 5.3.1.
13. 5.3.3 Audit Connected FAT32 and ExFAT Drives (Manual) — CIS ships this as a Manual audit; it is an organizational review of connected removable drives rather than a mechanically checkable condition.

### Checks that require decision

CIS has left the parameters of the following checks up to the benchmark implementer. CIS recommends that an organization make a conscious decision for these benchmarks, but does not make a specific recommendation.

Fleet has provided both an "enabled" and "disabled" version of these benchmarks. When both policies are added, at least one will fail. Once your organization has made a decision, you can delete one or the other policy.
The policy will be appended with a `-enabled` or `-disabled` label, such as `2.1.1.1-enabled`.

- 2.1.1.1 Audit iCloud Passwords & Keychain
- 2.1.1.2 Audit iCloud Drive
- 2.5.1 Audit Siri
- 2.8.1 Audit Universal Control

Furthermore, CIS has decided to not require the following password complexity settings:
- 5.2.3 Ensure Complex Password Must Contain Alphabetic Characters Is Configured
- 5.2.4 Ensure Complex Password Must Contain Numeric Character Is Configured
- 5.2.5 Ensure Complex Password Must Contain Special Character Is Configured
- 5.2.6 Ensure Complex Password Must Contain Uppercase and Lowercase Characters Is Configured

However, Fleet has provided these as policies. If your organization declines to implement these, simply delete the corresponding policies.

### v2.1.0 update notes

These policies were updated from v2.0.0 to v2.1.0. The relevant changes:

- **2.3.5 Device Management** — added as an informational sub-section only (no numbered recommendation), so there is no corresponding policy.
- **2.7.1 Ensure Screen Saver Hot Corners Are Secure** — CIS rescoped this to the *current user* only (previously all users) and moved it to Level 1. The query now checks only the current console user's `com.apple.dock` hot corners. Because the check reads the console user's Dock preferences, a non-root console user must be logged in when the policy is evaluated (see the console-user caveat in `ee/cis/CIS-BENCHMARKS.md`).
- **3.4 Ensure Security Auditing Logs Are Retained for 30 Days** — retitled; the requirement was relaxed to `expire-after:` ≥ 30 days (a size clause such as `OR 5G` is now optional). The query now checks for a day value ≥ 30 rather than the old `60d OR 5G`.
- **3.5 Ensure Access to Audit Records Is Controlled** — CIS updated only the *remediation* to `chmod 700`, but its *audit* still checks for `-r--r-----` (mode 440), so the two contradict each other in the CIS document. Fleet's query follows the audit (audit_control 0400, `/var/audit` contents 0440), so the query is unchanged.
- **5.1.6 No World Writable Folders in the System Folder** — CIS added `2>/dev/null` to suppress errors; the fleetd `find_cmd` table already handles this, so the query is unchanged. Note: the CIS audit excludes `downloadDir|locks`, whereas Fleet's query excludes only `Drop Box`; this pre-existing exclusion difference was left as-is (it predates the v2.1.0 delta).
- **5.1.7 No World Writable Folders in the Library Folder** — CIS updated the audit to ignore the non-accessible `/Library/AppStore` folder; the query now excludes `/Library/AppStore`.
- **5.3.1 / 5.3.2 storage encryption** — CIS removed the old CoreStorage recommendation and split disk encryption into internal (5.3.1) and external (5.3.2). Fleet's 5.3.1 covers internal APFS volumes (see caveat under Limitations); the old CoreStorage policy was removed. 5.3.2 (external) and 5.3.3 (FAT32/ExFAT) are documented under Limitations.
- **5.6 Ensure the "root" Account Is Disabled** — CIS updated the audit to detect a lingering *secure token* even when root is not enabled, and the remediation now removes it (`fdesetup remove -user root`). Fleet's query already checks that root's `AuthenticationAuthority` key is absent, which covers the secure-token case; the resolution was updated accordingly.

`cis_id` values were added to the policies modified in this update (2.7.1, 3.4, 5.1.7, 5.3.1, 5.6); most other policies in this file predate the `cis_id` convention.
