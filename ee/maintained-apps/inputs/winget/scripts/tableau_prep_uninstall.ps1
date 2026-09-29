# Tableau Prep Builder registers a WiX Burn bundle whose DisplayName carries the
# version (e.g. "Tableau Prep Builder 2024.3 (24.34.25.0210.0740)"), so its
# uninstall command is matched by prefix in the registry rather than hard-coded.
# Running the bundle removes the app and its Programs and Features entry together.

$softwareNameLike = "Tableau Prep Builder 20*"
$publisherLike = "Salesforce*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$selected = $null
foreach ($key in $uninstallKeys) {
    # The bundle and the MSI it installs register under the same DisplayName and
    # publisher, and only the bundle's uninstaller removes both. BundleUpgradeCode
    # and QuietUninstallString are written by the bundle and not by the MSI, so
    # requiring one of them selects the bundle. Dropping that condition selects
    # the MSI entry instead, whose recorded command ("MsiExec.exe /I{...}") is an
    # install and hangs indefinitely.
    if ($key.DisplayName -and $key.DisplayName -like $softwareNameLike -and
        $key.Publisher -like $publisherLike -and
        ($key.BundleUpgradeCode -or $key.QuietUninstallString)) {
        $selected = $key
        break
    }
}

if (-not $selected) {
    Write-Host "Uninstall entry not found for $softwareNameLike"
    Exit 1
}

# Burn bundles register QuietUninstallString with the switches already applied;
# UninstallString needs them added.
$uninstallCommand = $selected.QuietUninstallString
$defaultArgs = ""
if (-not $uninstallCommand) {
    $uninstallCommand = $selected.UninstallString
    $defaultArgs = "/uninstall /quiet"
}

if (-not $uninstallCommand) {
    Write-Host "No uninstall command recorded for $($selected.DisplayName)"
    Exit 1
}

# Split the uninstall string into exe + args, handling quoted and unquoted paths.
$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    # Unquoted path that may contain spaces: capture through the .exe.
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

$existingArgs = ("$existingArgs $defaultArgs").Trim()
if ($existingArgs -notmatch '(?i)/norestart') {
    $existingArgs = "$existingArgs /norestart"
}

Write-Host "Selected entry DisplayName: $($selected.DisplayName)"
Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $existingArgs"

$processOptions = @{
    FilePath = $exePath
    ArgumentList = $existingArgs
    PassThru = $true
    Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

# 3010 and 1641 mean the uninstall succeeded and a reboot is required to finish.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
    Exit 0
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
