# Microsoft's November slmgr fix is optional and mostly Catalog-only, so check for it by build number

*November's Patch Tuesday broke `slmgr.vbs` activation commands on Windows 10, Windows 11 23H2, and Windows Server. Microsoft shipped out-of-band fixes, but most of them won't reach devices unless someone installs them on purpose.*

## Key takeaways

- **Scripted activation is what broke.** After the November 2026 security update, `slmgr.vbs /ipk` and `slmgr.vbs /skms-domain` can fail with error 0x80041008, which hits imaging pipelines, task sequences, and KMS setup scripts.
- **Activated devices stay activated.** Microsoft says devices that were already activated keep their status, so the risk sits with new builds and with any machine whose key or KMS settings need to change.
- **The fixes are opt-in.** The November 13 out-of-band updates are optional cumulative updates, and every affected version except Windows 11 23H2 gets its fix from the Microsoft Update Catalog.
- **Each fix has a specific build number.** Reading the build and update revision from each device tells you which hosts have the fix, instead of assuming the out-of-band package went out with everything else.
- **Activation status is worth checking directly.** A short PowerShell script reports whether Windows is licensed on each device, so a failed activation doesn't sit unnoticed until the grace period runs out.

<a purpose="cta-button" href="https://fleetdm.com/windows">See Windows management in Fleet</a>

Microsoft confirmed on November 13 that its November 2026 security updates break some Windows activation and licensing operations. Devices running the affected commands can return error 0x80041008, "Invalid parameter (SWbemObjectEx)." The same day, Microsoft released out-of-band updates to fix it.

That sounds like a closed loop. It isn't, because of how those updates ship. They're optional, most of them live only in the Microsoft Update Catalog, and the devices that need them most, fresh builds and reimaged machines, are exactly the ones a monthly patch ring won't cover. Which of your devices actually got the fix?

## What broke

The problem is in the Software Licensing Management Tool, `slmgr.vbs`. Microsoft lists two example commands that can fail after the November update:

- `slmgr.vbs /ipk <key>`, which installs a product key
- `slmgr.vbs /skms-domain <domain>`, which points a device at a KMS host by DNS domain

Affected versions are Windows 10 21H2 and 22H2, Windows 11 23H2, and Windows Server 2019, 2022, and 2025. Windows 11 24H2 and later aren't on the list.

The organizations most exposed are the ones that automate activation. That includes task sequences that set a key during deployment, post-install scripts that configure KMS, and golden images prepared with `slmgr`. Microsoft says customers whose images already contain the KMS client key and configuration, and who don't run these commands, aren't expected to be affected.

## Already-activated devices are fine, for now

Microsoft says devices that are already activated remain activated. That limits the blast radius, but it doesn't remove it. KMS activations renew on a cycle and typically last up to 180 days between successful check-ins. A device that needs its key or KMS settings changed, or a new device built from an image that runs `slmgr`, can fail to activate quietly and stay that way until someone notices a watermark or a licensing prompt.

Microsoft also advises against uninstalling the November security update to work around the problem. That update closes a large batch of security flaws, and rolling it back trades a licensing headache for real exposure.

## The fixes don't arrive on their own

The November 13 out-of-band updates are optional cumulative updates. Each one includes the November security content, so a device can install it directly without the original monthly update first. According to IT-Connect's breakdown, here's how they ship:

| Version | Update | Build after install | Channel |
|---|---|---|---|
| Windows 11 23H2 | KB5136483 | 22631.7714 | Windows Update |
| Windows 10 22H2 (ESU) | KB5136493 | 19045.7850 | Microsoft Update Catalog |
| Windows 10 Enterprise LTSC 2021 | KB5136493 | 19044.7850 | Microsoft Update Catalog |
| Windows Server 2025 | KB5136489 | 26100.33828 | Microsoft Update Catalog |
| Windows Server 2022 | KB5136487 | 20348.5873 | Microsoft Update Catalog |
| Windows Server 2019 and Windows 10 Enterprise LTSC 2019 | KB5136490 | 17763.9488 | Microsoft Update Catalog |

Confirm the KB and build for each version against Microsoft's release health page before you deploy.

For everything except Windows 11 23H2, someone has to download the package and push it through WSUS, Configuration Manager, or another deployment tool. That's a manual step outside the normal monthly cadence, which is how a fix ends up on the servers someone remembered and not on the ones they didn't.

## Check the build on each device

The reliable way to know whether a device has the fix is to read its build number and update build revision (UBR) directly. Fleet's agent can read both from the Windows registry. Run this as a live report against your Windows hosts:

```sql
SELECT
  (SELECT data FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\CurrentBuild') AS build,
  (SELECT data FROM registry WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\UBR') AS ubr;
```

Compare each result to the table above. A Windows Server 2022 host with build 20348 needs a UBR of 5873 or higher, and a Windows 11 23H2 device with build 22631 needs 7714 or higher. A later cumulative update that includes the fix will also show a higher UBR, so "at or above" is the right test.

You can also look for the specific package in the `patches` table, for example `SELECT hotfix_id, installed_on FROM patches WHERE hotfix_id = 'KB5136487';`. Treat that as a secondary check, since a later cumulative update can replace the out-of-band package. Neither query has been tested against these specific builds, so confirm the results on a known patched device first.

To track this continuously, save the build check as a Fleet policy for each affected version. Hosts that fail are the ones still waiting on the out-of-band update.

## Confirm activation, not just the patch

The build tells you whether a device has the fix. It doesn't tell you whether activation already failed before the fix arrived. For that, ask Windows directly. [Fleet scripts](https://fleetdm.com/guides/scripts) can run a short PowerShell check on each host:

```powershell
Get-CimInstance SoftwareLicensingProduct -Filter "PartialProductKey IS NOT NULL AND Name LIKE 'Windows%'" |
  Select-Object Name, LicenseStatus
```

A `LicenseStatus` of 1 means licensed. Anything else on a recently built or rekeyed device is worth a look, and once the out-of-band update is installed, you can rerun your activation script on just those hosts.

## Optional updates need the same follow-through

Out-of-band fixes exist because something went wrong with the regular release. When they ship as optional, Catalog-only packages, they fall outside the process most teams use to prove patches landed. Checking the build on each device closes that gap. It turns "we pushed the fix" into a list of hosts that have it and hosts that don't, which is the only version of the answer that holds up.

## See it live

- **Learn more:** Read how to [run scripts with Fleet](https://fleetdm.com/guides/scripts) on Windows, macOS, and Linux.
- **Get a demo:** [Talk to the Fleet team](https://fleetdm.com/contact) about verifying Windows updates across your devices.

## Sources

- Microsoft, Windows release health message center, November 2026 activation known issue.
- BleepingComputer, "Microsoft confirms Windows activation failures after November updates."
- IT-Connect, "Windows Activation Error 0x80041008 Fixed by Microsoft."

<meta name="articleTitle" value="Microsoft's November slmgr fix is optional and mostly Catalog-only, so check for it by build number">
<meta name="authorFullName" value="Aube Paul">
<meta name="authorGitHubUsername" value="robinedev">
<meta name="category" value="industry news">
<meta name="publishedOn" value="2026-11-16">
<meta name="description" value="November's update broke slmgr activation. The fixes are optional and mostly Catalog-only, so check each device's build to confirm they landed.">
