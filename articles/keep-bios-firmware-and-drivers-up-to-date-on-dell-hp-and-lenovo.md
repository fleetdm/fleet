# Keep BIOS, firmware, and drivers up to date on Dell, HP, and Lenovo

Firmware runs before Windows or Linux and before your EDR, so if there's a hole in it, every security tool you bought is sitting on top of that hole. Keeping that layer patched is a big part of why Dell, HP, and Lenovo ship their own update tools.

I took the question of what happens to BIOS updates after Intune on a long walk with Claude, and the answer was simpler than I expected. Intune's OEM integrations, like HP Connect and Dell Command | Update, come down to running the vendor's update tooling, on a schedule or when a device falls behind. Nothing about that needs Intune. Fleet does the same thing with software, scripts, and policies, on Linux as well as Windows.

![Six months of walking and running distance, trending up to a 6.7-mile daily average in November](../website/assets/images/articles/walking-and-running-distance-6-months-585x926@2x.png)

## Prerequisites

- Fleet Premium, for policy automations
- Hosts enrolled with fleetd, with [scripts enabled](https://fleetdm.com/guides/scripts)
- A fleet for your Windows and Linux PCs

## Step 1: Label your hardware

Make a dynamic label per OEM so each tool only lands on its own hardware. Navigate to **Hosts**, click **Filter by platform or label**, then **Add label**, and use these queries:

- Dell: `SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'Dell%';`
- HP: `SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'HP%' OR hardware_vendor LIKE 'Hewlett%';`
- Lenovo: `SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'Lenovo%';`

## Step 2: Add the update tools

### Dell and Lenovo

Navigate to **Software**, click **Add software**, and add the Fleet-maintained app **Dell Command Update** for Windows, targeting the **Dell** label.

Add **Lenovo System Update** the same way, targeting the **Lenovo** label. If you also want [Lenovo Commercial Vantage](https://docs.lenovocdrt.com/guides/lcv/deployment/) on your PCs, upload it as a custom package with the install command `VantageInstaller.exe Install -Vantage -SuHelper` and manage it with [Lenovo's policies](https://docs.lenovocdrt.com/guides/lcv/configuration/).

Adding a Fleet-maintained app doesn't install it. Add a policy that fails when the tool is missing, then turn on **Install software** for it in step 5:

- Dell: `SELECT 1 FROM programs WHERE name LIKE 'Dell Command %Update' AND publisher = 'Dell Inc.';`
- Lenovo: `SELECT 1 FROM programs WHERE name = 'Lenovo System Update';`

### HP

HP doesn't document a silent installer for HP Image Assistant (HPIA), so the HP script in step 3 uses the [HP Client Management Script Library](https://developers.hp.com/hp-client-management/doc/client-management-script-library) (CMSL) to install it the first time it runs.

### Linux

The OEM tools above are Windows-only. On Linux, use [fwupd](https://fwupd.github.io/libfwupdplugin/fwupdmgr.html), which pulls firmware from the [Linux Vendor Firmware Service (LVFS)](https://fwupd.org/). [Dell publishes BIOS updates to LVFS](https://www.dell.com/support/kbdoc/en-us/000122025/how-to-update-firmware-under-linux), [HP recommends fwupd](https://ftp.ext.hp.com/pub/caps-softpaq/cmit/linuxtools/HP_LinuxTools.html) over its own Linux flash tool, and Lenovo ThinkPad firmware is [listed on LVFS](https://fwupd.org/lvfs/search?value=lenovo). If a host doesn't have fwupd, install it from your distribution's packages.

## Step 3: Add the update scripts

Navigate to **Controls** > **Scripts** and upload one script per OEM.

`dell-update-bios-firmware-drivers.ps1`:

```powershell
$dcu = "${env:ProgramFiles(x86)}\Dell\CommandUpdate\dcu-cli.exe"
if (-not (Test-Path $dcu)) { $dcu = "$env:ProgramFiles\Dell\CommandUpdate\dcu-cli.exe" }
if (-not (Test-Path $dcu)) { Write-Output 'Dell Command | Update not installed'; exit 1 }

& $dcu /applyUpdates -silent -reboot=disable -autoSuspendBitLocker=enable `
  -updateType=bios,firmware,driver -outputLog=C:\Windows\Temp\dcu-apply.log
$code = $LASTEXITCODE

# 0 = success, 1 = restart required, 5 = restart pending, 500 = no updates
if ($code -in 0, 1, 5, 500) { exit 0 }
exit $code
```

`-autoSuspendBitLocker=enable` suspends BitLocker for the next restart, so a BIOS update doesn't land the user on the recovery screen. To install only security and critical updates, add `-updateSeverity=security,critical`. Dell lists the other exit codes in the [Dell Command | Update CLI reference](https://www.dell.com/support/manuals/en-us/command-update/dcu_rg/command-line-interface-error-codes?guid=guid-fbb96b06-4603-423a-baec-cbf5963d8948&lang=en-us).

`lenovo-update-bios-firmware-drivers.ps1`:

```powershell
$tvsu = "${env:ProgramFiles(x86)}\Lenovo\System Update\tvsu.exe"
if (-not (Test-Path $tvsu)) { Write-Output 'Lenovo System Update not installed'; exit 1 }
Start-Process -FilePath $tvsu -Wait -ArgumentList `
  '/CM -search A -action INSTALL -packagetypes 2,3,4 -includerebootpackages 3 -noreboot -noicon -nolicense -exporttowmi'
```

`-packagetypes 2,3,4` limits System Update to drivers, BIOS, and firmware. `-noreboot` only holds back restarts for reboot type 3, so read the [System Update command-line reference](https://docs.lenovocdrt.com/guides/sus/su_dg/su_dg_ch5/) before you add other `-includerebootpackages` values. `-exporttowmi` records what was installed in the `Lenovo_Updates` class in `root\Lenovo`.

`hp-update-bios-firmware-drivers.ps1`:

```powershell
$ErrorActionPreference = 'Stop'
$hpia = 'C:\ProgramData\HP\HPIA'

if (-not (Get-Module -ListAvailable -Name HPCMSL)) {
  Install-PackageProvider -Name NuGet -Force | Out-Null
  Set-PSRepository -Name PSGallery -InstallationPolicy Trusted
  Install-Module -Name HPCMSL -AcceptLicense
}
if (-not (Test-Path "$hpia\HPImageAssistant.exe")) {
  Import-Module HPCMSL
  Install-HPImageAssistant -Extract -DestinationPath $hpia
}

$p = Start-Process -FilePath "$hpia\HPImageAssistant.exe" -Wait -PassThru -ArgumentList `
  "/Operation:Analyze /Action:Install /Category:BIOS,Firmware,Drivers /Selection:All /Silent /ReportFolder:$hpia\Report /SoftPaqDownloadFolder:$hpia\SoftPaqs"

# 0 = success, 256 = nothing to install, 3010 = installed, restart required
if ($p.ExitCode -in 0, 256, 3010) { exit 0 }
exit $p.ExitCode
```

The rest of HPIA's switches and return codes are in the [HP Image Assistant user guide](https://ftp.hp.com/pub/caps-softpaq/cmit/whitepapers/HPIAUserGuide2.pdf).

`linux-update-firmware.sh`:

```bash
#!/bin/bash
fwupdmgr refresh
fwupdmgr update --assume-yes --no-reboot-check
rc=$?

# 2 = nothing to do
if [ "$rc" -eq 2 ]; then exit 0; fi
exit "$rc"
```

> **Note:** Fleet stops a script after 300 seconds by default. BIOS and driver downloads can take longer, so raise [`script_execution_timeout`](https://fleetdm.com/docs/configuration/agent-configuration#script-execution-timeout) in your agent options, for example to `1800`.

If your PCs have a BIOS admin password, the OEM tools need it to flash BIOS (Dell Command | Update has `-biosPassword`, and HPIA's `/BIOSPwdFile` takes a file made with HP's `HpqPswd.exe`). Keep it out of Git with a [Fleet custom variable](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles): create the variable first, then write `'$FLEET_SECRET_BIOS_PASSWORD'` (single quotes) in the script, and Fleet fills in the value before the script runs.

## Step 4: Add a policy that finds outdated BIOS

A policy passes when its query returns a row. This one fails on any PC of a given model whose BIOS is older than the version you set:

```sql
SELECT 1 FROM system_info s, platform_info p
WHERE s.hardware_model != 'YOUR-MODEL'
   OR version_compare(p.version, 'X.Y.Z') >= 0;
```

Version strings look different on each OEM, so first run `SELECT s.hardware_model, p.version, p.date FROM system_info s, platform_info p;` as a live report and copy the model and version exactly as your hosts report them. The `date` column shows which PCs have the oldest BIOS. Take the minimum version from the OEM's support page. [`platform_info`](https://fleetdm.com/tables/platform_info) works on Windows and Linux, so one query covers both.

Navigate to **Policies**, select your fleet, and click **Add policy**. Save one policy per model, targeted to the OEM's label, with the platform set to Windows or Linux.

Fleet doesn't track the latest BIOS release for each model, so this policy is only as current as the version you type in. Bump it when the OEM ships a new one. Fleet already runs in production on Dell, HP, and Lenovo hardware in healthcare, finance, and government, and this hasn't been a blocker there. If you'd like a hand, [reach out](https://fleetdm.com/support).

## Step 5: Turn on automations

1. On the **Policies** page, click **Manage automations** and select **Install software**.
2. Check the Dell and Lenovo "installed" policies, pick the matching app, and click **Save**.
3. Click **Manage automations** again and select **Run script**.
4. Check each BIOS policy, pick the matching script, and click **Save**.

Fleet [runs the automation](https://fleetdm.com/guides/policy-automation-run-script) when a host first fails the policy. If the script fails, for example because the update tool wasn't installed yet, Fleet tries again, up to three attempts in total. Updates that need a restart finish the next time the user restarts.

The BIOS policy only fires when a PC falls behind, so drivers can drift in between. To catch up, select hosts on the **Hosts** page, click **Run script**, and pick **Schedule for later**. Dell Command | Update can also check on its own schedule with `dcu-cli.exe /configure -scheduleWeekly=Mon,23:45`.

## Manage it with GitOps

Add the labels to the `labels` list you already have in `default.yml`, since GitOps can remove labels that aren't listed. Do the same with `agent_options`:

```yaml
labels:
  - name: Dell
    query: SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'Dell%';
    label_membership_type: dynamic
  - name: HP
    query: SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'HP%' OR hardware_vendor LIKE 'Hewlett%';
    label_membership_type: dynamic
  - name: Lenovo
    query: SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'Lenovo%';
    label_membership_type: dynamic
agent_options:
  script_execution_timeout: 1800
  # ...the rest of your existing agent options
```

`fleets/workstations.yml`:

```yaml
name: Workstations
controls:
  scripts:
    - path: ../lib/windows/scripts/dell-update-bios-firmware-drivers.ps1
    - path: ../lib/windows/scripts/hp-update-bios-firmware-drivers.ps1
    - path: ../lib/windows/scripts/lenovo-update-bios-firmware-drivers.ps1
    - path: ../lib/linux/scripts/linux-update-firmware.sh
software:
  fleet_maintained_apps:
    - slug: dell-command-update/windows
      labels_include_any:
        - Dell
    - slug: lenovo-system-update/windows
      labels_include_any:
        - Lenovo
policies:
  - name: Dell - Dell Command | Update is installed
    platform: windows
    query: SELECT 1 FROM programs WHERE name LIKE 'Dell Command %Update' AND publisher = 'Dell Inc.';
    labels_include_any:
      - Dell
    install_software:
      fleet_maintained_app_slug: dell-command-update/windows
  - name: Dell - BIOS is up to date
    platform: windows
    query: SELECT 1 FROM system_info s, platform_info p WHERE s.hardware_model != 'YOUR-MODEL' OR version_compare(p.version, 'X.Y.Z') >= 0;
    labels_include_any:
      - Dell
    run_script:
      path: ../lib/windows/scripts/dell-update-bios-firmware-drivers.ps1
  - name: Linux - BIOS is up to date
    platform: linux
    query: SELECT 1 FROM system_info s, platform_info p WHERE s.hardware_model != 'YOUR-MODEL' OR version_compare(p.version, 'X.Y.Z') >= 0;
    run_script:
      path: ../lib/linux/scripts/linux-update-firmware.sh
```

Add the Lenovo and HP policies the same way. HP doesn't need an install policy, since its script installs HPIA.

## Verify

1. On a host's **Host details** page, open the **Activity** tab and check the script run and its output.
2. After the host restarts, run the live report from step 4 again and confirm the new BIOS version.

## Further reading

If you're coming from Intune, these are the OEM integrations this guide replaces:

- [HP Connect for Microsoft Intune user guide](https://connect.admin.hp.com/static/HPConnectUserGuide.pdf)
- [Dell Command | Update with Microsoft Intune](https://www.dell.com/support/manuals/en-us/command-update/admx_rg/using-microsoft-intune?guid=guid-8e1f9749-35cc-46cd-82cd-bc9576b32575&lang=en-us)
- [Lenovo Commercial Vantage Policy Manager for Intune](https://docs.lenovocdrt.com/guides/lcvpm/)

For the tools themselves, see the [Lenovo System Update deployment guide](https://docs.lenovocdrt.com/guides/sus/su_dg/) and the [fwupdmgr reference](https://fwupd.github.io/libfwupdplugin/fwupdmgr.html).

None of this needs Windows Autopilot. For new PCs, see [Windows zero touch without Entra or Autopilot](https://fleetdm.com/guides/windows-zero-touch-without-entra-or-autopilot), and if you're wondering what it costs to drop Microsoft 365 E3 altogether, we [priced it out](https://fleetdm.com/articles/leaving-e3-entirely).

If you're a Fleet customer, we can help you get this set up quickly as part of your Fast Track. Either way, if you run into any trouble, let us know. We're happy to help, and there's a friendly community of kind people who are happy to help too: [get support](https://fleetdm.com/support).

<meta name="articleTitle" value="Keep BIOS, firmware, and drivers up to date on Dell, HP, and Lenovo">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-26">
<meta name="description" value="Deploy Dell Command | Update, HP Image Assistant, Lenovo System Update, and fwupd with Fleet, and run updates automatically when a policy fails.">
