# CIS Benchmarks

_Available in Fleet Premium_.

CIS Benchmarks represent the consensus-based effort of cybersecurity experts globally to help you protect your systems against threats more confidently.
For more information about CIS Benchmarks check out [Center for Internet Security](https://www.cisecurity.org/cis-benchmarks)'s website.

Fleet has implemented native support for CIS Benchmarks for the following platforms:
- macOS 13.0 Ventura
- macOS 14.0 Sonoma
- macOS 15.0 Sequoia
- Windows 10 Enterprise
- Windows 11 Enterprise

[Where possible](#limitations), each CIS Benchmark is implemented with a [policy query](https://fleetdm.com/docs/rest-api/rest-api#policies) in Fleet. 

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

## Requirements

Following are the requirements to use the CIS Benchmarks in Fleet:

- Devices must be running [`fleetd`](https://fleetdm.com/docs/using-fleet/orbit), Fleet's lightweight agent.
- Some CIS Benchmarks explicitly involve verifying MDM-based controls, so devices must be enrolled to an MDM solution.
- On macOS, the orbit component of fleetd must have "Full Disk Access", see [Grant Full Disk Access to Osquery on macOS](https://fleetdm.com/guides/enroll-hosts#grant-full-disk-access-to-osquery-on-macos).

## Limitations

Certain benchmarks cannot be automated by a policy in Fleet. For a list of specific benchmarks which are not covered, please visit the README for each benchmark:

- [macOS 13.0 Ventura](https://github.com/fleetdm/fleet/blob/main/ee/cis/macos-13/README.md)
- [macOS 14.0 Sonoma](https://github.com/fleetdm/fleet/blob/main/ee/cis/macos-14/README.md)
- [macos 15.0 Sequoia](https://github.com/fleetdm/fleet/blob/main/ee/cis/macos-15/README.md)
- [Windows 10 Enterprise](https://github.com/fleetdm/fleet/blob/main/ee/cis/win-10/README.md)
- [Windows 11 Enterprise](https://github.com/fleetdm/fleet/blob/main/ee/cis/win-11/README.md)

## Performance testing
In August 2023, we completed scale testing on 10k Windows hosts and 70k macOS hosts. Ultimately, we validated both server and host performance at that scale.

Detailed results are [here](https://docs.google.com/document/d/1OSpyzMkHjVhG_-EIBkLu7X3hj_XfVASGl3IXIYChpck/edit?usp=sharing).

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="lucasmrod">
<meta name="authorFullName" value="Lucas Rodriguez">
<meta name="publishedOn" value="2024-04-02">
<meta name="articleTitle" value="CIS Benchmarks">
<meta name="description" value="Read about how Fleet's implementation of CIS Benchmarks offers consensus-based cybersecurity guidance.">
