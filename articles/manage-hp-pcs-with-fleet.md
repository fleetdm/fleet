# Manage HP PCs with Fleet

Firmware is the part of the PC everything else trusts without checking, so I think about it more than is probably healthy, and I'm always a little delighted to meet other people who do too. Lately, most of them have been IT teams moving off Intune, wondering what will happen to their HP Connect policies and HP Sure Start checks.

HP Connect lets Intune push HP's BIOS updates and settings, and Sure Start is a guard built into the PC that notices when the BIOS has been tampered with and puts it back. The good news is that neither one really lived in Intune. HP Connect is HP's scripts, run by Intune, and Sure Start keeps its notes on the PC itself. Fleet can run the same scripts and read the same notes, and it can keep your Linux PCs patched and locked down too.

## Prerequisites

- Fleet Premium, for policy automations.
- Hosts enrolled with fleetd, with [scripts enabled](https://fleetdm.com/guides/scripts).
- For Sure Start events, HP Notifications installed on your HP PCs.
- If your HP PCs have a BIOS admin password, a custom variable named `BIOS_PASSWORD` in **Controls** > **Variables** (scripts call it `$FLEET_SECRET_BIOS_PASSWORD`). Create it first, because Fleet won't save a script that uses a variable it doesn't know yet.

> **Note:** Fleet doesn't yet store a different BIOS password for each host ([#59972](https://github.com/fleetdm/fleet/issues/59972)), so for now one [custom variable](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) holds the same password for every device. The password itself still stays out of your scripts and out of Git. Fleet already runs in production on HP PCs at healthcare, finance, and government organizations, and this hasn't stood in their way. If you'd like to talk through your setup, [we're happy to help](https://fleetdm.com/support).

## Step 1: Label your HP PCs

A label is a group of hosts that Fleet keeps up to date on its own, and it's how HP's checks and scripts will reach only HP hardware. Go to **Hosts**, click **Filter by platform or label**, then **Add label**. Name it **HP** and use this query:

```sql
SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'HP%' OR hardware_vendor LIKE 'Hewlett%';
```

HP has gone by both names over the years, so the query checks for both.

Every policy in this guide should target this label. When you save one, set **Target** to **Custom** and check **HP**, so your other PCs don't fail an HP check or get sent HP's scripts.

## Step 2: Keep the BIOS, firmware, and drivers up to date

The BIOS is the small program that starts the PC before Windows does, and like everything else, it needs patches. HP has a tool that finds and installs them, along with firmware and driver updates, called [HP Image Assistant](https://ftp.hp.com/pub/caps-softpaq/cmit/whitepapers/HPIAUserGuide2.pdf) (HPIA). It has no documented way to install without someone clicking through it, so this script uses HP's PowerShell toolkit, the [Client Management Script Library](https://developers.hp.com/hp-client-management/doc/client-management-script-library) (CMSL), to set HPIA up the first time it runs. After that, HPIA does the updating.

Go to **Controls** > **Scripts** and upload `hp-update-bios-firmware-drivers.ps1`:

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

If your PCs have a BIOS password, HPIA reads it from a file, so make one with HP's `HpqPswd.exe` and add `/BIOSPwdFile:` followed by its path to the HPIA arguments above.

Some PCs will fall behind anyway, and a policy will tell you which. A policy is a yes-or-no question Fleet asks each host on a schedule, and a host that answers "no" fails. To see which models and BIOS versions you have, run this as a live report, which asks your hosts right now:

```sql
SELECT s.hardware_model, p.version, p.date FROM system_info s, platform_info p;
```

Then go to **Policies**, select your fleet, and click **Add policy**. Add one policy per model, using the model name from the live report and HP's minimum version, written the way your hosts report it:

```sql
SELECT 1 FROM system_info s, platform_info p
WHERE s.hardware_model != 'YOUR-MODEL'
   OR version_compare(p.version, 'X.Y.Z') >= 0;
```

Fleet doesn't track HP's latest BIOS for each model, so this policy is only as current as the version you type in. If bumping it every time HP ships a new one gets old, [tell us](https://fleetdm.com/support).

Last, add a policy automation, which tells Fleet what to do when a host fails. On the **Policies** page, click each BIOS policy's **Automation** column, choose **Run script** from the **Automation** dropdown, pick the script, and click **Save**. Fleet runs the script when a host first fails, trying up to three times in all. Updates that need a restart finish when the user next restarts.

> **Note:** Fleet stops a script after 300 seconds by default, and driver downloads can take longer, so raise [`script_execution_timeout`](https://fleetdm.com/docs/configuration/agent-configuration#script-execution-timeout) in your agent options, for example to `1800`.

## Step 3: Lock down BIOS settings

A handful of BIOS settings do most of the work of keeping a PC honest. Secure Boot only lets trusted software start the PC. The TPM is a chip that holds keys, including BitLocker's. Turning off network (PXE) and USB boot stops anyone from starting the PC from something they brought with them. This script sets each of those the safe way, along with Sure Start's protection for the settings themselves. Upload `hp-bios-security-settings.ps1`:

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
  'Secure Boot'                         = 'Enable'
  'TPM State'                           = 'Enable'
  'Virtualization Technology (VTx)'     = 'Enable'
  'Network (PXE) Boot'                  = 'Disable'
  'USB Storage Boot'                    = 'Disable'
  'Sure Start BIOS Settings Protection' = 'Enable'
}
foreach ($name in $settings.Keys) {
  Set-HPBIOSSettingValue -Name $name -Value $settings[$name] -Password $pw
}
```

Fleet swaps in the real password for `$FLEET_SECRET_BIOS_PASSWORD` before the script reaches the device. If your PCs don't have a BIOS password, drop `$pw` and the `-Password` argument.

The setting names come from HP's [BIOS administration guide](https://kaas.hpcloud.hp.com/pdf-public/pdf_12904670_en-US-1.pdf), and they're case sensitive, parentheses and all. `Virtualization Technology (VTx)` only exists on Intel models, and `Sure Start BIOS Settings Protection` needs a BIOS admin password set.

To check each setting, add a policy. On HP PCs, the [`wmi_bios_info`](https://fleetdm.com/tables/wmi_bios_info) table reads BIOS settings by HP's names:

```sql
SELECT 1 FROM wmi_bios_info WHERE name = 'Network (PXE) Boot' AND value = 'Disable';
```

To see the exact names and values, run `SELECT name, value FROM wmi_bios_info;` as a live report on one PC of each model. Secure Boot and the TPM have tables of their own:

```sql
SELECT 1 FROM secureboot WHERE secure_boot = 1;
```

```sql
SELECT 1 FROM tpm_info WHERE enabled = 1 AND activated = 1;
```

Attach the settings script to these policies as in step 2. Now if someone changes a setting, the policy fails the next time it runs and Fleet puts the setting back. The BIOS password is what stops them from changing it in the first place, so it's worth having both.

## Step 4: Watch for HP Sure Start events

So, how would you know if someone tampered with the BIOS? [HP Sure Start](https://h10032.www1.hp.com/ctg/Manual/c06216928.pdf) keeps a protected copy of the BIOS, compares the real one against it, and restores it when they don't match. It keeps a log of what it finds, and HP Notifications copies that log into Windows, under **Applications and Services Logs** > **HP Sure Start**.

Go to **Reports**, click **Add report**, and use this query to list Sure Start's recovery and tamper events:

```sql
SELECT datetime, eventid, data
FROM windows_eventlog
WHERE channel = 'HP Sure Start'
  AND eventid IN (30, 31, 32, 34, 36, 38);
```

Keep the `channel` line, because [`windows_eventlog`](https://fleetdm.com/tables/windows_eventlog) returns nothing without it. The events mean:

| Event ID | Meaning |
|---|---|
| 30, 31 | The main or backup BIOS was corrupted or missing |
| 32 | The flash layout differs from the factory's |
| 34, 36 | Sure Start recovered the main BIOS or repaired the layout |
| 38 | The BIOS the PC woke up with isn't the one it booted with |

These IDs come from HP's [Sure Start event table](https://h10032.www1.hp.com/ctg/Manual/c05163901.pdf), published in 2016, which is a long time ago in firmware years, so check them against your models. The `datetime` column is when HP Notifications copied the event, and `data` says when it happened.

To hear about trouble without reading the report every morning, add a policy that fails on any corruption event:

```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM windows_eventlog
  WHERE channel = 'HP Sure Start' AND eventid IN (30, 31, 38)
);
```

Sure Start has a long memory. The events stay in the log, and if someone clears it, HP Notifications copies them back from the firmware, so a host that fails this policy keeps failing. Treat a failure as a reason for a person to go look at that PC, and use the report to see what happened.

## Step 5: Do the same on Linux

CMSL, HPIA, and Sure Start's event reporting are Windows-only, and the scripts above are written for Windows. Each policy automation runs one script, so your Linux PCs need a script and a policy of their own.

For firmware, HP [recommends fwupd](https://ftp.ext.hp.com/pub/caps-softpaq/cmit/linuxtools/HP_LinuxTools.html) over its own Linux flash tool. [fwupd](https://fwupd.github.io/libfwupdplugin/fwupdmgr.html) installs firmware from the [Linux Vendor Firmware Service](https://fwupd.org/), a shared library of updates that hardware makers publish to. Upload `linux-update-firmware.sh`, add a Linux copy of the BIOS version policy from step 2, and attach the script to it:

```bash
#!/bin/bash
fwupdmgr refresh
fwupdmgr update --assume-yes --no-reboot-check
rc=$?

# 2 = nothing to do
if [ "$rc" -eq 2 ]; then exit 0; fi
exit "$rc"
```

For BIOS settings, HP's own Linux tools are experimental and need Secure Boot turned off unless you sign their kernel module, which rather defeats the purpose. The kernel's [`hp-bioscfg`](https://kernel.org/doc/Documentation/ABI/testing/sysfs-class-firmware-attributes) driver is easier. It gives each setting a folder, named the way HP names it, under `/sys/class/firmware-attributes/hp-bioscfg/attributes/`. To check a setting, read its `current_value` file with the `file_lines` table (the `secureboot` policy from step 3 works here too):

```sql
SELECT 1 FROM file_lines
WHERE path = '/sys/class/firmware-attributes/hp-bioscfg/attributes/Network (PXE) Boot/current_value'
  AND line = 'Disable';
```

To change a setting, write the new value to that same file. If a BIOS admin password is set, write it to `/sys/class/firmware-attributes/hp-bioscfg/authentication/` first, as the [kernel docs](https://kernel.org/doc/Documentation/ABI/testing/sysfs-class-firmware-attributes) describe. Try it on one PC before you roll it out, and if Linux gives you trouble, [let us know](https://fleetdm.com/support) and we'll help.

## Manage it with GitOps

If you manage Fleet with GitOps, where your settings live as YAML files in Git, everything above fits in two files and one secret. The label goes in `default.yml`:

```yaml
labels:
  - name: HP
    query: SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'HP%' OR hardware_vendor LIKE 'Hewlett%';
    label_membership_type: dynamic
```

Add it to the `labels` you already have, since GitOps removes any label that isn't listed. Everything else goes in a fleet file, here `fleets/workstations.yml`. Each script an automation runs must also be listed in that fleet's `controls.scripts`:

```yaml
name: Workstations
agent_options:
  script_execution_timeout: 1800
  # ...plus the rest of this fleet's existing agent options
controls:
  scripts:
    - path: ../lib/windows/scripts/hp-update-bios-firmware-drivers.ps1
    - path: ../lib/windows/scripts/hp-bios-security-settings.ps1
    - path: ../lib/linux/scripts/linux-update-firmware.sh
policies:
  - name: HP - BIOS is up to date
    platform: windows
    query: SELECT 1 FROM system_info s, platform_info p WHERE s.hardware_model != 'YOUR-MODEL' OR version_compare(p.version, 'X.Y.Z') >= 0;
    labels_include_any:
      - HP
    run_script:
      path: ../lib/windows/scripts/hp-update-bios-firmware-drivers.ps1
  - name: HP - Network boot is off
    platform: windows
    query: SELECT 1 FROM wmi_bios_info WHERE name = 'Network (PXE) Boot' AND value = 'Disable';
    labels_include_any:
      - HP
    run_script:
      path: ../lib/windows/scripts/hp-bios-security-settings.ps1
  - name: HP - No Sure Start BIOS corruption events
    platform: windows
    query: SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM windows_eventlog WHERE channel = 'HP Sure Start' AND eventid IN (30, 31, 38));
    labels_include_any:
      - HP
reports:
  - name: HP Sure Start events
    platform: windows
    query: SELECT datetime, eventid, data FROM windows_eventlog WHERE channel = 'HP Sure Start' AND eventid IN (30, 31, 32, 34, 36, 38);
    interval: 86400
    labels_include_any:
      - HP
# ...plus any other keys your fleet file already has
```

The secret is the BIOS password. Store it as a GitHub or GitLab secret named `FLEET_SECRET_BIOS_PASSWORD`, as described in [Secrets in scripts and configuration profiles](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles).

## Treat a surprise BitLocker prompt as a clue

None of this replaces [disk encryption](https://fleetdm.com/guides/enforce-disk-encryption), and BitLocker makes a handy tripwire of its own. A recovery prompt nobody expected can be the first sign that something in the boot chain, the firmware and everything it starts, has changed. Before the help desk reads out a recovery key, have them look the host up in Fleet for a Secure Boot change, a new Sure Start event, or a BIOS update. If none of those explains it, investigate before unlocking.

The rest of the PC will go on trusting the firmware without checking. That's fine, as long as something else is checking, and now Fleet is.

## Further reading

If you're coming from Intune, these pick up where this guide leaves off:

- [HP Connect for Microsoft Intune user guide](https://connect.admin.hp.com/static/HPConnectUserGuide.pdf), to compare with what you had
- [Windows zero touch without Entra or Autopilot](https://fleetdm.com/guides/windows-zero-touch-without-entra-or-autopilot), for setting up new PCs
- [Leaving E3 entirely](https://fleetdm.com/articles/leaving-e3-entirely), what it costs to drop Microsoft 365 E3
- [Manage Dell PCs with Fleet](https://fleetdm.com/guides/manage-dell-pcs-with-fleet) and [Manage Lenovo PCs with Fleet](https://fleetdm.com/guides/manage-lenovo-pcs-with-fleet), if HP isn't your only brand

If you're a Fleet customer, we can help you get this set up quickly as part of your Fast Track. Either way, if you run into any trouble, let us know. We're happy to help, and there's a friendly community of kind people who are happy to help too: [get support](https://fleetdm.com/support).

<meta name="articleTitle" value="Manage HP PCs with Fleet">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-27">
<meta name="description" value="Update the BIOS, lock down settings, and watch HP Sure Start on Windows and Linux with Fleet, no Intune required.">
