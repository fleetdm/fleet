# macOS 15 Sequoia benchmark

Fleet's policies have been written against v2.0.0 of the benchmark. You can refer to the [CIS website](https://www.cisecurity.org/cis-benchmarks) for full details about this version.

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
