# Manage Dell PCs with Fleet

The BIOS is the small program built into a PC that wakes everything up when you press the power button. It runs before Windows or Linux, and before your EDR, so if there's a hole in it, every security tool you bought is sitting on top of that hole. That's why Dell ships its own tools for updating the BIOS, locking down its settings, and checking that nobody has tampered with it.

I took the question of what happens to all of that after Intune on a long walk with Claude, and the answer was simpler than I expected. Dell's Intune integrations come down to installing Dell's tools, running them with a script, and running them again when a device drifts. Nothing about that needs Intune. Fleet does the same thing with software, scripts, and policies, on Linux as well as Windows.

![Walking and running distance for November, peaking at 16.7 miles on November 22](../website/assets/images/articles/walking-and-running-distance-november-585x711@2x.jpg)

In my defense, firmware is a big topic (see November 22).

## Prerequisites

- Fleet Premium, for policy automations.
- Hosts enrolled with fleetd, with [scripts enabled](https://fleetdm.com/guides/scripts), in a fleet. (Automations don't work on policies for "All fleets".)
- If your Dell PCs have a BIOS admin password, a custom variable named `BIOS_PASSWORD`. Add it in **Controls** > **Variables** first, because Fleet won't save a script that mentions a variable it hasn't heard of.

> **Note:** Fleet can't yet store a different BIOS password for each host ([#59972](https://github.com/fleetdm/fleet/issues/59972)), so for now one [custom variable](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles) holds the same password for every PC. That still keeps it out of your scripts and Git, which is what matters most. Fleet already runs in production on Dell hardware in healthcare, finance, and government, and this hasn't been a blocker there. If you'd like to talk through your setup, [reach out](https://fleetdm.com/support).

## Step 1: Label your Dell PCs

A label is a group of hosts that match a query, and this one rounds up your Dells. Go to **Hosts**, click **Filter by platform or label**, then **Add label**. Name it Dell and use this query:

```sql
SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'Dell%';
```

## Step 2: Install Dell's tools

Dell makes two tools you'll lean on. [Dell Command | Update](https://www.dell.com/support/manuals/en-us/command-update/dcu_rg/dell-command-update-cli-commands?guid=guid-92619086-5f7c-4a05-bce2-0d560c15e8ed&lang=en-us) fetches BIOS, firmware, and driver updates, and [Dell Command | Configure](https://www.dell.com/support/manuals/en-us/command-configure/dcc_5.x_ref_guide/) (`cctk`) changes BIOS settings.

Dell Command | Update is a Fleet-maintained app, so go to **Software**, click **Add software**, and add **Dell Command Update** for Windows, targeting the **Dell** label. For Dell Command | Configure, download it from Dell and add it under **Software** > **Add software** > **Custom package**, also targeting **Dell**.

Adding an app doesn't install it, so Dell Command | Update also needs a policy. Policies are yes-or-no questions Fleet asks every host about once an hour, and this one asks whether the app is installed. When you save it, and every policy below, set **Target** to **Custom** and check **Dell**.

```sql
SELECT 1 FROM programs WHERE name LIKE 'Dell Command %Update' AND publisher = 'Dell Inc.';
```

Then, on the **Policies** page, click the policy's **Automation** column, choose **Install software** from the **Automation** dropdown, pick **Dell Command Update**, and click **Save**. From then on, any Dell that answers "no" gets it.

## Step 3: Keep the BIOS, firmware, and drivers up to date

Go to **Controls** > **Scripts** and upload `dell-update-bios-firmware-drivers.ps1`:

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

BitLocker can ask for its recovery key after a BIOS update, which is a rough way to start a Monday, so `-autoSuspendBitLocker=enable` has it sit out the next restart. Add `-updateSeverity=security,critical` if you only want the urgent stuff, and see Dell's [CLI reference](https://www.dell.com/support/manuals/en-us/command-update/dcu_rg/command-line-interface-error-codes?guid=guid-fbb96b06-4603-423a-baec-cbf5963d8948&lang=en-us) for the other exit codes.

So, how do you know when a PC falls behind? Start by running this as a live report (a one-off question your hosts answer right away). It lists each PC's model and BIOS version, so you know which models you have:

```sql
SELECT s.hardware_model, p.version, p.date FROM system_info s, platform_info p;
```

Then add a policy for each model. Every other model passes it automatically, and this model passes only if its BIOS is at least version X.Y.Z. Copy the model name from the live report and the minimum version from Dell's support page, exactly:

```sql
SELECT 1 FROM system_info s, platform_info p
WHERE s.hardware_model != 'YOUR-MODEL'
   OR version_compare(p.version, 'X.Y.Z') >= 0;
```

Fleet doesn't track Dell's latest BIOS for each model, so this policy is only as current as the version you type in. Bump it when Dell ships a new one. If that upkeep worries you, [let us know](https://fleetdm.com/support).

Set each BIOS policy's automation to **Run script** with `dell-update-bios-firmware-drivers.ps1`. Fleet runs it when a host starts failing and tries up to three times. Anything that needs a restart finishes the next time the user restarts.

> **Note:** Fleet stops a script after 5 minutes by default, and BIOS downloads can take longer. Raise [`script_execution_timeout`](https://fleetdm.com/docs/configuration/agent-configuration#script-execution-timeout) in the fleet's agent options, for example to `1800` seconds.

## Step 4: Lock down BIOS settings

Upload `dell-bios-security-settings.ps1`:

```powershell
$cctk = "${env:ProgramFiles(x86)}\Dell\Command Configure\X86_64\cctk.exe"
$pw = '$FLEET_SECRET_BIOS_PASSWORD'

& $cctk --SecureBoot=Enabled --TpmSecurity=Enabled --TpmActivation=Enabled `
  --Virtualization=Enabled --VtForDirectIo=Enabled --UefiNwStack=Disabled `
  --UefiBootPathSecurity=AlwaysExceptInternalHdd "--ValSetupPwd=$pw"
exit $LASTEXITCODE
```

Fleet fills in `$FLEET_SECRET_BIOS_PASSWORD` before the script reaches the PC. (No BIOS password? Drop `$pw` and `--ValSetupPwd`.)

The script turns on Secure Boot (only trusted, signed software gets to start the PC), the TPM (a tiny chip that guards encryption keys), and virtualization. It also turns off PXE boot, which starts the PC from the network.

Dell has no single switch for USB boot, so `--UefiBootPathSecurity=AlwaysExceptInternalHdd` asks for the BIOS admin password before booting from anything but the internal drive. A stranger with a USB stick gets a password prompt and not much else. For other settings worth turning on, see Dell's [security profile](https://www.dell.com/support/manuals/en-us/command-configure/dcc_5.x_ref_guide/security-profile?guid=guid-66ba0e99-ad94-42f0-a162-00a673fd138e).

To check each setting, add a policy. On Dells from 2018 on, [`wmi_bios_info`](https://fleetdm.com/tables/wmi_bios_info) reads BIOS settings by the same names `cctk` uses:

```sql
SELECT 1 FROM wmi_bios_info WHERE name = 'UefiNwStack' AND value = 'Disabled';
```

Older Dells can use different names, so run `SELECT name, value FROM wmi_bios_info;` as a live report on one PC of each model first. Secure Boot and the TPM have their own tables:

```sql
SELECT 1 FROM secureboot WHERE secure_boot = 1;
```

```sql
SELECT 1 FROM tpm_info WHERE enabled = 1 AND activated = 1;
```

Set these policies' automation to **Run script** with `dell-bios-security-settings.ps1`. If someone flips a setting, the policy fails at the next check and Fleet flips it back. The BIOS password keeps them from flipping it in the first place, so use both.

## Step 5: Check Dell Trusted Device BIOS verification

[Dell Trusted Device](https://www.dell.com/support/kbdoc/en-us/000130811/how-to-run-bios-verification-using-dell-trusted-device) compares the BIOS on the PC with a known-good copy on Dell's servers. Fleet policies can't yet read arbitrary WMI classes (WMI is Windows' catalog of system details), so you'll run a script instead ([#59971](https://github.com/fleetdm/fleet/issues/59971)). If you need this as a policy, [tell us](https://fleetdm.com/support).

For Dell Trusted Device 2.1 through 6.0, upload `dell-trusted-device-bios-verification.ps1`:

```powershell
$exe = "$env:ProgramFiles\Dell\TrustedDevice\Dell.SecurityCenter.Agent.Console.exe"
if (-not (Test-Path $exe)) { Write-Output 'Dell Trusted Device console not found'; exit 1 }

# 0 = passed, 1 = failed, 2 = result tampered with, 7 = Dell's server unavailable
& $exe -headless
exit $LASTEXITCODE
```

Dell retired this console in 6.1. On newer versions, download the Trusted Device Agent Scripts from [Dell](https://www.dell.com/support/home/en-us/product-support/product/trusted-device/drivers) and upload `Get-TrustedDeviceResults.ps1`. It's the [same script Dell gives Intune customers](https://www.dell.com/support/kbdoc/en-us/000222830/capture-dell-trusted-device-telemetry-with-microsoft-intune), and the `TrustedDeviceComplianceRules.json` beside it says which results pass.

To check every Dell at once, filter **Hosts** by the **Dell** label, select them all, click **Run script**, and choose **Run now** or **Schedule for later**. A failed verification shows up as a failed script run.

## Step 6: Do the same on Linux

Dell Command | Update and Dell Trusted Device are Windows-only. On Linux, Dell [publishes BIOS updates](https://www.dell.com/support/kbdoc/en-us/000122025/how-to-update-firmware-under-linux) to the [Linux Vendor Firmware Service](https://fwupd.org/), a public library of firmware, and [fwupd](https://fwupd.github.io/libfwupdplugin/fwupdmgr.html) installs them. Dell Command | Configure also has a [Linux version](https://www.dell.com/support/manuals/en-us/command-configure/dcc_5.x_ig/installing-dell-command-configure-on-linux-systems?guid=guid-9d3f8b53-abb4-4d54-9049-b1e0bfbfc100&lang=en-us) for RHEL 8 and 9 and Ubuntu 20.04, 22.04, and 24.04, which you upload like any other package.

A policy runs one script, and PowerShell won't run on Linux, so make a Linux copy of the BIOS version policy from step 3 and attach `linux-update-firmware.sh`:

```bash
#!/bin/bash
fwupdmgr refresh
fwupdmgr update --assume-yes --no-reboot-check
rc=$?
# 2 = nothing to do
[ "$rc" -eq 2 ] && exit 0
exit "$rc"
```

`dell-bios-security-settings.sh` sets the same options:

```bash
#!/bin/bash
pw='$FLEET_SECRET_BIOS_PASSWORD'
/opt/dell/dcc/cctk --SecureBoot=Enabled --TpmSecurity=Enabled --TpmActivation=Enabled \
  --Virtualization=Enabled --VtForDirectIo=Enabled --UefiNwStack=Disabled \
  --UefiBootPathSecurity=AlwaysExceptInternalHdd "--ValSetupPwd=$pw"
```

To check settings, read the kernel's [`dell-wmi-sysman`](https://kernel.org/doc/Documentation/ABI/testing/sysfs-class-firmware-attributes) driver with `file_lines`. The `secureboot` policy from step 4 works here too.

```sql
SELECT 1 FROM file_lines
WHERE path = '/sys/class/firmware-attributes/dell-wmi-sysman/attributes/UefiNwStack/current_value'
  AND line = 'Disabled';
```

For a wider check, run `fwupdmgr security --json` as a script. It returns fwupd's [Host Security ID](https://fwupd.github.io/libfwupdplugin/hsi.html), a report card for firmware protections from HSI:0 (insecure) to HSI:4 (secure).

## Treat a surprise BitLocker prompt as a signal

None of this replaces disk encryption, so [keep enforcing it](https://fleetdm.com/guides/enforce-disk-encryption). The two back each other up. A BitLocker recovery prompt nobody expected can be the first sign that something in the boot chain changed. Before the help desk reads out a recovery key, have them look the host up in Fleet. Did Secure Boot turn off, did the BIOS verification fail, or was there a BIOS update? If nothing explains it, dig in before unlocking.

## Manage it with GitOps

To manage all of this as code, add the label to the `labels` in `default.yml`:

```yaml
labels:
  - name: Dell
    query: SELECT 1 FROM system_info WHERE hardware_vendor LIKE 'Dell%';
    label_membership_type: dynamic
```

Keep the labels you already have in that list, since GitOps removes labels that aren't listed. Policy automations live in a fleet file, which must also list each script under `controls.scripts`. For example, `fleets/workstations.yml`:

```yaml
name: Workstations
agent_options:
  script_execution_timeout: 1800
  # ...the rest of this fleet's agent options
controls:
  scripts:
    - path: ../lib/windows/scripts/dell-update-bios-firmware-drivers.ps1
    - path: ../lib/windows/scripts/dell-bios-security-settings.ps1
software:
  fleet_maintained_apps:
    - slug: dell-command-update/windows
      labels_include_any:
        - Dell
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
  - name: Dell - UEFI network boot is off
    platform: windows
    query: SELECT 1 FROM wmi_bios_info WHERE name = 'UefiNwStack' AND value = 'Disabled';
    labels_include_any:
      - Dell
    run_script:
      path: ../lib/windows/scripts/dell-bios-security-settings.ps1
  - name: Secure Boot is on
    platform: windows,linux
    query: SELECT 1 FROM secureboot WHERE secure_boot = 1;
```

Store the BIOS password as a GitHub or GitLab secret named `FLEET_SECRET_BIOS_PASSWORD` and pass it to your GitOps workflow, as described in [Custom variables](https://fleetdm.com/guides/secrets-in-scripts-and-configuration-profiles).

## Further reading

If you're coming from Intune, this guide replaces:

- [Dell Command | Update with Microsoft Intune](https://www.dell.com/support/manuals/en-us/command-update/admx_rg/using-microsoft-intune?guid=guid-8e1f9749-35cc-46cd-82cd-bc9576b32575&lang=en-us)
- [Dell Command | Endpoint Configure for Microsoft Intune](https://www.dell.com/support/kbdoc/en-us/000214308/dell-command-endpoint-configure-for-microsoft-intune)
- [Capture Dell Trusted Device telemetry with Microsoft Intune](https://www.dell.com/support/kbdoc/en-us/000222830/capture-dell-trusted-device-telemetry-with-microsoft-intune)

None of this needs Windows Autopilot or Entra. For new PCs, see [Windows zero touch without Entra or Autopilot](https://fleetdm.com/guides/windows-zero-touch-without-entra-or-autopilot), and if you're wondering what it costs to drop Microsoft 365 E3 altogether, we [priced it out](https://fleetdm.com/articles/leaving-e3-entirely). Got [HP](https://fleetdm.com/guides/manage-hp-pcs-with-fleet) or [Lenovo](https://fleetdm.com/guides/manage-lenovo-pcs-with-fleet) PCs too? They have guides of their own.

If you're a Fleet customer, we can help you get this set up quickly as part of your Fast Track. Either way, if you run into any trouble, let us know. We're happy to help, and there's a friendly community of kind people who are happy to help too: [get support](https://fleetdm.com/support).

<meta name="articleTitle" value="Manage Dell PCs with Fleet">
<meta name="authorFullName" value="Mike McNeil">
<meta name="authorGitHubUsername" value="mikermcneil">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-27">
<meta name="description" value="Update the BIOS, lock down settings, and check Dell Trusted Device on Windows and Linux with Fleet, no Intune required.">
