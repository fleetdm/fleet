# CIS Benchmarks

_Available in Fleet Premium_.

## Overview

CIS Benchmarks represent the consensus-based effort of cybersecurity experts globally to help you protect your systems against threats more confidently.
For more information about CIS Benchmarks check out [Center for Internet Security](https://www.cisecurity.org/cis-benchmarks)'s website.

Fleet has implemented native support for CIS Benchmarks for the following platforms:
- macOS 13.0 Ventura
- macOS 14.0 Sonoma
- Windows 10 Enterprise
- Windows 11 Enterprise

[Where possible](#limitations), each CIS Benchmark is implemented with a [policy query](./REST-API.md#policies) in Fleet. 

These benchmarks are intended to gauge your organization's security posture, rather than the current state of a given host. A host may fail a CIS Benchmark policy despite having the correct settings enabled if there is no configuration profile or Group Policy Object (GPO) in place to enforce the setting. For example, this is the query for  **CIS - Ensure FileVault Is Enabled (MDM Required)**:

```sql
SELECT 1 WHERE 
      EXISTS (
        SELECT 1 FROM managed_policies WHERE 
            domain='com.apple.MCX' AND 
            name='dontAllowFDEDisable' AND 
            (value = 1 OR value = 'true') AND 
            username = ''
        )
      AND NOT EXISTS (
        SELECT 1 FROM managed_policies WHERE 
            domain='com.apple.MCX' AND 
            name='dontAllowFDEDisable' AND 
            (value != 1 AND value != 'true')
        )
      AND EXISTS (
        SELECT 1 FROM disk_encryption WHERE 
            user_uuid IS NOT "" AND 
            filevault_status = 'on' 
        );  
```

Two things are being evaluated in this policy:

1. Is FileVault currently enabled?
2. Is there a profile in place that prevents FileVault from being disabled?

If either of these conditions fails, the host is considered to be failing the policy.

## Requirements

Following are the requirements to use the CIS Benchmarks in Fleet:

- To use these policies, Fleet must have an up-to-date paid license (≥Fleet Premium).
- Devices must be running [`fleetd`](https://fleetdm.com/docs/using-fleet/orbit), the lightweight agent that bundles the latest osqueryd.
- Some CIS Benchmarks explicitly involve verifying MDM-based controls, so devices must be enrolled to an MDM solution.  (Any MDM solution works, it doesn't have to be Fleet.)
- On macOS, the orbit executable in Fleetd must have "Full Disk Access", see [Grant Full Disk Access to Osquery on macOS](./Adding-hosts.md#grant-full-disk-access-to-osquery-on-macos).

### MDM required
Some of the policies created by Fleet use the [managed_policies](https://www.fleetdm.com/tables/managed_policies) table. This checks whether an MDM solution has turned on the setting to enforce the policy.
Using MDM is the recommended way to manage and enforce CIS Benchmarks. To learn how to set up MDM in Fleet, visit [here](/docs/using-fleet/mdm-macos-setup).

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

# Windows 11 (note the same file name. Rename as needed.)
wget https://raw.githubusercontent.com/fleetdm/fleet/main/ee/cis/win-11/cis-policy-queries.yml

# Apply the downloaded policies to Fleet for all files.
fleetctl apply --context <context> -f <path-to-macOS-13-policies> --policies-team <team-name>
fleetctl apply --context <context> -f <path-to-windows-10-policies> --policies-team <team-name>
fleetctl apply --context <context> -f <path-to-windows-11-policies> --policies-team <team-name>
```

To apply the policies on a specific team use the `--policies-team` flag:
```sh
fleetctl apply --policies-team "Workstations" -f cis-policy-queries.yml
```

## Limitations

Certain benchmarks require human action to audit, and cannot be automated by a policy in Fleet. For a list of specific benchmarks which are not covered, please visit the README for each benchmark:

- [macOS 13.0 Ventura](https://github.com/fleetdm/fleet/blob/main/ee/cis/macos-13/README.md)
- [macOS 14.0 Sonoma](https://github.com/fleetdm/fleet/blob/main/ee/cis/macos-14/README.md)
- [Windows 10 Enterprise](https://github.com/fleetdm/fleet/blob/main/ee/cis/win-10/README.md)
- [Windows 11 Enterprise](https://github.com/fleetdm/fleet/blob/main/ee/cis/win-11/README.md)

### Audit vs. remediation
Each benchmark has two elements:
1. Audit - how to find out whether the host is in compliance with the benchmark
2. Remediation - if the host is out of compliance with the benchmark, how to fix it

Since Fleetd is currently read-only without the ability to execute actions on the host, Fleet does not implement the remediation portions of CIS benchmarks.

To implement automated remediation, you can install a separate agent such as Munki, Chef, Puppet, etc. which has write functionality.


## Levels 1 and 2
CIS designates various benchmarks as Level 1 or Level 2 to describe the level of thoroughness and burden that each benchmark represents.

Each benchmark is tagged as `CIS_Level1` or `CIS_Level2`. 

### Level 1

Items in this profile intend to:
- be practical and prudent;
- provide a clear security benefit; and
- not inhibit the utility of the technology beyond acceptable means.

### Level 2

This profile extends the "Level 1" profile. Items in this profile exhibit one or more of the following characteristics:
- are intended for environments or use cases where security is paramount or acts as defense in depth measure
- may negatively inhibit the utility or performance of the technology.


## Performance testing
In August 2023, we completed scale testing on 10k Windows hosts and 70k macOS hosts. Ultimately, we validated both server and host performance at that scale.

Detailed results are [here](https://docs.google.com/document/d/1OSpyzMkHjVhG_-EIBkLu7X3hj_XfVASGl3IXIYChpck/edit?usp=sharing).

<meta name="pageOrderInSection" value="1700">
<meta name="title" value="CIS Benchmarks">
<meta name="description" value="Read about how Fleet's implementation of CIS Benchmarks offers consensus-based cybersecurity guidance, covering macOS 13.0 Ventura & Windows Enterprise.">
<meta name="navSection" value="Security compliance">
