$softwareNameLike = "Burp Suite Professional*"
$publisherLike    = "*PortSwigger*"

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
    Exit 0
}

# Stop running Burp processes so the uninstaller doesn't fail on locked files.
foreach ($proc in @("BurpSuitePro", "BurpSuiteProfessional", "Burp")) {
    Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
}

if ($selected.InstallLocation -and (Test-Path -LiteralPath $selected.InstallLocation)) {
    $loc = $selected.InstallLocation.TrimEnd('\')
    Get-Process | Where-Object { $_.Path -and $_.Path -like "$loc\*" } |
        ForEach-Object { Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue }
}
Start-Sleep -Seconds 2

$uninstallCommand = if ($selected.QuietUninstallString) {
    $selected.QuietUninstallString
} else {
    $selected.UninstallString
}

# Parse uninstaller exe path. install4j typically quotes the path because the
# install dir is under "Program Files".
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

# Ensure install4j's silent flags are present (merge with whatever the
# registry's UninstallString already supplied).
$argumentList = @()
if ($existingArgs) { $argumentList += ($existingArgs -split '\s+') }
if ($argumentList -notcontains "-q") { $argumentList += "-q" }
if (-not ($argumentList | Where-Object { $_ -like "-Dinstall4j.suppressUnattendedReboot*" })) {
    $argumentList += "-Dinstall4j.suppressUnattendedReboot=true"
}

Write-Host "Selected entry DisplayName: $($selected.DisplayName)"
Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $($argumentList -join ' ')"

$processOptions = @{
    FilePath     = $exePath
    ArgumentList = $argumentList
    PassThru     = $true
    Wait         = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
