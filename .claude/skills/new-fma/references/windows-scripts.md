# Windows install and uninstall scripts

Fleet runs install and uninstall scripts as **SYSTEM**, through orbit, in 64-bit Windows PowerShell 5.1. Most Windows FMA breakage traces back to that context. The CI validator runs as an interactive admin instead ([SKILL.md](../SKILL.md#what-ci-does-and-doesnt-prove)), so this class of bug ships silently unless you test for it.

- [Scope: machine-wide first](#scope-machine-wide-first)
- [When the app has no machine-wide mode](#when-the-app-has-no-machine-wide-mode)
- [The WOW64 trap](#the-wow64-trap)
- [Uninstall must run in the hive that owns the registration](#uninstall-must-run-in-the-hive-that-owns-the-registration)
- [Install scripts](#install-scripts)
- [Uninstall scripts](#uninstall-scripts)
- [Testing as SYSTEM](#testing-as-system)
- [PowerShell traps](#powershell-traps)

## Scope: machine-wide first

Try the installer's all-users switch first (`ALLUSERS=1`/`2`, `/ALLUSERS`, `/allusers`, `G2MINSTALLFORALLUSERS=1`). Machine-wide installs land in `Program Files` with an HKLM registration, and everything downstream just works.

**Verify that the switch was honored.** Signal accepts `/S /allusers` and silently ignores it, still installing per-user. Install it on a real host as SYSTEM and look at where the payload and the registration actually went. Don't trust the input's `installer_scope` either: it selects a winget manifest entry and is sometimes a default rather than a fact. `bluej`, `julia-app`, and `readest` are declared `installer_scope: user` yet install machine-wide to `Program Files` under HKLM, because their custom scripts already handle SYSTEM deliberately (`bluej` passes `ALLUSERS=2`). Scope alone is not evidence of a bug.

## When the app has no machine-wide mode

Electron/NSIS/Squirrel apps and some Inno apps only ever install into the running user's profile. Run as SYSTEM, they land in `C:\Windows\system32\config\systemprofile\AppData\Local\...`, where **no signed-in user can launch them**: the install "succeeds" and is useless. The symptoms vary, and none of them says "wrong scope": Notion's installer crashes outright (`0xC0000005`, installing nothing), `amazon-chime` hangs until the timeout, and `granola` returns 0 and installs into SYSTEM's profile.

The fix is to hand the installer to the signed-in user via a scheduled task. `figma`, `slack`, `brave`, `arc`, `postman`, `notion`, and others follow this shape:

```powershell
$owner = Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' -ErrorAction SilentlyContinue |
    Invoke-CimMethod -MethodName GetOwner -ErrorAction SilentlyContinue |
    Where-Object { $_.User } | Select-Object -First 1
if (-not $owner) { Throw "<App> installs per user and no user is signed in to this host. Sign in and try again." }
$userAccount = "$($owner.Domain)\$($owner.User)"
# Fleet's installer directory is not readable by that user - stage a copy under $env:PUBLIC.
```

Two variations worth knowing:
- **The installer also demands elevation.** `portfolioperformance` fails as a plain user with `ERROR_ELEVATION_REQUIRED` (`0x800702E4`) *and* strands itself when run as SYSTEM. Add `-RunLevel Highest` to `New-ScheduledTaskPrincipal`; it only works if the signed-in user is an admin.
- **`/currentuser` can be counterproductive.** `granola` shipped `/S /currentuser`; plain `/S` as the signed-in user is what lands it correctly.

## The WOW64 trap

Most Electron NSIS stubs are **32-bit** (PE machine `0x014C`; check with `scripts/pe_info.py` on the first few MB). Run as SYSTEM, the WOW64 file system redirector rewrites their `%LOCALAPPDATA%` writes into `C:\Windows\SysWOW64\config\systemprofile\...`, but the `UninstallString` they record still names the unredirected `C:\Windows\system32\...` path. Uninstall scripts run in 64-bit PowerShell, where no redirection applies, so that path doesn't resolve:

```
Error running uninstaller: This command cannot be run due to the error: The system cannot find the file specified.
```

Any uninstall script for a per-user app needs this fallback, so that hosts already carrying a stranded install can be cleaned up:

```powershell
if (-not (Test-Path -LiteralPath $exePath)) {
    $redirected = $exePath -replace '(?i)\\system32\\', '\SysWOW64\'
    if ($redirected -ne $exePath -and (Test-Path -LiteralPath $redirected)) { $exePath = $redirected }
}
```

## Uninstall must run in the hive that owns the registration

A per-user app's ARP entry lives in the installing user's hive, so a SYSTEM-context script must enumerate every hive, not `HKCU` (which *is* SYSTEM's hive when running as SYSTEM):

```powershell
foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
    if ($hive.Name -match '_Classes$') { continue }
    $roots.Add("Registry::$($hive.Name)\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall")
    $roots.Add("Registry::$($hive.Name)\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall")
}
```

Finding the entry is not enough. **These uninstallers read the directory to remove out of the hive of whoever runs them**, so run as SYSTEM against a real user's install they exit **0 and delete nothing**. Signal left 473 MB behind while reporting success. Run the uninstaller as the user who owns the entry: derive the SID from the key path with `'HKEY_USERS\\(S-1-5-21-[\d-]+|S-1-12-1-[\d-]+)\\'`, translate it to an account, and launch via a scheduled task. Match both prefixes: Entra ID users are `S-1-12-1-`, and a user hive that falls through to SYSTEM lets a standard user plant an `UninstallString` that runs as SYSTEM. Everything else (HKLM, the service SIDs `S-1-5-18`/`S-1-5-19`/`S-1-5-20`, and `.DEFAULT`) runs directly as SYSTEM: that is its correct context, and a standard user can't write to it. The stranded legacy installs are the ones under `S-1-5-18`/`.DEFAULT`.

Two details that avoid per-app data:
- **Stop processes by install directory, not by name.** Most apps aren't running when you look, so a process-name list is usually empty and useless. Matching on the directory also leaves another user's copy of the same app alone.
- **Sweep shortcuts by resolving each `.lnk` target** against the removed directory, rather than by filename. That catches the vendor subfolders that installers create under `Start Menu\Programs`.

## Install scripts

- **Wait for the uninstall registry entry, not for files on disk.** Installers write files before they register the ARP entry, and that entry is what osquery's `programs` table (and the validator) reads. A script that returns as soon as the exe exists races detection and fails intermittently with "App version X was not found by osquery". Poll both the native and `WOW6432Node` hives for the DisplayName the exists query matches, and keep polling to the deadline even after the setup process exits, because child processes finish the registration (`onedrive_install.ps1`).
- **An installer that never exits.** Some installers launch the app, or an Inno `[Run]` postinstall entry without `skipifsilent`, and `Start-Process -Wait` then blocks until the 10-minute timeout, with empty output. No Inno switch skips `[Run]`. Use poll-and-kill: `Start-Process -PassThru`, poll the registry until the app is registered, stop the installer's process tree (it holds a lock on the installer file), and exit 0 (`darktable_install.ps1`, `mozilla-vpn_install.ps1`).
- **Reboot-required is success.** Map `3010` and `1641` to 0 with `$expectedExitCodes = @(0, 1641, 3010)` (`okta_verify_install.ps1`). Never drop `/norestart` to avoid them: Fleet installs must not reboot the host.
- **An MSI that refuses reboot suppression** fails with 1603 right after `Action start: LaunchConditions`, because `/norestart` becomes `REBOOT=ReallySuppress`. Read the verbose log (`/lv`) for the property the vendor offers to defer the reboot-requiring step, and pass it in a custom script (`egnyte_install.ps1`: `ED_UPDATE_ON_BOOT=1`).
- **A transient self-extractor crash.** electron-builder NSIS installers occasionally die with `-1073741819` (`0xC0000005`) before extracting anything, then pass on rerun. Before adding a retry, confirm the same installer passed elsewhere (or compare `7zz l` of the old and new installer). Then retry up to 3 times, only on that exit code (`evernote_install.ps1`). A deterministic failure never passes; don't paper over it with retries.
- **Stage the installer with `Copy-Item … -ErrorAction Stop`** when a scheduled task runs it. See [PowerShell traps](#powershell-traps).

## Uninstall scripts

**Parse `UninstallString` defensively.** It comes in three shapes; this broke every JetBrains app, whose `C:\Program Files\JetBrains\PhpStorm 2026.1.2\bin\Uninstall.exe` is unquoted *and* contains spaces:

```powershell
if ($u -match '^\s*"([^"]+)"\s*(.*)$') {            # quoted
} elseif ($u -match '(?i)^\s*(.+?\.exe)\s*(.*)$') { # unquoted, may contain spaces: capture through .exe
} elseif ($u -match '^\s*(\S+)\s*(.*)$') {          # bare token (e.g. MsiExec.exe /X{GUID})
}
```

**Prefer `QuietUninstallString`** when it exists.

**MSI "maintenance form."** Some MSIs record `MsiExec.exe /I{ProductCode}` rather than `/X`. A helper that prepends `/X` produces `MsiExec.exe /X /I{GUID}`, which hangs for about 11 minutes and fails (Foxit). Resolve the ProductCode (the ARP key name if it's a GUID, else a `\{[0-9A-Fa-f-]+\}` match on the string) and run a clean `msiexec /x {GUID} /qn /norestart`. Never reuse the `/I` from the registry.

**Burn bundles register twice.** The inner MSI keeps an ARP key with the same DisplayName and Publisher as the bundle, and its `UninstallString` is `MsiExec.exe /I{guid}`. Select the bundle's key with `($key.BundleUpgradeCode -or $key.QuietUninstallString)`; Burn writes both and Windows Installer writes neither. Then use its `QuietUninstallString`. Don't glob `Package Cache` by filename: the bundle's ProductCode changes every release (`jabra-direct_uninstall.ps1`).

**NUL-padded registry values.** Some installers write `REG_SZ` values padded with NULs. Fork records `DisplayName`, `Publisher`, *and* `DisplayVersion` as `"Fork\0\0\0…"`, so a pattern built from the observed string never matches and an exact publisher comparison fails. Strip NULs on both sides when matching (`($key.DisplayName -replace "`0", "").Trim()`), and give that app's exists query a second look.

**NSIS uninstallers relaunch themselves from `%TEMP%`**, so the process you waited on exits while removal is still in flight. Poll until *either* the install directory or the registration disappears, then force-remove any remainder. Waiting only on the directory burns the full timeout for an app that clears its registration first (Notion: 120s vs 6s).

## Testing as SYSTEM

**CI can't catch SYSTEM-context bugs.** If your change concerns scope, profile location, or uninstall path resolution, a green run proves nothing. Use a Windows VM: `prlctl exec "<vm>" cmd /c "..."` (Parallels) runs as `NT AUTHORITY\SYSTEM`, exactly orbit's context. After install, assert three things: the registration is under a real user SID (`S-1-5-21-…`, or `S-1-12-1-…` for an Entra ID user) or HKLM as intended; there is **nothing** under `S-1-5-18`/`.DEFAULT`; and the payload is in the expected place. Then assert that the uninstall leaves no registration, directory, or shortcut. If you have no VM, say in the PR that scope is unverified.

**Mind the test host's architecture.** An ARM64 VM (Parallels on Apple silicon) can only produce false *failures*, never false passes. The per-user/SYSTEM-profile behavior and the `System32`→`SysWOW64` redirection are identical on x64, but x64-only installers with a native-architecture `LaunchCondition` refuse to run. Inno says so in `/LOG`: *"This program can only be installed on versions of Windows designed for the following processor architectures: x64"*. That's not an app defect; note that the app needs an x64 host. `kiro` and `antigravity-ide` use the same Inno 6.4.0.1 with identical switches, and only `kiro` runs on ARM64.

Run osquery there too: `osqueryi.exe --json "<sql>"` (the osquery MSI puts it in `C:\Program Files\osquery\`). Put the SQL in a `.ps1` and run it with `-File`, because nesting the queries' single quotes through `-Command` or a bash heredoc breaks every time.

## PowerShell traps

Each of these silently produced a wrong answer in practice, not an error.

- **`Start-Process -PassThru` + `WaitForExit($ms)` leaves `ExitCode` empty**, even once `HasExited` is `$true` and even after a parameterless `WaitForExit()`. Use `-Wait -PassThru` when you can. Treat any blank exit code as "not measured": this one falsely reported an entire 30-app validation run as failing.
- **`Exit` with a code above `Int32.MaxValue` exits 0 in Windows PowerShell 5.1.** A task that fails to launch leaves an HRESULT in `LastTaskResult` (0x80070002 = 2147942402), so a script that exits with it reports success. Put `if ($exitCode -gt [int]::MaxValue) { $exitCode = 1 }` before the final `Exit`.
- **`Copy-Item` of a missing `$env:INSTALLER_PATH` doesn't throw inside `try`.** It's a non-terminating error, so the script goes on to schedule a task for a file that isn't there. Add `-ErrorAction Stop`.
- **`'\b/S\b'` never matches a switch preceded by a space.** Neither the space nor the `/` is a word character, so there's no boundary between them, and a `-notmatch` guard appends a duplicate switch forever. Anchor on whitespace: `'(?i)(^|\s)/S($|\s)'`.
- **A single result is a scalar, so `.Count` is `$null`.** Wrap in `@(...)` before counting or comparing, or a one-match check silently reads as zero.
- **`New-ScheduledTaskAction -Argument ""` is rejected outright.** Omit the parameter when there are no arguments.
- **Don't wait for a scheduled task to be observed `Running`**: a fast task enters and leaves that state between polls. Poll until `LastTaskResult -ne 267009` (`SCHED_S_TASK_RUNNING`) instead.
- **A nested `"` inside a `$(...)` subexpression terminates the surrounding string.** Build the value into a variable first. A parse error means the script never ran at all, which is easy to mistake for a silent failure. Parse-check before trusting a run: `[System.Management.Automation.Language.Parser]::ParseFile($p,[ref]$null,[ref]$errors)`.
- **Carry over every element of a multi-value `ArgumentList`.** The original `ArgumentList = "/silent", "/skip-app-launch"` is an array. Taking only the first element dropped `/skip-app-launch`, which would have let Spotify launch itself after installing.
