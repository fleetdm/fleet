
# install switches:
#   /VERYSILENT          = no UI at all
#   /SUPPRESSMSGBOXES    = suppress every message box
#   /NORESTART           = don't reboot (returns 3010 instead, treat as success)
#   /ALLUSERS            = machine-scope install. Requires elevation; the
#                          installer self-elevates (manifest:
#                          ElevationRequirement: elevatesSelf), and Fleet/
#                          SYSTEM already runs elevated anyway.
#   /MERGETASKS=!runcode = keep default selectable tasks but deselect
#                          "runcode" (which auto-launches Windsurf after
#                          install). Inno's "!" prefix means "deselect".

$exeFilePath = "${env:INSTALLER_PATH}"

# 0 = success; 3010 = success but reboot required.
$ExpectedExitCodes = @(0, 3010)

try {
    $processOptions = @{
        FilePath     = "$exeFilePath"
        ArgumentList = "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART",
                       "/ALLUSERS", "/MERGETASKS=!runcode"
        PassThru     = $true
        Wait         = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    # Defensive: even with !runcode, Inno installers occasionally launch the
    # app via post-install scripts. Stop Windsurf if it slipped through so
    # files aren't locked for later detection/uninstall.
    Start-Sleep -Seconds 5
    Stop-Process -Name "Windsurf" -Force -ErrorAction SilentlyContinue

    if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
