# Windows install and uninstall scripts

Fleet runs install and uninstall scripts as **SYSTEM**, through orbit, in 64-bit Windows PowerShell 5.1, in session 0 with no desktop, with a one-hour limit. Most Windows FMA breakage traces back to that context. The CI validator runs as an interactive admin with UAC off instead ([SKILL.md](../SKILL.md#what-ci-does-and-doesnt-prove)), so this class of bug ships silently unless you test for it.

- [Scope: machine-wide first](#scope-machine-wide-first)
- [When the app has no machine-wide mode](#when-the-app-has-no-machine-wide-mode)
- [The WOW64 trap](#the-wow64-trap)
- [Uninstall must run in the hive that owns the registration](#uninstall-must-run-in-the-hive-that-owns-the-registration)
- [Installers that never exit](#installers-that-never-exit)
- [Install scripts](#install-scripts)
- [Uninstall scripts](#uninstall-scripts)
- [MSIX](#msix)
- [Testing as SYSTEM](#testing-as-system)
- [PowerShell traps](#powershell-traps)

## Scope: machine-wide first

Try the installer's all-users mode first (`ALLUSERS=1`, `/ALLUSERS`, `/allusers`, `/MULTIUSER`, `G2MINSTALLFORALLUSERS=1`). Machine-wide installs land in `Program Files` with an HKLM registration, and everything downstream just works.

**Verify that the switch was honored.** Signal accepts `/S /allusers` and silently installs per-user anyway. An MSI with no `ALLUSERS`, or `ALLUSERS=2` + `MSIINSTALLPERUSER=1`, or a WixUI_Advanced or Velopack MSI under `/quiet`, installs per-user into SYSTEM's profile ([identity-verification.md](identity-verification.md#msi)); pass `ALLUSERS=1 MSIINSTALLPERUSER=""`. Install it on a real host as SYSTEM and look at where the payload and the registration went. Don't trust `installer_scope` either: it selects a winget manifest entry and is sometimes a default rather than a fact. `bluej`, `julia-app`, and `readest` are declared `user` yet install machine-wide because their scripts handle SYSTEM deliberately.

**Fixing a per-user MSI on hosts that already have the stranded copy.** With the same ProductCode already installed per-user, `msiexec /i … ALLUSERS=1` just reconfigures the existing per-user product in place, and vendor custom actions may read HKCU and force per-user again. Remove the stranded copy first: `(New-Object -ComObject WindowsInstaller.Installer).ProductInfo($code, 'AssignmentType')` returns `0` for per-user (SYSTEM's own) and `1` for per-machine (`ProductsEx` returns nothing from PowerShell COM, and `S-1-5-18` as the SID throws). Then install (`tower_install.ps1`).

**Apps that exist in both scopes** (VS Code and its forks, browsers, Zoom, Dropbox, and others) can leave a stale copy in the other scope after Fleet installs one. The exists and patched queries stay scope-blind on purpose ([queries.md](queries.md#exists-name-and-publisher-matching)); the install script removes the other scope's copy, best-effort, preferring the vendor's quiet uninstall string. Delete a per-user ARP key only after verifying the uninstaller actually removed the app, or the policy reads green over a copy that's still there.

## When the app has no machine-wide mode

Electron/NSIS (oneClick without per-machine), Squirrel, and some Inno apps only ever install into the running user's profile. Run as SYSTEM, they land in `C:\Windows\system32\config\systemprofile\AppData\Local\...`, where **no signed-in user can launch them**: the install "succeeds" and is useless. The symptoms vary and none says "wrong scope": Notion crashes (`0xC0000005`), `amazon-chime` hangs until the timeout, `granola` returns 0 into SYSTEM's profile, MEGAsync exits 2.

The fix is to hand the installer to the signed-in user via a scheduled task. `figma`, `slack`, `brave`, `arc`, `postman`, `notion`, `todoist`, and others follow this shape:

```powershell
$owner = Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' -ErrorAction SilentlyContinue |
    Invoke-CimMethod -MethodName GetOwner -ErrorAction SilentlyContinue |
    Where-Object { $_.User } | Select-Object -First 1
if (-not $owner) { Throw "<App> installs per user and no user is signed in to this host. Sign in and try again." }
$userAccount = "$($owner.Domain)\$($owner.User)"
```

- **Stage the installer where only that user and admins can read it.** Fleet downloads to `C:\Windows\SystemTemp\<random>`, which only SYSTEM and Administrators can read. Copy it (`Copy-Item … -ErrorAction Stop`) into a new directory under `$env:ProgramData` and grant the target user ReadAndExecute, then delete it in `finally`. Don't stage in a world-writable location such as `$env:PUBLIC`: another local user could swap the file before the task runs it as the target user.
- **Run install and uninstall tasks at the same run level.** An installer that also demands elevation (`portfolioperformance`: `ERROR_ELEVATION_REQUIRED`, `0x800702E4`) needs `-RunLevel Highest`, which only works if the user is an admin. If the install task runs elevated but the uninstall task doesn't, the uninstaller shows a UAC prompt on the user's desktop and the task times out every time (NVM for Windows). CI has UAC off, so it never sees this.
- **Treat a task timeout as a failure.** A fallback that force-deletes the folder and registration and exits 0 hides a hang.
- **`/currentuser` can be counterproductive.** `granola` shipped `/S /currentuser`; plain `/S` as the signed-in user is what lands it correctly.
- **Installers that register per-user scheduled tasks at install time** fail under SYSTEM (Firefox's Default Browser Agent: `0x80070534`, then a crash dialog on first launch). Look for a vendor switch that skips the registration (`/RegisterDefaultAgent=false`) instead of scripting around it.
- **Squirrel**: before reinstalling, stop any process under `%LOCALAPPDATA%\<App>` and fail fast if one won't die. If `Update.exe --install` fails, `Setup.exe` shows a modal error dialog even with `--silent`, and the task never finishes. `Update.exe --uninstall -s` leaves `app-<ver>`, `.dead`, and `Update.exe` behind, so remove the folder afterward.

## The WOW64 trap

Most Electron NSIS stubs are **32-bit** (PE machine `0x014C`; check with `scripts/pe_info.py`). Run as SYSTEM, the WOW64 file system redirector rewrites their `%LOCALAPPDATA%` writes into `C:\Windows\SysWOW64\config\systemprofile\...`, but the `UninstallString` they record names the unredirected `C:\Windows\system32\...` path. Uninstall scripts run in 64-bit PowerShell, where no redirection applies, so that path doesn't resolve ("The system cannot find the file specified"). Any uninstall script for a per-user app needs this fallback, so hosts already carrying a stranded install can be cleaned up:

```powershell
if (-not (Test-Path -LiteralPath $exePath)) {
    $redirected = $exePath -replace '(?i)\\system32\\', '\SysWOW64\'
    if ($redirected -ne $exePath -and (Test-Path -LiteralPath $redirected)) { $exePath = $redirected }
}
```

Two failures from the same redirection:
- **A 32-bit InstallShield setup** caches its MSI under redirected `systemprofile`, and the 64-bit msiserver can't open the path: exit 1619. Add `/b"<dir>"` before `/v"…"`, with the dir under `[IO.Path]::GetTempPath()` (not redirected for SYSTEM), and remove it in `finally` (`sonos_install.ps1`).
- **An NSIS `InstallLib … LIBRARY_X64`** turns redirection off to copy a 64-bit DLL into the per-user dir, hits the real (nonexistent) `System32\config\systemprofile` path, and aborts with exit 2 (MEGAsync). Use the vendor's all-users switch.

## Uninstall must run in the hive that owns the registration

A per-user app's ARP entry lives in the installing user's hive, so a SYSTEM-context script must enumerate every hive, not `HKCU` (which *is* SYSTEM's hive when running as SYSTEM):

```powershell
foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
    if ($hive.Name -match '_Classes$') { continue }
    $roots.Add("Registry::$($hive.Name)\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall")
    $roots.Add("Registry::$($hive.Name)\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall")
}
```

Finding the entry is not enough. **These uninstallers read the directory to remove out of the hive of whoever runs them**, so run as SYSTEM against a real user's install they exit **0 and delete nothing** (Signal left 473 MB behind). Run the uninstaller as the user who owns the entry: derive the SID from the key path with `'HKEY_USERS\\(S-1-5-21-[\d-]+|S-1-12-1-[\d-]+)\\'`, translate it to an account, and launch via a scheduled task. Match both prefixes: Entra ID users are `S-1-12-1-`, and a user hive that falls through to SYSTEM lets a standard user plant an `UninstallString` that runs as SYSTEM. For the same reason, delete a user's leftover folder as that user (`cmd /d /c rd /s /q` in the task), never as SYSTEM. Everything else (HKLM, `S-1-5-18`/`-19`/`-20`, `.DEFAULT`) runs directly as SYSTEM: that's its correct context, and a standard user can't write to it.

Details that avoid per-app data:
- **Stop processes by install directory, not by name.** Most apps aren't running when you look, so a name list is usually empty. Matching on the directory also leaves another user's copy alone.
- **Sweep shortcuts in the owning profile only, matching `TargetPath` or `WorkingDirectory`.** Read as SYSTEM, a user's `.lnk` `TargetPath` resolves through SYSTEM's known folders (`C:\Users\bob\AppData\Local\Programs\X\x.exe` comes back as `…\systemprofile\…`), so a TargetPath-only sweep never matches the user's shortcuts, and cleaning a stranded SYSTEM copy deletes every user's shortcuts. `WorkingDirectory` keeps the real path (`ollama_uninstall.ps1`'s `Remove-Shortcuts`).
- **Legacy registrations** can lack `InstallLocation` and quote nothing (RStudio's pre-NsisMultiUser key under `WOW6432Node`): fall back to the parent of `UninstallString`.

## Installers that never exit

A silent install that waits on something invisible runs until fleetd's one-hour limit (CI's limit is about 11 minutes: 10 plus a 1-minute wait delay) with empty output and the installer file still locked. In session 0 nobody can click the dialog. Known causes:

| Cause | Examples | Fix |
|---|---|---|
| Silent switch ignored (wrong framework) | darktable (`/S` to an Inno installer) | Confirm the framework ([identity-verification.md](identity-verification.md#exe-installers)) |
| Inno `[Run]` postinstall without `skipifsilent`; installer launches the app | darktable, iMazing Converter, Mozilla VPN | Poll-and-kill (below) |
| NSIS MessageBox with no `/SD` default under `/S` | JetBrains IDEs on a same-version reinstall ("installation dir is not empty"), Android Studio while `studio64.exe` runs, Pale Moon's AVX check on ARM64, x64 builds on ARM64 | Remove the cause first: stop processes under the version dir and delete it with retries (fail if files remain); stop by name for name-based checks; `/IGNORE_ARCH` (JetBrains) on ARM64; fail fast on ARM64 when there's no build |
| Inno `[Code]` plain `MsgBox` (not suppressible) | KNIME's "remove current installation?" on upgrade, Rtools' "target directory already exists" | Run the existing uninstaller first and wait for its process tree and ARP key to disappear |
| NSIS running-app abort in `.onInit` (exit 2, fast) | RStudio (`nsProcess` by name) | Stop by name and by install dir, retry once on exit 2 |
| Inno can't close a session-1 app from SYSTEM | "RestartManager … Session Mismatch": exit 5, exit 1 for AppMutex apps, or exit 0 + pending reboot then exit 8 forever (GIMP) | Stop the app (by install dir) before installing |
| Squirrel install failure dialog | Fellow, Discord-family apps | Stop processes under the app folder first |
| A self-extractor wrapping a setup that never returns as SYSTEM | Paint.NET's SFX → `SetupShim.exe` | Use the vendor's MSI (`nested_installer_type: wix`) |

The **poll-and-kill** pattern for installers that finish their work but don't exit: `Start-Process -PassThru`, poll both Uninstall hives for the DisplayName the exists query matches, then stop the installer's whole process tree (it holds a lock on the installer file) and exit 0 (`darktable_install.ps1`, `mozilla-vpn_install.ps1`). Use a `[Diagnostics.Stopwatch]` for deadlines, not `Get-Date`. When a "registered?" check accepts any matching ARP entry, exclude entries that existed before the installer started: the runner image already ships some apps (Rtools), and a pre-existing entry turns a hang into a false pass.

To prove which dialog is hanging on a VM, enumerate window titles and text (`EnumWindows`/`EnumChildWindows`) from a SYSTEM scheduled task, or capture the screen and look for `consent.exe`. Don't reach for a longer timeout or a validator cap before ruling out a dialog.

## Install scripts

- **Wait for the uninstall registry entry, not for files on disk.** Installers write files before they register the ARP entry, and that entry is what osquery's `programs` reads. Poll both the native and `WOW6432Node` hives for the DisplayName (and DisplayVersion) the exists query matches, and keep polling to the deadline even after the setup process exits, because child processes finish the registration (`onedrive_install.ps1`).
- **Reboot-required is success**: map `3010` and `1641` to 0 with `$expectedExitCodes = @(0, 1641, 3010)`. Never drop `/norestart`: Fleet installs must not reboot the host. Two exceptions: a Burn bundle that exits 3010 while its ARP key is still there did nothing (a restart is pending from an earlier run: VC++ redistributable), so exit 1 with "restart pending"; and Google Drive's 1610 means "restart before reinstalling", which Fleet can't fix, so explain it in the output.
- **An MSI that refuses reboot suppression** fails with 1603 right after `Action start: LaunchConditions`. Read the verbose log (`/lv`) for the property the vendor offers to defer the reboot-requiring step (`egnyte_install.ps1`: `ED_UPDATE_ON_BOOT=1`).
- **A custom bootstrapper that rejects standard switches** (Okta Verify 7 exits 1 on a bare `/norestart`): pass only what it accepts (`/quiet`, `/log <path>`), and print the Burn log's tail on failure.
- **Same-version reinstalls are a real path** (Fleet retries, admins reinstall): an MSI custom action that launches the app aborts with 1602 (`DO_NOT_LAUNCH=1` for WireGuard); an installer that leaves state behind can refuse the next install (PostgreSQL's kept `data` dir triggers a broken upgrade path: move an orphaned `data` aside first).
- **Replacing a product that installs side by side** (Audacity's Inno → MSI): remove the legacy install *after* the new one succeeds, never before, so a failed install leaves the old app working.
- **Prerequisites the quiet path skips** (Tower's MSI installs the .NET runtime only in its UI sequence): check for them and print a clear message.
- **A transient self-extractor crash** (`-1073741819`, `0xC0000005`, before extracting anything) passes on rerun. Confirm it's transient first ([validator-failures.md](validator-failures.md#transient-failures-and-reruns)); if a retry is warranted, retry up to 3 times only on that code (`evernote_install.ps1`).
- **Stage the installer with `Copy-Item … -ErrorAction Stop`** when a scheduled task runs it.

## Uninstall scripts

**Parse `UninstallString` defensively.** It comes in three shapes; this broke every JetBrains app, whose `C:\Program Files\JetBrains\PhpStorm 2026.1.2\bin\Uninstall.exe` is unquoted *and* contains spaces:

```powershell
if ($u -match '^\s*"([^"]+)"\s*(.*)$') {            # quoted
} elseif ($u -match '(?i)^\s*(.+?\.exe)\s*(.*)$') { # unquoted, may contain spaces: capture through .exe
} elseif ($u -match '^\s*(\S+)\s*(.*)$') {          # bare token (e.g. MsiExec.exe /X{GUID})
}
```

**Prefer `QuietUninstallString`** when it exists.

**Match the exact entry.** A pattern like `REAPER*` uninstalls whichever arch's copy sorts first, including unmanaged ones. Match the DisplayName (and publisher) the exists query uses. Strip NULs on both sides when matching (`($key.DisplayName -replace "`0", "").Trim()`): some installers pad `REG_SZ` values (Fork).

**MSI "maintenance form."** Some MSIs record `MsiExec.exe /I{ProductCode}` rather than `/X`. A helper that prepends `/X` produces `MsiExec.exe /X /I{GUID}`, which hangs and fails. Resolve the ProductCode (the key name if it's a GUID, else a `\{[0-9A-Fa-f-]+\}` match) and run a clean `msiexec /x {GUID} /qn /norestart`. Custom uninstall scripts get only `$PACKAGE_ID` substituted; hardcode an UpgradeCode and use `WindowsInstaller.Installer`'s `RelatedProducts` to resolve it.

**Burn bundles register twice.** The inner MSI keeps an ARP key with the same DisplayName and Publisher, and its `UninstallString` is `MsiExec.exe /I{guid}`. Select the bundle's key with `($key.BundleUpgradeCode -or $key.QuietUninstallString)` (Burn writes both; Windows Installer writes neither), then use its `QuietUninstallString`. Don't glob `Package Cache` by filename: the bundle's ProductCode changes every release (`jabra-direct_uninstall.ps1`). A completed Burn uninstall deletes its key even when it needs a restart, so "3010 and the key is still there" means nothing was removed.

**NSIS uninstallers relaunch themselves from `%TEMP%`** (`helper.exe` → `Un_A.exe`), so the process you waited on exits while removal is still in flight. Poll until *either* the install directory or the registration disappears, then force-remove any remainder (Notion: 120 s vs 6 s). An NSIS uninstall run in session 0 can remove the files but leave the ARP key. Session 0 is where fleetd runs scripts, so the exists query keeps reporting the app: once the install directory is gone, delete the exact matched uninstall key yourself (`qemu_uninstall.ps1`), then confirm it's gone before exiting 0.

## MSIX

- orbit downloads to `C:\Windows\SystemTemp\<random>` (SYSTEM and Administrators only), and `Add-AppxPackage -Path` opens the file as the calling user, so a per-user registration task can't read it unless the user is an elevated admin; standard users fail silently while the install still exits 0 after provisioning. Stage a copy in a protected `%ProgramData%\fleet-<app>-<guid>` directory with the user granted ReadAndExecute.
- Uninstall in two steps, as `slack_uninstall.ps1` does: `Remove-AppxProvisionedPackage -Online` so new profiles don't get the app, then `Remove-AppxPackage -AllUsers`. Don't count on `-AllUsers` to deprovision. It removes the app now for signed-in users and writes an `AppxAllUserStore\EndOfLife\<SID>\<PackageFullName>` marker. Reinstalling the same version for that user then shows "Installed (pending removal)" and the app vanishes later. The install script must clear `EndOfLife\*\<PackageName>_*` markers. Signed-out users' copies stay listed until their next sign-in.
- Presence for the uninstall's success check: provisioned, or registered for a loaded-hive SID (that's what osquery's `programs` reads). Add `-ErrorAction Stop` to every `Get-Appx*` call: a non-terminating error returns nothing, and the script reports success with the app still installed. Prove it fails closed by shadowing the cmdlet with a function that writes an error.
- `Get-AppxProvisionedPackage -Online` objects have no `PackageFamilyName` property; filter on `DisplayName` or `PackageName`.
- An MSI whose custom action installs a per-user MSIX fails as SYSTEM (eM Client 11: `0x80070057` → 1603). The shipped design builds an MSI transform that drops that action, installs as SYSTEM, then runs the vendor's per-user step in a task per signed-in user.

## Testing as SYSTEM

**CI can't catch SYSTEM-context bugs.** If your change concerns scope, profile location, dialogs, or uninstall path resolution, a green run proves nothing. Use a Windows VM. With Parallels, `prlctl exec "<vm>" <cmd>` runs as `NT AUTHORITY\SYSTEM` in session 0, exactly orbit's context; `prlctl exec "<vm>" --current-user <cmd>` runs as the desktop user, unelevated. After install, assert: the registration is under a real user SID or HKLM as intended; there is **nothing** under `S-1-5-18`/`.DEFAULT`; the payload is where you expect. After uninstall, assert no registration, directory, or shortcut remains. Then run the queries ([SKILL.md](../SKILL.md#validating-the-queries)). If you have no VM, say in the PR that scope is unverified.

Driving a VM reliably:
- Run PowerShell from a `.ps1` with `-File`. `prlctl exec` eats double quotes, nested single quotes in SQL break `-Command`, and an `-EncodedCommand` over about 4 KB fails with "Unable to open new session" (which looks like a VM fault but isn't).
- Launch GUI apps through an elevated interactive scheduled task, not `--current-user` (apps that require elevation hang on UAC there). Register tasks with `New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries`: a `schtasks /create` task inherits "don't start on battery" and sits at Queued on a laptop host. `LastTaskResult 267009` means still running.
- Hash-check downloads inside the guest: they can truncate silently. Copy a verified file from the host if needed.
- Use a `Stopwatch` for deadlines: the guest clock can jump for a few minutes after the VM resumes.
- A shared VM may be running other people's tests: check `Get-ScheduledTask | ? State -eq Running` and recent MsiInstaller events, use your own working directory and task names, and collect results in the same step as the install. A 1618 there usually means someone else's msiexec.
- When a GUI-free step takes about 300 s and then fails, capture the screen and look for a UAC prompt (`consent.exe`).

**Mind the test host's architecture.** An ARM64 VM (Parallels on Apple silicon) can only produce false *failures*, never false passes, for x64 installers: per-user/SYSTEM-profile behavior and `System32` → `SysWOW64` redirection are identical, but x64-only installers with an architecture check refuse to run (Inno's `/LOG` says so), kernel drivers and VPNs fail, and x86 processes see no AVX. Note in the PR that the app needs an x64 host, and if the app is frozen, get one CI run with a temporary unfreeze commit ([validator-failures.md](validator-failures.md#revert-and-freeze)). ARM64 hosts are real customer hosts too: an installer that shows an architecture or CPU-feature dialog under `/S` hangs there for an hour.

## PowerShell traps

Each of these silently produced a wrong answer in practice, not an error.

- **`Start-Process -PassThru` + `WaitForExit($ms)` can leave `ExitCode` empty**, even once `HasExited` is `$true`. Use `-Wait -PassThru` when you can, and treat any blank exit code as "not measured": this once reported an entire 30-app run as failing. Call `winget` directly (`& winget …; $LASTEXITCODE`), not through `Start-Process`, whose exit code is `$null` with redirected streams.
- **PowerShell 7's `Start-Process -Wait` doesn't wait for descendants**; Windows PowerShell 5.1's does. Scripts run in 5.1, but harnesses you write may not.
- **`Exit` with a code above `Int32.MaxValue` exits 0 in Windows PowerShell 5.1.** A crashed installer (`0xC0000005` = 3221225477) or a task HRESULT in `LastTaskResult` (`0x80070002`) reads as success. Put `if ($exitCode -gt [int]::MaxValue) { $exitCode = 1 }` before the final `Exit`.
- **`Copy-Item` of a missing `$env:INSTALLER_PATH` doesn't throw inside `try`.** It's a non-terminating error, so the script goes on to schedule a task for a file that isn't there. Add `-ErrorAction Stop`.
- **`'\b/S\b'` never matches a switch preceded by a space.** Neither the space nor the `/` is a word character. Anchor on whitespace: `'(?i)(^|\s)/S($|\s)'`.
- **A single result is a scalar, so `.Count` is `$null`.** Wrap in `@(...)` before counting. `ConvertFrom-Json` of an empty osquery result yields one `$null`: filter nulls before counting.
- **`New-ScheduledTaskAction -Argument ""` is rejected outright.** Omit the parameter when there are no arguments.
- **Don't wait for a scheduled task to be observed `Running`**: a fast task enters and leaves that state between polls. Poll until `LastTaskResult -ne 267009` (`SCHED_S_TASK_RUNNING`), or until `LastRunTime` advances.
- **A nested `"` inside a `$(...)` subexpression terminates the surrounding string.** Build the value into a variable first. A parse error means the script never ran at all. Parse-check before trusting a run: `[System.Management.Automation.Language.Parser]::ParseFile($p,[ref]$null,[ref]$errors)`.
- **Carry over every element of a multi-value `ArgumentList`.** Taking only the first of `"/silent", "/skip-app-launch"` would have let Spotify launch itself after installing.
- **Overriding `$env:ProgramFiles` for a child process doesn't stick**: Windows resets it per process. To test a path-dependent branch, run a copy of the script with that path substituted.
