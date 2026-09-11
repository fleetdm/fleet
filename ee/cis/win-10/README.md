# Windows 10 Enterprise benchmarks

Fleet's policies have been written against v5.0.0 of the benchmark. You can refer to the [CIS website](https://www.cisecurity.org/cis-benchmarks) for full details about this version.

For requirements and usage details, see the [CIS Benchmarks](https://fleetdm.com/docs/using-fleet/cis-benchmarks) documentation.

### Limitations

> All Automated items in this version of the benchmark are covered. Fleet does not ship BitLocker (BL) or Next Generation (NG) profile recommendations.

### Important: Group Policy removal does not clear registry values

When a Group Policy entry is removed from `Registry.pol` and `gpupdate /force` is run, Windows does **not** clean up the registry value it previously wrote. This means the osquery-based policy check will continue to report the device as compliant even after the Group Policy is set back to "Not Configured."

This is expected Windows behavior and is consistent with the CIS benchmark audit procedure, which checks the registry value regardless of how it was set. To truly revert a setting, the registry value must be explicitly deleted or changed -- simply removing the Group Policy is not sufficient.

### v5.0.0 update notes

#### Removed (27 policies)

- **All 21 recommendations under 18.10.42 Microsoft Defender Antivirus** -- CIS removed these from v5.0.0 (Ticket #27270). This includes MAPS, Real-Time Protection, Scan, Reporting, MpEngine, Network Inspection System, Features, and Remediation sub-sections. Microsoft Defender Application Guard (18.10.43) and Exploit Guard (18.10.44) policies are retained.
- **2.3.8 (L1)** Microsoft network client: Digitally sign communications (if server agrees) -- removed by CIS (Ticket #25346).
- **2.3.9 (L1)** Microsoft network server: Digitally sign communications (if client agrees) -- removed by CIS (Ticket #25347).
- **2.2 (L1)** Change the time zone -- removed by CIS (Ticket #26345).
- **18.10.91 (L1)** Allow networking in Windows Sandbox -- removed by CIS (Ticket #26086).
- **18.10.94.4 (L1)** Select when Preview Builds and Feature Updates are received -- removed by CIS (Ticket #26311).
- **18.10.16 (L1)** Disable OneSettings Downloads -- removed by CIS (Ticket #26336).

#### Added (2 policies)

- **18.9.17.1 (L1)** Enable / disable CLFS logfile authentication (Ticket #26810).
- **18.11.1 (L1)** Disable HTTP proxy features: Disable WPAD (Ticket #26860). Requires the CIS.admx/adml Group Policy template from the CIS Build Kits (January 2026 or newer).

#### Level changes

- **2.3.7.1** Interactive logon: Do not require CTRL+ALT+DEL -- moved L1 to L2 (Ticket #26130).
- **18.10.50** Prevent the usage of OneDrive for file storage -- moved L1 to L2 (Ticket #26332).
- **18.10.16** Limit Diagnostic Log Collection -- moved L1 to L2 (Ticket #26882).
- **18.10.16** Limit Dump Collection -- moved L1 to L2 (Ticket #26883).
- **18.10.16** Enable OneSettings Auditing -- moved L1 to L2 (Ticket #27156).
- **18.10.57.3.10** Set time limit for active but idle Remote Desktop Services sessions -- moved L2 to L1 (Ticket #27272).

#### Title and query updates

- Bulk title normalization (Ticket #26462): "Domain member:" prefix added, "Microsoft network client/server:" colons added, "Network security:" colon added, "Interactive logon:" colon added, "empty list" changed to "'No One'", "includes" changed to "to include" with expanded principal lists.
- Firewall logging name policies changed from checking a specific log file path to checking that a path is configured (Tickets #26325, #26326, #26327).
- Audit Sensitive Privilege Use changed from "Success and Failure" to "Success" (Ticket #25745).
- 18.9.7 renamed from "Prevent device metadata retrieval from the Internet" to "Prevent automatic download of applications associated with device metadata" (Ticket #26306).
- "Audit PNP Activity" corrected to "Audit User Account Management" (mislabeled in v4.0.0).
- Resolution text fixes: corrected ~12 policies where resolution text said "set to an empty list" but the expected value is a specific principal list (e.g., "Administrators").

#### v5.0.0 items not implemented

These items from the v5.0.0 Change History are **not** represented in `cis-policy-queries.yml`:

- Items in the BitLocker (BL) and Next Generation (NG) profiles -- Fleet does not ship these for this benchmark.
- User section items (19.x) that were removed -- Fleet does not track removals for items it never shipped.
- **18.10.57.3.10** Set time limit for disconnected sessions (removed by CIS, Ticket #27275) -- was not present in the v4.0.0 YAML.
- Section renumbering from Windows 11 Release 25H2 Administrative Templates (Ticket #26305) -- section numbers are not stored in the YAML; only policy names and registry paths matter.

### v4.0.0 update notes

These items from the v4.0.0 Change History are **not** represented in `cis-policy-queries.yml`, with the reason for each:

- **18.6.8 (L1) Ensure 'Require Encryption' is set to 'Enabled'** -- listed in the v4.0.0 Change History (Appendix), but the recommendation has no corresponding section in the body of the v4.0.0 document (the `18.6.8 Lanman Workstation` section only contains `18.6.8.1 Enable insecure guest logons`). With no Description/Audit/Remediation in the benchmark, there is no registry location to query, so no policy could be authored.
- **18.9.26.2 (NG) Ensure 'Configures LSASS to run as a protected process' is set to 'Enabled: Enabled with UEFI Lock'** -- the Change History labels this `(L1)`, but the body heading tags it **Next Generation (NG)**, which Fleet does not ship for this benchmark.

### Checks that require a Group Policy template

Several items require Group Policy templates in place in order to audit them.
These items are tagged with the label `CIS_group_policy_template_required` in the YAML file, and details about the required Group Policy templates can be found in each item's `resolution`.
