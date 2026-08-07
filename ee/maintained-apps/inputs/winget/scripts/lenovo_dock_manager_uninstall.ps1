$displayNameLike = "Lenovo Dock Manager*"
$publisherLike = "Lenovo*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$ExpectedExitCodes = @(0, 1641, 3010, 1223)
# Bounded under the validator's ~300s cap so the script can print diagnostics
# and clean up before it is force-killed.
$waitSeconds = 220

function Test-StillInstalled {
  foreach ($p in $paths) {
    $m = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
      $_.DisplayName -like $displayNameLike -and $_.Publisher -like $publisherLike
    }
    if ($m) { return $true }
  }
  return $false
}

$entry = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -like $displayNameLike -and $_.Publisher -like $publisherLike
  }
  if ($items) { $entry = $items | Select-Object -First 1; break }
}

if (-not $entry -or -not $entry.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

# Tear down the Dock Manager components before uninstalling. A soft Stop-Service
# is not enough: the scheduler component (dockmgr.schd) and service recovery
# restart the service within seconds, and the running service then locks files
# so the Inno uninstaller's second phase blocks immediately. So we DISABLE the
# service (so it cannot restart), disable the scheduled task, and force-kill all
# components. Uses only built-in Windows tools (sc.exe, taskkill, *-ScheduledTask).
$svcs = Get-CimInstance Win32_Service -ErrorAction SilentlyContinue | Where-Object {
  $_.PathName -like '*dockmgr*' -or $_.PathName -like '*Dock Manager*'
}
foreach ($s in $svcs) {
  Write-Host "Disabling + stopping service: $($s.Name) ($($s.State)) -> $($s.PathName)"
  & sc.exe config "$($s.Name)" start= disabled | Out-Null
  & sc.exe stop "$($s.Name)" | Out-Null
}

Get-ScheduledTask -ErrorAction SilentlyContinue | Where-Object {
  $_.TaskName -like '*Dock*Manager*' -or $_.TaskName -like '*dockmgr*'
} | ForEach-Object {
  Write-Host "Disabling scheduled task: $($_.TaskName)"
  Disable-ScheduledTask -TaskName $_.TaskName -TaskPath $_.TaskPath -ErrorAction SilentlyContinue | Out-Null
}

foreach ($n in @('dockmgr', 'dockmgr.svc', 'dockmgr.schd')) {
  Write-Host "Killing process: $n.exe"
  & taskkill.exe /F /IM "$n.exe" /T 2>$null | Out-Null
}
Start-Sleep -Seconds 3

# Inno's UninstallString is the bare unins000.exe path (quoted, no args).
$uninstPath = ($entry.UninstallString -replace '"', '').Trim()

if (-not (Test-Path -LiteralPath $uninstPath)) {
  Write-Host "Uninstaller not found at: $uninstPath"
  Exit 1
}

$logFile = Join-Path $env:TEMP "LenovoDockManager-Uninstall.log"
Remove-Item -LiteralPath $logFile -Force -ErrorAction SilentlyContinue

# The Inno uninstaller pops a CUSTOM Yes/No dialog ("Do you want to keep local
# user data and settings?") that /VERYSILENT /SUPPRESSMSGBOXES does NOT dismiss:
# those flags only suppress message boxes created via Inno's SuppressibleMsgBox()
# helper, and Lenovo coded this one with the plain (non-suppressible) MsgBox().
# There is no command-line flag that will get past it, so we dismiss it ourselves
# while the uninstaller runs by clicking the dialog's button via the Win32 API.
# SendMessage(BM_CLICK) works even in session 0 with no interactive desktop
# (where SendKeys / AppActivate / SetForegroundWindow do not), which is how
# Fleet's orbit agent runs uninstall scripts (as SYSTEM). We only ever click
# "Yes"/"OK" -- never "No"/"Cancel" -- so we can never abort the uninstall; for
# the keep-data prompt, "Yes" just proceeds (the program and its registry entry
# are still removed, which is what Test-StillInstalled verifies).
$dismisserLoaded = $false
try {
  Add-Type -Language CSharp @"
using System;
using System.Diagnostics;
using System.Runtime.InteropServices;
using System.Text;

public static class InnoDlg {
    [DllImport("user32.dll")] static extern bool EnumWindows(EnumWindowsProc cb, IntPtr p);
    [DllImport("user32.dll")] static extern bool EnumChildWindows(IntPtr h, EnumWindowsProc cb, IntPtr p);
    [DllImport("user32.dll", CharSet=CharSet.Unicode)] static extern int GetClassName(IntPtr h, StringBuilder s, int n);
    [DllImport("user32.dll", CharSet=CharSet.Unicode)] static extern int GetWindowText(IntPtr h, StringBuilder s, int n);
    [DllImport("user32.dll")] static extern uint GetWindowThreadProcessId(IntPtr h, out uint pid);
    [DllImport("user32.dll")] static extern IntPtr SendMessage(IntPtr h, uint msg, IntPtr wp, IntPtr lp);
    delegate bool EnumWindowsProc(IntPtr h, IntPtr p);
    const uint BM_CLICK = 0x00F5;

    static string ClassOf(IntPtr h) { var sb = new StringBuilder(256); GetClassName(h, sb, sb.Capacity); return sb.ToString(); }
    static string TextOf(IntPtr h)  { var sb = new StringBuilder(512); GetWindowText(h, sb, sb.Capacity); return sb.ToString(); }

    // Finds standard dialog windows (#32770) owned by the Inno uninstaller -- the
    // original "unins000" or its relaunched temporary "_iu*.tmp" second phase --
    // and clicks their Yes/OK button. Returns the number of such dialogs found.
    public static int DismissOnce() {
        int found = 0;
        EnumWindows((h, p) => {
            if (ClassOf(h) != "#32770") return true;
            uint pid; GetWindowThreadProcessId(h, out pid);
            string pname = "";
            try { pname = Process.GetProcessById((int)pid).ProcessName.ToLowerInvariant(); } catch {}
            if (!(pname.StartsWith("unins") || pname.StartsWith("_iu"))) return true;
            found++;
            EnumChildWindows(h, (c, q) => {
                if (ClassOf(c) != "Button") return true;
                string t = TextOf(c).Replace("&", "").Trim();
                if (t.Equals("Yes", StringComparison.OrdinalIgnoreCase) ||
                    t.Equals("OK",  StringComparison.OrdinalIgnoreCase)) {
                    SendMessage(c, BM_CLICK, IntPtr.Zero, IntPtr.Zero);
                }
                return true;
            }, IntPtr.Zero);
            return true;
        }, IntPtr.Zero);
        return found;
    }
}
"@
  $dismisserLoaded = $true
} catch {
  Write-Host "Warning: could not load dialog dismisser ($($_.Exception.Message)); uninstaller may block on prompts."
}

Write-Host "Uninstall command: $uninstPath"
Write-Host "Uninstall args: /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /LOG=`"$logFile`""

try {
    $process = Start-Process -FilePath $uninstPath `
      -ArgumentList "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /LOG=`"$logFile`"" `
      -PassThru

    # Poll for exit while dismissing any uninstaller dialog that pops up, instead
    # of a single blocking WaitForExit -- otherwise the custom "keep user data?"
    # prompt stalls the uninstaller until it is force-killed.
    $deadline = (Get-Date).AddSeconds($waitSeconds)
    $exited = $false
    while ((Get-Date) -lt $deadline) {
        if ($process.HasExited) { $exited = $true; break }
        if ($dismisserLoaded) {
            try {
                $n = [InnoDlg]::DismissOnce()
                if ($n -gt 0) { Write-Host "Dismissed $n uninstaller dialog(s)" }
            } catch { }
        }
        Start-Sleep -Milliseconds 750
    }

    if ($exited) {
        Write-Host "Uninstaller process exited. ExitCode: $($process.ExitCode)"
    } else {
        Write-Host "Uninstaller process still running after $waitSeconds s."
    }

    # The original unins000.exe relaunches to a temp copy; give it a moment to
    # finish removal, then re-check the registry (the true completion signal).
    Start-Sleep -Seconds 10

    # Surface the Inno uninstall log so we can see where it stopped if it failed.
    if (Test-Path -LiteralPath $logFile) {
        Write-Host "----- Inno uninstall log (tail) -----"
        Get-Content -LiteralPath $logFile -Tail 40 -ErrorAction SilentlyContinue | ForEach-Object { Write-Host $_ }
        Write-Host "-------------------------------------"
    } else {
        Write-Host "No Inno uninstall log was produced at $logFile"
    }

    if (-not (Test-StillInstalled)) {
        Write-Host "Dock Manager uninstalled successfully."
        Exit 0
    }

    # Stop any lingering uninstaller (original plus the temp _iu*.tmp copy).
    Stop-Process -Name "unins000" -Force -ErrorAction SilentlyContinue
    Get-Process -ErrorAction SilentlyContinue | Where-Object { $_.Name -like "_iu*" } |
      Stop-Process -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 5

    if (-not (Test-StillInstalled)) {
        Write-Host "Dock Manager uninstalled successfully."
        Exit 0
    }

    Write-Host "Dock Manager still present after uninstall attempt."
    Exit 1
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
