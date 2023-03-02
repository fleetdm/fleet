# CIS Benchmarks

The CIS Benchmarks represent the consensus-based effort of cybersecurity experts globally to help you protect your systems against threats more confidently.
For more information about CIS Benchmarks check out [Center for Internet Security](https://www.cisecurity.org/cis-benchmarks)'s website.

Fleet implements CIS Benchmarks using [Policies](./REST-API.md#policies). Each specific CIS benchmark check is implemented with a policy query in Fleet.
<img src=https://user-images.githubusercontent.com/2073526/220428249-7a1b6433-24fe-4686-8dfb-b555c199f47d.png />

All CIS Benchmarks implemented by Fleet are limited to a Fleet Premium or Fleet Ultimate license.

The Center for Internet Security website offers documentation for all CIS Benchmarks in PDF format. Such PDFs document all the checks, their description, rationale and how to remediate them.

Fleet has implemented CIS benchmarks for the following platforms:
- CIS Apple macOS 13.0 Ventura Benchmark v1.0.0 - 11-14-2022 (82 checks) 
- CIS Microsoft Windows 10 Enterprise Benchmark v1.12.0 - 02-15-2022 (In progress)

## Manual vs Automated

There are two types of CIS Benchmark checks, "Manual" and "Automated".
- Automated: Represents recommendations for which assessment of a technical control can be fully automated and validated to a pass/fail state
- Manual: Represents recommendations for which assessment of a technical control cannot be fully automated and requires all or some manual steps to validate that the configured state is set as expected.

Fleet only implements "Automated" checks. "Manual" checks cannot be automated as a Fleet policy. As such, they require administrators to implement other processes to conduct the check.

## Check Levels 1 and 2

### Level 1

Items in this profile intend to:
- be practical and prudent;
- provide a clear security benefit; and
- not inhibit the utility of the technology beyond acceptable means.

### Level 2

This profile extends the "Level 1" profile. Items in this profile exhibit one or more of the following characteristics:
- are intended for environments or use cases where security is paramount o acts as defense in depth measure
- may negatively inhibit the utility or performance of the technology.

## Requirements

Following are the requirements to use the CIS Benchmarks in Fleet:

- Fleet must be Premium or Ultimate licensed.
- Devices must be running [Fleetd](https://fleetdm.com/docs/using-fleet/orbit), the osquery manager from Fleet. Fleetd can be built with [fleetctl](https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer).
- Devices must be enrolled to an MDM solution.
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

## CIS Apple macOS 13.0 Ventura Benchmark

You can import macOS 13 Ventura CIS Benchmark the following way:
```sh
# Download policy queries from Fleet's repository (e.g. for macOS 13)
wget https://raw.githubusercontent.com/fleetdm/fleet/main/ee/cis/macos-13/cis-policy-queries.yml

# Apply the downloaded policies to Fleet.
fleetctl apply -f cis-policy-queries.yml
```

The above will add all the automated CIS Benchmark checks as Fleet policies.

### macOS 13.0 Ventura Benchmark manual checks that require customer decision

- 2.1.1.1 Audit iCloud Keychain (Level 2): Ensure that the iCloud keychain is used consistently with organizational requirements.
    The customer will decide whether iCloud keychain should be enabled or disabled and use only the relevant query
    2.1.1.1-enabled OR 2.1.1.1-disabled
- 2.1.1.2 Audit iCloud Drive (Level 2): Ensure that the iCloud Drive is used consistently with organizational requirements.
    The customer will decide whether iCloud Drive should be enabled or disabled and use only the relevant query
    2.1.1.2-enabled OR 2.1.1.2-disabled
- 2.5.1 Audit Siri (Level 1): Ensure that the Siri is used consistently with organizational requirements.
    The customer will decide whether Siri should be enabled or disabled and use only the relevant query
    2.5.1-enabled OR 2.5.1-disabled
- 2.8.1 Audit Universal Control (Level 1): Ensure that the Universal Control is used consistently with organizational requirements.
    The customer will decide whether Universal Control should be enabled or disabled and use only the relevant query
    2.8.1-enabled OR 2.8.1-disabled

### macOS 13.0 Ventura Benchmark manual checks

The following CIS benchmark checks cannot be automated and must be addressed manually (they are flagged as "Manual"):
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
