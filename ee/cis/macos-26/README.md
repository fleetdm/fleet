# macOS 26 Tahoe

Fleet's policies have been written against v1.0.0. You can refer to the [CIS website](https://www.cisecurity.org/cis-benchmarks) for full details about this version.

For requirements and usage details, see the [CIS Benchmarks](https://fleetdm.com/docs/using-fleet/cis-benchmarks) documentation.

### Limitations

The following CIS benchmarks cannot be checked with a policy in Fleet:

- TODO

### Checks that require decision

CIS has left the parameters of the following checks up to the benchmark implementer. CIS recommends that an organization make a conscious decision for these benchmarks, but does not make a specific recommendation.

Fleet has provided both an "enabled" and "disabled" version of these benchmarks. When both policies are added, at least one will fail. Once your organization has made a decision, you can delete one or the other policy query.
The policy will be appended with a `-enabled` or `-disabled` label, such as `2.1.1.1-enabled`.

- TODO

However, Fleet has provided these as policies. If your organization declines to implement these, simply delete the corresponding policies.