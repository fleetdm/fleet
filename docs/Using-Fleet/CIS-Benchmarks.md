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

The manual checks which are not included in Fleet are documented below. 

## Requirements

Following are the requirements to use the CIS Benchmarks in Fleet:

- Fleet must be Premium licensed.
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