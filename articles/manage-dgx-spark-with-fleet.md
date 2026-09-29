# Manage DGX Spark with Fleet

NVIDIA's DGX Spark manageability guide is built around SSH: an orchestrator connects to each device, runs a tool, and parses the JSON it prints. This guide sets up the same coverage in Fleet. You'll enroll DGX Spark into your existing workstations fleet, scope DGX-specific reports and policies with a label, collect diagnostics bundles with file carving, and run controlled updates, all managed in Git.

Every query and script here was tested on a DGX Spark running DGX OS 7.5.0 (Ubuntu 24.04.5 on arm64, kernel `7.0.0-1019-nvidia`, NVIDIA driver 580.178.04) with fleetd (Orbit 1.60.0, osquery 5.23.1). Your image will differ, so re-run anything version-specific before you roll it out.

## Prerequisites

- Fleet Premium 4.82.0 or later. This guide uses fleets, label-scoped policies, and policy script automations.
- `fleetctl` installed and a [GitOps repository](https://fleetdm.com/docs/configuration/yaml-files) that manages your Fleet instance.
- An existing workstations fleet, such as `fleets/workstations.yml`, and its enroll secret.
- A DGX Spark, or a partner system built on the NVIDIA GB10 chip, running DGX OS 7, with outbound HTTPS access to your Fleet server.
- An S3 bucket configured as Fleet's [file carving backend](https://fleetdm.com/docs/configuration/fleet-server-configuration#s3-file-carving-backend), if you need to keep diagnostics bundles longer than 24 hours.

> **Warning:** Decide disk encryption and SSH hardening before you image the device. A stock DGX Spark ships with an unencrypted ext4 root, and Linux disk encryption can only be enabled during OS setup. If the OS is already installed unencrypted, you have to reinstall it.

## How NVIDIA's tools map to Fleet

If you're working from NVIDIA's manageability guide, use this table to find the Fleet equivalent for each tool.

| NVIDIA tool | Fleet equivalent |
| :- | :- |
| `device_identity.py`, `os_build_identity.py` | Host details, plus reports on `system_info`, `os_version`, and `kernel_info`. A script reads the DGX OS build string. |
| `hardware_config.py` | Reports on `pci_devices`, `memory_devices`, `block_devices`, and `interface_details` |
| `firmware_reporter.py` | Report on `platform_info`, plus a script that wraps `fwupdmgr get-devices` |
| `driver_inventory_reporter.py` | Reports on `deb_packages` (installed) and `kernel_modules` (loaded), plus a policy for drift |
| `software_inventory_reporter.py` | Fleet's built-in software inventory and vulnerability detection |
| `NVAIAread`, `NVAIAwrite` | Labels and fleets. Wrap the NVIDIA tools in a script only if the UEFI tag must stay authoritative. |
| `spark_diagctl.py` health | Policies, with automations for tickets, webhooks, and remediation scripts |
| `spark_diagctl.py` bundle | A script that builds the bundle, plus file carving to retrieve it |
| `reset_reason_reporter.py` | Report on `uptime`, plus a script that reads the journal |
| `spark_updatectl.py` | A scheduled batch script run, with precheck and postcheck policies |
| Landscape reference scripts | Reports and policies on `apt_sources`, `secureboot`, `disk_encryption`, and `systemd_units` |
| Rings and waves | A dynamic label for the device class and manual labels for each wave |
| Evidence and ticket linkage | The activity feed, audit log streaming, and the Jira and Zendesk integrations |

> **Note:** DGX OS reports as Ubuntu, so Fleet matches distribution packages against Ubuntu's OVAL data. The DGX kernel is a custom `-nvidia` variant, and NVIDIA's packages come from NVIDIA's repositories, so both fall back to NVD matching, which produces more false positives.

## Build the fleetd package

DGX Spark is arm64, so build the package with the `--arch` flag:

```sh
fleetctl package \
  --type=deb \
  --arch=arm64 \
  --fleet-url=https://fleet.example.com \
  --enroll-secret=<workstations-fleet-enroll-secret> \
  --enable-scripts
```

Use your workstations fleet's enroll secret. Don't create a separate DGX Spark fleet. A Spark is a Linux workstation, and it should inherit your existing Linux baseline, patch cadence, and agent configuration. The label you create later scopes the DGX-specific pieces.

> **Note:** `--enable-scripts` is required. Script execution is off by default, and every script in this guide, plus remote lock and wipe on Linux, depends on it.

Host the package somewhere your devices can download it from, such as an internal artifact server.

## Enroll DGX Spark

NVIDIA's guide provisions DGX Spark with a cloud-init NoCloud seed on a USB drive labeled `CIDATA`. Keep that seed and add fleetd to it. This example also writes the SSH drop-in that the root login policy checks later:

```yaml
#cloud-config
write_files:
  - path: /etc/ssh/sshd_config.d/10-hardening.conf
    permissions: "0644"
    content: |
      PermitRootLogin no

runcmd:
  - [ sh, -c, "curl -fsSL -o /tmp/fleet-osquery.deb https://your-artifact-host/fleet-osquery_arm64.deb" ]
  - [ dpkg, -i, /tmp/fleet-osquery.deb ]
  - [ shred, -u, /tmp/fleet-osquery.deb ]
```

The package contains your enroll secret, which is why the seed shreds it after install. After enrollment, the secret stays in `/etc/default/orbit`, readable only by root.

There's no approval step. The device appears in Fleet on its first check-in, in the fleet that matches the enroll secret. For more cloud-init and Kickstart patterns, see [Enrolling Linux devices for Fleet management](https://fleetdm.com/articles/enrolling-linux-devices-for-fleet-management).

## Spot-check tables as root

Before you write reports, check what the tables return on your image. Run osquery as root, with the same lens path fleetd uses:

```sh
sudo /opt/orbit/bin/osqueryd/linux-arm64/stable/osqueryd -S \
  --augeas_lenses=/opt/orbit/lenses \
  --json "SELECT * FROM platform_info;"
```

> **Warning:** Don't test with a bare `osqueryi` as your own user. osquery reads SMBIOS data from `/sys/firmware/dmi/tables/DMI`, which only root can read. As a regular user, `system_info` hardware columns, `platform_info`, and `memory_devices` come back empty, and `augeas` returns zero rows without `--augeas_lenses`. fleetd runs as root with the lens path set, so these tables work under the agent.

You can also run any query in this guide as a live report from the Fleet UI, which runs it through fleetd.

## Lay out the GitOps repository

Give DGX Spark a directory under `lib/linux/`, not its own fleet file. Its reports, policies, and scripts are pulled into the workstations fleet and scoped by label:

```
fleet-config/
  default.yml
  fleets/
    workstations.yml
  lib/
    linux/
      dgx-spark/
        labels.yml
        policies/
          baseline.yml
          health.yml
        reports/
          inventory.yml
        scripts/
          dgx-build-string.sh
          dgx-firmware-inventory.sh
          dgx-firmware-check-updates.sh
          gpu-health.sh
          reset-context.sh
          dgx-diag-bundle.sh
          dgx-os-update.sh
```

Add the DGX Spark files to your workstations fleet:

```yaml
# fleets/workstations.yml
name: Workstations

policies:
  - paths: ../lib/linux/dgx-spark/policies/*.yml
  # ... your existing workstation policies ...

reports:
  - paths: ../lib/linux/dgx-spark/reports/*.yml
  # ... your existing workstation reports ...

controls:
  scripts:
    - paths: ../lib/linux/dgx-spark/scripts/*.sh
    # ... your existing workstation scripts ...
```

> **Note:** Scripts are scoped to a fleet, not a label. Every script in `controls.scripts` appears for every host in the workstations fleet, including Macs. Name DGX scripts clearly, and start anything that changes state with a guard such as `[ -f /etc/dgx-release ] || { echo "not a DGX system"; exit 1; }`.

## Define the DGX Spark labels

Use a dynamic label for the device class and manual labels for rollout waves, since wave membership is a human decision:

```yaml
# lib/linux/dgx-spark/labels.yml
- name: DGX Spark
  description: DGX Spark and partner systems built on NVIDIA GB10
  label_membership_type: dynamic
  platform: linux
  # On DGX Spark, hardware_model is the literal string NVIDIA_DGX_Spark.
  query: >
    SELECT 1 FROM system_info
    WHERE hardware_model LIKE '%Spark%' OR hardware_model LIKE '%GB10%';

- name: DGX Spark pilot
  description: First wave for DGX OS and driver changes
  label_membership_type: manual
  # hardware_serial matches DGX_SERIAL_NUMBER in /etc/dgx-release.
  hosts:
    - "1983426000956"
    - "1983426000957"

- name: DGX Spark wave 1
  description: Second wave for DGX OS and driver changes
  label_membership_type: manual
  hosts: []
```

Reference the file from the `labels` key in `default.yml`:

```yaml
# default.yml
labels:
  - path: ./lib/linux/dgx-spark/labels.yml
  # ... your existing labels ...
```

Every report and policy in this guide depends on the DGX Spark label. If a Spark is reimaged onto a different OS, it stops matching the label and stops being graded against the DGX baseline.

> **Warning:** Omitting the `labels` key from `default.yml` deletes your existing labels, unless you've turned on the labels exception in **Settings** > **Integrations** > **Change management**. Any label a policy or report references must also be defined in `labels`.

## Add inventory reports

These reports replace NVIDIA's identity, hardware, firmware, and driver collectors. Setting `automations_enabled: true` sends results to your log destination, which you need for drift history beyond the 1,000 results Fleet keeps per report.

```yaml
# lib/linux/dgx-spark/reports/inventory.yml
- name: DGX Spark identity
  description: Hardware identity for CMDB and asset acceptance.
  platform: linux
  interval: 86400
  automations_enabled: true
  labels_include_any:
    - DGX Spark
  # cpu_brand is omitted because ARM has no brand string.
  query: >
    SELECT hardware_serial, uuid, hardware_vendor, hardware_model,
           hardware_version, board_serial, cpu_physical_cores,
           cpu_logical_cores, physical_memory, hostname, computer_name
    FROM system_info;

- name: DGX Spark OS build
  description: OS version and running kernel.
  platform: linux
  interval: 86400
  automations_enabled: true
  labels_include_any:
    - DGX Spark
  # os_version.build is omitted because Ubuntu doesn't populate it.
  query: >
    SELECT os.name, os.version, os.major, os.minor, os.patch, os.arch,
           k.version AS kernel_version, k.arguments AS kernel_cmdline
    FROM os_version os, kernel_info k;

- name: DGX Spark PCI devices
  description: GPU, NIC, and other PCI devices with their bound driver.
  platform: linux
  interval: 86400
  labels_include_any:
    - DGX Spark
  # model is empty for every PCI device on this platform, including the GPU.
  query: SELECT pci_slot, pci_class, vendor, driver FROM pci_devices ORDER BY pci_class;

- name: DGX Spark memory
  description: Memory from SMBIOS.
  platform: linux
  interval: 86400
  labels_include_any:
    - DGX Spark
  # memory_type is omitted because osquery doesn't map LPDDR5.
  query: SELECT device_locator, size, configured_clock_speed, manufacturer FROM memory_devices;

- name: DGX Spark storage
  description: Block devices, excluding snap loop devices.
  platform: linux
  interval: 86400
  labels_include_any:
    - DGX Spark
  query: SELECT name, model, size, type, vendor FROM block_devices WHERE name NOT LIKE '%loop%';

- name: DGX Spark network interfaces
  description: Interfaces with link speed, PCI slot, and addresses.
  platform: linux
  interval: 86400
  labels_include_any:
    - DGX Spark
  query: >
    SELECT id.interface, id.mac, id.type, id.mtu, id.link_speed, id.pci_slot,
           ia.address, ia.mask
    FROM interface_details id
    LEFT JOIN interface_addresses ia ON id.interface = ia.interface;

- name: DGX Spark firmware
  description: UEFI firmware version and revision.
  platform: linux
  interval: 86400
  automations_enabled: true
  labels_include_any:
    - DGX Spark
  query: SELECT vendor, version, revision, date FROM platform_info;

- name: DGX Spark driver packages
  description: Installed NVIDIA driver, kernel module, and CUDA packages.
  platform: linux
  interval: 86400
  automations_enabled: true
  labels_include_any:
    - DGX Spark
  query: >
    SELECT name, version, arch FROM deb_packages
    WHERE name LIKE 'nvidia-%'
       OR name LIKE 'libnvidia-%'
       OR name LIKE 'cuda-%'
       OR name LIKE 'linux-modules-nvidia-%';

- name: DGX Spark loaded kernel modules
  description: NVIDIA, NVMe, and Mellanox kernel modules.
  platform: linux
  interval: 3600
  labels_include_any:
    - DGX Spark
  query: >
    SELECT name, status, used_by FROM kernel_modules
    WHERE name LIKE 'nvidia%' OR name IN ('nvme', 'mlx5_core', 'mlx5_ib');

- name: DGX Spark containers
  description: Running containers and the images behind them.
  platform: linux
  interval: 3600
  labels_include_any:
    - DGX Spark
  query: SELECT id, name, image, image_id, state, status FROM docker_containers;

- name: DGX Spark apt sources
  description: Package repositories the device trusts.
  platform: linux
  interval: 86400
  labels_include_any:
    - DGX Spark
  query: SELECT name, base_uri, release, components, architectures FROM apt_sources;
```

A few things to know about what these return on DGX Spark:

- **Memory:** The 128 GB is unified memory shared by the CPU and GPU. SMBIOS reports it as one device, so `memory_devices` returns one row. Use `system_info.physical_memory` for capacity.
- **Drivers:** DGX OS 7 has no DKMS. The driver ships as `nvidia-driver-580-open` plus a pre-built `linux-modules-nvidia-580-open-nvidia-hwe-24.04` package. Anything that looks for `nvidia-dkms-*` finds nothing.
- **Containers:** Fleet doesn't scan container image layers for CVEs. Use this report to see what's running, and hand the image list to your registry scanner.
- **Apt sources:** A stock Spark trusts NVIDIA's `baseos`, CUDA, and Workbench repositories plus a Canonical `nvidia-desktop-edge` PPA. `apt_sources` has no signing metadata, so verify keyrings with a script.

## Add baseline and health policies

A policy passes when its query returns at least one row. Each policy below returns a row when the device is compliant. Fleet has no configuration profiles for Linux, so these policies detect drift, and automations fix it.

```yaml
# lib/linux/dgx-spark/policies/baseline.yml
- name: DGX OS approved kernel
  platform: linux
  description: Device is running a kernel from the approved DGX OS list.
  resolution: Schedule the device into the next DGX OS update wave.
  query: SELECT 1 FROM kernel_info WHERE version IN ('7.0.0-1019-nvidia');
  labels_include_any:
    - DGX Spark

- name: DGX Spark approved UEFI firmware
  platform: linux
  description: UEFI firmware is an approved version.
  resolution: Apply the approved firmware through fwupd in the next change window.
  query: SELECT 1 FROM platform_info WHERE version IN ('5.36_0ACUM027');
  labels_include_any:
    - DGX Spark

- name: NVIDIA driver loaded
  platform: linux
  description: The nvidia kernel module is live.
  resolution: Reboot the device to load the newly installed kernel module package.
  query: SELECT 1 FROM kernel_modules WHERE name = 'nvidia' AND status = 'Live';
  labels_include_any:
    - DGX Spark

- name: Secure Boot enabled
  platform: linux
  description: UEFI Secure Boot is enabled.
  resolution: Turn on Secure Boot in UEFI setup and re-enroll the MOK if required.
  query: SELECT 1 FROM secureboot WHERE secure_boot = 1;
  labels_include_any:
    - DGX Spark

- name: Disk encryption enabled
  platform: linux
  description: At least one disk is encrypted.
  resolution: Reinstall DGX OS with disk encryption turned on.
  query: SELECT 1 FROM disk_encryption WHERE encrypted = 1;
  labels_include_any:
    - DGX Spark

- name: SSH root login disabled
  platform: linux
  description: The SSH hardening drop-in sets PermitRootLogin to no.
  resolution: Add PermitRootLogin no to /etc/ssh/sshd_config.d/10-hardening.conf.
  query: >
    SELECT 1 FROM augeas
    WHERE path = '/etc/ssh/sshd_config.d/10-hardening.conf'
      AND label = 'PermitRootLogin'
      AND value = 'no';
  labels_include_any:
    - DGX Spark
```

> **Warning:** Set `labels_include_any` on each policy. Setting it once at the top of the `policies` list doesn't apply it to the policies inside. `platform: linux` isn't a substitute: without the label, every Linux workstation in the fleet is graded against the DGX kernel list.

Use explicit allow-lists for the kernel and firmware, and update them in Git when you approve a new build. DGX OS 7 has already moved from kernel 6.11 to 6.17 to 7.0, so a policy that matches a kernel line fails across every device on the next update. The firmware version string, such as `5.36_0ACUM027`, doesn't sort in a way a `>=` comparison can handle.

The SSH policy checks the drop-in file from your cloud-init seed, not `/etc/ssh/sshd_config`. The `augeas` table parses one literal path, and in the stock DGX OS image `PermitRootLogin` only appears as a comment in the main file.

On a stock image, expect the disk encryption and SSH policies to fail. That's the finding, and the fix belongs in provisioning.

```yaml
# lib/linux/dgx-spark/policies/health.yml
- name: DGX Spark root filesystem has 20 GiB free
  platform: linux
  description: The root filesystem has at least 20 GiB available.
  resolution: Remove unused container images, models, or datasets.
  query: >
    SELECT 1 FROM mounts
    WHERE path = '/' AND (blocks_available * blocks_size) >= 21474836480;
  labels_include_any:
    - DGX Spark

- name: DGX Spark time sync running
  platform: linux
  description: A time synchronization service is active.
  resolution: Start systemd-timesyncd.
  query: >
    SELECT 1 FROM systemd_units
    WHERE id IN ('systemd-timesyncd.service', 'chrony.service', 'chronyd.service')
      AND active_state = 'active';
  labels_include_any:
    - DGX Spark

- name: DGX Spark has no failed systemd units
  platform: linux
  description: No systemd units are in a failed state.
  resolution: Run systemctl --failed and fix or reset the failed units.
  query: >
    SELECT 1 WHERE NOT EXISTS (
      SELECT 1 FROM systemd_units WHERE active_state = 'failed'
    );
  labels_include_any:
    - DGX Spark
```

After GitOps runs, open each policy in Fleet and confirm the number of targeted hosts matches your Spark count. A higher number means a policy is missing its label.

To open a Jira or Zendesk ticket when a policy fails, set up a [ticket automation](https://fleetdm.com/guides/automations). Ticket automations fire only when a host newly fails a policy, and Fleet checks for them once a day by default.

## Add scripts

Some signals only come from NVIDIA's own tools, so run them as scripts. Fleet keeps only the last 10,000 characters of script output, so keep each script's output small.

Read the DGX OS build string. It includes both the installed build and the pending OTA version:

```sh
#!/bin/bash
# lib/linux/dgx-spark/scripts/dgx-build-string.sh
set -euo pipefail
cat /etc/dgx-release
```

List component firmware from fwupd. The raw `fwupdmgr get-devices --json` output is about 39,000 bytes on a current Spark, so this script keeps four fields per device, which brings it to about 1,000 bytes:

```sh
#!/bin/bash
# lib/linux/dgx-spark/scripts/dgx-firmware-inventory.sh
set -euo pipefail
fwupdmgr get-devices --json 2>/dev/null | \
  python3 -c 'import json,sys; d=json.load(sys.stdin); print(json.dumps([{"name":x.get("Name"),"version":x.get("Version"),"vendor":x.get("Vendor"),"plugin":x.get("Plugin")} for x in d.get("Devices",[])], separators=(",",":")))'
```

Take a GPU health snapshot:

```sh
#!/bin/bash
# lib/linux/dgx-spark/scripts/gpu-health.sh
# ecc.errors.* is omitted because it returns [N/A] on GB10.
set -euo pipefail
nvidia-smi --query-gpu=name,driver_version,temperature.gpu,clocks_throttle_reasons.active \
  --format=csv,noheader
```

> **Note:** On GB10, `nvidia-smi` reports memory usage as `Not Supported` because the CPU and GPU share memory, and ECC counters return `[N/A]`. Don't build health checks on either.

Collect reset and stability context. osquery's `last` table filters out boot records, so boot history comes from the journal. The Xid lines are NVIDIA GPU fault codes, which tell you whether the GPU faulted before a reboot:

```sh
#!/bin/bash
# lib/linux/dgx-spark/scripts/reset-context.sh
set -euo pipefail
journalctl --list-boots --no-pager | tail -n 10
echo "---"
journalctl -k -b -1 --no-pager -p err 2>/dev/null | tail -n 40 || true
echo "---"
grep -i -E 'xid|nvrm' /var/log/kern.log 2>/dev/null | tail -n 20 || true
```

To run a script on one Spark, open the host's details page and select **Actions** > **Run Script**. Output and exit codes appear in the host's activity.

## Retrieve diagnostics bundles with file carving

NVIDIA's guide pulls diagnostics bundles over `scp`. In Fleet, a script builds the bundle and prints a small JSON pointer, then a file carve retrieves the file.

### Configure the agent

Add the carver flags and a longer script timeout to the workstations fleet's `agent_options`. `nvidia-bug-report.sh` regularly takes longer than the default 300-second timeout:

```yaml
# fleets/workstations.yml
agent_options:
  script_execution_timeout: 3600
  command_line_flags:
    disable_carver: false
    carver_disable_function: false
    carver_start_endpoint: /api/v1/osquery/carve/begin
    carver_continue_endpoint: /api/v1/osquery/carve/block
    carver_block_size: 5242880
    carver_compression: true
  # ... your existing agent options ...
```

> **Note:** Agent options apply to the whole fleet, not to a label. The longer timeout applies to every workstation in the fleet. That's usually fine, since a higher ceiling doesn't make other scripts run longer. Command line flags take effect when fleetd restarts.

With S3 as the carve backend, `carver_block_size` must be at least 5 MiB (5,242,880 bytes). With the default MySQL backend, keep it below MySQL's `max_allowed_packet`.

### Build the bundle

This script writes the bundle to a fixed path, so the same carve query works for every device:

```sh
#!/bin/bash
# lib/linux/dgx-spark/scripts/dgx-diag-bundle.sh
set -euo pipefail
[ -f /etc/dgx-release ] || { echo "not a DGX system"; exit 1; }

RUN_ID="$(date -u +%Y%m%dT%H%M%SZ)"
SERIAL="$(cat /sys/class/dmi/id/product_serial 2>/dev/null || echo unknown)"
OUT="/var/tmp/fleet-carve/dgx-diag-latest.tar.gz"
mkdir -p /var/tmp/fleet-carve

nvidia-bug-report.sh --output-file "/var/tmp/nvidia-bug-report-${RUN_ID}.log.gz" >/dev/null 2>&1
tar -czf "${OUT}" -C /var/tmp "nvidia-bug-report-${RUN_ID}.log.gz" 2>/dev/null

SHA="$(sha256sum "${OUT}" | cut -d' ' -f1)"
BYTES="$(stat -c%s "${OUT}")"

printf '{"run_id":"%s","serial":"%s","path":"%s","sha256":"%s","bytes":%s}\n' \
  "${RUN_ID}" "${SERIAL}" "${OUT}" "${SHA}" "${BYTES}"
```

`nvidia-bug-report.sh` needs root, which fleetd already has. The printed byte count tells you the bundle size before you carve it, and the SHA-256 hash lets you confirm the file you retrieve matches the file the device built.

### Carve the bundle

Carves run as live reports, so the host must be online. Run the carve against the host:

```sh
fleetctl report --hosts <hostname> \
  --query "SELECT * FROM carves WHERE carve = 1 AND path = '/var/tmp/fleet-carve/dgx-diag-latest.tar.gz';"
```

List carves and download the bundle:

```sh
fleetctl get carves
fleetctl get carve --outfile dgx-diag.tar <carve-id>
```

The download is a `.tar` archive, compressed with Zstandard if `carver_compression` is on. Compare the SHA-256 hash of the file inside against the hash the script printed.

> **Warning:** On the default backend, carve contents expire 24 hours after the first block arrives. NVIDIA suggests keeping incident bundles for 90 to 180 days, so configure the S3 carve backend and a bucket lifecycle policy that matches your retention target.

A few more limits to plan for:

- A carve fails if any single block fails after three attempts. On an unreliable network, keep bundles as small as the investigation allows.
- You can't attach a carve to a policy failure. A realistic workflow is: the policy fails, an automation opens a ticket, and an operator runs the bundle script and the carve.
- Anyone who can run live reports on these hosts can carve files from them. Scope fleet access and roles to match.

For more, see [File carving in Fleet](https://fleetdm.com/guides/file-carving).

## Run controlled updates

NVIDIA's `spark_updatectl.py` wraps updates in a change window with precheck and postcheck evidence. In Fleet, those pieces are a scheduled batch script run and the policies you already added.

### Add the update scripts

This script captures the kernel and driver before and after `dist-upgrade`:

```sh
#!/bin/bash
# lib/linux/dgx-spark/scripts/dgx-os-update.sh
# Run inside an approved change window only.
set -euo pipefail
[ -f /etc/dgx-release ] || { echo "not a DGX system"; exit 1; }

export DEBIAN_FRONTEND=noninteractive

# DGX OS 7 has no DKMS. Match the driver and pre-built module packages by wildcard.
BEFORE_KERNEL="$(uname -r)"
BEFORE_DRIVER="$(dpkg-query -W -f='${Package} ${Version}\n' 'nvidia-driver-*-open' 2>/dev/null | head -n1 || echo none)"
BEFORE_MODULE="$(dpkg-query -W -f='${Package} ${Version}\n' 'linux-modules-nvidia-*-open-nvidia-hwe-*' 2>/dev/null | head -n1 || echo none)"

apt-get update -qq
apt-get -y -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold dist-upgrade

AFTER_DRIVER="$(dpkg-query -W -f='${Package} ${Version}\n' 'nvidia-driver-*-open' 2>/dev/null | head -n1 || echo none)"
AFTER_MODULE="$(dpkg-query -W -f='${Package} ${Version}\n' 'linux-modules-nvidia-*-open-nvidia-hwe-*' 2>/dev/null | head -n1 || echo none)"
REBOOT_REQUIRED=false
[ -f /var/run/reboot-required ] && REBOOT_REQUIRED=true

printf '{"before_kernel":"%s","before_driver":"%s","before_module":"%s","after_driver":"%s","after_module":"%s","reboot_required":%s}\n' \
  "${BEFORE_KERNEL}" "${BEFORE_DRIVER}" "${BEFORE_MODULE}" "${AFTER_DRIVER}" "${AFTER_MODULE}" "${REBOOT_REQUIRED}"
```

This script checks for firmware updates through LVFS. The apply step is commented out so you can turn it on only when a change record authorizes it:

```sh
#!/bin/bash
# lib/linux/dgx-spark/scripts/dgx-firmware-check-updates.sh
set -euo pipefail
fwupdmgr refresh --force >/dev/null 2>&1 || true
fwupdmgr get-updates --json 2>/dev/null || echo '{"Devices":[]}'
# fwupdmgr update --assume-yes --no-reboot-check
```

> **Note:** UEFI capsule updates apply at the next boot. Plan firmware as two windows: one to stage the update and one to reboot.

### Run the change window

1. Confirm the baseline policies pass for every host in the **DGX Spark pilot** label. If they don't, fix them before you promote. This is your precheck.
2. Go to **Hosts** and filter by the **DGX Spark pilot** label.
3. Select the hosts, then select **Run Script**.
4. Hover over `dgx-os-update.sh`, select **Run Script**, then select **Schedule for later** and pick the start of your change window.
5. Track the run under **Controls** > **Scripts** > **Batch progress**. You can cancel pending hosts from there.
6. After the devices reboot, confirm the baseline policies pass again. This is your postcheck.
7. Add the new kernel version to the approved list in `baseline.yml`, merge the pull request, and repeat for the next wave.

The batch record, the activity feed, and the policy history together are the change record.

> **Warning:** Fleet doesn't roll back an apt transaction or a firmware capsule. Rollback for DGX OS stays with NVIDIA's mechanism, and your rollout waves are the main control.

### Optional: Run updates from a policy

You can attach the update script to the approved kernel policy so Fleet runs it when a device drifts. The script must be in the same fleet's `controls.scripts`, and the path is relative to the policy file:

```yaml
# lib/linux/dgx-spark/policies/baseline.yml
- name: DGX OS approved kernel
  platform: linux
  description: Device is running a kernel from the approved DGX OS list.
  resolution: Schedule the device into the next DGX OS update wave.
  query: SELECT 1 FROM kernel_info WHERE version IN ('7.0.0-1019-nvidia');
  labels_include_any:
    - DGX Spark
  run_script:
    path: ../scripts/dgx-os-update.sh
```

By default, the script runs on the first failure or when a host changes from passing to failing. Set `continuous_automations_enabled: true` to run it on every failing check-in. Failed script runs retry up to three times total, so make remediation scripts safe to repeat.

An unattended `dist-upgrade` is a different risk than a batch someone released on purpose. For most teams, a ticket automation on the policy plus a scheduled batch for the change is the safer default.

## Verify

1. Go to **Hosts**, filter by the **DGX Spark** label, and confirm every Spark appears.
2. Open a Spark's host details and confirm **Hardware model** shows `NVIDIA_DGX_Spark`.
3. Go to **Reports**, open **DGX Spark identity**, and confirm it returns a row for each Spark.
4. Go to **Policies**, filter by the workstations fleet, and confirm each DGX policy targets only your Sparks.
5. Run `dgx-diag-bundle.sh` on one pilot device, carve the bundle, and confirm the SHA-256 hash matches. Do this before an incident, not during one.

## Troubleshoot

**Tables return empty results when you test by hand**

You're probably running `osqueryi` as your own user. Run the agent's `osqueryd` binary as root with `--augeas_lenses=/opt/orbit/lenses`, as shown in [Spot-check tables as root](#spot-check-tables-as-root), or run the query as a live report from Fleet.

**Macs or other Linux workstations fail DGX policies**

A policy is missing `labels_include_any`. Add it to that policy directly, not to the top of the `policies` list.

**The approved kernel policy fails on every Spark after an update**

NVIDIA shipped a new kernel. Add the new version to the allow-list in Git once you've approved it.

**The update script reports `none` for the driver**

The script is looking for a DKMS package. DGX OS 7 uses `nvidia-driver-<branch>-open` and a pre-built `linux-modules-nvidia-<branch>-open-*` package, so match those names.

**Script output is cut off**

Fleet keeps the last 10,000 characters. Filter output in the script, as the firmware inventory script does, or write large output to a file and carve it.

**The diagnostics bundle script times out**

Raise `script_execution_timeout` in the fleet's `agent_options`. The maximum is 18,000 seconds.

**A carve doesn't appear in `fleetctl get carves`**

The carve appears after fleetd sends the first block, which can take a while. Check the `carves` table on the host for status, and confirm the carver flags are set in `command_line_flags` and fleetd has restarted.

## Further reading

- [Frontier hardware deserves frontier device management](https://fleetdm.com/articles/frontier-hardware-deserves-frontier-device-management)
- [File carving in Fleet](https://fleetdm.com/guides/file-carving)
- [Scripts](https://fleetdm.com/guides/scripts)
- [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script)
- [Encrypt your Fleet-managed Linux device](https://fleetdm.com/guides/linux-disk-encryption-end-user)
- [Linux vulnerability detection with OVAL and Fleet](https://fleetdm.com/engineering/linux-vulnerability-detection-with-oval-and-fleet)
- [GitOps YAML files](https://fleetdm.com/docs/configuration/yaml-files)

<meta name="articleTitle" value="Manage DGX Spark with Fleet">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-09-29">
<meta name="category" value="guides">
<meta name="description" value="Enroll NVIDIA DGX Spark into your workstations fleet, then add inventory, baseline policies, diagnostics, and controlled updates with Fleet.">
