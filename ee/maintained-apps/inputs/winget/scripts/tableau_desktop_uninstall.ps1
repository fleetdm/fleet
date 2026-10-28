# Tableau Desktop is a per-machine WiX Burn bundle. Running the cached bundle
# recorded in its machine-wide (HKLM) ARP entry (BundleCachePath) with
# /uninstall removes the product AND its ARP registration in one shot.
#
# Reference: https://silentinstallhq.com/tableau-desktop-2024-silent-uninstall-powershell/

$arpPaths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

try {
    $bundlePath = $null
    foreach ($arpPath in $arpPaths) {
        $entry = Get-ItemProperty "$arpPath\*" -ErrorAction SilentlyContinue | Where-Object {
            $_.DisplayName -like 'Tableau 20*' -and $_.BundleCachePath
        } | Select-Object -First 1
        if ($entry) { $bundlePath = $entry.BundleCachePath; break }
    }

    if (-not $bundlePath -or -not (Test-Path $bundlePath)) {
        Write-Host "Tableau Desktop bundle not found in the uninstall registry."
        Exit 1
    }

    Write-Host "Uninstalling: $bundlePath"
    $p = Start-Process -FilePath $bundlePath `
        -ArgumentList '/uninstall', '/quiet', '/norestart' `
        -PassThru -Wait
    $exitCode = $p.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
