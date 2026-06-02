# Zed (Zed Industries) ships an Inno Setup installer (InstallerType: inno).
# Silent install switches:
#   /VERYSILENT          = no UI at all
#   /SUPPRESSMSGBOXES    = suppress every message box
#   /NORESTART           = don't reboot (returns 3010 instead, treat as success)
#   /ALLUSERS            = machine-scope install. Zed's Inno installer has both
#                          machine and per-user modes baked in; without
#                          /ALLUSERS it silently falls back to per-user when
#                          the install context isn't elevated -- /ALLUSERS pins
#                          the scope. Fleet/SYSTEM provides the elevation.

$exeFilePath = "${env:INSTALLER_PATH}"

# 0 = success; 3010 = success but reboot required.
$ExpectedExitCodes = @(0, 3010)

try {
    $processOptions = @{
        FilePath     = "$exeFilePath"
        ArgumentList = "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", "/ALLUSERS"
        PassThru     = $true
        Wait         = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    Start-Sleep -Seconds 5
    Stop-Process -Name "Zed" -Force -ErrorAction SilentlyContinue

    if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
