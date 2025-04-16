# Security

## Directory contents

- [status.md](status.md): Current status of vulnerabilities reported on Fleet software components by security scanners (trivy). This document is currently auto-generated from files in the `vex/` directory.
- `code/`: Files used for vulnerability scanning on Fleet's source code.
- `vex/`: OpenVEX files to report status of vulnerabilities detected by Trivy on Fleet docker images.

## Vulnerability scanning

The following Github CI actions perform daily vulnerability scanning on Fleet software components.

- [trivy-scan.yml](https://github.com/fleetdm/fleet/blob/main/.github/workflows/trivy-scan.yml): Scan source code for vulnerabilities.
- [build-and-check-fleetctl-docker-and-deps.yml](https://github.com/fleetdm/fleet/blob/main/.github/workflows/build-and-check-fleetctl-docker-and-deps.yml): Scans for `HIGH` and `CRITICAL` vulnerabilities in `fleetctl` docker image dependencies (`fleetdm/fleetctl`, `fleetdm/wix`, and `fleetdm/bomutils`).
- [goreleaser-snapshot-fleet.yaml](https://github.com/fleetdm/fleet/blob/main/.github/workflows/goreleaser-snapshot-fleet.yaml): Scans for HIGH and CRITICAL vulnerabilities in `fleetdm/fleet` docker image before pushing to the Docker registry (runs daily and is triggered for every change in Fleet's source code).
- [check-vulnerabilities-in-released-docker-images.yml](https://github.com/fleetdm/fleet/blob/main/.github/workflows/check-vulnerabilities-in-released-docker-images.yml): Scans for `CRITICAL` vulnerabilities in the last 5 minor released versions of the `fleetdm/fleet` and on the latest release of `fleetdm/fleetctl`.

## Process to run when a CVE is reported

### 1. Update report (status.md)

If trivy reports a HIGH or CRITICAL CVE on one of Fleet's docker images (reported by the previously mentioned Github Actions), then we need to assess the report and track it with a status of "not affected", "affected", "fixed", or "under investigation".

We use the OpenVEX format to track the status of reported vulnerabilities (`vex/` folder).

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

When new VEX files are generated or updated we can update the `security/status.md` file by running:
```sh
make vex-report
```

### 2. Update software

If the detected vulnerability can be fixed by updating the base docker image or removing/changing components in the docker image then we do so and the update will be present on the next release. (It is good practice to keep software up-to-date.)

### 3. Process for "affected" CRITICAL vulnerabilities

#### fleetdm/fleet

Following is the process to run when a `CRITICAL` CVE affects any of the five last releases of `fleetdm/fleet` docker image (reported by [check-vulnerabilities-in-released-docker-images.yml](https://github.com/fleetdm/fleet/blob/main/.github/workflows/check-vulnerabilities-in-released-docker-images.yml)).

1. We will use the information reported by the scanner and update our `status.md` to keep users/customers informed.
2. If the `CRITICAL` vulnerability (that has a fix) is on the `latest` release, we'll file a critical/P0 bug and release a patch ASAP (within 1 business day). The previous four versions scanned won't be retroactively patched, only `latest` will be patched.

#### fleetdm/fleetctl

Following is the process to run when a `CRITICAL` CVE affects the released `fleetdm/fleetctl:latest` docker image:

1. After `security/status.md` is updated, notify users/customers about the CVE in the `fleetdm/fleetctl` image and possible remediations.
2. Create a Github issue with a `P0`/`security` label to track the fix.
3. The fix will be released on the next release of the `fleetdm/fleetctl` docker image.