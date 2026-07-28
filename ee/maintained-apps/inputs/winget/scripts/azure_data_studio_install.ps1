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
$installTimeoutSeconds = 600
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

if (-not $process.WaitForExit($installTimeoutSeconds * 1000)) {
  Write-Host "Installer process did not exit within ${installTimeoutSeconds}s, stopping it."
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  Start-Sleep -Seconds 2
}

$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

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

# 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
