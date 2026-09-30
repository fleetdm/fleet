# Linux zero-touch enrollment with cloud-init: assessment and gaps

Status: exploration, not a committed plan. Tested 2026-09-29 on Fleet 4.91.1, fleetd 1.61.0, Ubuntu 24.04 and AlmaLinux 10 (arm64, QEMU).

## Verdict

Zero-touch Linux enrollment is achievable today, but it is a customer-built integration, not a Fleet feature. cloud-init installs fleetd at first boot and the host enrolls in 20 to 30 seconds. Every piece around it (image, user-data delivery, package hosting, secret handling) is the customer's to build and secure. The user-facing guide is `articles/enroll-linux-hosts-on-first-boot-with-cloud-init.md`.

Product context: `handbook/company/pricing-features-table.yml` lists zero-touch for macOS, iOS/iPadOS, and Windows only.

## What was validated

| Scenario | Result |
|---|---|
| Ubuntu 24.04, cloud-init NoCloud seed, per-fleet package with secret baked in | Enrolled in 21s |
| Ubuntu 24.04 and AlmaLinux 10, one package built with no URL or secret, config from cloud-init | Enrolled in 20 to 32s. SELinux enforcing, no denials |
| Fleet server down at first boot | fleetd retried on its own, host appeared 5s after Fleet returned |
| Wrong enroll secret | Never enrolls. An enrollment gate in `runcmd` turns this into `cloud-init status: error` |
| Unattended Ubuntu Server autoinstall (laptop path) | Enrolled 25s after first boot, cloud-init disabled itself afterward |
| Media-less NoCloud (`ds=nocloud;s=URL` in SMBIOS) and a two-line `#include` user-data, both fetching config from an HTTP endpoint | Both enrolled. cloud-init already supports Fleet-served config |
| Two VMs, same hardware UUID, default identifier | One host row, constant overwriting (21 warnings in 2 minutes) |
| Same, with `ORBIT_HOST_IDENTIFIER=instance` | Two separate hosts |
| Two clones of a disk with fleetd already enrolled | Three machines share one host row, silently. Scrubbing the node key and osquery DB before capture fixes it |
| Fleet requires end user auth, headless host | Enrollment blocked forever. `ORBIT_BYPASS_END_USER_AUTH=true` fixes it |
| Remote script execution on the autoinstalled host via `fleetctl run-script` | Worked |

Not tested: x86_64, Ubuntu Desktop autoinstall, Debian, Fedora, real cloud metadata services, Linux setup experience (read from code only), HTTPS `#include` with a private CA.

## What the customer must build

1. An image that runs cloud-init, and a way to deliver user-data (cloud field, `CIDATA` volume, autoinstall, kernel command line).
2. Package hosting reachable at first boot, plus a pipeline that rebuilds the package as fleetd ships.
3. Secret handling. user-data is not confidential. It sits root-only in `/var/lib/cloud`, is recreated on every boot, and on clouds any local process can read it from the metadata service.
4. Network and TLS trust from new hosts to Fleet and `updates.fleetdm.com`.
5. Cleanup of ephemeral hosts. Host expiry has a one-day minimum.

## Sharp edges found

- `runcmd` items are separate lines with no `set -e`. A failed `sha256sum` did not stop `dpkg -i`, and cloud-init reported `done`. Fleet's DGX Spark article and any naive snippet have this shape. The guide uses one `set -eu` script.
- cloud-init `packages:` cannot install a `.deb` from a URL (reports an error). It works for RPM but skips verification.
- Deleting cloud-init's copies of user-data is pointless. They are recreated each boot.
- `/etc/default/orbit` is not a conffile, so post-install edits are overwritten on reinstall. The working override is a systemd drop-in adding an `EnvironmentFile`, which is undocumented.
- The orbit unit orders after `network.service`, which does not exist on most distros, not `network-online.target`. Harmless because fleetd retries.
- The autoinstall user has password-protected sudo.
- Docs bug: `articles/windows-linux-setup-experience.md:45` says `--bypass-end-user-auth` only works when `mdm.allow_orbit_end_user_auth_bypass` is false. The server code and `docs/Configuration/fleet-server-configuration.md` say it works when true (the default).
- Docs bug: `articles/which-public-resources-to-expose-to-hosts.md:10` says `download.fleetdm.com` hosts `.deb` and `.rpm` base installers. Only `.msi` and `.pkg` exist (`fleetd-base.deb` and `.rpm` return 404, checked 2026-09-29).

