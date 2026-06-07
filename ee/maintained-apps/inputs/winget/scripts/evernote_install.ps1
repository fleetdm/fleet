# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Evernote is an electron-builder NSIS (assisted/multi-user) installer.
# /AllUsers selects the per-machine scope (required for Fleet's SYSTEM context),
# /S runs it silently. Order matters: the scope switch MUST come before /S so the
# installer resolves the per-machine target before entering silent mode. With the
# reverse order ("/S /allusers") the scope switch is ignored, the installer falls
# back to a per-user code path and crashes with 0xC0000005 (access violation) under
# the SYSTEM account. The "/AllUsers /S" ordering matches the documented machine-wide
# deployment switches (Intune / ManageEngine / silentinstallhq).
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/AllUsers /S"
  PassThru = $true
  Wait = $true
}

# Start process and track exit code
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# Prints the exit code
Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
