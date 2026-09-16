# TeamViewer Host registers itself in Add/Remove Programs with the DisplayName
# "TeamViewer Host" (the MSI ProductName; there is no ARPDISPLAYNAME override)
# and the Publisher "TeamViewer" (the MSI Manufacturer).
#
# The trailing wildcard tolerates a version suffix if TeamViewer ever adds one,
# and the "Host" in the pattern keeps this from matching the full TeamViewer
# client, which registers as plain "TeamViewer" and ships as its own
# Fleet-maintained app (teamviewer/windows).
$softwareNameLike = "TeamViewer Host*"

# Matched with a trailing wildcard so this filter stays strictly looser than the
# app's exists query (publisher = 'TeamViewer'). Uninstall must never be more
# selective than detection, or the app reports installed with no way to remove it.
$publisherLike = "TeamViewer*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

try {

$uninstall = $null
foreach ($p in $paths) {
    $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
        $_.DisplayName -like $softwareNameLike -and $_.Publisher -like $publisherLike
    }
    if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall) {
    Write-Host "Uninstall entry not found for $softwareNameLike"
    Exit 0
}

if (-not $uninstall.UninstallString -and -not $uninstall.QuietUninstallString) {
    Write-Host "Error: uninstall entry for $($uninstall.DisplayName) has no UninstallString or QuietUninstallString"
    Exit 1
}

# Get the uninstall command. Some uninstallers do not include
# 'QuietUninstallString' and require a flag to run silently.
$uninstallCommand = if ($uninstall.QuietUninstallString) {
    $uninstall.QuietUninstallString
} else {
    $uninstall.UninstallString
}

# UninstallString comes in three shapes. TeamViewer's is unquoted and
# contains a space ("C:\Program Files\TeamViewer\uninstall.exe"), so
# capture through the .exe rather than splitting on the first space.
$existingArgs = ''
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    # Quoted path, optionally followed by args
    $uninstallCommand = $Matches[1]
    $existingArgs = $Matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    # Unquoted path that may contain spaces
    $uninstallCommand = $Matches[1]
    $existingArgs = $Matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    # Bare token (e.g. MsiExec.exe /X{GUID})
    $uninstallCommand = $Matches[1]
    $existingArgs = $Matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

# NSIS installers require /S for a silent uninstall. Append it unless the
# registered command already runs silently.
if ($existingArgs -notmatch '(?i)(^|\s)/S(\s|$)') {
    $uninstallArgs = "$existingArgs /S".Trim()
} else {
    $uninstallArgs = $existingArgs
}

Write-Host "Uninstall command: $uninstallCommand"
Write-Host "Uninstall args: $uninstallArgs"

$processOptions = @{
    FilePath = $uninstallCommand
    ArgumentList = $uninstallArgs
    PassThru = $true
    Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"
Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
