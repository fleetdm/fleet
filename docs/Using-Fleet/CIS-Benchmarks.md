# CIS Benchmarks

## Overview
CIS Benchmarks represent the consensus-based effort of cybersecurity experts globally to help you protect your systems against threats more confidently.
For more information about CIS Benchmarks check out [Center for Internet Security](https://www.cisecurity.org/cis-benchmarks)'s website.

Fleet has implemented native support for CIS benchmarks for the following platforms:
- CIS Apple macOS 13.0 Ventura Benchmark v1.0.0 - 11-14-2022 (96 checks)
- CIS Microsoft Windows 10 Enterprise Benchmark v1.12.0 - 02-15-2022 (496 checks - in progress)

[Where possible](#limitations), each CIS benchmark is implemented with a [policy query](./REST-API.md#policies) in Fleet. 

## Requirements

Following are the requirements to use the CIS Benchmarks in Fleet:

- Fleet must be Premium or Ultimate licensed.
- Devices must be running [Fleetd](https://fleetdm.com/docs/using-fleet/orbit), the osquery manager from Fleet. Fleetd can be built with [fleetctl](https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer).
- Devices must be enrolled to an MDM solution. TODO: Why?
- On macOS, the orbit executable in Fleetd must have "Full Disk Access", see [Grant Full Disk Access to Osquery on macOS](Adding-hosts.md#grant-full-disk-access-to-osquery-on-macos).

## How to add CIS Benchmarks

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
Fleet's current set of benchmarks only implements benchmark auditing steps that can be automated.

For a list of specific checks which are not covered by Fleet, please visit the section devoted to each benchmark.

### Audit vs. Remediation
Each benchmark has two elements:
1. Audit - how to find out whether the host is in compliance with the benchmark
2. Remediation - if the host is out of compliance with the benchmark, how to fix it

Since Fleetd is currently read-only without the ability to execute actions on the host, Fleet does not implement the remediation portions of CIS benchmarks.

To implement remediation, you can install a separate agent such as Munki, Chef, Puppet, etc. which has write functionality.

### Manual vs. Automated

For both the audit and remediation elements of a CIS Benchmark, there are two types:
1. Automated - the element can be audited or remediated without human intervention
2. Manual - the element requires human intervention to be audited or remediated

Fleet only implements automated audit checks. Manual checks require administrators to implement other processes to conduct the check.

## CIS Levels 1 and 2
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

## CIS Apple macOS 13.0 Ventura Benchmark

### Checks that require customer decision

CIS has left the parameters of the following checks up to the benchmark implementer. CIS recommends that an organization make a conscious decision for these benchmarks, but does not make a specific recommendation.

Fleet has provided both an "enabled" and "disabled" version of these benchmarks. When both policies are added, at least one will fail. Once your organization has made a decision, you can delete one or the other policy query.
The policy will be appended with a `-enabled` or `-disabled` label, such as `2.1.1.1-enabled`.

- 2.1.1.1 Audit iCloud Keychain (Level 2)
- 2.1.1.2 Audit iCloud Drive (Level 2)
- 2.5.1 Audit Siri (Level 1)
- 2.8.1 Audit Universal Control (Level 1)

### macOS 13.0 Ventura Benchmark manual checks

The following CIS benchmark checks cannot be automated and must be addressed manually:
- 2.1.1.1 Audit iCloud Keychain (Level 2): Ensure that the iCloud keychain is used consistently with organizational requirements.
- 2.1.1.2 Audit iCloud Drive (Level 2): Organizations should review third party storage solutions pertaining to existing data confidentiality and integrity requirements.
- 2.1.2 Audit App Store Password Settings (Level 2): Users who are not authorized to download software may have physical access to an unlocked computer where someone who is authorized recently made a purchase. If that is a concern, a password should be required at all times for App Store access in the Password Settings controls.
- 2.3.3.12 Ensure Computer Name Does Not Contain PII or Protected Organizational Information (Level 2): By default, the name of a macOS computer is derived from the first user created. A documented plan to better enable a complete device inventory without exposing user or organizational information is part of mature security.
- 2.5.1 Audit Siri Settings (Level 1): Where "normal" user activity is already limited, Siri use should be controlled as well.
- 2.6.1.3 Audit Location Services Access (Level 2): Privacy controls should be monitored for appropriate settings.
- 2.6.6 Audit Lockdown Mode (Level 2): Apple introduced Lockdown Mode as a security feature in their OS releases in 2022 that provides additional security protection that Apple has describes as "extreme". Users and organizations that suspect some users are targets of advanced attacks must consider using this control.
- 2.6.7 Ensure an Administrator Password Is Required to Access System-Wide Preferences (Level 1): By requiring a password to unlock system-wide System Preferences, the risk is mitigated of a user changing configurations that affect the entire system and requires an admin user to re-authenticate to make changes. Note: In previous OS versions of the macOS Benchmarks, this has been an automated recommendation. In the initial release of macOS 13.0 Ventura, this setting does not apply properly. Once the setting starts applying properly, then the recommendation will move back to automated.
- 2.8.1 Audit Universal Control Settings (Level 1): The use of devices together when some are organizational and some are not may complicate device management standards.
- 2.11.2 Audit Touch ID and Wallet & Apple Pay Settings (Level 1): Touch ID allows for an account-enrolled fingerprint to access a key that uses a previously provided password.
- 2.13.1 Audit Passwords System Preference Setting (Level 1): Organizations should remove what passwords can be saved on user computer and the ability of attackers to potentially steal organizational credentials. Limits on password storage must be evaluated based on both user risk and Enterprise risk.
- 2.14.1 Audit Notification & Focus Settings (Level 1): Some work environments will handle sensitive or confidential information with applications that can provide notifications to anyone who can see the computer screen. Organizations must review the likelihood that information may be exposed inappropriately and suppress notifications where risk is not organizationally accepted.
- 3.7 Audit Software Inventory (Level 2): Scan systems on a monthly basis and determine the number of unauthorized pieces of software that are installed. Verify that if an unauthorized piece of software is found one month, it is removed from the system the next. Note: This can be accomplished via the Software Inventory feature provided by Fleet.
- 5.2.3 Ensure Complex Password Must Contain Alphabetic Characters Is Configured (Level 2), 5.2.4 Ensure Complex Password Must Contain Numeric Character Is Configured (Level 2), 5.2.5 Ensure Complex Password Must Contain Special Character Is Configured (Level 2) and 5.2.6 Ensure Complex Password Must Contain Uppercase and Lowercase Characters Is Configured: The CIS macOS community has decided to not require the additional password complexity settings (Recommendations 5.3 - 5.6). Because of that, we have left the complexity recommendations as a manual assessment. Since there are a large amount of admins in the greater macOS world that do need these settings, we include both the guidance for the proper setting as well as probes for CIS-CAT to test.
- 5.3.1 Ensure all user storage APFS volumes are encrypted (Level 1): In order to protect user data from loss or tampering volumes, carrying data should be encrypted.
- 5.3.2 Ensure all user storage CoreStorage volumes are encrypted (Level 1): In order to protect user data from loss or tampering, volumes carrying data should be encrypted.
- 6.2.1 Ensure Protect Mail Activity in Mail Is Enabled (Level 2): Email is routinely abused by attackers, spammers and marketers. The "Protect Mail Activity" control reduces risk by hiring the current IP address of your Mac and privately downloading remote content.
- 6.3.2 Audit History and Remove History Items (Level 2): Old browser history becomes stale and the use or misuse of the data can lead to unwanted outcomes. Search engine results are maintained and often provide much more relevant current information than old website visit information.
- 6.3.5 Audit Hide IP Address in Safari Setting (Level 2): Trackers can correlate your visits through various applications including websites and is a threat to your privacy. 

Please refer to the "CIS Apple macOS 13.0 Ventura Benchmark v1.0.0 - 11-14-2022" PDF for descriptions and instructions on how to remediate.

<meta name="pageOrderInSection" value="1700">
<meta name="title" value="CIS Benchmarks">
