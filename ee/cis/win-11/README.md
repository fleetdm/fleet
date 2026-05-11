# Windows 11 Enterprise benchmarks

Fleet's policies have been written against v5.0.1 of the benchmark. You can refer to the [CIS website](https://www.cisecurity.org/cis-benchmarks) for full details about this version.

For requirements and usage details, see the [CIS Benchmarks](https://fleetdm.com/docs/using-fleet/cis-benchmarks) documentation.

### Limitations

> None. All items in this version of the benchmark are able to be automated.

### Important: Group Policy removal does not clear registry values

When a Group Policy entry is removed from `Registry.pol` and `gpupdate /force` is run, Windows does **not** clean up the registry value it previously wrote. This means the osquery-based policy check will continue to report the device as compliant even after the Group Policy is set back to "Not Configured."

This is expected Windows behavior and is consistent with the CIS benchmark audit procedure, which checks the registry value regardless of how it was set. To truly revert a setting, the registry value must be explicitly deleted or changed — simply removing the Group Policy is not sufficient.

### Checks that require a Group Policy template

Several items require Group Policy templates in place in order to audit them.
These items are tagged with the label `CIS_group_policy_template_required` in the YAML file, and details about the required Group Policy templates can be found in each item's `resolution`.
