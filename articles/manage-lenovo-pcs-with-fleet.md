# Manage Lenovo PCs with Fleet

I'm one of those people who enjoys poking around in BIOS setup. I realize it's not a popular hobby, though if you clicked on a guide about Lenovo BIOS settings, you might be one too. So I was a little too pleased when companies moving off Intune prompted this question around what would happen to their BIOS settings.

The BIOS is the small program that wakes a PC up before Windows or Linux does. It decides whether Secure Boot (which only lets trusted software start) is on, whether the TPM (a tiny chip that guards encryption keys) works, and what the PC can start from. Your disk encryption and EDR count on all of that, and like anything else, the BIOS needs patches. So there are two jobs, keeping the BIOS updated and keeping its settings locked down, and Lenovo makes tools for both: System Update and Commercial Vantage for updates, and the Think BIOS Config Tool for settings. Intune just runs those tools. Fleet can too, on Linux as well as Windows.

## Prerequisites

- Fleet Premium, for policy automations.
- Hosts enrolled with fleetd, with [scripts enabled](https://fleetdm.com/guides/scripts).
- If your PCs have a BIOS supervisor password, a custom variable named `BIOS_PASSWORD` in **Controls** > **Variables**. Add it before uploading the scripts below, since Fleet won't save a script that uses a variable it hasn't heard of.

> **Note:** Fleet can't yet store a different BIOS password for each host ([#59972](https://github.com/fleetdm/fleet/issues/59972)), so one [custom variable](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) holds the same password for every device. It still stays out of your scripts and Git, which matters most. Fleet is already used in production on Lenovo hardware in healthcare, finance, and government, and this hasn't been a blocker there. If you'd like to talk through your setup, [reach out](https://fleetdm.com/support).

## Step 1: Label your Lenovo PCs

A label is a group of hosts that match a query, a bit like a saved search that keeps itself up to date. Navigate to **Hosts**, click **Filter by platform or label**, then **Add label**. Name it **Lenovo** and use this query:

```sql
SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'Lenovo%';
```

## Step 2: Keep the BIOS, firmware, and drivers up to date

[Lenovo System Update](https://docs.lenovocdrt.com/guides/sus/su_dg/) is a Fleet-maintained app. Navigate to **Software**, click **Add software**, and add **Lenovo System Update** for Windows, targeting the **Lenovo** label.

Adding an app doesn't install it. For that, you need a policy (a yes-or-no question Fleet asks every host on a schedule) and a policy automation (what Fleet does when a host says "no"). Add a policy that fails when System Update is missing. Under **Target**, choose **Custom** and check **Lenovo**, so your other PCs don't fail it and get Lenovo software pushed to them:

```sql
SELECT 1 FROM programs WHERE name = 'Lenovo System Update';
```

Then, on the **Policies** page, click the policy's **Automation** column, choose **Install software**, pick **Lenovo System Update**, and click **Save**.

If you'd rather use [Lenovo Commercial Vantage](https://docs.lenovocdrt.com/guides/lcv/deployment/), upload it as a custom package with the install command `VantageInstaller.exe Install -Vantage -SuHelper` and manage it with [Lenovo's policies](https://docs.lenovocdrt.com/guides/lcv/configuration/).

Next, navigate to **Controls** > **Scripts** and upload `lenovo-update-bios-firmware-drivers.ps1`:

```powershell
$tvsu = "${env:ProgramFiles(x86)}\Lenovo\System Update\tvsu.exe"
if (-not (Test-Path $tvsu)) { Write-Output 'Lenovo System Update not installed'; exit 1 }
Start-Process -FilePath $tvsu -Wait -ArgumentList `
  '/CM -search A -action INSTALL -packagetypes 2,3,4 -includerebootpackages 3 -noreboot -noicon -nolicense'
```

`-packagetypes 2,3,4` picks drivers, BIOS, and firmware. The two reboot flags install updates that need a restart without forcing one, so nobody's laptop restarts in the middle of a meeting, and the update finishes whenever they next restart. (`-noreboot` only works for the one kind of restart this script includes; Lenovo's [command-line reference](https://docs.lenovocdrt.com/guides/sus/su_dg/su_dg_ch5/) has the details.)

To notice when a BIOS falls behind, you'll use another policy. Start by running this as a live report (a one-off query with answers right away) to see what your hosts call themselves:

```sql
SELECT s.hardware_model, p.version, p.date FROM system_info s, platform_info p;
```

Then add one policy per model, also targeted at **Lenovo**, with the model exactly as the report shows it and the minimum BIOS version from Lenovo's support page:

```sql
SELECT 1 FROM system_info s, platform_info p
WHERE s.hardware_model != 'YOUR-MODEL'
   OR version_compare(p.version, 'X.Y.Z') >= 0;
```

Fleet doesn't track Lenovo's latest BIOS for each model, so this policy is only as current as the version you type in. Bump it when Lenovo ships a new one. If keeping that number current by hand is a problem for your team, [tell us](https://fleetdm.com/support).

Finally, click each BIOS policy's **Automation** column, choose **Run script**, pick the script, and click **Save**. Fleet runs the script when a host first fails the policy and tries up to three times.

> **Note:** Fleet stops a script after 300 seconds by default, and driver downloads can take longer. Raise [`script_execution_timeout`](https://fleetdm.com/docs/configuration/agent-configuration#script-execution-timeout) in your agent options, for example to `1800`.

## Step 3: Lock down BIOS settings

ThinkPads let you change BIOS settings through [WMI](https://docs.lenovocdrt.com/ref/bios/wmi/wmi_guide/) (Windows' built-in way to read and change system settings), so there's nothing extra to install. Upload `lenovo-bios-security-settings.ps1`:

```powershell
$set = Get-WmiObject -Class Lenovo_SetBiosSetting -Namespace root\wmi
$save = Get-WmiObject -Class Lenovo_SaveBiosSettings -Namespace root\wmi
$pw = '$FLEET_SECRET_BIOS_PASSWORD'

foreach ($s in 'SecureBoot,Enable', 'SecurityChip,Enable', 'VirtualizationTechnology,Enable',
               'IPv4NetworkStack,Disable', 'IPv6NetworkStack,Disable') {
  $r = $set.SetBiosSetting($s).return
  if ($r -ne 'Success') { Write-Output "$s -> $r"; exit 1 }
}
(Get-WmiObject -Class Lenovo_WmiOpcodeInterface -Namespace root\wmi).WmiOpcodeInterface("WmiOpcodePasswordAdmin:$pw;") | Out-Null
$r = $save.SaveBiosSettings().return
Write-Output "Save -> $r"
if ($r -ne 'Success') { exit 1 }
```

Fleet swaps in the real password for `$FLEET_SECRET_BIOS_PASSWORD` before the script reaches the device. If you don't use a supervisor password, drop `$pw` and the `WmiOpcodeInterface` line. The order matters: Lenovo's WMI guide sets the values, then gives the password, then saves.

The setting names come from Lenovo's [ThinkPad BIOS settings reference](https://docs.lenovocdrt.com/ref/bios/bios_guide/). `SecurityChip` is Lenovo's name for the TPM. On AMD models, use `AmdVt` instead of `VirtualizationTechnology`, and expect different names altogether on ThinkCentre and ThinkStation PCs.

Lenovo doesn't have a single switch for USB or network boot. USB boot lives in `BootOrder`. Network (PXE) boot, which lets a PC start from a server instead of its own disk, needs the UEFI network stacks this script turns off. If your team reimages PCs over the network, that will stop working, so try the script on one PC of each model first.

To check each setting, add a policy (targeted at **Lenovo**, like the others) that reads it with [`wmi_bios_info`](https://fleetdm.com/tables/wmi_bios_info):

```sql
SELECT 1 FROM wmi_bios_info WHERE name = 'VirtualizationTechnology' AND value = 'Enable';
```

To see the exact names and values, run `SELECT name, value FROM wmi_bios_info;` as a live report on one PC of each model. Secure Boot and the TPM have their own tables:

```sql
SELECT 1 FROM secureboot WHERE secure_boot = 1;
```

```sql
SELECT 1 FROM tpm_info WHERE enabled = 1 AND activated = 1;
```

Give these policies the settings script as a **Run script** automation, like in step 2. If someone changes a setting, the policy fails the next time it runs and Fleet changes it back. Use both: the supervisor password keeps people from changing settings in the first place, and the policy tidies up after anyone who manages it anyway.

If you already built a Think BIOS Config Tool file for Intune, keep it. The tool is a PowerShell module, so a Fleet script can run it too.

## Step 4: Do the same on Linux

System Update, Commercial Vantage, and the Think BIOS Config Tool are Windows-only. On Linux:

- **Updates:** Lenovo firmware is [listed on the Linux Vendor Firmware Service](https://fwupd.org/lvfs/search?value=lenovo), and [fwupd](https://fwupd.github.io/libfwupdplugin/fwupdmgr.html) installs it.
- **Settings:** On its Linux-certified PCs from 2020 on, Lenovo supports changing BIOS settings through the kernel's `thinklmi` driver. See Lenovo's [ThinkLMI guide](https://download.lenovo.com/pccbbs/mobiles_pdf/lenovo_bios_setup_linux_wmi_sysfs_1.2.pdf).

Upload `linux-update-firmware.sh` and attach it to a Linux copy of the BIOS version policy from step 2:

```bash
#!/bin/bash
fwupdmgr refresh
fwupdmgr update --assume-yes --no-reboot-check
rc=$?

# 2 = nothing to do
if [ "$rc" -eq 2 ]; then exit 0; fi
exit "$rc"
```

`lenovo-bios-security-settings.sh` sets most of the same settings through `thinklmi`:

```bash
#!/bin/bash
set -e
base=/sys/class/firmware-attributes/thinklmi
pw='$FLEET_SECRET_BIOS_PASSWORD'
echo "$pw" > "$base/authentication/Admin/current_password"
echo Enable > "$base/attributes/SecureBoot/current_value"
echo Enable > "$base/attributes/VirtualizationTechnology/current_value"
echo Disable > "$base/attributes/IPv4NetworkStack/current_value"
echo Disable > "$base/attributes/IPv6NetworkStack/current_value"
```

Drop the two password lines if you don't use one. The driver can turn Secure Boot on but not off, which is the direction you want anyway. To check a setting, read it with `file_lines` (the `secureboot` policy from step 3 works on Linux too):

```sql
SELECT 1 FROM file_lines
WHERE path = '/sys/class/firmware-attributes/thinklmi/attributes/VirtualizationTechnology/current_value'
  AND line = 'Enable';
```

For a wider check-up, run `fwupdmgr security --json` as a script. It grades the PC's firmware protections from HSI:0 (insecure) to HSI:4 (secure) using fwupd's [Host Security ID](https://fwupd.github.io/libfwupdplugin/hsi.html).

## Treat a surprise BitLocker prompt as a clue

Keep [enforcing disk encryption](https://fleetdm.com/guides/enforce-disk-encryption), because BitLocker and a locked-down BIOS back each other up. An unexpected BitLocker recovery prompt can be the first sign that something in the startup chain changed. Before the help desk reads out a recovery key, have them check Fleet for a recent Secure Boot change, BIOS setting change, or BIOS update. If nothing explains it, investigate before unlocking.

## Manage it with GitOps

If you manage Fleet with GitOps, the label goes in `default.yml`:

```yaml
labels:
  - name: Lenovo
    query: SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'Lenovo%';
    label_membership_type: dynamic
```

Merge it into your existing `labels`, since GitOps deletes labels that aren't listed. Everything else lives in a fleet file, and each script a policy runs must also be listed in that fleet's `controls.scripts`. A fleet file for your workstations might look like this:

```yaml
name: Workstations
agent_options:
  script_execution_timeout: 1800
  # ...plus the rest of your existing agent options
controls:
  scripts:
    - path: ../lib/windows/scripts/lenovo-update-bios-firmware-drivers.ps1
    - path: ../lib/windows/scripts/lenovo-bios-security-settings.ps1
    - path: ../lib/linux/scripts/linux-update-firmware.sh
    - path: ../lib/linux/scripts/lenovo-bios-security-settings.sh
software:
  fleet_maintained_apps:
    - slug: lenovo-system-update/windows
      labels_include_any:
        - Lenovo
policies:
  - name: Lenovo - System Update is installed
    platform: windows
    query: SELECT 1 FROM programs WHERE name = 'Lenovo System Update';
    labels_include_any:
      - Lenovo
    install_software:
      fleet_maintained_app_slug: lenovo-system-update/windows
  - name: Lenovo - BIOS is up to date
    platform: windows
    query: SELECT 1 FROM system_info s, platform_info p WHERE s.hardware_model != 'YOUR-MODEL' OR version_compare(p.version, 'X.Y.Z') >= 0;
    labels_include_any:
      - Lenovo
    run_script:
      path: ../lib/windows/scripts/lenovo-update-bios-firmware-drivers.ps1
  - name: Lenovo - Virtualization is on
    platform: windows
    query: SELECT 1 FROM wmi_bios_info WHERE name = 'VirtualizationTechnology' AND value = 'Enable';
    labels_include_any:
      - Lenovo
    run_script:
      path: ../lib/windows/scripts/lenovo-bios-security-settings.ps1
  - name: Lenovo Linux - Secure Boot is on
    platform: linux
    query: SELECT 1 FROM secureboot WHERE secure_boot = 1;
    labels_include_any:
      - Lenovo
    run_script:
      path: ../lib/linux/scripts/lenovo-bios-security-settings.sh
# ...plus any other keys your fleet file already has
```

Store the BIOS password as a GitHub or GitLab secret named `FLEET_SECRET_BIOS_PASSWORD` and add it to your workflow's `env`, as described in [Custom variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles).

## Further reading

Once all this is running, nobody should need to open BIOS setup on these PCs again. That's good news for your help desk, and only mildly disappointing for people like me.

If you're coming from Intune, these Lenovo guides cover the same tools:

- [Think BIOS Config Tool](https://docs.lenovocdrt.com/guides/tbct_v2/) and its [Intune packaging guide](https://docs.lenovocdrt.com/dpm/manage/think-bios-config-tool.html)
- [Lenovo Commercial Vantage Policy Manager for Intune](https://docs.lenovocdrt.com/guides/lcvpm/)
- [Lenovo System Update deployment guide](https://docs.lenovocdrt.com/guides/sus/su_dg/)

None of this needs Windows Autopilot or Entra. For new PCs, see [Windows zero touch without Entra or Autopilot](https://fleetdm.com/guides/windows-zero-touch-without-entra-or-autopilot), and if you're wondering what it costs to drop Microsoft 365 E3 altogether, we [priced it out](https://fleetdm.com/articles/leaving-e3-entirely). Managing Dell or HP too? See [Manage Dell PCs with Fleet](https://fleetdm.com/guides/manage-dell-pcs-with-fleet) and [Manage HP PCs with Fleet](https://fleetdm.com/guides/manage-hp-pcs-with-fleet).

If you're a Fleet customer, we can help you get this set up quickly as part of your Fast Track. Either way, if you run into any trouble, let us know. We're happy to help, and there's a friendly community of kind people who are happy to help too: [get support](https://fleetdm.com/support).

<meta name="articleTitle" value="Manage Lenovo PCs with Fleet">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-27">
<meta name="description" value="Update the BIOS, lock down settings, and check firmware health on Lenovo PCs running Windows or Linux with Fleet, no Intune required.">
