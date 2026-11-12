$softwareNameLike = "Burp Suite Community*"
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

# Earlier install scripts could leave a second copy of Burp in C:\Program, so
# remove every one.
[array]$selected = $uninstallKeys | Where-Object {
    $_.DisplayName -and $_.DisplayName -like $softwareNameLike -and $_.Publisher -like $publisherLike -and $_.UninstallString
}

if (-not $selected) {
    Write-Host "Uninstall entry not found for $softwareNameLike"
    Exit 0
}

# Stop running Burp processes so the uninstaller doesn't fail on locked files.
foreach ($proc in @("BurpSuiteCommunity", "BurpSuiteCommunityEdition", "Burp")) {
    Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
}

foreach ($entry in $selected) {
    if ($entry.InstallLocation -and (Test-Path -LiteralPath $entry.InstallLocation)) {
        $loc = $entry.InstallLocation.TrimEnd('\')
        Get-Process | Where-Object { $_.Path -and $_.Path -like "$loc\*" } |
            ForEach-Object { Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue }
    }
}
Start-Sleep -Seconds 2

foreach ($entry in $selected) {
    $uninstallCommand = if ($entry.QuietUninstallString) {
        $entry.QuietUninstallString
    } else {
        $entry.UninstallString
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

    Write-Host "Selected entry DisplayName: $($entry.DisplayName)"
    Write-Host "Uninstall command: $exePath"
    Write-Host "Uninstall args: $($argumentList -join ' ')"

    $processOptions = @{
        FilePath     = $exePath
        ArgumentList = $argumentList
        PassThru     = $true
        Wait         = $true
    }

    $process = Start-Process @processOptions
    Write-Host "Uninstall exit code: $($process.ExitCode)"
    if ($ExpectedExitCodes -notcontains $process.ExitCode) { $exitCode = $process.ExitCode }
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
