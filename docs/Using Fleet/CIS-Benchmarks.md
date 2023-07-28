# CIS Benchmarks

> Available in Fleet Premium

## Overview
CIS Benchmarks represent the consensus-based effort of cybersecurity experts globally to help you protect your systems against threats more confidently.
For more information about CIS Benchmarks check out [Center for Internet Security](https://www.cisecurity.org/cis-benchmarks)'s website.

Fleet has implemented native support for CIS Benchmarks for the following platforms:
- macOS 13.0 Ventura (96 checks)
- Windows 10 Enterprise (496 checks)

[Where possible](#limitations), each CIS Benchmark is implemented with a [policy query](./REST-API.md#policies) in Fleet. 

## Requirements

Following are the requirements to use the CIS Benchmarks in Fleet:

- To use these policies, Fleet must have an up-to-date paid license (≥Fleet Premium).
- Devices must be running [`fleetd`](https://fleetdm.com/docs/using-fleet/orbit), the lightweight agent that bundles the latest osqueryd.
- Some CIS Benchmarks explicitly involve verifying MDM-based controls, so devices must be enrolled to an MDM solution.  (Any MDM solution works, it doesn't have to be Fleet.)
- On macOS, the orbit executable in Fleetd must have "Full Disk Access", see [Grant Full Disk Access to Osquery on macOS](./Adding-hosts.md#grant-full-disk-access-to-osquery-on-macos).

### MDM required
Some of the policies created by Fleet use the [managed_policies](https://www.fleetdm.com/tables/managed_policies) table. This checks whether an MDM solution has turned on the setting to enforce the policy.
Using MDM is the recommended way to manage and enforce CIS Benchmarks. To learn how to set up MDM in Fleet, visit [here](/docs/using-fleet/mdm-setup).

### Fleetd required
Fleet's CIS Benchmarks require our [osquery manager, Fleetd](https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer). This is because Fleetd includes tables which are not part of vanilla osquery in order to accomplish auditing the benchmarks.

## How to add CIS Benchmarks

All CIS policies are stored under our restricted licensed folder `ee/cis/`.

How to import them to Fleet:
```sh
# Download policy queries from Fleet's repository 
# macOS 13
wget https://raw.githubusercontent.com/fleetdm/fleet/main/ee/cis/macos-13/cis-policy-queries.yml

# Windows 10 (note the same file name. Rename as needed.)
wget https://raw.githubusercontent.com/fleetdm/fleet/main/ee/cis/win-10/cis-policy-queries.yml

# Apply the downloaded policies to Fleet for both files.
fleetctl apply --context <context> -f <path-to-macOS-13-policies> --policies-team <team-name>
fleetctl apply --context <context> -f <path-to-windows-10-policies> --policies-team <team-name>
```

To apply the policies on a specific team use the `--policies-team` flag:
```sh
fleetctl apply --policies-team "Workstations" -f cis-policy-queries.yml
```

## Limitations
Fleet's current set of benchmarks only implements benchmark *auditing* steps that can be *automated*.

In practice, Fleet is able to cover a large majority of benchmarks:
* macOS 13 Ventura - 96 of 104
* Windows 10 Enterprise - All CIS items (496) 

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

* macOS 13 Ventura - 96 of 104 are automated
* Windows 10 Enterprise - All CIS items (496) are automated 


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

## Windows 10 Enterprise benchmark

Fleet's policies have been written against v1.12.0 of the benchmark. You can refer to the [CIS website](https://www.cisecurity.org/cis-benchmarks) for full details about this version.

### Checks that require a Group Policy Template

38 items require Group Policy Template in place in order to audit them.
These items are tagged with the label `CIS_group_policy_template_required` in the YAML file, and details about the required Group Policy templates can be found in each item's `resolution`.

```
18.3.1 CIS - Ensure 'Apply UAC restrictions to local accounts on network logons' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MS Security Guide\Apply UAC restrictions to local accounts on network logons'

18.3.2 CIS - Ensure 'Configure SMB v1 client driver' is set to 'Enabled: Disable driver (recommended)'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MS Security Guide\Configure SMB v1 client driver'

18.3.3 CIS - Ensure 'Configure SMB v1 server' is set to 'Disabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MS Security Guide\Configure SMB v1 server'

18.3.4 CIS - Ensure 'Enable Structured Exception Handling Overwrite Protection (SEHOP)' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MS Security Guide\Enable Structured Exception Handling Overwrite Protection (SEHOP)'

18.3.5 CIS - Ensure 'Limits print driver installation to Administrators' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MS Security Guide\Limits print driver installation to Administrators'

18.3.6 CIS - Ensure 'NetBT NodeType configuration' is set to 'Enabled: P-node (recommended)'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MS Security Guide\NetBT NodeType configuration'

18.3.7 CIS - Ensure 'WDigest Authentication' is set to 'Disabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MS Security Guide\WDigest Authentication (disabling may require KB2871997)'

18.4.1 CIS - Ensure 'MSS: (AutoAdminLogon) Enable Automatic Logon (not recommended)' is set to 'Disabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MSS (Legacy)\MSS: (AutoAdminLogon) Enable Automatic Logon (not recommended)'

18.4.2 CIS - Ensure 'MSS: (DisableIPSourceRouting IPv6) IP source routing protection level (protects against packet spoofing)' is set to 'Enabled: Highest protection, source routing is completely disabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MSS (Legacy)\MSS: (DisableIPSourceRouting IPv6) IP source routing protection level (protects against packet spoofing)'

18.4.3 CIS - Ensure 'MSS: (DisableIPSourceRouting) IP source routing protection level (protects against packet spoofing)' is set to 'Enabled: Highest protection, source routing is completely disabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MSS (Legacy)\MSS: (DisableIPSourceRouting) IP source routing protection level (protects against packet spoofing)'

18.4.4 CIS - Ensure 'MSS: (DisableSavePassword) Prevent the dial-up password from being saved' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MSS (Legacy)\MSS:(DisableSavePassword) Prevent the dial-up password from being saved'

18.4.5 CIS - Ensure 'MSS: (EnableICMPRedirect) Allow ICMP redirects to override OSPF generated routes' is set to 'Disabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MSS (Legacy)\MSS: (EnableICMPRedirect) Allow ICMP redirects to override OSPF generated routes'

18.4.6 CIS - Ensure 'MSS: (KeepAliveTime) How often keep-alive packets are sent in milliseconds' is set to 'Enabled: 300,000 or 5 minutes'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MSS (Legacy)\MSS: (KeepAliveTime) How often keep-alive packets are sent in milliseconds'

18.4.7 CIS - Ensure 'MSS: (NoNameReleaseOnDemand) Allow the computer to ignore NetBIOS name release requests except from WINS servers' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MSS (Legacy)\MSS: (NoNameReleaseOnDemand) Allow the computer to ignore NetBIOS name release requests except from WINS servers'

18.4.8 CIS - Ensure 'MSS: (PerformRouterDiscovery) Allow IRDP to detect and configure Default Gateway addresses (could lead to DoS)' is set to 'Disabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MSS (Legacy)\MSS: (PerformRouterDiscovery) Allow IRDP to detect and configure Default Gateway addresses (could lead to DoS)'

18.4.9 CIS - Ensure 'MSS: (SafeDllSearchMode) Enable Safe DLL search mode (recommended)' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MSS (Legacy)\MSS: (SafeDllSearchMode) Enable Safe DLL search mode (recommended)'

18.4.10 CIS - Ensure 'MSS: (ScreenSaverGracePeriod) The time in seconds before the screen saver grace period expires (0 recommended)' is set to 'Enabled: 5 or fewer seconds'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MSS (Legacy)\MSS: (ScreenSaverGracePeriod) The time in seconds before the screen saver grace period expires (0 recommended)'

18.4.11 CIS - Ensure 'MSS: (TcpMaxDataRetransmissions IPv6) How many times unacknowledged data is retransmitted' is set to 'Enabled: 3'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MSS (Legacy)\MSS:(TcpMaxDataRetransmissions IPv6) How many times unacknowledged data is retransmitted'

18.4.12 CIS - Ensure 'MSS: (TcpMaxDataRetransmissions) How many times unacknowledged data is retransmitted' is set to 'Enabled: 3'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MSS (Legacy)\MSS:(TcpMaxDataRetransmissions) How many times unacknowledged data is retransmitted'

18.4.13 CIS - Ensure 'MSS: (WarningLevel) Percentage threshold for the security event log at which the system will generate a warning' is set to 'Enabled: 90% or less'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\MSS (Legacy)\MSS: (WarningLevel) Percentage threshold for the security event log at which the system will generate a warning'

18.8.21.2 CIS - Ensure 'Configure registry policy processing: Do not apply during periodic background processing' is set to 'Enabled: FALSE'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Group Policy\Configure registry policy processing'

18.8.22.1.1 CIS - Ensure 'Turn off access to the Store' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Internet Communication Management\Internet Communication settings\Turn off access to the Store'

18.8.22.1.2 CIS - Ensure 'Turn off downloading of print drivers over HTTP' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Internet Communication Management\Internet Communication settings\Turn off downloading of print drivers over HTTP'

18.8.22.1.3 CIS - Ensure 'Turn off handwriting personalization data sharing' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Internet Communication Management\Internet Communication settings\Turn off handwriting personalization data sharing'

18.8.22.1.4 CIS - Ensure 'Turn off handwriting recognition error reporting' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Internet Communication Management\Internet Communication settings\Turn off handwriting recognition error reporting'

18.8.22.1.5 CIS - Ensure 'Turn off Internet Connection Wizard if URL connection is referring to Microsoft.com' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Internet Communication Management\Internet Communication settings\Turn off Internet Connection Wizard if URL connection is referring to Microsoft.com'

18.8.22.1.6 CIS - Ensure 'Turn off Internet download for Web publishing and online ordering wizards' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Internet Communication Management\Internet Communication settings\Turn off Internet download for Web publishing and online ordering wizards'

18.8.22.1.7 CIS - Ensure 'Turn off printing over HTTP' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Internet Communication Management\Internet Communication settings\Turn off printing over HTTP'

18.8.22.1.8 CIS - Ensure 'Turn off Registration if URL connection is referring to Microsoft.com' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Internet Communication Management\Internet Communication settings\Turn off Registration if URL connection is referring to Microsoft.com'

18.8.22.1.9 CIS - Ensure 'Turn off Search Companion content file updates' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Internet Communication Management\Internet Communication settings\Turn off Search Companion content file updates'

18.8.22.1.10 CIS - Ensure 'Turn off the "Order Prints" picture task' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Internet Communication Management\Internet Communication settings\Turn off the "Order Prints" picture task'

18.8.22.1.11 CIS - Ensure 'Turn off the "Publish to Web" task for files and folders' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Internet Communication Management\Internet Communication settings\Turn off the "Publish to Web" task for files and folders'

18.8.22.1.12 CIS - Ensure 'Turn off the Windows Messenger Customer Experience Improvement Program' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Internet Communication Management\Internet Communication settings\Turn off the Windows Messenger Customer Experience Improvement Program'

18.8.22.1.13 CIS - Ensure 'Turn off Windows Customer Experience Improvement Program' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Internet Communication Management\Internet Communication settings\Turn off Windows Customer Experience Improvement Program'

18.8.22.1.14 CIS - Ensure 'Turn off Windows Error Reporting' is set to 'Enabled'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Internet Communication Management\Internet Communication settings\Turn off Windows Error Reporting'

18.8.25.1 CIS - Ensure 'Support device authentication using certificate' is set to 'Enabled: Automatic'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Kerberos\Support device authentication using certificate'

18.8.26.1 CIS - Ensure 'Enumeration policy for external devices incompatible with Kernel DMA Protection' is set to 'Enabled: Block All'
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Kernel DMA Protection\Enumeration policy for external devices incompatible with Kernel DMA Protection'

18.8.27.1 CIS - Ensure 'Disallow copying of user input methods to the system account for sign-in' is set to 'Enabled' (Automated)
Requires this GPO in place: 'Computer Configuration\Policies\Administrative Templates\System\Locale Services\Disallow copying of user input methods to the system account for sign-in'
```

<meta name="pageOrderInSection" value="1700">
<meta name="title" value="CIS Benchmarks">
<meta name="description" value="Read about how Fleet's implementation of CIS Benchmarks offers consensus-based cybersecurity guidance, covering macOS 13.0 Ventura & Windows 10 Enterprise.">
<meta name="navSection" value="Security compliance">
