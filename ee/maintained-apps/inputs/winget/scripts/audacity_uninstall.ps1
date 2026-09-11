# Removes every Audacity 4 MSI product registered under its UpgradeCode, then
# any legacy Audacity 3.x Inno Setup install that an earlier version of this
# Fleet-maintained app may have deployed.

$upgradeCode = '{C9D47FF5-2A6D-4A42-9038-93C7D3C3FB23}'
$timeoutSeconds = 300
$successCodes = @(0, 3010, 1641)

Get-Process -Name "Audacity4", "audacity" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

$inst = New-Object -ComObject "WindowsInstaller.Installer"
foreach ($product_code in $inst.RelatedProducts("$upgradeCode")) {
    $process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $product_code, "/norestart") -PassThru

    $completed = $process.WaitForExit($timeoutSeconds * 1000)
    if (-not $completed) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Write-Host "Uninstall for $product_code timed out"
        Exit 1603
    }

    if ($successCodes -notcontains $process.ExitCode) {
        Write-Host "Uninstall for $product_code exited $($process.ExitCode)"
        Exit $process.ExitCode
    }
    Write-Host "Uninstalled $product_code"
}

# Legacy Audacity 3.x (Inno Setup, publisher "Audacity Team")
$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)
$legacy = $null
foreach ($p in $paths) {
    $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
        $_.DisplayName -like 'Audacity 3*' -and
        $_.Publisher -eq 'Audacity Team' -and
        $_.UninstallString -like '*unins*.exe*'
    }
    if ($items) { $legacy = $items | Select-Object -First 1; break }
}

if ($legacy) {
    $uninstaller = $legacy.UninstallString.Trim('"')
    if (Test-Path $uninstaller) {
        Write-Host "Removing legacy install: $($legacy.DisplayName)"
        $process = Start-Process -FilePath $uninstaller `
            -ArgumentList "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART" `
            -PassThru -Wait -NoNewWindow
        if ($process.ExitCode -ne 0) {
            Write-Host "Legacy uninstall exited $($process.ExitCode)"
            Exit $process.ExitCode
        }
    } else {
        Write-Host "Warning: legacy uninstaller not found at $uninstaller"
    }
}

Exit 0
