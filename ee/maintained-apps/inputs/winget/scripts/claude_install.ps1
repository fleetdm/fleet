# MSIX: provision machine-wide so the app is available to all users at sign-in, then
# opportunistically register for the currently logged-on console user (via a scheduled
# task in their session) so the app is immediately visible without requiring sign-out.
#
# The Fleet agent runs as Local System on Windows, and Add-AppxPackage cannot run in that
# context (HRESULT 0x80073CF9). The scheduled task is the supported way to register a
# package in a user session from a system-context script.

$softwareName = "Claude"
$taskName = "fleet-install-$softwareName.msix"
$scriptPath = "$env:PUBLIC\install-$softwareName.ps1"
$exitCodeFile = "$env:PUBLIC\install-exitcode-$softwareName.txt"

try {

  $msixPath = $env:INSTALLER_PATH
  if (-not $msixPath) {
    throw "INSTALLER_PATH is not set"
  }

  Write-Host "Provisioning MSIX for all users..."
  $result = Add-AppxProvisionedPackage -Online -PackagePath $msixPath -SkipLicense -Regions "all" -ErrorAction Stop
  $result | Out-String | Write-Host

  # Win32_ComputerSystem.UserName returns the console user (DOMAIN\User) or null when no
  # interactive session is active. Other RDP/fast-user-switch sessions won't get the
  # immediate registration; those users will pick it up from the provisioned install at
  # their next sign-in.
  $userName = (Get-CimInstance Win32_ComputerSystem).UserName
  if (-not $userName -or $userName -notlike "*\*") {
    Write-Host "No interactive user logged on; provisioned install will register for each user at sign-in."
    Start-Sleep -Seconds 5
    Exit 0
  }

  Write-Host "Registering MSIX for logged-on user '$userName' via scheduled task..."

  $userScript = @"
`$msixPath = "$msixPath"
`$exitCodeFile = "$exitCodeFile"
try {
  Add-AppxPackage -Path `$msixPath -ErrorAction Stop | Out-String | Write-Host
  Set-Content -Path `$exitCodeFile -Value 0
} catch {
  Write-Host "Add-AppxPackage failed: `$(`$_.Exception.Message)"
  Set-Content -Path `$exitCodeFile -Value 1
}
"@

  Set-Content -Path $scriptPath -Value $userScript -Force

  $action = New-ScheduledTaskAction -Execute "powershell.exe" `
    -Argument "-WindowStyle Hidden -ExecutionPolicy Bypass -File `"$scriptPath`""
  $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries
  $principal = New-ScheduledTaskPrincipal -UserId $userName -RunLevel Highest
  $task = New-ScheduledTask -Action $action -Settings $settings -Principal $principal
  Register-ScheduledTask -TaskName $taskName -InputObject $task -User $userName -Force | Out-Null
  Start-ScheduledTask -TaskName $taskName

  $startDate = Get-Date
  $state = (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue).State
  while ($state -ne "Running") {
    Start-Sleep -Seconds 1
    if ((New-Timespan -Start $startDate).TotalSeconds -gt 30) {
      Write-Host "Per-user registration task did not start within 30s; provisioned install is still valid."
      break
    }
    $state = (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue).State
  }

  while ($state -eq "Running") {
    Start-Sleep -Seconds 2
    if ((New-Timespan -Start $startDate).TotalSeconds -gt 90) {
      Write-Host "Per-user registration task did not complete within 90s; provisioned install is still valid."
      break
    }
    $state = (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue).State
  }

  if (Test-Path $exitCodeFile) {
    $code = (Get-Content $exitCodeFile -ErrorAction SilentlyContinue | Select-Object -First 1).Trim()
    if ($code -eq "0") {
      Write-Host "Per-user registration completed for '$userName'."
    } else {
      Write-Host "Per-user registration did not complete cleanly (exit code: $code). Provisioned install is still valid."
    }
  }

  Start-Sleep -Seconds 5
  Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1
} finally {
  Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
  Remove-Item -Path $scriptPath -Force -ErrorAction SilentlyContinue
  Remove-Item -Path $exitCodeFile -Force -ErrorAction SilentlyContinue
}
