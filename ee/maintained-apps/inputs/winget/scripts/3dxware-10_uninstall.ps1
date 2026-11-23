# 3DxWare 10 is a WiX Burn bundle; its components register separately
# (3Dconnexion 3DxWinCore, 3DxSoftware, plug-ins). Uninstalling the bundle
# entry removes all of them. The uninstaller stops the 3DxWare driver.

$displayName = "3Dconnexion 3DxWare 10"
$publisher = "3Dconnexion"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

function Get-BundleKeys {
    @(Get-ChildItem -Path $paths -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -eq $displayName -and $_.Publisher -eq $publisher })
}

try {
    $matchedKeys = Get-BundleKeys
    if ($matchedKeys.Count -eq 0) {
        Write-Host "$displayName isn't installed."
        Exit 0
    }

    # Burn writes BundleUpgradeCode and QuietUninstallString; Windows Installer writes neither.
    $bundle = $matchedKeys | Where-Object { $_.BundleUpgradeCode -or $_.QuietUninstallString } | Select-Object -First 1
    if (-not $bundle) {
        Write-Host "Found $displayName, but not its bundle uninstall entry."
        Exit 1
    }

    $uninstallCommand = $bundle.QuietUninstallString
    $defaultArgs = ""
    if (-not $uninstallCommand) {
        $uninstallCommand = $bundle.UninstallString
        $defaultArgs = "/uninstall /quiet"
    }
    if (-not $uninstallCommand) {
        Write-Host "No uninstall command recorded for $displayName."
        Exit 1
    }

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
        Throw "Could not parse uninstall command: $uninstallCommand"
    }

    $uninstallArgs = ("$existingArgs $defaultArgs").Trim()
    if ($uninstallArgs -notmatch '(?i)/norestart') {
        $uninstallArgs = "$uninstallArgs /norestart"
    }

    Write-Host "Uninstall command: $exePath"
    Write-Host "Uninstall args: $uninstallArgs"

    $process = Start-Process -FilePath $exePath -ArgumentList $uninstallArgs -PassThru -Wait
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    # 3010 = success, restart required; 1641 = success, restart initiated.
    if ($exitCode -notin @(0, 1641, 3010)) {
        Exit $exitCode
    }

    $sw = [Diagnostics.Stopwatch]::StartNew()
    while ($sw.Elapsed.TotalSeconds -lt 60) {
        if ((Get-BundleKeys).Count -eq 0) {
            Write-Host "$displayName was uninstalled."
            Exit 0
        }
        Start-Sleep -Seconds 2
    }

    Write-Host "$displayName is still registered after uninstall."
    Exit 1

} catch {
    Write-Host "Error: $_"
    Exit 1
}
