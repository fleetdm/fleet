# CIS Benchmarks

CIS Benchmarks represent the consensus-based effort of cybersecurity experts to help you protect your systems against threats more confidently.
For more information about CIS Benchmarks check out [Center for Internet Security](https://www.cisecurity.org/cis-benchmarks)'s website.

Fleet has implemented native support for CIS Benchmarks for the following platforms:
- macOS 13.0 Ventura
- macOS 14.0 Sonoma
- macOS 15.0 Sequoia
- macOS 26.0 Tahoe
- Windows 10 Enterprise
- Windows 11 Enterprise
- Windows 11 (Intune)

[Where possible](#limitations), each CIS Benchmark is implemented with a [policy](https://fleetdm.com/docs/rest-api/rest-api#policies) in Fleet. 

These policy queries are intended to assess your organization's security posture against the CIS benchmarks. Because the policy queries alone do not remediate security issues, a host may fail a CIS Benchmark policy if there is no device profile or script in place to enforce the setting. By enabling [automations](https://fleetdm.com/guides/automations#basic-article) in Fleet, these policy queries can be used as the basis for managing security compliance and remediation in Fleet.

For example, this is the query for  **CIS - Ensure FileVault Is Enabled (MDM Required)**:

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

This policy is evaluating 2 attributes:

1. Is FileVault currently enabled?
2. Is there a profile in place that prevents FileVault from being disabled?

If either of these conditions fails, the host is considered to be failing the policy.

## How to add CIS Benchmarks

CIS policies are stored in GitOps format in [`docs/solutions/cis/`](https://github.com/fleetdm/fleet/tree/main/docs/solutions/cis), with one folder per platform. Each folder has a `policies/` folder with the policy YAML.

To add the policies to a fleet, copy the YAML file into your GitOps repository and reference it in the fleet's YAML file. For example:

```yaml
policies:
  - path: ../lib/cis-policy-queries.yml
```

To add only some of the policies, copy the ones you want into your own YAML file. Learn more about [Fleet's YAML files](https://fleetdm.com/docs/configuration/yaml-files#policies).

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

- [macOS 13.0 Ventura](https://github.com/fleetdm/fleet/blob/main/docs/solutions/cis/macos-13/README.md)
- [macOS 14.0 Sonoma](https://github.com/fleetdm/fleet/blob/main/docs/solutions/cis/macos-14/README.md)
- [macOS 15.0 Sequoia](https://github.com/fleetdm/fleet/blob/main/docs/solutions/cis/macos-15/README.md)
- [macOS 26.0 Tahoe](https://github.com/fleetdm/fleet/blob/main/docs/solutions/cis/macos-26/README.md)
- [Windows 10 Enterprise](https://github.com/fleetdm/fleet/blob/main/docs/solutions/cis/win-10/README.md)
- [Windows 11 Enterprise](https://github.com/fleetdm/fleet/blob/main/docs/solutions/cis/win-11/README.md)
- [Windows 11 (Intune)](https://github.com/fleetdm/fleet/blob/main/docs/solutions/cis/win-11-intune/README.md)

## Performance testing
In August 2023, we completed [scale testing on 10k Windows hosts and 70k macOS hosts](https://docs.google.com/document/d/1OSpyzMkHjVhG_-EIBkLu7X3hj_XfVASGl3IXIYChpck/edit?usp=sharing). Ultimately, we validated both server and host performance at that scale.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="lucasmrod">
<meta name="authorFullName" value="Lucas Rodriguez">
<meta name="publishedOn" value="2024-04-02">
<meta name="articleTitle" value="CIS Benchmarks">
<meta name="description" value="Read about how Fleet's implementation of CIS Benchmarks offers consensus-based cybersecurity guidance.">
