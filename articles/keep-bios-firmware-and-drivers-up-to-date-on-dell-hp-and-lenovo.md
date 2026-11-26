# Keep BIOS, firmware, and drivers up to date on Dell, HP, and Lenovo

I have a soft spot for the BIOS update. It's the least glamorous patch in IT, and it runs before anything else on the computer, including every security tool you paid for. If the firmware has a hole in it, the rest of the stack is standing on that hole.

That's why Dell, HP, and Lenovo each ship their own update tools, and why Intune customers wire them up with HP Connect, Dell Command | Update, and Lenovo's deployment guides. Under the hood, those integrations deploy the OEM's tool as an app, run it with a script, and use remediations to run it again when a device falls behind. Fleet does the same thing with custom software, scripts, policies, and policy automations, on Windows and on Linux.

## Prerequisites

- Fleet Premium. Policy automations that run scripts or install software are Premium features.
- `fleetd` with scripts enabled. If you use Fleet's MDM features, scripts are on by default.
- A fleet for your Windows and Linux PCs.

## Step 1: Label your hardware

Make three dynamic labels so each OEM's tool only lands on its own hardware. Navigate to **Hosts**, click **Filter by platform or label**, then **Add label**, and use these queries:

- **Dell:** `SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'Dell%';`
- **HP:** `SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'HP%' OR hardware_vendor LIKE 'Hewlett%';`
- **Lenovo:** `SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'Lenovo%';`

## Step 2: Install the update tool

