
[YellowKey (CVE-2026-45585)](https://msrc.microsoft.com/update-guide/vulnerability/CVE-2026-45585) is an unpatched BitLocker bypass on Windows 11 and Server 2025 (and, per the researcher's disclosure, Server 2022). Microsoft shipped a mitigation on May 20, 2026 and no full patch is out yet. This pattern flags exposed hosts in Fleet, reports on them daily, and applies Microsoft's mitigation through a script.

## The threat

YellowKey abuses `autofstx.exe` in the Windows Recovery Environment. With brief physical access, an attacker drops a crafted `FsTx` directory on a USB stick or the EFI System Partition and reboots into WinRE. autofstx replays NTFS transaction logs that delete `winpeshl.ini`, so WinRE falls back to `cmd.exe` with the BitLocker volume unlocked. Windows 10 ships a different WinRE component and is not affected.

USB-block GPOs and BIOS USB-boot blocks do not stop it: WinRE ignores the OS USB policy and the attack does not boot from the stick. TPM-only BitLocker is the target. Microsoft's mitigation strips `autofstx.exe` from WinRE's `BootExecute` chain. TPM + PIN blocks the published proof of concept but not the researcher's withheld variant, so treat it as raising attacker cost.

## What you'll deploy

| File | Role |
|---|---|
| [`allenhouchins/fleet-extensions/windows_yellowkey`](https://github.com/allenhouchins/fleet-extensions/tree/main/windows_yellowkey) | osquery extension upstream; exposes the `windows_yellowkey` table |
| [`allenhouchins/fleet-extensions/windows_yellowkey/install-windows-yellowkey-extension.ps1`](https://raw.githubusercontent.com/allenhouchins/fleet-extensions/main/windows_yellowkey/install-windows-yellowkey-extension.ps1) | Installs the extension (lives upstream with the binary) |
| [`docs/solutions/windows/reports/windows-yellowkey.reports.yml`](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/solutions/windows/reports/windows-yellowkey.reports.yml) | Daily per-host report |
| [`docs/solutions/windows/policies/windows-yellowkey-extension.policies.yml`](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/solutions/windows/policies/windows-yellowkey-extension.policies.yml) | Keeps the extension installed |
| [`docs/solutions/windows/scripts/mitigate-windows-yellowkey.ps1`](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/solutions/windows/scripts/mitigate-windows-yellowkey.ps1) | Applies Microsoft's mitigation |

## Detect

The extension reads OS, WinRE state, BitLocker key protectors, and the `BootExecMitigated` marker on every query, then returns one `state` per host. There is no internal cache or staleness gate. A host returns a row once the extension loads; the policy below keeps it loaded.

| state | Meaning |
|---|---|
| `not_affected` | Windows 10, older Windows Server, or unrecognised SKU |
| `mitigated` | autofstx stripped, marker set |
| `mitigated_winre_off` | WinRE disabled |
| `bitlocker_off` | no protected BitLocker volume |
| `exposed` | affected OS, BitLocker on, no mitigation |

`winre_enabled` is a separate column and can be `Enabled`, `Disabled`, or `unknown`. When BitLocker is on and WinRE is unknown the host is reported `exposed` (the safe default).

The report is one line:

```
SELECT state, state_reason, needs_action, winre_enabled, tpm_only, mitigated FROM windows_yellowkey;
```

## Mitigate

`mitigate-windows-yellowkey.ps1` adapts [Microsoft's reference script](https://msrc.microsoft.com/update-guide/vulnerability/CVE-2026-45585). It mounts the WinRE image with `reagentc /mountre`, loads the offline SYSTEM hive, strips `autofstx` from every ControlSet's `BootExecute`, unmounts with `/commit` if changes were made (otherwise `/discard`), then runs `reagentc /disable` and `/enable` to re-seal the BitLocker measurement chain.

The script verifies each ControlSet by read-back and writes `HKLM\SOFTWARE\Fleet\YellowKey\BootExecMitigated = 1` only when every one is clean. Exit codes: `0` done, `3` OS not affected, `4` failed. There is no opt-in gate, since Microsoft's strip is safe on every affected host, and no unmitigate path: when Microsoft ships a patch, apply it and clear the marker.

## Deploy

The `windows-yellowkey-extension` policy checks `osquery_registry` for the `windows_yellowkey` table and passes when it is loaded. Failing hosts run [`install-windows-yellowkey-extension.ps1`](https://raw.githubusercontent.com/allenhouchins/fleet-extensions/main/windows_yellowkey/install-windows-yellowkey-extension.ps1) from Allen's repo. The installer downloads the architecture-matching release, validates the PE header, places the binary, registers it in `C:\Program Files\osquery\extensions.load`, and restarts the `Fleet osquery` service. osqueryd autoloads the extension on the next start.

Allen's CI rebuilds and republishes `releases/latest` on every push to `main`, so failing hosts pick up new binaries automatically with no edits to this repo.

## Roll it out

The reports, policy, and mitigation script live in [fleetdm/fleet](https://github.com/fleetdm/fleet/tree/main/docs/solutions/windows); the installer lives in [allenhouchins/fleet-extensions](https://github.com/allenhouchins/fleet-extensions/tree/main/windows_yellowkey). Drop all four into your GitOps repo and reference them from your fleet config:

```yaml
policies:
  - path: ../windows/policies/windows-yellowkey-extension.policies.yml
reports:
  - path: ../windows/reports/windows-yellowkey.reports.yml
controls:
  scripts:
    - path: ../windows/scripts/install-windows-yellowkey-extension.ps1
    - path: ../windows/scripts/mitigate-windows-yellowkey.ps1
```

1. Apply with `fleetctl gitops -f fleets/workstations.yml`.
1. The policy runs on the next interval; failing hosts run the installer (default 60 seconds for the first check).
1. Open the report. Run `mitigate-windows-yellowkey.ps1` against `exposed` hosts from Fleet > Controls > Scripts.
1. Re-run the report. Those hosts move to `mitigated`.

## If a host stays failing

Check the host in an elevated PowerShell:


1. The loader file at osquery's compiled default path lists the binary
```powershell
Get-Content 'C:\Program Files\osquery\extensions.load'
```
2. The binary is in place (~5.6 MB amd64, ~5.2 MB arm64)
```powershell
Get-Item 'C:\Program Files\osquery\extensions\windows_yellowkey.ext.exe' | Select-Object Length, LastWriteTime
```
3. The script's last run is recorded under ***Fleet > Hosts > [host] > Activity > Scripts***

If `extensions.load` is missing or does not list the binary, the script did not finish; read its stdout under Activity > Scripts. If both look right but the table is still missing, search `C:\Windows\System32\config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log` for `unsafe permissions` or `Timed out waiting for extension`. Re-running the installer re-asserts the ACLs and the loader entry.

## Operational notes

- Inspect one host with `SELECT * FROM windows_yellowkey`. If the extension will not load, `reagentc /info` and `Get-BitLockerVolume` give the same signals without osquery.
- `reagentc /disable` also removes push-button reset, in-WinRE BitLocker recovery, System Restore from boot, and Recovery Drive restore. Re-enable with `reagentc /enable` before you need them.
- Move high-risk hosts to TPM + PIN where the threat model includes physical access. It blocks the public PoC and raises attacker cost on the withheld variant.

<meta name="articleTitle" value="Detect and mitigate the YellowKey BitLocker bypass with Fleet">
<meta name="authorFullName" value="Adam Baali">
<meta name="authorGitHubUsername" value="AdamBaali">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-06-02">
<meta name="description" value="Use Fleet to detect and fix CVE-2026-45585, the YellowKey BitLocker bypass, across your Windows fleet with policies and scripts.">
