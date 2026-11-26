# Monitor firmware security with Fleet policies

The scariest screen on a Windows laptop is the blue one asking for a 48-digit number. Often it means a BIOS update forgot to suspend BitLocker. Once in a while, it's the only sign that someone changed what the computer boots, and the help desk reads out the recovery key anyway.

Firmware is the part of the PC everything else trusts without checking, so I think about it more than is probably healthy. I'm always a little delighted when someone else does too. Lately that's been teams moving off Intune, asking what happens to the Dell Trusted Device and HP Sure Start checks they set up there. Those signals come from the device, not from Intune, so Fleet can read them too. This guide turns them, along with Secure Boot and fwupd's checks on Linux, into Fleet policies, reports, and scripts.

## Prerequisites

- Windows and Linux hosts enrolled in Fleet.
- **HP** and **Dell** labels, so each check only runs on the right hardware. Step 1 of [Keep BIOS, firmware, and drivers up to date on Dell, HP, and Lenovo](https://fleetdm.com/guides/keep-bios-firmware-and-drivers-up-to-date-on-dell-hp-and-lenovo) shows how to make them.
- For HP Sure Start events: HP Notifications installed on your HP PCs. Without it, Sure Start events stay in the firmware and never reach the Windows event log.
- For Dell Trusted Device: [Dell Trusted Device](https://www.dell.com/support/kbdoc/en-us/000130811/how-to-run-bios-verification-using-dell-trusted-device) installed on your Dell PCs.
- For Linux firmware checks: `fwupd` on your Linux hosts.

## Step 1: Fail any PC with Secure Boot off

Secure Boot makes the firmware refuse to start a boot loader that isn't signed. A bootkit has a much easier time with it off, and users also turn it off to dual-boot.

Navigate to **Policies**, select your fleet, click **Add policy**, and use:

```sql
SELECT 1 FROM secureboot WHERE secure_boot = 1;
```

The [`secureboot`](https://fleetdm.com/tables/secureboot) table works on Windows, Linux, and macOS. To turn Secure Boot back on remotely, use the scripts in [Configure BIOS settings on Dell, HP, and Lenovo with Fleet scripts](https://fleetdm.com/guides/configure-bios-settings-on-dell-hp-and-lenovo-with-fleet-scripts).

## Step 2: Watch for HP Sure Start tamper and recovery events

HP Sure Start checks the BIOS against a protected copy and restores it when they don't match. HP Notifications copies those events into Windows, under **Applications and Services Logs** > **HP Sure Start**, each time Windows boots or resumes. HP built that log so management tools could read it.

> **Note:** Confirm the channel name before you roll this out. Open Event Viewer on one of your HP PCs, right-click the **HP Sure Start** log, select **Properties**, and copy the **Full Name**. If it differs from `HP Sure Start`, use that name in the queries below.

Navigate to **Reports**, click **Add report**, and use this to list every recovery and tamper event Sure Start has logged:

```sql
SELECT datetime, eventid, data
FROM windows_eventlog
WHERE channel = 'HP Sure Start'
  AND eventid IN (30, 31, 32, 34, 36, 38);
```

Keep the `channel` constraint. The [`windows_eventlog`](https://fleetdm.com/tables/windows_eventlog) table returns nothing without it.

These event IDs come from HP's [Sure Start event table](https://h10032.www1.hp.com/ctg/Manual/c05163901.pdf):

- 30: the primary BIOS was corrupted or missing.
- 31: the backup BIOS was corrupted or missing.
- 32: the flash layout differs from the factory layout.
- 34: Sure Start recovered the primary BIOS.
- 36: Sure Start repaired the flash layout.
- 38: the BIOS on resume differs from the one that booted.

HP published this list in 2016, so check it against your models. The timestamp you see, in Event Viewer or in the report, is when HP Notifications copied the event, not when it happened. The real time is in the event details, in the `data` column.

Then add a policy, targeted at the **HP** label, that fails when any of the corruption events (30, 31, or 38) shows up:

```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM windows_eventlog
  WHERE channel = 'HP Sure Start' AND eventid IN (30, 31, 38)
);
```

A host that fails this policy keeps failing, even after you've looked into it. The events stay in the log, and if someone clears it, HP Notifications copies them back from the firmware. Use the policy to find out when an HP PC needs a person to look at it, and the report to see which events it logged. A pass only means something if HP Notifications is installed, since a PC without it has an empty log.

A second **HP** policy can check that Sure Start's settings protection is on, using [`wmi_bios_info`](https://fleetdm.com/tables/wmi_bios_info) to read the BIOS setting:

```sql
SELECT 1 FROM wmi_bios_info
WHERE name = 'Sure Start BIOS Settings Protection' AND value = 'Enable';
```

## Step 3: Check Dell Trusted Device BIOS verification

Dell Trusted Device compares the BIOS on the PC with a known-good copy kept off the device, on Dell's servers.

Fleet policies can't read arbitrary WMI classes yet ([#59971](https://github.com/fleetdm/fleet/issues/59971)), so you'll read the result with a Fleet script instead. Both OEM tools are also Windows-only. HP says Sure Start's event reporting isn't supported on Linux, and Dell Trusted Device is Windows software, so Linux hosts rely on Secure Boot and fwupd (Step 4).

Neither gap has been a blocker for teams already running Fleet in production on Dell, HP, and Lenovo hardware in healthcare, finance, and government. If you have questions about your environment, [reach out](https://fleetdm.com/support). We're happy to help.

For Dell Trusted Device versions 2.1 through 6.0, Dell's [BIOS verification KB](https://www.dell.com/support/kbdoc/en-us/000130811/how-to-run-bios-verification-using-dell-trusted-device) documents a console command and its return codes. Upload this as `dell-trusted-device-bios-verification.ps1` in **Controls** > **Scripts**:

```powershell
$exe = "$env:ProgramFiles\Dell\TrustedDevice\Dell.SecurityCenter.Agent.Console.exe"
if (-not (Test-Path $exe)) { Write-Output "Dell Trusted Device console not found"; exit 1 }

& $exe -headless
$code = $LASTEXITCODE
switch ($code) {
  0 { Write-Output 'BIOS verification passed' }
  1 { Write-Output 'BIOS verification failed' }
  2 { Write-Output 'Verification result was tampered with' }
  3 { Write-Output 'Result unknown' }
  7 { Write-Output 'Dell server unavailable' }
  11 { Write-Output 'Platform not supported' }
  default { Write-Output "Unexpected return code $code" }
}
exit $code
```

Dell deprecated this console in version 6.1 and folded BIOS verification into its Security Score. On newer versions, download Dell's Trusted Device Agent Scripts from [Dell's driver page](https://www.dell.com/support/home/en-us/product-support/product/trusted-device/drivers) and upload `Get-TrustedDeviceResults.ps1` as a Fleet script. It's the [same script Dell gives Intune customers](https://www.dell.com/support/kbdoc/en-us/000222830/capture-dell-trusted-device-telemetry-with-microsoft-intune), and the `TrustedDeviceComplianceRules.json` file that ships with it shows which values in its output count as passing.

To check every Dell PC at once, navigate to **Hosts**, filter by the **Dell** label, select the hosts, click **Run script**, and choose the script to run now or on a schedule. With the console script, a failed verification shows up as a failed script run in each host's activity feed.

## Step 4: Check firmware health on Linux

Step 1 already covers Secure Boot on Linux. For the rest of the firmware, use fwupd's [Host Security ID (HSI)](https://fwupd.github.io/libfwupdplugin/hsi.html) report, which scores firmware protections from HSI:0 (insecure) to HSI:4 (secure). Upload this as `linux-firmware-security-report.sh`:

```bash
#!/bin/bash
fwupdmgr security --json
```

Run it on your Linux hosts the same way. Each entry under `SecurityAttributes` in the output has an `HsiLevel` and an `HsiResult`, which tell you which protections passed.

## Step 5: Pair it with disk encryption

Firmware checks tell you when the boot chain changes, and disk encryption keeps the data unreadable when a laptop goes missing. You want both. Fleet enforces BitLocker on Windows and escrows LUKS keys on Linux, as covered in [Enforce disk encryption](https://fleetdm.com/guides/enforce-disk-encryption).

The two also cover for each other, because a surprise BitLocker recovery prompt is often the first visible sign that something in the boot chain changed. Train your help desk to treat it as a possible tampering signal. Before reading out the key, look the host up in Fleet. Did Secure Boot turn off, is there a new Sure Start event, or was there a BIOS update? If nothing explains it, investigate before you unlock it.

## Manage it with GitOps

`default.yml`:

```yaml
labels:
  - name: HP
    query: SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'HP%' OR hardware_vendor LIKE 'Hewlett%';
    label_membership_type: dynamic
```

`fleets/workstations.yml`:

```yaml
controls:
  scripts:
    - path: ../lib/windows/scripts/dell-trusted-device-bios-verification.ps1
    - path: ../lib/linux/scripts/linux-firmware-security-report.sh
policies:
  - name: Secure Boot is on
    platform: windows,linux
    query: SELECT 1 FROM secureboot WHERE secure_boot = 1;
  - name: HP - No Sure Start BIOS corruption events
    platform: windows
    query: SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM windows_eventlog WHERE channel = 'HP Sure Start' AND eventid IN (30, 31, 38));
    labels_include_any:
      - HP
  - name: HP - Sure Start BIOS settings protection is on
    platform: windows
    query: SELECT 1 FROM wmi_bios_info WHERE name = 'Sure Start BIOS Settings Protection' AND value = 'Enable';
    labels_include_any:
      - HP
reports:
  - name: HP Sure Start events
    platform: windows
    query: SELECT datetime, eventid, data FROM windows_eventlog WHERE channel = 'HP Sure Start' AND eventid IN (30, 31, 32, 34, 36, 38);
    interval: 86400
    labels_include_any:
      - HP
```

## Further reading

If you're coming from Intune, these are the OEM integrations this guide replaces:

- [Capture Dell Trusted Device telemetry with Microsoft Intune](https://www.dell.com/support/kbdoc/en-us/000222830/capture-dell-trusted-device-telemetry-with-microsoft-intune)
- [HP Connect for Microsoft Intune user guide](https://connect.admin.hp.com/static/HPConnectUserGuide.pdf)

For background on what the checks measure, see HP's [Sure Start white paper](https://h10032.www1.hp.com/ctg/Manual/c06216928.pdf) and fwupd's [Host Security ID docs](https://fwupd.github.io/libfwupdplugin/hsi.html).

None of this needs Windows Autopilot or Entra. See [Windows zero touch without Entra or Autopilot](https://fleetdm.com/guides/windows-zero-touch-without-entra-or-autopilot), and if you're wondering whether you need Microsoft 365 E3 at all, [Leaving E3 entirely](https://fleetdm.com/articles/leaving-e3-entirely).

If you're a Fleet customer, we can help you get this set up quickly as part of your Fast Track. Either way, if you run into any trouble, let us know. We're happy to help, and there's a friendly community of kind people who are happy to help too: [get support](https://fleetdm.com/support).

<meta name="articleTitle" value="Monitor firmware security with Fleet policies">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-26">
<meta name="description" value="Catch firmware tampering on Windows and Linux PCs with Fleet: Secure Boot, HP Sure Start, Dell Trusted Device, and fwupd checks.">
