# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# iMazing HEIC Converter ships an Inno Setup 6.1 installer. Its [Run] section
# has a postinstall entry that launches "{app}\iMazing HEIC Converter.exe" and
# is NOT flagged "skipifsilent", so the GUI starts even under /VERYSILENT.
# PowerShell's "Start-Process -Wait" waits for the process AND its descendants,
# so that lingering GUI keeps the install step blocked on a headless host even
# though the install itself succeeds.
#
# So we pass "/nolaunch" (a switch the installer's own script parses, alongside
# "/silent" and "/verysilent") to keep the postinstall launch from firing, wait
# on the Setup process ITSELF rather than its descendants, and stop the GUI if
# it started anyway. "/nolaunch" is lowercase to match the literal the
# installer script compares against.

$exeFilePath = "${env:INSTALLER_PATH}"

$timeoutSeconds = 300

# Recursively stop the Setup process and any children (e.g. Inno's setup.tmp
# helper) so the installer file lock is released.
function Stop-ProcessTree {
    param([int]$ParentId)
    Get-CimInstance Win32_Process -Filter "ParentProcessId = $ParentId" -ErrorAction SilentlyContinue |
        ForEach-Object { Stop-ProcessTree -ParentId $_.ProcessId }
    Stop-Process -Id $ParentId -Force -ErrorAction SilentlyContinue
}

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    # NOTE: intentionally launched WITHOUT -Wait; see comment above.
    $process = Start-Process -FilePath "$exeFilePath" `
        -ArgumentList "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /nolaunch" -PassThru
    Write-Host "Launched iMazing HEIC Converter installer (PID: $($process.Id))"

    $exited = $process.WaitForExit($timeoutSeconds * 1000)

    # Stop the app in case the installer launched it anyway, so it can't hold a
    # file lock or interfere with the rest of the run. This only ends the
    # running process; it does not uninstall anything.
    Stop-Process -Name "iMazing HEIC Converter" -Force -ErrorAction SilentlyContinue

    if (-not $exited) {
        Write-Host "Installer did not exit within ${timeoutSeconds}s; stopping it."
        Stop-ProcessTree -ParentId $process.Id
        Exit 1
    }

    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
