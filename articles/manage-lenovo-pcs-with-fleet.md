# Manage Lenovo PCs with Fleet

I'm one of those people who enjoys poking around in BIOS setup. I realize it's not a popular hobby, though if you clicked on a guide about Lenovo BIOS settings, you might be one too. So I was a little too pleased when companies moving off Intune prompted this question around what would happen to their BIOS settings.

Secure Boot, the TPM, and the list of devices a PC will start from are what your disk encryption and EDR depend on, and the BIOS itself needs patches like anything else. Lenovo builds tools for both because they keep devices secure. In Intune, you'd usually run them with Lenovo System Update, Commercial Vantage, and the Think BIOS Config Tool. None of that needs Intune. This guide shows how to do the same with Fleet, on Linux as well as Windows.

## Prerequisites

- Fleet Premium, for policy automations.
- Hosts enrolled with fleetd, with [scripts enabled](https://fleetdm.com/guides/scripts).
- If your Lenovo PCs have a BIOS supervisor password, a custom variable named `BIOS_PASSWORD`. Add it in **Controls** > **Variables** before you upload the scripts below, because Fleet won't save a script that uses a variable that doesn't exist yet.

> **Note:** Fleet doesn't yet store a unique BIOS password for each host ([#59972](https://github.com/fleetdm/fleet/issues/59972)), so for now one [custom variable](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) holds the same password for every device. That keeps it out of your scripts and out of Git, which is the part that matters most. Fleet is already used in production on Lenovo hardware in healthcare, finance, and government, and this hasn't been a blocker there. If you'd like to talk through your setup, [reach out](https://fleetdm.com/support).

## Step 1: Label your Lenovo PCs

Navigate to **Hosts**, click **Filter by platform or label**, then **Add label**. Name it **Lenovo** and use this query:

```sql
SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'Lenovo%';
```

Every policy and app below targets this label.

## Step 2: Keep the BIOS, firmware, and drivers up to date

[Lenovo System Update](https://docs.lenovocdrt.com/guides/sus/su_dg/) is a Fleet-maintained app. Navigate to **Software**, click **Add software**, and add **Lenovo System Update** for Windows, targeting the **Lenovo** label.

Adding an app doesn't install it, so add a policy that fails when it's missing:

```sql
SELECT 1 FROM programs WHERE name = 'Lenovo System Update';
```

Then, on the **Policies** page, click **Manage automations**, select **Install software**, check the policy, pick **Lenovo System Update**, and click **Save**.

If you'd rather use [Lenovo Commercial Vantage](https://docs.lenovocdrt.com/guides/lcv/deployment/), upload it as a custom package with the install command `VantageInstaller.exe Install -Vantage -SuHelper` and manage it with [Lenovo's policies](https://docs.lenovocdrt.com/guides/lcv/configuration/).

Next, navigate to **Controls** > **Scripts** and upload `lenovo-update-bios-firmware-drivers.ps1`:

```powershell
$tvsu = "${env:ProgramFiles(x86)}\Lenovo\System Update\tvsu.exe"
if (-not (Test-Path $tvsu)) { Write-Output 'Lenovo System Update not installed'; exit 1 }
Start-Process -FilePath $tvsu -Wait -ArgumentList `
  '/CM -search A -action INSTALL -packagetypes 2,3,4 -includerebootpackages 3 -noreboot -noicon -nolicense -exporttowmi'
```

`-packagetypes 2,3,4` limits System Update to drivers, BIOS, and firmware. `-noreboot` only holds back restarts for packages Lenovo marks as reboot type 3, so read the [command-line reference](https://docs.lenovocdrt.com/guides/sus/su_dg/su_dg_ch5/) before you add other `-includerebootpackages` values. `-exporttowmi` records what was installed in the `Lenovo_Updates` class in `root\Lenovo`.

Now add a policy that fails when a model's BIOS falls behind. Run this live report first to see what your hosts report:

```sql
SELECT s.hardware_model, p.version, p.date FROM system_info s, platform_info p;
```

Then add one policy per model, copying the model name and the minimum version from Lenovo's support page exactly as they appear:

```sql
SELECT 1 FROM system_info s, platform_info p
WHERE s.hardware_model != 'YOUR-MODEL'
   OR version_compare(p.version, 'X.Y.Z') >= 0;
```

Fleet doesn't track Lenovo's latest BIOS for each model, so this policy is only as current as the version you type in. Bump it when Lenovo ships a new one. Fleet is already used in production on Lenovo hardware in regulated industries, and this hasn't been a blocker there. If you'd like a hand, [reach out](https://fleetdm.com/support).

Finally, click **Manage automations** on the **Policies** page, select **Run script**, check the BIOS policies, pick the script, and click **Save**. Fleet runs the script when a host first fails the policy and retries it up to three times. Updates that need a restart finish when the user next restarts.

> **Note:** Fleet stops a script after 300 seconds by default, and driver downloads can take longer. Raise [`script_execution_timeout`](https://fleetdm.com/docs/configuration/agent-configuration#script-execution-timeout) in your agent options, for example to `1800`.

## Step 3: Lock down BIOS settings

ThinkPads expose BIOS settings through [WMI](https://docs.lenovocdrt.com/ref/bios/wmi/wmi_guide/), so you don't need to install anything. Upload `lenovo-bios-security-settings.ps1`:

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

Before the script reaches the device, Fleet swaps `$FLEET_SECRET_BIOS_PASSWORD` for the variable's value. If your PCs don't have a supervisor password, drop `$pw` and the `WmiOpcodeInterface` line. The order matters: Lenovo's [WMI guide](https://docs.lenovocdrt.com/ref/bios/wmi/wmi_guide/) sets the values, then gives the password, then saves.

The setting names come from Lenovo's [ThinkPad BIOS settings reference](https://docs.lenovocdrt.com/ref/bios/bios_guide/). Use `AmdVt` instead of `VirtualizationTechnology` on AMD models. ThinkCentre and ThinkStation PCs use different names. Lenovo doesn't have a single switch for USB or network boot. The script turns off the UEFI IPv4 and IPv6 network stacks, which network (PXE) boot runs on, so test-boot one model before you roll it out. USB boot is controlled through `BootOrder`.

To check each setting, add a policy. [`wmi_bios_info`](https://fleetdm.com/tables/wmi_bios_info) reads Lenovo's BIOS settings by name:

```sql
SELECT 1 FROM wmi_bios_info WHERE name = 'VirtualizationTechnology' AND value = 'Enable';
```

Run `SELECT name, value FROM wmi_bios_info;` as a live report on one PC of each model to see the exact names. Secure Boot and the TPM have their own tables:

```sql
SELECT 1 FROM secureboot WHERE secure_boot = 1;
```

```sql
SELECT 1 FROM tpm_info WHERE enabled = 1 AND activated = 1;
```

Turn on **Run script** for these policies with the settings script, the same way as in step 2. If someone changes a setting, the policy fails at the next check-in and Fleet puts it back. The supervisor password is what stops them from changing it in the first place, so use both.

If you already have a Think BIOS Config Tool file from Intune, Lenovo's [Think BIOS Config Tool](https://docs.lenovocdrt.com/guides/tbct_v2/) module is PowerShell too, so you can run it from a Fleet script instead.

## Step 4: Do the same on Linux

System Update, Commercial Vantage, and the Think BIOS Config Tool are Windows-only. On Linux:

- **Updates:** ThinkPad firmware is [listed on the Linux Vendor Firmware Service](https://fwupd.org/lvfs/search?value=lenovo), and [fwupd](https://fwupd.github.io/libfwupdplugin/fwupdmgr.html) installs it.
- **Settings:** Lenovo supports changing BIOS settings through the kernel's `thinklmi` driver on its Linux-certified PCs from 2020 on. See Lenovo's [ThinkLMI guide](https://download.lenovo.com/pccbbs/mobiles_pdf/lenovo_bios_setup_linux_wmi_sysfs_1.2.pdf).

Upload `linux-update-firmware.sh`, add a Linux copy of the BIOS version policy from step 2, and attach the script to it:

```bash
#!/bin/bash
fwupdmgr refresh
fwupdmgr update --assume-yes --no-reboot-check
rc=$?

# 2 = nothing to do
if [ "$rc" -eq 2 ]; then exit 0; fi
exit "$rc"
```

`lenovo-bios-security-settings.sh` sets the same settings through `thinklmi`:

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

To check a setting, read the driver with `file_lines`. The `secureboot` policy from step 3 works on Linux too.

```sql
SELECT 1 FROM file_lines
WHERE path = '/sys/class/firmware-attributes/thinklmi/attributes/VirtualizationTechnology/current_value'
  AND line = 'Enable';
```

For a wider check of firmware protections, run `fwupdmgr security --json` as a script. It scores the PC from HSI:0 (insecure) to HSI:4 (secure) using fwupd's [Host Security ID](https://fwupd.github.io/libfwupdplugin/hsi.html).

## Treat a surprise BitLocker prompt as a signal

None of this replaces disk encryption, so [keep enforcing it](https://fleetdm.com/guides/enforce-disk-encryption). The two also back each other up: a BitLocker recovery prompt nobody expected can be the first sign that something in the boot chain changed. Before the help desk reads out a recovery key, have them look the host up in Fleet. Did Secure Boot turn off, did a BIOS setting change, or was there a BIOS update? If nothing explains it, investigate before unlocking.

## Manage it with GitOps

`default.yml`:

```yaml
labels:
  - name: Lenovo
    query: SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'Lenovo%';
    label_membership_type: dynamic
agent_options:
  script_execution_timeout: 1800
  # ...the rest of your existing agent options
```

Add these to the `labels` and `agent_options` you already have, since GitOps removes labels that aren't listed. Policy automations live in a fleet file, and each script must also be listed in that fleet's `controls.scripts`. `fleets/workstations.yml`:

```yaml
name: Workstations
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
```

Store the BIOS password as a GitHub or GitLab secret named `FLEET_SECRET_BIOS_PASSWORD`, as described in [Secrets in scripts and configuration profiles](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles).

## Further reading

If you're coming from Intune, these are the Lenovo tools this guide replaces:

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
