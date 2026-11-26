# Configure BIOS settings on Dell, HP, and Lenovo with Fleet scripts

The first time I opened a BIOS setup screen, it was blue and gray, and it felt like I'd walked into the boiler room of the computer. Nobody was supposed to be down there. That's still mostly true, which is exactly why the settings in that room matter: Secure Boot, which devices the PC can start from, and whether the TPM is on all decide whether your disk encryption and EDR can trust what booted.

Intune customers manage these settings with HP Connect, Dell Command | Endpoint Configure, and Lenovo's Think BIOS Config Tool. Each one runs the OEM's command-line tool or PowerShell library through a script or remediation. Fleet runs the same tools with scripts, checks the result with policies, and puts settings back with policy automations when they drift.

## Prerequisites

- Fleet Premium, for policy automations.
- `fleetd` with scripts enabled.
- The OEM tool for your hardware:
  - **Dell:** [Dell Command | Configure](https://www.dell.com/support/manuals/en-us/command-configure/dcc_5.x_ref_guide/security-profile?guid=guid-66ba0e99-ad94-42f0-a162-00a673fd138e) (`cctk`) for Windows and Linux. Upload it to Fleet as a custom package.
  - **HP:** [HP Client Management Script Library](https://developers.hp.com/hp-client-management/doc/client-management-script-library) (CMSL). The scripts below install it from the PowerShell Gallery if it's missing.
  - **Lenovo:** nothing. ThinkPad BIOS settings are exposed through [WMI](https://docs.lenovocdrt.com/ref/bios/wmi/wmi_guide/) on Windows and through the kernel's `thinklmi` driver on Linux.

> **Warning:** If a device has a BIOS admin password, every script that changes a setting needs it. Fleet doesn't yet store a unique BIOS password for each host ([#59972](https://github.com/fleetdm/fleet/issues/59972)). Don't hard-code a real password in a script. Store it as a [custom variable](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) (`$FLEET_SECRET_BIOS_PASSWORD`) so it stays out of the script text and the Fleet UI. Fleet is already used in production with Dell, HP, and Lenovo hardware in healthcare, finance, and government, and this isn't a blocker there. If you have questions about how to handle it in your environment, [reach out](https://fleetdm.com/support). We're happy to help.

## Step 1: Find each setting's exact name

Setting names differ by OEM and sometimes by model. Before you write a script, run this live report on one device from each model:

```sql
SELECT name, value FROM wmi_bios_info;
```

On Windows, [`wmi_bios_info`](https://fleetdm.com/tables/wmi_bios_info) reads the OEM's own BIOS settings on Dell, HP, and Lenovo hardware. On Linux, list the settings with `ls /sys/class/firmware-attributes/*/attributes/`. The Dell, HP, and Lenovo kernel drivers ([`dell-wmi-sysman`, `hp-bioscfg`, and `thinklmi`](https://kernel.org/doc/Documentation/ABI/testing/sysfs-class-firmware-attributes)) all use that path.

## Step 2: Write the scripts

Navigate to **Controls** > **Scripts** and upload a script for each OEM. These turn on Secure Boot, the TPM, and virtualization, and turn off network (PXE) boot.

### Dell

`dell-bios-security-settings.ps1`:

```powershell
$cctk = "${env:ProgramFiles(x86)}\Dell\Command Configure\X86_64\cctk.exe"
$pw = "$env:FLEET_SECRET_BIOS_PASSWORD"
$auth = if ($pw) { "--ValSetupPwd=$pw" } else { $null }

& $cctk --SecureBoot=Enabled --TpmSecurity=Enabled --TpmActivation=Enabled `
  --Virtualization=Enabled --VtForDirectIo=Enabled --UefiNwStack=Disabled $auth
exit $LASTEXITCODE
```

`dell-bios-security-settings.sh` for Linux:

```bash
#!/bin/bash
auth=""
[ -n "$FLEET_SECRET_BIOS_PASSWORD" ] && auth="--ValSetupPwd=$FLEET_SECRET_BIOS_PASSWORD"
/opt/dell/dcc/cctk --SecureBoot=Enabled --TpmSecurity=Enabled --TpmActivation=Enabled \
  --Virtualization=Enabled --VtForDirectIo=Enabled --UefiNwStack=Disabled $auth
```

Dell Command | Configure has no single "boot from USB" switch. To stop someone from picking a USB stick in the F12 boot menu, set a BIOS admin password and `--UefiBootPathSecurity=AlwaysExceptInternalHdd`, which asks for that password before booting anything but the internal drive. Dell's [security profile](https://www.dell.com/support/manuals/en-us/command-configure/dcc_5.x_ref_guide/security-profile?guid=guid-66ba0e99-ad94-42f0-a162-00a673fd138e) lists recommended values for all of these.

### HP

`hp-bios-security-settings.ps1`:

```powershell
$ErrorActionPreference = 'Stop'
if (-not (Get-Module -ListAvailable -Name HPCMSL)) {
  Install-PackageProvider -Name NuGet -Force | Out-Null
  Install-Module -Name HPCMSL -Force -AcceptLicense
}
Import-Module HPCMSL

$settings = [ordered]@{
  'Secure Boot'                     = 'Enable'
  'TPM State'                       = 'Enable'
  'Virtualization Technology (VTx)' = 'Enable'
  'Network (PXE) Boot'              = 'Disable'
  'USB Storage Boot'                = 'Disable'
}
foreach ($name in $settings.Keys) {
  Set-HPBIOSSettingValue -Name $name -Value $settings[$name] -Password "$env:FLEET_SECRET_BIOS_PASSWORD"
}
```

HP's setting names come from the [HP BIOS administration guide](https://kaas.hpcloud.hp.com/pdf-public/pdf_12904670_en-US-1.pdf), and they're case sensitive. `Virtualization Technology (VTx)` applies to Intel models. On Linux, HP's [Linux tools](https://ftp.ext.hp.com/pub/caps-softpaq/cmit/linuxtools/HP_LinuxTools.html) (`hp-repsetup`) are experimental and need Secure Boot turned off, so use the `hp-bioscfg` kernel driver to read settings instead (see Step 3).

### Lenovo

`lenovo-bios-security-settings.ps1`:

```powershell
$set = Get-WmiObject -Class Lenovo_SetBiosSetting -Namespace root\wmi
$save = Get-WmiObject -Class Lenovo_SaveBiosSettings -Namespace root\wmi
$pw = "$env:FLEET_SECRET_BIOS_PASSWORD"

foreach ($s in 'SecureBoot,Enable', 'SecurityChip,Enable', 'VirtualizationTechnology,Enable', 'IPv4NetworkStack,Disable', 'IPv6NetworkStack,Disable') {
  $r = $set.SetBiosSetting($s).return
  if ($r -ne 'Success') { Write-Output "$s -> $r"; exit 1 }
}
if ($pw) {
  (Get-WmiObject -Class Lenovo_WmiOpcodeInterface -Namespace root\wmi).WmiOpcodeInterface("WmiOpcodePasswordAdmin:$pw;") | Out-Null
}
$r = $save.SaveBiosSettings().return
Write-Output "Save -> $r"
if ($r -ne 'Success') { exit 1 }
```

The setting names come from Lenovo's [ThinkPad BIOS settings reference](https://docs.lenovocdrt.com/ref/bios/bios_guide/). Use `AmdVt` instead of `VirtualizationTechnology` on AMD models. Lenovo has no single USB boot switch either; USB boot is controlled through `BootOrder`.

`lenovo-bios-security-settings.sh` for Linux, using Lenovo's [ThinkLMI guide](https://download.lenovo.com/pccbbs/mobiles_pdf/lenovo_bios_setup_linux_wmi_sysfs_1.2.pdf):

```bash
#!/bin/bash
base=/sys/class/firmware-attributes/thinklmi
if [ -n "$FLEET_SECRET_BIOS_PASSWORD" ]; then
  echo "$FLEET_SECRET_BIOS_PASSWORD" > "$base/authentication/Admin/current_password"
fi
echo Enable > "$base/attributes/SecureBoot/current_value"
echo Enable > "$base/attributes/VirtualizationTechnology/current_value"
echo Disable > "$base/attributes/IPv4NetworkStack/current_value"
echo Disable > "$base/attributes/IPv6NetworkStack/current_value"
```

Changes take effect at the next restart.

## Step 3: Verify each setting with a policy

A policy passes when its query returns a row. On Windows, check the setting directly with `wmi_bios_info`. Use the names you found in Step 1:

```sql
-- Lenovo, Windows
SELECT 1 FROM wmi_bios_info WHERE name = 'VirtualizationTechnology' AND value = 'Enable';
```

```sql
-- HP, Windows
SELECT 1 FROM wmi_bios_info WHERE name = 'Network (PXE) Boot' AND value = 'Disable';
```

For Secure Boot, use the cross-platform [`secureboot`](https://fleetdm.com/tables/secureboot) table. It works on Windows and Linux:

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

Swap `thinklmi` for `dell-wmi-sysman` or `hp-bioscfg` on Dell and HP hardware, and use the names from Step 1.

## Step 4: Reapply settings when they drift

1. Navigate to **Policies**, click **Manage automations**, and select **Run script**.
2. Check each BIOS policy and pick the OEM's script.
3. Click **Save**.

If someone turns a setting off, the policy fails at the next check-in and Fleet runs the script again. A BIOS admin password is what stops them from turning it off in the first place, so use both.

## Manage it with GitOps

`fleets/workstations.yml`:

```yaml
controls:
  scripts:
    - path: ../lib/windows/scripts/lenovo-bios-security-settings.ps1
    - path: ../lib/linux/scripts/lenovo-bios-security-settings.sh
policies:
  - name: Lenovo - Virtualization is on
    platform: windows
    query: SELECT 1 FROM wmi_bios_info WHERE name = 'VirtualizationTechnology' AND value = 'Enable';
    labels_include_any:
      - Lenovo
    run_script:
      path: ../lib/windows/scripts/lenovo-bios-security-settings.ps1
  - name: Linux - Secure Boot is on
    platform: linux
    query: SELECT 1 FROM secureboot WHERE secure_boot = 1;
    labels_include_any:
      - Lenovo
    run_script:
      path: ../lib/linux/scripts/lenovo-bios-security-settings.sh
```

Add the BIOS password as a GitHub or GitLab secret named `FLEET_SECRET_BIOS_PASSWORD`. Learn how in [Secrets in scripts and configuration profiles](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles). For the `Lenovo` label, see [Keep BIOS, firmware, and drivers up to date on Dell, HP, and Lenovo](https://fleetdm.com/guides/keep-bios-firmware-and-drivers-up-to-date-on-dell-hp-and-lenovo).

## Further reading

If you're coming from Intune, these are the tools this guide mirrors:

- [Dell Command | Endpoint Configure for Microsoft Intune](https://www.dell.com/support/kbdoc/en-us/000214308/dell-command-endpoint-configure-for-microsoft-intune)
- [HP Connect for Microsoft Intune user guide](https://connect.admin.hp.com/static/HPConnectUserGuide.pdf)
- [Lenovo Think BIOS Config Tool](https://docs.lenovocdrt.com/guides/tbct_v2/)
- [Lenovo BIOS WMI guide](https://docs.lenovocdrt.com/ref/bios/wmi/wmi_guide/)

None of this needs Windows Autopilot or Entra. See [Windows zero touch without Entra or Autopilot](https://fleetdm.com/guides/windows-zero-touch-without-entra-or-autopilot) and [Leaving E3 entirely](https://fleetdm.com/articles/leaving-e3-entirely).

If you're a Fleet customer, we can help you get this set up quickly as part of your Fast Track. Either way, if you run into any trouble, let us know. We're happy to help, and there's a friendly community of kind people who are happy to help too: [get support](https://fleetdm.com/support).

<meta name="articleTitle" value="Configure BIOS settings on Dell, HP, and Lenovo with Fleet scripts">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-26">
<meta name="description" value="Turn on Secure Boot, the TPM, and virtualization, and turn off network boot on Dell, HP, and Lenovo PCs with Fleet scripts and policies.">
