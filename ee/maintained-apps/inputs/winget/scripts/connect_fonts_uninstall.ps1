# Connect Fonts for Windows registers as "Monotype Connect" twice: the WiX Burn
# bundle and the MSI it installs share that DisplayName. Only the bundle's
# uninstaller removes both, so this script selects the bundle entry (the one
# with a QuietUninstallString) and falls back to msiexec if only the MSI remains.

$softwareName = "Monotype Connect"
$publisherLike = "*Monotype*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$bundle = $null
$msi = $null
foreach ($key in $uninstallKeys) {
    if (-not $key.DisplayName -or $key.DisplayName -ne $softwareName -or $key.Publisher -notlike $publisherLike) {
        continue
    }
    # Burn writes BundleUpgradeCode and QuietUninstallString; Windows Installer writes
    # neither. The MSI entry's UninstallString is "MsiExec.exe /I{...}", an install verb.
    if ($key.BundleUpgradeCode -or $key.QuietUninstallString) {
        $bundle = $key
        break
    }
    if (-not $msi -and $key.PSChildName -match '^\{[0-9A-Fa-f-]+\}$') {
        $msi = $key
    }
}

if ($bundle) {
    # QuietUninstallString already carries the silent switches; UninstallString needs them added.
    $uninstallCommand = $bundle.QuietUninstallString
    $defaultArgs = ""
    if (-not $uninstallCommand) {
        $uninstallCommand = $bundle.UninstallString
        $defaultArgs = "/uninstall /quiet"
    }
    if (-not $uninstallCommand) {
        Write-Host "No uninstall command recorded for $($bundle.DisplayName)"
        Exit 1
    }

    # Split the uninstall string into exe + args, handling quoted and unquoted paths.
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

    $existingArgs = ("$existingArgs $defaultArgs").Trim()
    if ($existingArgs -notmatch '(?i)/norestart') {
        $existingArgs = "$existingArgs /norestart"
    }

    Write-Host "Uninstall command: $exePath"
    Write-Host "Uninstall args: $existingArgs"

    $process = Start-Process -FilePath $exePath -ArgumentList $existingArgs -PassThru -Wait
    $exitCode = $process.ExitCode
} elseif ($msi) {
    $productCode = $msi.PSChildName
    Write-Host "Bundle entry not found; uninstalling MSI product code: $productCode"
    $process = Start-Process -FilePath "msiexec.exe" `
        -ArgumentList "/x $productCode /qn /norestart" `
        -PassThru -Wait
    $exitCode = $process.ExitCode
} else {
    Write-Host "Uninstall entry not found for $softwareName"
    Exit 1
}

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
