# Monitor firmware security with Fleet policies

The scariest screen on a Windows laptop is the blue one asking for a 48-digit number. Most of the time it's a BIOS update that forgot to suspend BitLocker. Once in a while, it's the only sign that someone changed what the computer boots. The help desk reads out the recovery key, the user types it in, and whatever changed is now running.

Firmware is the part of the PC your other security tools trust without checking. Dell, HP, and Lenovo all build ways to verify it: Secure Boot, HP Sure Start, and Dell Trusted Device. This guide shows how to turn their signals into Fleet policies and reports on Windows and Linux, so you notice when something below the operating system changes.

## Prerequisites

- `fleetd` on your Windows and Linux hosts.
- For HP Sure Start events: HP's [HP Notifications](https://h10032.www1.hp.com/ctg/Manual/c06216928.pdf) software. Without it, Sure Start events stay in the firmware and never reach the Windows event log.
- For Dell Trusted Device: the [Dell Trusted Device](https://www.dell.com/support/kbdoc/en-us/000130811/how-to-run-bios-verification-using-dell-trusted-device) agent.
- Fleet Premium, if you want policy automations.

## Step 1: Fail any PC with Secure Boot off

Secure Boot makes the firmware refuse to start a boot loader that isn't signed. Turning it off is the first step in most bootkit attacks, and it's also something users do to dual-boot.

Navigate to **Policies**, select your fleet, click **Add policy**, and use:

```sql
SELECT 1 FROM secureboot WHERE secure_boot = 1;
```

The [`secureboot`](https://fleetdm.com/tables/secureboot) table works on Windows, Linux, and macOS. A host fails this policy when it reports Secure Boot as off. To turn it back on, use the scripts in [Configure BIOS settings on Dell, HP, and Lenovo with Fleet scripts](https://fleetdm.com/guides/configure-bios-settings-on-dell-hp-and-lenovo-with-fleet-scripts).

## Step 2: Report HP Sure Start tamper and recovery events

HP Sure Start checks the BIOS against a protected copy at every boot and restores it when they don't match. With HP Notifications installed, it logs those events to Windows under **Applications and Services Logs** > **HP Sure Start**.

Run this as a report to see every recovery and tamper event Sure Start has logged:

```sql
SELECT datetime, eventid, data
FROM windows_eventlog
WHERE channel = 'HP Sure Start'
  AND eventid IN (30, 31, 32, 34, 36, 38);
```

The `channel` constraint is required. The [`windows_eventlog`](https://fleetdm.com/tables/windows_eventlog) table won't return anything without it.

These event IDs come from HP's [Sure Start white paper](https://h10032.www1.hp.com/ctg/Manual/c05163901.pdf):

- **30**: the primary BIOS was corrupted or missing.
- **31**: the backup BIOS was corrupted or missing.
- **32**: the flash layout differs from the factory layout.
- **34**: Sure Start recovered the primary BIOS.
- **36**: Sure Start repaired the flash layout.
- **38**: the BIOS on resume differs from the one that booted.

> **Note:** The time in Event Viewer is when HP Notifications copied the event from the firmware, not when it happened. The real time is in the event details. HP published this event ID list in 2016, so confirm it against your models.

To turn it into a policy that fails on any device with a recorded BIOS recovery, use:

```sql
SELECT 1 WHERE NOT EXISTS (
  SELECT 1 FROM windows_eventlog
  WHERE channel = 'HP Sure Start' AND eventid IN (30, 31, 38)
);
```

You can also check that Sure Start's own protections are on. On HP hardware, [`wmi_bios_info`](https://fleetdm.com/tables/wmi_bios_info) reads BIOS settings directly:

```sql
SELECT 1 FROM wmi_bios_info
WHERE name = 'Sure Start BIOS Settings Protection' AND value = 'Enable';
```

## Step 3: Check Dell Trusted Device BIOS verification

Dell Trusted Device compares the BIOS on the PC with a known-good copy kept off the device. In Intune, Dell reports the result with a custom compliance script, [`Get-TrustedDeviceResults.ps1`](https://www.dell.com/support/kbdoc/en-us/000222830/capture-dell-trusted-device-telemetry-with-microsoft-intune).

Fleet policies can't read arbitrary WMI classes yet ([#59971](https://github.com/fleetdm/fleet/issues/59971)), so get the result with a Fleet script instead. Fleet is already used in production with Dell hardware in healthcare, finance, and government, and this isn't a blocker there. If you have questions about how to handle it in your environment, [reach out](https://fleetdm.com/support). We're happy to help.

For Dell Trusted Device versions 2.1 through 6.0, Dell's [BIOS verification KB](https://www.dell.com/support/kbdoc/en-us/000130811/how-to-run-bios-verification-using-dell-trusted-device) documents a console command and its return codes. Upload this as `dell-trusted-device-bios-verification.ps1` in **Controls** > **Scripts**:

```powershell
$exe = "$env:ProgramFiles\Dell\TrustedDevice\Dell.SecurityCenter.Agent.Console.exe"
if (-not (Test-Path $exe)) { Write-Output "Dell Trusted Device console not found"; exit 1 }

& $exe -headless
$code = $LASTEXITCODE
switch ($code) {
  0 { Write-Output 'BIOS verification passed' }
  1 { Write-Output 'BIOS verification FAILED' }
  2 { Write-Output 'Result tampered' }
  default { Write-Output "BIOS verification error $code" }
}
exit $code
```

Dell deprecated this console in version 6.1 and folded BIOS verification into its Security Score. On 6.1 and later, download the **Trusted Device Agent Scripts** from [Dell's driver page](https://www.dell.com/support/home/en-us/product-support/product/trusted-device/drivers) and upload `Get-TrustedDeviceResults.ps1` as a Fleet script. It's the same script Dell gives Intune customers.

To check every Dell PC at once, navigate to **Hosts**, select your Dell hosts, click **Run script**, and choose the script. You can run it now or schedule it. A failed result shows up as a failed script run in each host's activity feed.

## Step 4: Check firmware health on Linux

On Linux, the `secureboot` policy from Step 1 covers Secure Boot. For everything else, use fwupd's [Host Security ID (HSI)](https://fwupd.github.io/libfwupdplugin/hsi.html) report, which scores firmware protections from HSI:0 (insecure) to HSI:4 (secure). Upload this as `linux-firmware-security-report.sh`:

```bash
#!/bin/bash
fwupdmgr security --json
```

Run it on a Linux host and read the JSON in the script output. Each entry under `SecurityAttributes` has an `HsiLevel` and an `HsiResult`, so you can see which protections passed and which didn't.

## Step 5: Pair it with disk encryption

None of this replaces disk encryption. Firmware checks tell you when the boot chain changes, and BitLocker and LUKS keep the data unreadable if the device is lost. Use both. To enforce encryption, see [Enforce disk encryption](https://fleetdm.com/guides/enforce-disk-encryption).

Train your help desk to treat a surprise BitLocker recovery prompt as a possible tampering signal, not a routine reset. Before reading out the key, check the host in Fleet: did Secure Boot just turn off? Is there a new Sure Start event? Was there a BIOS update? If nothing explains it, investigate before you unlock it.

## Manage it with GitOps

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
  - name: HP - No Sure Start BIOS recovery events
    platform: windows
    query: SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM windows_eventlog WHERE channel = 'HP Sure Start' AND eventid IN (30, 31, 38));
    labels_include_any:
      - HP
  - name: HP - Sure Start BIOS settings protection is on
    platform: windows
    query: SELECT 1 FROM wmi_bios_info WHERE name = 'Sure Start BIOS Settings Protection' AND value = 'Enable';
    labels_include_any:
      - HP
```

For the `HP` label, see [Keep BIOS, firmware, and drivers up to date on Dell, HP, and Lenovo](https://fleetdm.com/guides/keep-bios-firmware-and-drivers-up-to-date-on-dell-hp-and-lenovo).

## Further reading

If you're coming from Intune, these are the integrations this guide mirrors:

- [Capture Dell Trusted Device telemetry with Microsoft Intune](https://www.dell.com/support/kbdoc/en-us/000222830/capture-dell-trusted-device-telemetry-with-microsoft-intune)
- [HP Connect for Microsoft Intune user guide](https://connect.admin.hp.com/static/HPConnectUserGuide.pdf)
- [HP Sure Start white paper](https://h10032.www1.hp.com/ctg/Manual/c06216928.pdf)
- [fwupd Host Security ID](https://fwupd.github.io/libfwupdplugin/hsi.html)

You don't need Windows Autopilot or Entra for any of this. See [Windows zero touch without Entra or Autopilot](https://fleetdm.com/guides/windows-zero-touch-without-entra-or-autopilot).

If you're a Fleet customer, we can help you get this set up quickly as part of your Fast Track. Either way, if you run into any trouble, let us know. We're happy to help, and there's a friendly community of kind people who are happy to help too: [get support](https://fleetdm.com/support).

<meta name="articleTitle" value="Monitor firmware security with Fleet policies">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-26">
<meta name="description" value="Use Fleet policies, reports, and scripts to watch Secure Boot, HP Sure Start, Dell Trusted Device, and Linux firmware health.">
