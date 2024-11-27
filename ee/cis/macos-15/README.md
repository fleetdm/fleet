# macOS 15 Sequoia benchmark

Fleet's policies have been written against v1.0.0 of the benchmark. You can refer to the [CIS website](https://www.cisecurity.org/cis-benchmarks) for full details about this version.

For requirements and usage details, see the [CIS Benchmarks](https://fleetdm.com/docs/using-fleet/cis-benchmarks) documentation.

### Limitations

The following CIS benchmarks cannot be checked with a policy in Fleet:
1. 2.1.2 Audit App Store Password Settings
2. 2.3.3.12 Ensure Computer Name Does Not Contain PII or Protected Organizational Information
3. 2.6.6 Audit Lockdown Mode
4. 2.11.2 Audit Touch ID and Wallet & Apple Pay Settings
5. 2.13.1 Audit Passwords System Preference Setting
6. 2.14.1 Audit Notification & Focus Settings
7. 3.7 Audit Software Inventory
8. 6.2.1 Ensure Protect Mail Activity in Mail Is Enabled

### Checks that require decision

CIS has left the parameters of the following checks up to the benchmark implementer. CIS recommends that an organization make a conscious decision for these benchmarks, but does not make a specific recommendation.

Fleet has provided both an "enabled" and "disabled" version of these benchmarks. When both policies are added, at least one will fail. Once your organization has made a decision, you can delete one or the other policy query.
The policy will be appended with a `-enabled` or `-disabled` label, such as `2.1.1.1-enabled`.

- 2.1.1.1 Audit iCloud Keychain
- 2.1.1.2 Audit iCloud Drive
- 2.5.1 Audit Siri
- 2.8.1 Audit Universal Control

Furthermore, CIS has decided to not require the following password complexity settings:
- 5.2.3 Ensure Complex Password Must Contain Alphabetic Characters Is Configured
- 5.2.4 Ensure Complex Password Must Contain Numeric Character Is Configured
- 5.2.5 Ensure Complex Password Must Contain Special Character Is Configured
- 5.2.6 Ensure Complex Password Must Contain Uppercase and Lowercase Characters Is Configured

However, Fleet has provided these as policies. If your organization declines to implement these, simply delete the corresponding policies.