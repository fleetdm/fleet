# Tableau Reader is a per-machine WiX Burn bundle. Each release family
# (2024.3, 2025.1, ...) has its own upgrade code and installs side by side, and
# Fleet reports every "Tableau Reader 20*" release as one title. Run each
# bundle's cached copy recorded in its HKLM ARP entry (BundleCachePath) with
# /uninstall, which removes the product and its ARP registration. The inner
# MSI's hidden ARP entry has the same DisplayName but no BundleCachePath, so it
# is skipped.

$arpPaths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

try {
    $bundlePaths = @(foreach ($arpPath in $arpPaths) {
        Get-ItemProperty "$arpPath\*" -ErrorAction SilentlyContinue | Where-Object {
            $_.DisplayName -like 'Tableau Reader 20*' -and $_.BundleCachePath -and
                (Test-Path -LiteralPath $_.BundleCachePath)
        } | ForEach-Object { $_.BundleCachePath }
    })

    if ($bundlePaths.Count -eq 0) {
        Write-Host "Tableau Reader bundle not found in the uninstall registry."
        Exit 1
    }

    $exitCode = 0
    foreach ($bundlePath in $bundlePaths) {
        Write-Host "Uninstalling: $bundlePath"
        $p = Start-Process -FilePath $bundlePath `
            -ArgumentList '/uninstall', '/quiet', '/norestart' `
            -PassThru -Wait
        Write-Host "Uninstall exit code: $($p.ExitCode)"
        if ($p.ExitCode -notin 0, 3010, 1641) { $exitCode = $p.ExitCode }
    }

    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
