# remove-remote-access-apps.ps1
# Stops and uninstalls dedicated remote-access apps from a Windows host.
# Run by the "Windows - Remote access apps not installed" policy; keep the lists below in sync with that policy's query.
#
# Zoom ("Zoom Workplace", Zoom.exe) has a screen sharing feature and is explicitly allowed. It is not in any list
# below, and the guard in Test-IsZoom skips it even if a list is edited wrongly. Slack, Google Meet, and Microsoft
# Teams aren't touched. Windows Remote Desktop isn't touched.
#
# Uninstall methods: MSI entries (RealVNC, TightVNC, Splashtop Streamer, Chrome Remote Desktop Host, ScreenConnect)
# run msiexec /x; TeamViewer runs its NSIS uninstaller with /S; AnyDesk runs AnyDesk.exe --remove --silent. Those
# match the methods in ee/maintained-apps. Any other entry is reported and left alone instead of guessing flags.
# Only machine-wide installs (HKLM) are uninstalled. A portable exe (for example AnyDesk or TeamViewer QuickSupport
# run from Downloads) is stopped but not deleted, so it can come back and fail the policy again.
# Idempotent: safe to re-run. Built for Fleet "Run script" (runs as SYSTEM).
# Exit: 0 = removed or nothing to do, 2 = something needs a look (read output), 1 = error.

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 2
$script:ExitCode = 0
$UninstallTimeoutMs = 120000

function Log([string]$Message) { Write-Host $Message }
function Warn([string]$Message) { Write-Host "WARN: $Message"; $script:ExitCode = 2 }

# Lowercase process names (no .exe) from the policy query.
$ProcessNames = @(
    'teamviewer', 'teamviewer host', 'anydesk', 'splashtop streamer', 'vnc server',
    'vncviewer', 'rvncconnect', 'tightvnc', 'chrome remote desktop'
)

# Mirrors the programs table conditions in the policy. Method: msi, nsis (TeamViewer), or anydesk.
# Name uses -like patterns; Publisher is an exact match when set.
$Targets = @(
    @{ Name = 'TeamViewer';                   Publisher = 'TeamViewer';            Method = 'nsis' },
    @{ Name = 'TeamViewer Host';              Publisher = 'TeamViewer';            Method = 'nsis' },
    @{ Name = 'AnyDesk';                      Publisher = 'AnyDesk Software GmbH'; Method = 'anydesk' },
    @{ Name = 'Splashtop Streamer';           Publisher = 'Splashtop Inc.';        Method = 'msi' },
    @{ Name = 'RealVNC Server *';             Publisher = 'RealVNC';               Method = 'msi' },
    @{ Name = 'RealVNC Viewer *';             Publisher = 'RealVNC';               Method = 'msi' },
    @{ Name = 'RealVNC Connect Viewer *';     Publisher = 'RealVNC';               Method = 'msi' },
    @{ Name = 'TightVNC';                     Publisher = 'GlavSoft LLC.';         Method = 'msi' },
    @{ Name = 'Chrome Remote Desktop Host';   Publisher = 'Google LLC';            Method = 'msi' },
    @{ Name = 'ScreenConnect Client *';       Publisher = $null;                   Method = 'msi' },
    @{ Name = 'UltraVNC';                     Publisher = $null;                   Method = 'msi' },
    @{ Name = 'RustDesk';                     Publisher = $null;                   Method = 'msi' },
    @{ Name = 'Parsec';                       Publisher = $null;                   Method = 'msi' },
    @{ Name = 'NoMachine';                    Publisher = $null;                   Method = 'msi' }
)

function Test-IsZoom([string]$Name) {
    return ($Name -like 'Zoom*')
}

function Get-InstalledEntries {
    $paths = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
    )
    foreach ($path in $paths) {
        foreach ($item in @(Get-ItemProperty -Path $path -ErrorAction SilentlyContinue)) {
            if ($item.PSObject.Properties['DisplayName'] -and $item.DisplayName) { $item }
        }
    }
}

function Find-Matches {
    foreach ($entry in @(Get-InstalledEntries)) {
        if (Test-IsZoom $entry.DisplayName) { continue }
        $publisher = if ($entry.PSObject.Properties['Publisher']) { $entry.Publisher } else { $null }
        foreach ($t in $Targets) {
            if ($entry.DisplayName -notlike $t.Name) { continue }
            if ($t.Publisher -and $publisher -ne $t.Publisher) { continue }
            [pscustomobject]@{ Entry = $entry; Method = $t.Method }
            break
        }
    }
}

function Invoke-Uninstall([string]$FilePath, [string]$Arguments) {
    Log "Running: $FilePath $Arguments"
    $p = Start-Process -FilePath $FilePath -ArgumentList $Arguments -PassThru -WindowStyle Hidden
    if (-not $p.WaitForExit($UninstallTimeoutMs)) {
        Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue
        Warn "uninstaller timed out after $($UninstallTimeoutMs / 1000) seconds: $FilePath"
        return
    }
    if (@(0, 3010, 1641) -notcontains $p.ExitCode) {
        Warn "uninstaller exited $($p.ExitCode): $FilePath $Arguments"
    }
}

function Get-ExecutablePath([string]$Command) {
    if ($Command -match '^\s*"([^"]+)"') { return $Matches[1] }
    if ($Command -match '(?i)^\s*(.+?\.exe)') { return $Matches[1] }
    return $null
}

function Remove-Match($Match) {
    $entry = $Match.Entry
    $name = $entry.DisplayName
    $uninstallString = if ($entry.PSObject.Properties['UninstallString']) { [string]$entry.UninstallString } else { '' }
    $msi = [regex]::Match($uninstallString, '(?i)MsiExec\.exe\s+/[IX]\s*(\{[0-9A-F-]+\})')

    Log "Uninstalling $name"
    if ($msi.Success) {
        Invoke-Uninstall 'msiexec.exe' "/x $($msi.Groups[1].Value) /qn /norestart"
    } elseif ($Match.Method -eq 'anydesk') {
        $exe = Get-ExecutablePath $uninstallString
        if ($exe -and (Test-Path -LiteralPath $exe)) { Invoke-Uninstall $exe '--remove --silent' }
        else { Warn "$name: executable not found in UninstallString: $uninstallString" }
    } elseif ($Match.Method -eq 'nsis') {
        $exe = Get-ExecutablePath $uninstallString
        if ($exe -and (Test-Path -LiteralPath $exe)) { Invoke-Uninstall $exe '/S' }
        else { Warn "$name: executable not found in UninstallString: $uninstallString" }
    } else {
        Warn "$name has no silent uninstall method this script trusts (UninstallString: $uninstallString). Remove it by hand."
    }
}

try {
    $running = @(Get-Process -ErrorAction SilentlyContinue | Where-Object { $ProcessNames -contains $_.ProcessName.ToLower() })
    foreach ($proc in $running) {
        Log "Stopping process $($proc.ProcessName) (pid $($proc.Id))"
        Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    }

    $found = @(Find-Matches)
    if ($found.Count -eq 0 -and $running.Count -eq 0) {
        Log 'No remote-access apps found.'
    }
    foreach ($m in $found) { Remove-Match $m }

    foreach ($m in @(Find-Matches)) {
        Warn "$($m.Entry.DisplayName) is still installed."
    }
} catch {
    Write-Host "Error: $_"
    Exit 1
}

Exit $script:ExitCode
