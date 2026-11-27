# Manage HP PCs with Fleet

Firmware is the part of the PC everything else trusts without checking, so I think about it more than is probably healthy. I'm always a little delighted when someone else does too. Lately, that's been teams moving off Intune who want to know what happens to the HP Connect policies and HP Sure Start checks they set up there.

The good news is that none of it really lives in Intune. HP Connect runs HP's scripts as Intune remediations, and Sure Start's signals come from the PC itself. Fleet can run the same scripts and read the same signals, on Linux as well as Windows, so you keep the security HP built these tools to provide.

## Prerequisites

- Fleet Premium, for policy automations.
- Hosts enrolled with fleetd, with [scripts enabled](https://fleetdm.com/guides/scripts).
- For Sure Start events, HP Notifications installed on your HP PCs. Without it, they never reach the Windows event log.
- If your HP PCs have a BIOS admin password, a custom variable named `BIOS_PASSWORD`. Add it in **Controls** > **Variables** before you upload the scripts below, because Fleet won't save a script that uses a variable that doesn't exist yet. Scripts use it as `$FLEET_SECRET_BIOS_PASSWORD`, because Fleet variables always start with `FLEET_SECRET_`.

> **Note:** Fleet doesn't yet store a unique BIOS password for each host ([#59972](https://github.com/fleetdm/fleet/issues/59972)), so for now one [custom variable](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) holds the same password for every device. That still keeps it out of your scripts and out of Git. Fleet is already used in production on HP hardware in healthcare, finance, and government, and this hasn't been a blocker there. If you'd like to talk through your setup, [reach out](https://fleetdm.com/support).

## Step 1: Label your HP PCs

A label is a group of hosts that Fleet keeps up to date on its own. Navigate to **Hosts**, click **Filter by platform or label**, then **Add label**. Name it **HP** and use this query:

```sql
SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'HP%' OR hardware_vendor LIKE 'Hewlett%';
```

HP has gone by both names over the years, so the query checks for both.

## Step 2: Keep the BIOS, firmware, and drivers up to date

The BIOS is the small program that starts the PC before Windows does, and like everything else, it needs patches. HP's update tool is [HP Image Assistant](https://ftp.hp.com/pub/caps-softpaq/cmit/whitepapers/HPIAUserGuide2.pdf) (HPIA). HPIA has no documented silent installer, but HP's [Client Management Script Library](https://developers.hp.com/hp-client-management/doc/client-management-script-library) (CMSL) can install it, so this script sets up both on first run. Navigate to **Controls** > **Scripts** and upload `hp-update-bios-firmware-drivers.ps1`:

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

If your PCs have a BIOS password, pass HPIA a password file with `/BIOSPwdFile`. HP's `HpqPswd.exe` makes the file, and the HPIA user guide covers every other switch.

To spot the PCs that fall behind anyway, add a policy. A policy is a yes-or-no question Fleet asks each host on a schedule, and a "no" means the host fails. Start with a live report to see what your hosts say:

```sql
SELECT s.hardware_model, p.version, p.date FROM system_info s, platform_info p;
```

Then navigate to **Policies**, select your fleet, and add one policy per model. Use the model name from the live report, and HP's minimum version written in the same format your hosts report:

```sql
SELECT 1 FROM system_info s, platform_info p
WHERE s.hardware_model != 'YOUR-MODEL'
   OR version_compare(p.version, 'X.Y.Z') >= 0;
```

Fleet doesn't track HP's latest BIOS for each model, so this policy is only as current as the version you type in. Bump it when HP ships a new one. If that's a problem for you, [tell us](https://fleetdm.com/support).

Finally, add a policy automation, which tells Fleet what to do when a host fails. On the **Policies** page, click each BIOS policy's **Automation** column, choose **Run script**, pick the script, and click **Save**. Fleet runs the script when a host first fails the policy, and tries up to three times. Updates that need a restart finish when the user next restarts.

> **Note:** Fleet stops a script after 300 seconds by default, and driver downloads can take longer, so raise [`script_execution_timeout`](https://fleetdm.com/docs/configuration/agent-configuration#script-execution-timeout) in your agent options, for example to `1800`.

## Step 3: Lock down BIOS settings

A few BIOS settings do a lot of the work of keeping a PC honest. Secure Boot only lets trusted software start the PC. The TPM is a chip that holds keys, including BitLocker's. Turning off network (PXE) and USB boot stops anyone from starting the PC from something they brought along. This script sets all of those, plus Sure Start's protection for the settings themselves. Upload `hp-bios-security-settings.ps1`:

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

Fleet fills in `$FLEET_SECRET_BIOS_PASSWORD` before the script reaches the device. If your PCs don't have a BIOS password, drop `$pw` and the `-Password` argument.

The setting names come from HP's [BIOS administration guide](https://kaas.hpcloud.hp.com/pdf-public/pdf_12904670_en-US-1.pdf), and they're case sensitive, parentheses and all. `Virtualization Technology (VTx)` is for Intel models, and `Sure Start BIOS Settings Protection` needs a BIOS admin password set.

To check each setting, add a policy. On HP PCs, [`wmi_bios_info`](https://fleetdm.com/tables/wmi_bios_info) reads BIOS settings by HP's names:

```sql
SELECT 1 FROM wmi_bios_info WHERE name = 'Network (PXE) Boot' AND value = 'Disable';
```

Run `SELECT name, value FROM wmi_bios_info;` as a live report on one PC per model to see the exact names and values. Secure Boot and the TPM have their own tables:

```sql
SELECT 1 FROM secureboot WHERE secure_boot = 1;
```

```sql
SELECT 1 FROM tpm_info WHERE enabled = 1 AND activated = 1;
```

Attach the settings script to these policies the same way as in step 2. If someone changes a setting, the policy fails the next time it runs and Fleet puts it back. The BIOS password is what stops them from changing it in the first place, so use both.

## Step 4: Watch for HP Sure Start events

So, how would you know if someone tampered with the BIOS? [HP Sure Start](https://h10032.www1.hp.com/ctg/Manual/c06216928.pdf) keeps a protected copy of the BIOS, checks the real one against it, and restores it when they don't match. It logs what it finds, and HP Notifications copies those events into Windows, under **Applications and Services Logs** > **HP Sure Start**.

Navigate to **Reports**, click **Add report**, and use this to list Sure Start's recovery and tamper events:

```sql
SELECT datetime, eventid, data
FROM windows_eventlog
WHERE channel = 'HP Sure Start'
  AND eventid IN (30, 31, 32, 34, 36, 38);
```

Keep the `channel` line, because [`windows_eventlog`](https://fleetdm.com/tables/windows_eventlog) returns nothing without it. Here's what each event means:

| Event ID | Meaning |
|---|---|
| 30, 31 | The main or backup BIOS was corrupted or missing |
| 32 | The flash layout differs from the factory's |
| 34, 36 | Sure Start recovered the main BIOS or repaired the layout |
| 38 | The BIOS the PC woke up with isn't the one it booted with |

These IDs come from HP's [Sure Start event table](https://h10032.www1.hp.com/ctg/Manual/c05163901.pdf), published in 2016 (a long time ago, in firmware years), so check them against your models. The `datetime` column is when HP Notifications copied the event, and `data` has when it happened.

Then add a policy, targeted at the **HP** label, that fails on any corruption event:

```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM windows_eventlog
  WHERE channel = 'HP Sure Start' AND eventid IN (30, 31, 38)
);
```

Sure Start has a long memory. Once a host fails this policy, it keeps failing, because the events stay in the log, and HP Notifications copies them back from the firmware if someone clears it. Treat a failure as a reason for a person to look at that PC, and use the report to see what happened.

## Step 5: Do the same on Linux

CMSL, HPIA, and Sure Start's event reporting are Windows-only, but Linux has its own way in. For updates, HP [recommends fwupd](https://ftp.ext.hp.com/pub/caps-softpaq/cmit/linuxtools/HP_LinuxTools.html) over its own Linux flash tool. [fwupd](https://fwupd.github.io/libfwupdplugin/fwupdmgr.html) installs firmware from the [Linux Vendor Firmware Service](https://fwupd.org/).

For settings, HP's own Linux tools are experimental and need Secure Boot off unless you sign their kernel module. The kernel's [`hp-bioscfg`](https://kernel.org/doc/Documentation/ABI/testing/sysfs-class-firmware-attributes) driver is easier. It gives each BIOS setting a folder, named the way HP names it, under `/sys/class/firmware-attributes/hp-bioscfg/attributes/`.

Each policy automation runs one script, and PowerShell doesn't run on Linux, so Linux gets its own policy. Upload `linux-update-firmware.sh`, add a Linux copy of the BIOS version policy from step 2, and attach the script to it:

```bash
#!/bin/bash
fwupdmgr refresh
fwupdmgr update --assume-yes --no-reboot-check
rc=$?

# 2 = nothing to do
if [ "$rc" -eq 2 ]; then exit 0; fi
exit "$rc"
```

To check a setting, read the driver with `file_lines`. The `secureboot` policy from step 3 works here too.

```sql
SELECT 1 FROM file_lines
WHERE path = '/sys/class/firmware-attributes/hp-bioscfg/attributes/Network (PXE) Boot/current_value'
  AND line = 'Disable';
```

To change a setting, write the new value to its `current_value` file. When a BIOS admin password is set, the kernel requires it first, so write it to `/sys/class/firmware-attributes/hp-bioscfg/authentication/` as the [kernel docs](https://kernel.org/doc/Documentation/ABI/testing/sysfs-class-firmware-attributes) describe. Try it on one PC before you roll it out. If you get stuck on Linux, [let us know](https://fleetdm.com/support).

## Treat a surprise BitLocker prompt as a signal

None of this replaces [disk encryption](https://fleetdm.com/guides/enforce-disk-encryption), and the two back each other up. A BitLocker recovery prompt nobody expected can be the first sign that something in the boot chain changed. Before the help desk reads out a recovery key, have them look the host up in Fleet for a Secure Boot change, a new Sure Start event, or a BIOS update. If nothing explains it, investigate before unlocking.

The rest of the PC will go on trusting the firmware without checking, which is fine, now that Fleet does.

## Manage it with GitOps

`default.yml`:

```yaml
labels:
  - name: HP
    query: SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'HP%' OR hardware_vendor LIKE 'Hewlett%';
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
# ...plus the agent_options and other keys your fleet file already has
```

Store the BIOS password as a GitHub or GitLab secret named `FLEET_SECRET_BIOS_PASSWORD`, as described in [Secrets in scripts and configuration profiles](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles).

## Further reading

If you're coming from Intune, this guide covers what you used [HP Connect for Microsoft Intune](https://connect.admin.hp.com/static/HPConnectUserGuide.pdf) for.

None of this needs Windows Autopilot or Entra. For new PCs, see [Windows zero touch without Entra or Autopilot](https://fleetdm.com/guides/windows-zero-touch-without-entra-or-autopilot), and if you're wondering what it costs to drop Microsoft 365 E3 altogether, we [priced it out](https://fleetdm.com/articles/leaving-e3-entirely). Managing Dell or Lenovo too? See [Manage Dell PCs with Fleet](https://fleetdm.com/guides/manage-dell-pcs-with-fleet) and [Manage Lenovo PCs with Fleet](https://fleetdm.com/guides/manage-lenovo-pcs-with-fleet).

If you're a Fleet customer, we can help you get this set up quickly as part of your Fast Track. Either way, if you run into any trouble, let us know. We're happy to help, and there's a friendly community of kind people who are happy to help too: [get support](https://fleetdm.com/support).

<meta name="articleTitle" value="Manage HP PCs with Fleet">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-27">
<meta name="description" value="Update the BIOS, lock down settings, and watch HP Sure Start on Windows and Linux with Fleet, no Intune required.">
