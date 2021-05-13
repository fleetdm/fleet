# Standard query library

Fleet's standard query library includes a growing collection of useful queries for organizations deploying Fleet and osquery.

### Queries

- [Detect machines with gatekeeper disabled](./detect-machines-with-gatekeeper-disabled.md) (macOS)
- [Detect presence of authorized SSH keys](./detect-presence-of-authorized-ssh-keys.md) (macOS, Linux)
- [Detect hosts with the firewall disabled](./detect-hosts-with-the-firewall-disabled.md) (macOS)
- [Detect Linux hosts with high severity vulnerable versions of OpenSSL](./detect-hosts-with-high-severity-vulnerable-versions-of-openssl.md) (Linux)
- [Get installed Chrome extensions](./get-installed-chrome-extensions.md) (macOS, Linux, Windows, FreeBSD)
- [Get installed FreeBSD software](./get-installed-freebsd-software.md) (FreeBSD)
- [Get installed Homebrew packages](./get-installed-homebrew-packages.md) (macOS)
- [Get installed Linux software](./get-installed-linux-software.md) (Linux)
- [Get installed macOS software](./get-installed-macos-software.md) (macOS)
- [Get installed Safari extensions](./get-installed-safari-extensions.md) (macOS)
- [Get installed Windows software](./get-installed-windows-software.md) (Windows)
- [Get laptops with failing batteries](./get-laptops-with-failing-batteries.md) (macOS)
- [Get macOS disk free space percentage](./get-macos-disk-free-space-percentage.md) (macOS)
- [Get System Logins and Logouts](./get-system-logins-and-logouts.md) (macOS)
- [Get wifi status](./get-wifi-status.md) (macOS)
- [Get Windows machines with unencrypted hard disks](./get-windows-machines-with-unencrypted-hard-disks.md) (Windows)
- [Get platform info](./get-platform-info.md) (macOS)
- [Get USB devices](./get-usb-devices.md) (macOS, Linux)
- [Count Apple applications installed](./count-apple-applications-installed.md) (macOS)
- [Get authorized keys](./get-authorized-keys.md) (macOS, Linux)
- [Get OS version](./get-os-version.md) (macOS, Linux, Windows, FreeBSD)
- [Get mounts](./get-mounts.md) (macOS, Linux)
- [Get startup items](./get-startup-items.md) (macOS, Linux, Windows, FreeBSD)
- [Get system uptime](./get-system-uptime.md) (macOS, Linux, Windows, FreeBSD)
- [Get crashes](./get-crashes.md) (macOS)

### Importing the queries in Fleet

#### After cloning the fleetdm/fleet repo, import the queries using fleetctl:
```
fleetctl apply -f fleet/handbook/queries/import-queries.yml
```

### Contributors

Want to add your own query?

1. Please copy the following YAML section and paste it at the bottom of the [import-queries.yml](./import-queries.yml) file.
```yaml
---
apiVersion: v1
kind: query
spec:
  name: What is your query called? Please use a human readable query name.
  platforms: What operating systems support your query? This can usually be determined by the osquery tables included in your query. Heading to the https://osquery.io/schema webpage to see which operating systems are supported by the tables you include.
  description: Describe your query. What does information does your query reveal?
  query: Insert query here
  purpose: What is the goal of running your query? Ex. Detection
  remediation: Are there any remediation steps to resolve the detection triggered by your query? If not, insert "N/A."
```
2. Replace each field and submit a pull request to the fleetdm/fleet GitHub repository.

For instructions on submitting pull requests to Fleet check out [the Committing Changes section](https://github.com/fleetdm/fleet/blob/58445ede82550cb574775a83ae4cf5433f325a7e/docs/4-Contribution/4-Committing-Changes.md#committing-changes) in the Contributors documentation.

### Additional resources

Listed below are great resources that contain additional queries.

- Osquery (https://github.com/osquery/osquery/tree/master/packs)
- Palantir osquery configuration (https://github.com/palantir/osquery-configuration/tree/master/Fleet)