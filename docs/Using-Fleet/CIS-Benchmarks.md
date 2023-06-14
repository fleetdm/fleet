# CIS benchmarks

> Available in Fleet Premium

## Overview
CIS Benchmarks represent the consensus-based effort of cybersecurity experts globally to help you protect your systems against threats more confidently.
For more information about CIS Benchmarks check out [Center for Internet Security](https://www.cisecurity.org/cis-benchmarks)'s website.

Fleet has implemented native support for CIS benchmarks for the following platforms:
- macOS 13.0 Ventura (96 checks)
- Windows 10 Enterprise (496 checks - in progress)

[Where possible](#limitations), each CIS benchmark is implemented with a [policy query](./REST-API.md#policies) in Fleet. 

## Requirements

Following are the requirements to use the CIS benchmarks in Fleet:

- To use these policies, Fleet must have an up-to-date paid license (≥Fleet Premium).
- Devices must be running [`fleetd`](https://fleetdm.com/docs/using-fleet/orbit), the lightweight agent that bundles the latest osqueryd.
- Some CIS benchmarks explicitly involve verifying MDM-based controls, so devices must be enrolled to an MDM solution.  (Any MDM solution works, it doesn't have to be Fleet.)
- On macOS, the orbit executable in Fleetd must have "Full Disk Access", see [Grant Full Disk Access to Osquery on macOS](./Adding-hosts.md#grant-full-disk-access-to-osquery-on-macos).

### MDM required
Some of the policies created by Fleet use the [managed_policies](https://www.fleetdm.com/tables/managed_policies) table. This checks whether an MDM solution has turned on the setting to enforce the policy.
Using MDM is the recommended way to manage and enforce CIS benchmarks. To learn how to set up MDM in Fleet, visit [here](/docs/using-fleet/mdm-setup).

### Fleetd required
Fleet's CIS benchmarks require our [osquery manager, Fleetd](https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer). This is because Fleetd includes tables which are not part of vanilla osquery in order to accomplish auditing the benchmarks.

## How to add CIS benchmarks

All CIS policies are stored under our restricted licensed folder `ee/cis/`.

How to import them to Fleet:
```sh
# Download policy queries from Fleet's repository (e.g. for macOS 13)
wget https://raw.githubusercontent.com/fleetdm/fleet/main/ee/cis/macos-13/cis-policy-queries.yml

# Apply the downloaded policies to Fleet.
fleetctl apply -f cis-policy-queries.yml
```

To apply the policies on a specific team use the `--policies-team` flag:
```sh
fleetctl apply --policies-team "Workstations" -f cis-policy-queries.yml
```

## Limitations
Fleet's current set of benchmarks only implements benchmark *auditing* steps that can be *automated*.

In practice, Fleet is able to cover a large majority of benchmarks:
* macOS 13 Ventura - 96 of 104
* Windows 10 Enterprise - TODO

For a list of specific checks which are not covered by Fleet, please visit the section devoted to each benchmark.

### Audit vs. remediation
Each benchmark has two elements:
1. Audit - how to find out whether the host is in compliance with the benchmark
2. Remediation - if the host is out of compliance with the benchmark, how to fix it

Since Fleetd is currently read-only without the ability to execute actions on the host, Fleet does not implement the remediation portions of CIS benchmarks.

To implement automated remediation, you can install a separate agent such as Munki, Chef, Puppet, etc. which has write functionality.

### Manual vs. automated

For both the audit and remediation elements of a CIS Benchmark, there are two types:
1. Automated - the element can be audited or remediated without human intervention
2. Manual - the element requires human intervention to be audited or remediated

Fleet only implements automated audit checks. Manual checks require administrators to implement other processes to conduct the check.

## Levels 1 and 2
CIS designates various benchmarks as Level 1 or Level 2 to describe the level of thoroughness and burden that each benchmark represents.

### Level 1

Items in this profile intend to:
- be practical and prudent;
- provide a clear security benefit; and
- not inhibit the utility of the technology beyond acceptable means.

### Level 2

This profile extends the "Level 1" profile. Items in this profile exhibit one or more of the following characteristics:
- are intended for environments or use cases where security is paramount or acts as defense in depth measure
- may negatively inhibit the utility or performance of the technology.

## macOS 13.0 Ventura benchmark

Fleet's policies have been written against v1.0 of the benchmark. Please refer to the "CIS Apple macOS 13.0 Ventura Benchmark v1.0.0 - 11-14-2022" PDF from the CIS website for full details.

### Checks that require customer decision

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

However, Fleet has provided these as policies. If your organization declines to implement these, simply delete the corresponding policy.

### macOS 13.0 Ventura manual checks

The following CIS benchmark checks cannot be automated and must be addressed manually:
- 2.1.2 Audit App Store Password Settings
- 2.3.3.12 Ensure Computer Name Does Not Contain PII or Protected Organizational Information
- 2.6.6 Audit Lockdown Mode
- 2.11.2 Audit Touch ID and Wallet & Apple Pay Settings
- 2.13.1 Audit Passwords System Preference Setting
- 2.14.1 Audit Notification & Focus Settings
- 3.7 Audit Software Inventory
- 6.2.1 Ensure Protect Mail Activity in Mail Is Enabled

<meta name="pageOrderInSection" value="1700">
<meta name="title" value="CIS Benchmarks">
