# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# /AllUsers installs ocenaudio machine-wide. Without it, the installer puts the
# app in the SYSTEM account's profile, where no one can open it. A copy left
# there by an earlier install is removed first.

$exeFilePath = "${env:INSTALLER_PATH}"
$legacyKey = 'Registry::HKEY_USERS\S-1-5-18\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\ocenaudio'

try {
  $legacy = Get-ItemProperty -Path $legacyKey -ErrorAction SilentlyContinue
  if ($legacy -and $legacy.UninstallString -match '^\s*"([^"]+)"') {
    # The 32-bit installer wrote it under SysWOW64 but registered the system32 path.
    $legacyExe = $matches[1] -replace '(?i)\\system32\\', '\SysWOW64\'
    $legacyDir = Split-Path $legacyExe -Parent
    Write-Host "Removing the copy of ocenaudio in $legacyDir"
    if (Test-Path -LiteralPath $legacyExe) {
      Start-Process -FilePath $legacyExe -ArgumentList "/CurrentUser /S _?=$legacyDir" -Wait -NoNewWindow
    }
    if ((Split-Path $legacyDir -Leaf) -eq "ocenaudio") {
      Remove-Item -LiteralPath $legacyDir -Recurse -Force -ErrorAction SilentlyContinue
    }
    Remove-Item -Path $legacyKey -Recurse -Force -ErrorAction SilentlyContinue
  }

  $processOptions = @{
    FilePath     = "$exeFilePath"
    ArgumentList = "/S /AllUsers"
    PassThru     = $true
    Wait         = $true
    NoNewWindow  = $true
  }

  $process = Start-Process @processOptions
  $exitCode = $process.ExitCode

  Write-Host "Install exit code: $exitCode"
  Exit $exitCode
} catch {
  Write-Host "Error: $_"
  Exit 1
}
