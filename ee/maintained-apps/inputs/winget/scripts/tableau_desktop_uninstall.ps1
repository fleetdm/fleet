# Tableau Desktop is a per-machine WiX Burn bundle. Each release family
# (2024.2, 2024.3, ...) has its own upgrade code and installs side by side, and
# Fleet reports every "Tableau 20*" release as one title. Run each bundle's
# cached copy recorded in its HKLM ARP entry (BundleCachePath) with /uninstall,
# which removes the product and its ARP registration. The inner MSI's hidden ARP
# entry has the same DisplayName but no BundleCachePath, so it is skipped.

$arpPaths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

try {
    $bundles = @(foreach ($arpPath in $arpPaths) {
        Get-ItemProperty "$arpPath\*" -ErrorAction SilentlyContinue | Where-Object {
            $_.DisplayName -like 'Tableau 20*' -and $_.BundleCachePath
        }
    })

    if ($bundles.Count -eq 0) {
        Write-Host "Tableau Desktop bundle not found in the uninstall registry."
        Exit 1
    }

    # A release that can't be removed (including one whose cached bundle is gone)
    # fails the uninstall, but the remaining releases are still removed.
    $exitCode = 0
    foreach ($bundle in $bundles) {
        $bundlePath = $bundle.BundleCachePath
        try {
            if (-not (Test-Path -LiteralPath $bundlePath)) {
                throw "cached bundle not found: $bundlePath"
            }
            Write-Host "Uninstalling: $bundlePath"
            $p = Start-Process -FilePath $bundlePath `
                -ArgumentList '/uninstall', '/quiet', '/norestart' `
                -PassThru -Wait
            Write-Host "Uninstall exit code: $($p.ExitCode)"
            if ($p.ExitCode -notin 0, 3010, 1641) { $exitCode = $p.ExitCode }
        } catch {
            Write-Host "Failed to uninstall $($bundle.DisplayName): $_"
            $exitCode = 1
        }
    }

    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