## What Fleet would need to build

Ordered by leverage per unit of work.

### 1. Published config-less Linux packages
`fleetd-base` exists only as `.pkg` and `.msi` (`pkg/fleetdbase/fleetd_base.go`, `.github/workflows/release-fleetd-base.yml`). Publish `.deb` and `.rpm` (amd64 and arm64) at a stable download URL with a checksum and ideally a signature. This removes the customer's build pipeline and staleness problem. Precedent: `fleetd-base.msi` takes `FLEET_URL` and `FLEET_SECRET`.

### 2. A supported Linux install-time configuration path
Today `fleetctl package` allows omitting URL and secret, and orbit reads `ORBIT_*` env vars, but no documented file is meant for this. Options: a documented drop-in or `/etc/fleet/*.conf` read by the unit, or install-time `FLEET_URL`/`FLEET_SECRET` env vars as `docs/solutions/fleetd-proxy-installers/linux` already does. Also mark `/etc/default/orbit` as a conffile, and fix the unit ordering.

### 3. A Fleet-generated cloud-config
The prototype proved cloud-init can pull its whole config from a URL. Smallest version: an **Add hosts > Linux > cloud-init** tab (or `fleetctl` command) that renders user-data for the chosen fleet and distro. Larger version: a Fleet endpoint serving `user-data`, `meta-data`, and `vendor-data` per fleet, so a machine needs only `ds=nocloud;s=https://fleet.example.com/...` or a two-line `#include`. Fleet then controls package version, checksum, and options. The URL becomes the credential, so it must be revocable (see 4). Open question: `#include` runs before `ca_certs`, so a private CA does not help the fetch.

### 4. Better enrollment credentials
Enroll secrets are plaintext rows with only `created_at` (`enroll_secrets` table). No expiry, single-use, scoping, or rate limit for Linux. One-time secrets exist only for Apple MDM (`mdm.apple_one_time_enroll_secrets`), and the TPM doc lists one-time secrets as future work. A rogue holder can enroll unlimited hosts, receive the fleet's agent options, and receive any setup-experience software automatically. Directions, roughly in increasing effort:
- Expiring and single-use secrets for Linux (Tailscale, CrowdStrike, and Elastic all offer expiry).
- Rate limits on the enroll endpoints.
- Secretless join: validate a cloud identity document (AWS, GCP, Azure) or a TPM, as Teleport and Vault do. Relates to the existing TPM-backed host identity work.
- Pending or approval state for unknown hosts.

### 5. Identity safety
Duplicate identifiers currently overwrite silently. Surface the existing "duplicate identifier" server warning in the UI or an activity, and consider a safer default for cloud-provisioned hosts. Clone-scrub logic in orbit (detecting a changed DMI UUID against the stored one) exists only for macOS.

### 6. Enrollment visibility
There is no `orbit status`. The only signals are the node key file, `journalctl -u orbit`, and the `orbit_info` table. A simple status command or file would let provisioning tools wait on enrollment without the `runcmd` gate.

### 7. Ephemeral host lifecycle
Sub-day host expiry, or removal when a cloud instance is terminated. Cloud metadata tables (`ec2_instance_metadata`, `azure_instance_metadata`) exist in the schema and could drive labels, but nothing uses them, and fleet assignment is sticky at first enroll.

### 8. Post-enrollment provisioning
Linux setup experience is software-only (no scripts), ignores labels, and skips hosts enrolled more than 24 hours. Fleets that require end user auth block headless enrollment. Decide whether cloud-provisioned servers need scripts or label scoping at setup.

## Context

- Ubuntu 26.04 Desktop with Landscape offers OIDC login plus a 6-digit attach code for workstation provisioning. Canonical's own path may set expectations for Ubuntu customers.
- Ubuntu autoinstall disables cloud-init after first boot on the target, which limits re-provisioning risk on laptops.
- Windows has an analogue (Cloudbase-Init) that could use `fleetd-base.msi` properties. Not explored.

## Suggested sequencing

1. Fix the two docs bugs and publish the guide (no engineering).
2. Ship items 1 and 2 together. Together they make the cloud-init recipe a three-line user-data.
3. Item 3 as a UI/CLI generator first, hosted endpoint later.
4. Item 4 (expiry and single-use) is the largest security win and should be scoped separately.
