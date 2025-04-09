# Security

## Directory contents

- `status.md`: Current status of vulnerabilities reported on Fleet software components by security scanners (trivy). This document is currently auto-generated from files in the `vex/` directory.
- `code/`: Files used for vulnerability scanning on Fleet's source code.
- `vex/`: OpenVEX files to report status of vulnerabilities detected by Trivy on Fleet docker images.

## Vulnerability scanning

The following Github CI actions perform daily vulnerability scanning on Fleet software components.

- [trivy-scan.yml](https://github.com/fleetdm/fleet/blob/main/.github/workflows/trivy-scan.yml): Scan source code for vulnerabilities.
- [build-and-check-fleetctl-docker-and-deps.yml](https://github.com/fleetdm/fleet/blob/main/.github/workflows/build-and-check-fleetctl-docker-and-deps.yml): Scan for vulnerabilities in `fleetctl` docker image dependencies (fleetdm/fleetctl, fleetdm/wix, and fleetdm/bomutils).
- [goreleaser-snapshot-fleet.yaml](https://github.com/fleetdm/fleet/blob/main/.github/workflows/goreleaser-snapshot-fleet.yaml): Scans for vulnerabilities in `fleetdm/fleet` docker image before pushing to the Docker registry (runs daily and is triggered for every change in Fleet's source code).

## Steps to add a report for a detected CVE

We use the OpenVEX format to track the status of reported vulnerabilities.

If trivy reports a HIGH or CRITICAL CVE on one of Fleet's docker images (reported by the previously mentioned Github Actions), then we need to assess the report and track it with a status of "not affected", "affected", "fixed", or "under investigation".

Once the status is determined, we use the [vexctl](https://github.com/openvex/vexctl) tool to create a VEX file.
```sh
brew install vexctl
```

Example for `CVE-2023-32698` on package `github.com/goreleaser/nfpm/v2` which we know doesn't affect `fleetdm/fleetctl`:
```sh
vexctl create --product="fleetctl,pkg:golang/github.com/goreleaser/nfpm/v2" \
  --vuln="CVE-2023-32698" \
  --status="not_affected" \
  --author="@getvictor" \
  --justification="vulnerable_code_cannot_be_controlled_by_adversary" \
  --status-note="When packaging linux files, fleetctl does not use global permissions. It was verified that packed fleetd package files do not have group/global write permissions." > security/vex/fleetctl/CVE-2023-32698.vex.json
```

Similarly, for `CVE-2024-8260` on package `github.com/open-policy-agent/opa` which we know doesn't affect `fleetdm/fleet`:
```sh
vexctl create --product="fleet,pkg:golang/github.com/open-policy-agent/opa" \
  --vuln="CVE-2024-8260" \
  --status="not_affected" \
  --author="@luacsmrod" \
  --justification="vulnerable_code_cannot_be_controlled_by_adversary" \
  --status-note="Fleet doesn't run on Windows, so it's not affected by this vulnerability." > security/vex/fleetctl/CVE-2024-8260.vex.json
```

Examples of `--product` flag values (which accept "PURLs"):
- `liblzma5` debian package: `pkg:deb/debian/liblzma5`.
- `github.com/goreleaser/nfpm/v2` golang package: `pkg:golang/github.com/goreleaser/nfpm/v2`.
- `xerces/xercesImpl` java package: `pkg:maven/xerces/xercesImpl`.

When new VEX files are generated or updated you have to run:
```sh
make vex-report
```
to update `security/status.md`.