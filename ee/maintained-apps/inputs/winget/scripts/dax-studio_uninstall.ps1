# Locates DAX Studio's Inno Setup uninstaller from the registry and runs it
# silently. DAX Studio embeds the version in its DisplayName (e.g.
# "DAX Studio 3.5.2.1205"), so we match by prefix and require the publisher.
# The ARP entry can land in HKLM or (when the Inno installer writes per-user
# registry while placing files machine-wide) in the installing user's hive, so
# we search HKLM, the WOW6432Node view, and HKCU.

$softwareNameLike = "DAX Studio*"
$publisherLike    = "DAX Studio*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

# 0 = success; 3010/1641 = success but reboot required.
$ExpectedExitCodes = @(0, 3010, 1641)
$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$selected = $null
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -and $key.DisplayName -like $softwareNameLike -and $key.Publisher -like $publisherLike) {
        $selected = $key
        break
    }
}

if (-not $selected -or -not $selected.UninstallString) {
    Write-Host "Uninstall entry not found for $softwareNameLike"
    Exit 1
}

# Best-effort: stop the app so the uninstaller doesn't fail on locked files.
Stop-Process -Name "daxstudio" -Force -ErrorAction SilentlyContinue

$uninstallCommand = if ($selected.QuietUninstallString) {
    $selected.QuietUninstallString
} else {
    $selected.UninstallString
}

# Parse uninstaller exe path (Inno quotes the path because it lives under
# "Program Files").
$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

# Inno Setup uninstallers take /VERYSILENT for a silent uninstall.
if ($existingArgs -notmatch '(?i)/VERYSILENT') {
    $existingArgs = ("$existingArgs /VERYSILENT /SUPPRESSMSGBOXES /NORESTART").Trim()
}

Write-Host "Selected entry DisplayName: $($selected.DisplayName)"
Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $existingArgs"

$process = Start-Process -FilePath $exePath -ArgumentList $existingArgs -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
