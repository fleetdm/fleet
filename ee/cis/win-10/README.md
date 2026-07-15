# Windows 10 Enterprise benchmarks

Fleet's policies have been written against v4.0.0 of the benchmark. You can refer to the [CIS website](https://www.cisecurity.org/cis-benchmarks) for full details about this version.

For requirements and usage details, see the [CIS Benchmarks](https://fleetdm.com/docs/using-fleet/cis-benchmarks) documentation.

### Limitations

> With the two exceptions noted below, all items in this version of the benchmark are able to be automated.

### v4.0.0 update notes

These items from the v4.0.0 Change History are **not** represented in `cis-policy-queries.yml`, with the reason for each:

- **18.6.8 (L1) Ensure 'Require Encryption' is set to 'Enabled'** — listed in the v4.0.0 Change History (Appendix), but the recommendation has no corresponding section in the body of the v4.0.0 document (the `18.6.8 Lanman Workstation` section only contains `18.6.8.1 Enable insecure guest logons`). With no Description/Audit/Remediation in the benchmark, there is no registry location to query, so no policy could be authored. Revisit if a later errata/print of the PDF adds the section.
- **18.9.26.2 (NG) Ensure 'Configures LSASS to run as a protected process' is set to 'Enabled: Enabled with UEFI Lock'** — the Change History labels this `(L1)`, but the body heading tags it **Next Generation (NG)**, which Fleet does not ship for this benchmark. Note also that starting with the Windows 11 Release 24H2 Administrative Templates the backing registry value moved from `HKLM\SYSTEM\CurrentControlSet\Control\Lsa:RunAsPPL` to `HKLM\SOFTWARE\Policies\Microsoft\Windows\System:RunAsPPL`.

Other v4.0.0 changes were applied to the YAML: 18 new Automated recommendations were added, 2 recommendations were removed (`18.10.66` Only display the private store within the Microsoft Store, and `18.10.42` Turn off Microsoft Defender AntiVirus), `18.10.17` Enable App Installer moved from Level 1 to Level 2, `Enable Certificate Padding` now accepts a `REG_DWORD` or `REG_SZ` value, and the `Log on as a service`, `Create symbolic links`, and MPR-notifications (`18.10.82.1`) titles were updated to their v4.0.0 wording.


### Checks that require a Group Policy template

Several items require Group Policy templates in place in order to audit them.
These items are tagged with the label `CIS_group_policy_template_required` in the YAML file, and details about the required Group Policy templates can be found in each item's `resolution`.
