# CIS Benchmarks

The CIS Benchmarks represent the consensus-based effort of cybersecurity experts globally to help you protect your systems against threats more confidently.
For more information about CIS benchmarks check out [Center for Internet Security](https://www.cisecurity.org/cis-benchmarks).

The CIS Benchmarks are implemented in Fleet using the [Policies feature](./REST-API.md#policies). Each specific CIS benchmark check maps to one policy query in Fleet.

Fleet has implemented CIS benchmarks for the following platforms:
- CIS Apple macOS 13.0 Ventura Benchmark v1.0.0 - 11-14-2022 (82 checks) 
- CIS Microsoft Windows 10 Enterprise Benchmark v1.12.0 - 02-15-2022 (Coming soon!)

The Center for Internet Security .

## Manual and Automated

...

## Requirements

- Orbit
- FDA on macOS

## Apple macOS 13 Ventura Benchmark v1.0.0

```sh
# Download policy queries from Fleet's repository.
wget https://raw.githubusercontent.com/fleetdm/fleet/main/ee/cis/macos-13/cis-policy-queries.yml

# Apply the downloaded policies to Fleet.
fleetctl apply -f ./ee/cis/macos-13/cis-policy-queries.yml
```

The applied policies will only run on macOS devices.