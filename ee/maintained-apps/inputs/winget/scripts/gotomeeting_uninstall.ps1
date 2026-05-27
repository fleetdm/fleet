# Best-effort uninstall for GoToMeeting.
#
# The winget installer is the GoToMeeting "Setup" bootstrapper (ARPSYSTEMCOMPONENT=1,
# so it hides itself from Programs and Features). It installs the actual GoToMeeting
# app, which self-registers a separate, visible uninstall entry (DisplayName like
# "GoToMeeting <version>"). We therefore can't uninstall via the bootstrapper's
# UpgradeCode; instead we locate the app's own registry entry and run its
# uninstaller. We search HKLM, the 32-bit hive, and HKCU because the app may be
# registered per-machine or per-user.

$softwareNameLike = "GoToMeeting*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$selected = $null
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -and $key.DisplayName -like $softwareNameLike) {
        $selected = $key
        break
    }
}

if (-not $selected) {
    Write-Host "Uninstall entry not found for $softwareNameLike"
    Exit 1
}

# Best-effort: stop running GoToMeeting processes so the uninstaller doesn't
# fail on locked files.
foreach ($proc in @("g2mstart", "g2mlauncher", "g2mcomm", "g2muicore", "GoToMeeting", "GoTo")) {
    Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
}

# Prefer QuietUninstallString (it already includes the correct silent switches).
$useQuiet = $false
if ($selected.QuietUninstallString) {
    $uninstallCommand = $selected.QuietUninstallString
    $useQuiet = $true
} elseif ($selected.UninstallString) {
    $uninstallCommand = $selected.UninstallString
} else {
    Write-Host "Selected entry has no UninstallString: $($selected.DisplayName)"
    Exit 1
}

# Split the uninstall string into exe + args. Handle quoted paths, unquoted
# paths that may contain spaces (capture through .exe), and a bare token.
$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

# If we fell back to UninstallString (no quiet variant), add a silent switch.
if (-not $useQuiet) {
    if ($exePath -match '(?i)msiexec') {
        if ($existingArgs -notmatch '/quiet' -and $existingArgs -notmatch '/qn') {
            $existingArgs = ("$existingArgs /quiet /norestart").Trim()
        }
    } elseif ($existingArgs -notmatch '/S\b' -and $existingArgs -notmatch '/silent' -and $existingArgs -notmatch '/quiet') {
        # Custom uninstaller: GoTo's uninstaller honors /S for silent operation.
        $existingArgs = ("$existingArgs /S").Trim()
    }
}

Write-Host "Selected entry DisplayName: $($selected.DisplayName)"
Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $existingArgs"

$processOptions = @{
    FilePath = $exePath
    PassThru = $true
    Wait = $true
}

if ($existingArgs -ne '') {
    $processOptions.ArgumentList = $existingArgs
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

# Treat msiexec reboot-required success codes as success.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
    Exit 0
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
