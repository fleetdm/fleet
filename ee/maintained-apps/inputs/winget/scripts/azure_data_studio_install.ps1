# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# Azure Data Studio is a Visual Studio Code fork, so it ships the same Inno Setup
# script -- including the "runcode" task, which launches the app once the install
# finishes. PowerShell's "Start-Process -Wait" waits for the process *and all of
# its descendants*, so the launched app kept the install script blocked
# indefinitely. "/MERGETASKS=!runcode" suppresses the launch (the same switch
# vscode_install.ps1 and vscodium_install.ps1 use); waiting on the installer
# process alone, then stopping any stray app process, covers the rest.
# Budget: the caller kills the whole script at 10 minutes
# (cmd/maintained-apps/validate/windows.go), so the worst case here -- full install
# wait, then the registration wait, plus process overhead -- has to stay under
# that. 420 + 120 leaves headroom; 600 + 120 would have been killed mid-recovery,
# in exactly the case this script exists to handle.
$installTimeoutSeconds = 420
$registrationTimeoutSeconds = 120

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Test-AzureDataStudioRegistered {
    $null -ne (Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "Azure Data Studio*" } |
        Select-Object -First 1)
}

try {

$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /MERGETASKS=!runcode" `
  -PassThru
# Touch .Handle so the exit code is still readable after the process ends:
# Start-Process -PassThru otherwise returns $null for .ExitCode.
$null = $process.Handle

$killed = $false
if (-not $process.WaitForExit($installTimeoutSeconds * 1000)) {
  Write-Host "Installer process did not exit within ${installTimeoutSeconds}s, stopping it."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  # Wait for the kill to land: reading .ExitCode while the process is still alive
  # throws, which would fail the script even though the install may have succeeded.
  $null = $process.WaitForExit(30 * 1000)
  $killed = $true
}

$exitCode = $null
if ($process.HasExited) {
  $exitCode = $process.ExitCode
  Write-Host "Install exit code: $exitCode"
} else {
  Write-Host "Installer process could not be stopped; falling back to the registration check."
}

# The installer can return before the Add/Remove Programs entry is written; wait
# for it so software inventory sees a complete install.
$elapsed = 0
while (-not (Test-AzureDataStudioRegistered) -and ($elapsed -lt $registrationTimeoutSeconds)) {
  Start-Sleep -Seconds 5
  $elapsed += 5
  Write-Host "Waiting for Azure Data Studio to register... ($elapsed seconds)"
}

# Belt and braces in case a future build ignores !runcode.
Stop-Process -Name "azuredatastudio" -Force -ErrorAction SilentlyContinue

if (-not (Test-AzureDataStudioRegistered)) {
  Write-Host "Azure Data Studio did not register in Add/Remove Programs."
  Exit 1
}

# Registration is the authoritative success signal above. If the installer had to
# be killed, or never reported a code, don't fail on a code that means nothing.
if ($killed -or $null -eq $exitCode) { Exit 0 }

# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
