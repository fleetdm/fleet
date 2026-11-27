# Configure BIOS settings on Dell, HP, and Lenovo with Fleet scripts

I'm one of those people who enjoys poking around in BIOS setup. I realize it's not a popular hobby, though if you clicked on a guide about BIOS settings, you might be one too. So I was a little too pleased when companies moving off Intune prompted this question around what would happen to their BIOS settings. Secure Boot, the TPM, and the list of devices a PC will start from are what your disk encryption and EDR quietly depend on. Dell, HP, and Lenovo build management tools for these settings because they keep devices secure, and leaving Intune doesn't mean losing that control.

In Intune, you'd usually manage them with Dell Command | Endpoint Configure, HP Connect, or Lenovo's Think BIOS Config Tool. HP's and Lenovo's tools both run as Intune remediation scripts. Fleet does the same job with a script, checks the result with a policy, and runs the script again when a setting drifts. This guide turns on Secure Boot, the TPM, and virtualization, and locks down network and USB boot, on Windows and Linux.

## Prerequisites

- Fleet Premium, for policy automations.
- fleetd with scripts enabled.
- The OEM tool for your hardware:
  - Dell: [Dell Command | Configure](https://www.dell.com/support/manuals/en-us/command-configure/dcc_5.x_ref_guide/) (`cctk`), for Windows and [Linux](https://www.dell.com/support/manuals/en-us/command-configure/dcc_5.x_ig/installing-dell-command-configure-on-linux-systems?guid=guid-9d3f8b53-abb4-4d54-9049-b1e0bfbfc100&lang=en-us). Upload it to Fleet as a custom package.
  - HP: the [HP Client Management Script Library](https://developers.hp.com/hp-client-management/doc/client-management-script-library) (CMSL), which is Windows-only. The script below installs it from the PowerShell Gallery if it's missing.
  - Lenovo: nothing extra. ThinkPad BIOS settings are exposed through [WMI](https://docs.lenovocdrt.com/ref/bios/wmi/wmi_guide/) on Windows and the kernel's `thinklmi` driver on Linux.
- A custom variable named `BIOS_PASSWORD`, if your devices have a BIOS admin password. Add it in **Controls** > **Variables** before you upload the scripts, which use it as `$FLEET_SECRET_BIOS_PASSWORD`. Fleet won't save a script that references a variable that doesn't exist yet.

> **Warning:** If a device has a BIOS admin password, every script that changes a setting needs it. Fleet doesn't yet store a unique BIOS password for each host ([#59972](https://github.com/fleetdm/fleet/issues/59972)), so for now one [custom variable](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) holds the same password for every device. Don't hard-code a real password in a script. Fleet is already used in production on Dell, HP, and Lenovo hardware in healthcare, finance, and government, and this hasn't been a blocker there. If you want to talk through your setup, [reach out](https://fleetdm.com/support).

## Step 1: Find each setting's exact name

Under the branding, Dell, HP, and Lenovo all store BIOS settings the same way, as a list of names and values. What differs is where you read them and what they're called, and the names can change even between models from the same OEM. On Windows, run this live report on one device of each model:

```sql
SELECT name, value FROM wmi_bios_info;
```

[`wmi_bios_info`](https://fleetdm.com/tables/wmi_bios_info) reads each OEM's own WMI provider on Dell, HP, and Lenovo hardware. On Linux, the Dell, HP, and Lenovo kernel drivers ([`dell-wmi-sysman`, `hp-bioscfg`, and `thinklmi`](https://kernel.org/doc/Documentation/ABI/testing/sysfs-class-firmware-attributes)) give each setting its own folder, so list the folders instead:

```sql
SELECT path FROM file WHERE path LIKE '/sys/class/firmware-attributes/%/attributes/%';
```

## Step 2: Upload the scripts

Navigate to **Controls** > **Scripts** and upload a script for each OEM. Before a script reaches the device, Fleet swaps `$FLEET_SECRET_BIOS_PASSWORD` for the variable's value. The single quotes around it keep special characters intact, though a password that contains a single quote will break them. If your devices don't have a BIOS admin password, remove `$pw` and every line or argument that uses it.

### Dell

`dell-bios-security-settings.ps1`:

```powershell
$cctk = "${env:ProgramFiles(x86)}\Dell\Command Configure\X86_64\cctk.exe"
$pw = '$FLEET_SECRET_BIOS_PASSWORD'

& $cctk --SecureBoot=Enabled --TpmSecurity=Enabled --TpmActivation=Enabled `
  --Virtualization=Enabled --VtForDirectIo=Enabled --UefiNwStack=Disabled `
  --UefiBootPathSecurity=AlwaysExceptInternalHdd "--ValSetupPwd=$pw"
exit $LASTEXITCODE
```

`dell-bios-security-settings.sh`:

```bash
#!/bin/bash
pw='$FLEET_SECRET_BIOS_PASSWORD'
/opt/dell/dcc/cctk --SecureBoot=Enabled --TpmSecurity=Enabled --TpmActivation=Enabled \
  --Virtualization=Enabled --VtForDirectIo=Enabled --UefiNwStack=Disabled \
  --UefiBootPathSecurity=AlwaysExceptInternalHdd "--ValSetupPwd=$pw"
```

Dell doesn't have a single switch for USB boot. Instead, `--UefiBootPathSecurity=AlwaysExceptInternalHdd` asks for the BIOS admin password before the PC boots from anything but its internal drive, so nobody can pick a USB stick from the boot menu without it. `--UefiNwStack=Disabled` turns off UEFI network boot. Dell's [security profile](https://www.dell.com/support/manuals/en-us/command-configure/dcc_5.x_ref_guide/security-profile?guid=guid-66ba0e99-ad94-42f0-a162-00a673fd138e) lists recommended values for the rest.

### HP

`hp-bios-security-settings.ps1`:

```powershell
$ErrorActionPreference = 'Stop'
if (-not (Get-Module -ListAvailable -Name HPCMSL)) {
  Install-PackageProvider -Name NuGet -Force | Out-Null
  Set-PSRepository -Name PSGallery -InstallationPolicy Trusted
  Install-Module -Name HPCMSL -AcceptLicense
}
Import-Module HPCMSL
$pw = '$FLEET_SECRET_BIOS_PASSWORD'

$settings = [ordered]@{
  'Secure Boot'                     = 'Enable'
  'TPM State'                       = 'Enable'
  'Virtualization Technology (VTx)' = 'Enable'
  'Network (PXE) Boot'              = 'Disable'
  'USB Storage Boot'                = 'Disable'
}
foreach ($name in $settings.Keys) {
  Set-HPBIOSSettingValue -Name $name -Value $settings[$name] -Password $pw
}
```

HP's setting names come from the [HP BIOS administration guide](https://kaas.hpcloud.hp.com/pdf-public/pdf_12904670_en-US-1.pdf), and they're case sensitive. `Virtualization Technology (VTx)` is for Intel models. Compare the values against your Step 1 report before you roll this out.

CMSL doesn't run on Linux, and HP's [Linux tools](https://ftp.ext.hp.com/pub/caps-softpaq/cmit/linuxtools/HP_LinuxTools.html) are experimental and need Secure Boot turned off unless you sign their kernel module. The `hp-bioscfg` kernel driver uses the same sysfs interface as the Lenovo Linux script below, with HP's setting names as folder names (for example, `attributes/Secure Boot`), so you can adapt that script for HP and check the result with a policy (Step 3). Test it on one device first. Fleet is already used in production with HP hardware in healthcare, finance, and government, and this hasn't been a blocker there. If it matters for your fleet, [let us know](https://fleetdm.com/support).

### Lenovo

`lenovo-bios-security-settings.ps1`:

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

The names come from Lenovo's [ThinkPad BIOS settings reference](https://docs.lenovocdrt.com/ref/bios/bios_guide/). Use `AmdVt` instead of `VirtualizationTechnology` on AMD models. Lenovo doesn't have a USB boot or PXE boot switch either. The script turns off the UEFI IPv4 and IPv6 network stacks, which Lenovo's UEFI PXE boot runs on, and USB boot is controlled through `BootOrder`. Confirm with a test boot on one model before you roll it out.

`lenovo-bios-security-settings.sh`, based on Lenovo's [ThinkLMI guide](https://download.lenovo.com/pccbbs/mobiles_pdf/lenovo_bios_setup_linux_wmi_sysfs_1.2.pdf):

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

BIOS changes generally take effect at the next restart.

## Step 3: Verify each setting with a policy

Navigate to **Policies**, select your fleet, and add one policy per setting. A policy passes when its query returns a row. On Windows, read the setting with `wmi_bios_info`, using the names and values from Step 1. Each of these examples is its own policy:

```sql
-- Lenovo
SELECT 1 FROM wmi_bios_info WHERE name = 'VirtualizationTechnology' AND value = 'Enable';
-- HP
SELECT 1 FROM wmi_bios_info WHERE name = 'Network (PXE) Boot' AND value = 'Disable';
-- Dell
SELECT 1 FROM wmi_bios_info WHERE name = 'UefiNwStack' AND value = 'Disabled';
```

On most Dell models from 2018 on, the names match the `cctk` options. Older Dell PCs with Dell Command | Monitor report different names and values, so check your Step 1 report.

Secure Boot has its own table, [`secureboot`](https://fleetdm.com/tables/secureboot), which works on Windows and Linux:

```sql
SELECT 1 FROM secureboot WHERE secure_boot = 1;
```

For the TPM on Windows, use [`tpm_info`](https://fleetdm.com/tables/tpm_info):

```sql
SELECT 1 FROM tpm_info WHERE enabled = 1 AND activated = 1;
```

On Linux, read the kernel's firmware attributes with `file_lines`:

```sql
SELECT 1 FROM file_lines
WHERE path = '/sys/class/firmware-attributes/thinklmi/attributes/VirtualizationTechnology/current_value'
  AND line = 'Enable';
```

On Dell and HP, swap `thinklmi` for `dell-wmi-sysman` or `hp-bioscfg` and use the names from Step 1.

## Step 4: Reapply settings when they drift

1. Navigate to **Policies** and select your fleet.
2. Click **Manage automations** and select **Run script**.
3. Check each BIOS policy, pick the matching script, and click **Save**.

Fleet runs the script when a policy starts failing. The policy may keep failing until the device restarts. Automations fix drift after the fact, and a BIOS admin password keeps people from changing settings in the first place, so use both.

## Manage it with GitOps

Policy automations live in a fleet file, and the script has to be listed in that fleet's `controls.scripts` too. `fleets/workstations.yml`:

```yaml
controls:
  scripts:
    - path: ../lib/windows/scripts/lenovo-bios-security-settings.ps1
    - path: ../lib/linux/scripts/lenovo-bios-security-settings.sh
policies:
  - name: Lenovo Windows - Virtualization is on
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

Add a policy for each remaining setting and OEM the same way. Store the password as a GitHub or GitLab secret and expose it to your workflow as `FLEET_SECRET_BIOS_PASSWORD`, as described in [Secrets in scripts and configuration profiles](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles). Define the `Lenovo` label in `default.yml`, as shown in [Keep BIOS, firmware, and drivers up to date on Dell, HP, and Lenovo](https://fleetdm.com/guides/keep-bios-firmware-and-drivers-up-to-date-on-dell-hp-and-lenovo).

## Further reading

If you're coming from Intune, these are the integrations this guide stands in for:

- [Dell Command | Endpoint Configure for Microsoft Intune](https://www.dell.com/support/kbdoc/en-us/000214308/dell-command-endpoint-configure-for-microsoft-intune)
- [HP Connect for Microsoft Intune user guide](https://connect.admin.hp.com/static/HPConnectUserGuide.pdf)
- [Lenovo Think BIOS Config Tool](https://docs.lenovocdrt.com/guides/tbct_v2/)

None of this needs Windows Autopilot or Entra. See [Windows zero touch without Entra or Autopilot](https://fleetdm.com/guides/windows-zero-touch-without-entra-or-autopilot) and [Leaving E3 entirely](https://fleetdm.com/articles/leaving-e3-entirely).

If you're a Fleet customer, we can help you get this set up quickly as part of your Fast Track. Either way, if you run into any trouble, let us know. We're happy to help, and there's a friendly community of kind people who are happy to help too: [get support](https://fleetdm.com/support).

<meta name="articleTitle" value="Configure BIOS settings on Dell, HP, and Lenovo with Fleet scripts">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-26">
<meta name="description" value="Turn on Secure Boot, the TPM, and virtualization, and lock down network and USB boot on Dell, HP, and Lenovo PCs with Fleet scripts and policies.">
