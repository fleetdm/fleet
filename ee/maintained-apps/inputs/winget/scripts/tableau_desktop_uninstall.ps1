# Tableau Desktop is a WiX Burn bundle. After install, the bundle EXE is cached
# at C:\ProgramData\Package Cache\<bundle-guid>\tableau-setup-std-tableau-*.exe.
# Running the cached bundle with /uninstall removes the product AND its ARP
# registration in one shot. This avoids hunting the registry (per-machine vs
# per-user hives, SYSTEM-context HKCU mismatch, etc.).
#
# Reference: https://silentinstallhq.com/tableau-desktop-2024-silent-uninstall-powershell/

try {
    $bundle = Get-ChildItem -Path "$env:ProgramData\Package Cache" `
        -Recurse -Filter 'tableau-setup-std-tableau*.exe' -ErrorAction SilentlyContinue |
        Select-Object -First 1

    if (-not $bundle) {
        Write-Host "Tableau Desktop bundle not found in Package Cache."
        Exit 1
    }

    Write-Host "Uninstalling: $($bundle.FullName)"
    $p = Start-Process -FilePath $bundle.FullName `
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