**Dell.** [Dell Command | Update](https://www.dell.com/support/manuals/en-us/command-update/dcu_rg/dell-command-update-cli-commands?guid=guid-92619086-5f7c-4a05-bce2-0d560c15e8ed&lang=en-us) is a Fleet-maintained app. Navigate to **Software**, click **Add software**, select **Fleet-maintained**, and add **Dell Command Update**. Target the **Dell** label.

**Lenovo.** Lenovo System Update is a Fleet-maintained app too. Add **Lenovo System Update** the same way and target the **Lenovo** label. If you standardize on [Lenovo Commercial Vantage](https://docs.lenovocdrt.com/guides/lcv/deployment/) instead, upload it as a custom package and use Lenovo's documented install command, `VantageInstaller.exe Install -Vantage -SuHelper`.

**HP.** HP doesn't document silent switches for the HP Image Assistant (HPIA) download. HP's [Client Management Script Library](https://developers.hp.com/hp-client-management/doc/client-management-script-library) (CMSL) has a cmdlet that installs it for you, so the HP script in the next step installs CMSL and HPIA the first time it runs.

**Linux.** Use [fwupd](https://fwupd.github.io/libfwupdplugin/fwupdmgr.html), which downloads firmware from the [Linux Vendor Firmware Service (LVFS)](https://fwupd.org/). Dell, HP, and Lenovo all publish firmware there. [Dell](https://www.dell.com/support/kbdoc/en-us/000122025/how-to-update-firmware-under-linux) publishes BIOS updates to LVFS, and [HP recommends fwupd](https://ftp.ext.hp.com/pub/caps-softpaq/cmit/linuxtools/HP_LinuxTools.html) over its own Linux flash tool for capsule updates. The OEM update tools above are Windows-only.

## Step 3: Add the update scripts

Navigate to **Controls** > **Scripts** and upload one script per OEM.

`dell-update-bios-firmware-drivers.ps1`:

```powershell
$dcu = "${env:ProgramFiles(x86)}\Dell\CommandUpdate\dcu-cli.exe"
if (-not (Test-Path $dcu)) { $dcu = "$env:ProgramFiles\Dell\CommandUpdate\dcu-cli.exe" }

& $dcu /applyUpdates -silent -reboot=disable -autoSuspendBitLocker=enable `
  -updateType=bios,firmware,driver -outputLog=C:\Windows\Temp\dcu-apply.log
$code = $LASTEXITCODE

# 0 = success, 1 = restart required, 500 = no updates found
if ($code -in 0, 1, 500) { exit 0 }
exit $code
```

`-autoSuspendBitLocker=enable` suspends BitLocker for the next restart, so a BIOS update doesn't send the user to the recovery screen. Dell lists every exit code in the [Dell Command | Update CLI reference](https://www.dell.com/support/manuals/en-us/command-update/dcu_rg/command-line-interface-error-codes?guid=guid-fbb96b06-4603-423a-baec-cbf5963d8948&lang=en-us).

`lenovo-update-bios-firmware-drivers.ps1`:

```powershell
$tvsu = "${env:ProgramFiles(x86)}\Lenovo\System Update\tvsu.exe"
Start-Process -FilePath $tvsu -Wait -ArgumentList `
  '/CM -search A -action INSTALL -packagetypes 1,2,3,4 -includerebootpackages 3 -noreboot -noicon -nolicense -exporttowmi'
```

> **Note:** `-includerebootpackages 3` installs packages that need a restart without forcing one. Lenovo marks some BIOS and firmware packages as forced restart (`1`), forced shutdown (`4`), or delayed forced restart (`5`). Add those values only on a schedule your users expect. See Lenovo's [System Update command-line reference](https://docs.lenovocdrt.com/guides/sus/su_dg/su_dg_ch5/).

`hp-update-bios-firmware-drivers.ps1`:

```powershell
$ErrorActionPreference = 'Stop'
$hpia = 'C:\ProgramData\HP\HPIA'

if (-not (Get-Module -ListAvailable -Name HPCMSL)) {
  Install-PackageProvider -Name NuGet -Force | Out-Null
  Install-Module -Name HPCMSL -Force -AcceptLicense
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

HP documents every switch and return code in the [HP Image Assistant user guide](https://ftp.hp.com/pub/caps-softpaq/cmit/whitepapers/HPIAUserGuide2.pdf).

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

> **Note:** Fleet stops a script after 5 minutes by default. Driver and BIOS downloads can take longer, so raise [`script_execution_timeout`](https://fleetdm.com/docs/configuration/agent-configuration#script-execution-timeout) in your agent options, for example to `1800`.

## Step 4: Write a policy that finds devices that need updates

A policy passes when its query returns a row. This one fails on any PC of a given model whose BIOS is older than the version you set:

```sql
SELECT 1 FROM system_info s, platform_info p
WHERE s.hardware_model != 'YOUR-MODEL'
   OR version_compare(p.version, 'X.Y.Z') >= 0;
```

Replace `YOUR-MODEL` and `X.Y.Z` with a model from your fleet and the minimum BIOS version from the OEM's support page. To see what your hosts report, run `SELECT s.hardware_model, p.vendor, p.version, p.date FROM system_info s, platform_info p;` as a live report. [`platform_info`](https://fleetdm.com/tables/platform_info) works on Windows and Linux, so one query covers both.

Navigate to **Policies**, select your fleet, click **Add policy**, and save the query. Target it with the OEM label, and set the platform to Windows (or Linux for the fwupd policy).

## Step 5: Run the update when the policy fails

1. On the **Policies** page, click **Manage automations** and select **Run script**.
2. Check each BIOS policy and pick the matching script.
3. Click **Save**.

When a host fails the policy, Fleet runs the script. If the script fails, Fleet retries it up to three times. Learn more in [Automatically run scripts](https://fleetdm.com/guides/policy-automation-run-script).

## Manage it with GitOps

`default.yml`:

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
```

`fleets/workstations.yml`:

```yaml
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
  - name: Dell - BIOS is up to date
    platform: windows
    query: SELECT 1 FROM system_info s, platform_info p WHERE s.hardware_model != 'YOUR-MODEL' OR version_compare(p.version, 'X.Y.Z') >= 0;
    labels_include_any:
      - Dell
    run_script:
      path: ../lib/windows/scripts/dell-update-bios-firmware-drivers.ps1
  - name: Linux - Firmware is up to date
    platform: linux
    query: SELECT 1 FROM system_info s, platform_info p WHERE s.hardware_model != 'YOUR-MODEL' OR version_compare(p.version, 'X.Y.Z') >= 0;
    run_script:
      path: ../lib/linux/scripts/linux-update-firmware.sh
```

Add the HP and Lenovo policies the same way.

## Further reading

If you're coming from Intune, these are the integrations this guide mirrors:

- [HP Connect for Microsoft Intune user guide](https://connect.admin.hp.com/static/HPConnectUserGuide.pdf)
- [Dell Command | Update with Microsoft Intune](https://www.dell.com/support/manuals/en-us/command-update/admx_rg/using-microsoft-intune?guid=guid-8e1f9749-35cc-46cd-82cd-bc9576b32575&lang=en-us)
- [Lenovo System Update deployment guide](https://docs.lenovocdrt.com/guides/sus/su_dg/)
- [fwupdmgr reference](https://fwupd.github.io/libfwupdplugin/fwupdmgr.html)

None of this needs Windows Autopilot. If you're setting up new PCs, see [Windows zero touch without Entra or Autopilot](https://fleetdm.com/guides/windows-zero-touch-without-entra-or-autopilot).

If you're a Fleet customer, we can help you get this set up quickly as part of your Fast Track. Either way, if you run into any trouble, let us know. We're happy to help, and there's a friendly community of kind people who are happy to help too: [get support](https://fleetdm.com/support).

<meta name="articleTitle" value="Keep BIOS, firmware, and drivers up to date on Dell, HP, and Lenovo">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-26">
<meta name="description" value="Deploy Dell Command | Update, HP Image Assistant, Lenovo System Update, and fwupd with Fleet, and run updates automatically when a policy fails.">
