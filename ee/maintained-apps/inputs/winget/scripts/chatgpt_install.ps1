# MSIX: provision machine-wide so the app is available to all users at sign-in, then
# opportunistically register for the currently logged-on console user (via a scheduled
# task in their session) so the app is immediately visible without requiring sign-out.
#
# The Fleet agent runs as Local System on Windows, and Add-AppxPackage cannot run in that
# context (HRESULT 0x80073CF9). The scheduled task is the supported way to register a
# package in a user session from a system-context script.

$softwareName = "ChatGPT"
$taskName = "fleet-install-$softwareName.msix"
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$stageDir = $null

try {

  $msixPath = $env:INSTALLER_PATH
  if (-not $msixPath) {
    throw "INSTALLER_PATH is not set"
  }

  Add-Type -AssemblyName System.IO.Compression.FileSystem
  $zip = [System.IO.Compression.ZipFile]::OpenRead($msixPath)
  try {
    $reader = New-Object System.IO.StreamReader($zip.GetEntry("AppxManifest.xml").Open())
    [xml]$manifest = $reader.ReadToEnd()
    $reader.Dispose()
  } finally {
    $zip.Dispose()
  }
  $identity = $manifest.Package.Identity

  # ChatGPT updates itself, so the host may already be on a newer version than this
  # installer. Windows won't provision an older version over it (0xc1570118 when the newer
  # version is provisioned, "Element not found" when it's installed only for a user).
  $newer = @(Get-AppxProvisionedPackage -Online | Where-Object { $_.DisplayName -eq $identity.Name }) +
    @(Get-AppxPackage -AllUsers -Name $identity.Name) |
    Where-Object { [version]$_.Version -gt [version]$identity.Version } |
    Sort-Object { [version]$_.Version } -Descending |
    Select-Object -First 1
  if ($newer) {
    Write-Host "$softwareName $($newer.Version) is already on this host, which is newer than this installer ($($identity.Version))."

    # A copy registered only for other users isn't available to the console user, not even
    # after they sign in again, so register the copy that's already on the host for them.
    $userName = (Get-CimInstance Win32_ComputerSystem).UserName
    if (-not $userName -or $userName -notlike "*\*") {
      Write-Host "No interactive user logged on. Nothing to install."
      Exit 0
    }
    $registered = {
      Get-AppxPackage -User $userName -Name $identity.Name |
        Where-Object { [version]$_.Version -ge [version]$identity.Version }
    }
    if (& $registered) {
      Write-Host "It's already registered for '$userName'. Nothing to install."
      Exit 0
    }

    $familyName = (Get-AppxPackage -AllUsers -Name $identity.Name | Select-Object -First 1).PackageFamilyName
    Write-Host "Registering $familyName for '$userName' via scheduled task..."
    $action = New-ScheduledTaskAction -Execute "powershell.exe" `
      -Argument "-NoProfile -WindowStyle Hidden -Command `"Add-AppxPackage -RegisterByFamilyName -MainPackage '$familyName'`""
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
    $principal = New-ScheduledTaskPrincipal -UserId $userName -RunLevel Highest
    $task = New-ScheduledTask -Action $action -Settings $settings -Principal $principal
    Register-ScheduledTask -TaskName $taskName -InputObject $task -User $userName -Force | Out-Null
    Start-ScheduledTask -TaskName $taskName

    $startDate = Get-Date
    while (-not (& $registered)) {
      if ((New-TimeSpan -Start $startDate).TotalSeconds -gt 90) {
        Stop-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
        Write-Host "Error: $softwareName wasn't registered for '$userName' within 90s."
        Exit 1
      }
      Start-Sleep -Seconds 2
    }
    Write-Host "Registered $softwareName for '$userName'."
    Exit 0
  }

  # Uninstalling for all users leaves this version marked for removal at each user's next
  # sign-in. Clear those marks so reinstalling the same version isn't undone.
  Get-ChildItem 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Appx\AppxAllUserStore\EndOfLife' -ErrorAction SilentlyContinue |
    Get-ChildItem -ErrorAction SilentlyContinue |
    Where-Object { $_.PSChildName -like "$($identity.Name)_*" } |
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue

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

  # The task runs elevated as that user, so stage its script in a new folder only SYSTEM
  # and Administrators can write to. Add-AppxPackage reads the package as that user, who
  # can't read Fleet's installer directory, so stage a copy of the package there too.
  $sid = (New-Object System.Security.Principal.NTAccount($userName)).Translate(
    [System.Security.Principal.SecurityIdentifier]).Value
  $security = New-Object System.Security.AccessControl.DirectorySecurity
  $security.SetAccessRuleProtection($true, $false)
  foreach ($grant in @(@("S-1-5-18", "FullControl"), @("S-1-5-32-544", "FullControl"), @($sid, "ReadAndExecute"))) {
    $security.AddAccessRule((New-Object System.Security.AccessControl.FileSystemAccessRule(
      (New-Object System.Security.Principal.SecurityIdentifier($grant[0])), $grant[1],
      "ContainerInherit, ObjectInherit", "None", "Allow")))
  }
  $stageDir = Join-Path $env:ProgramData ("fleet-chatgpt-" + [guid]::NewGuid().ToString("N"))
  [void][System.IO.Directory]::CreateDirectory($stageDir, $security)
  $stagedMsix = Join-Path $stageDir (Split-Path $msixPath -Leaf)
  Copy-Item -LiteralPath $msixPath -Destination $stagedMsix -Force -ErrorAction Stop
  $scriptPath = Join-Path $stageDir "install-$softwareName.ps1"

  $userScript = @'
param([string]$MsixPath)
try {
  Add-AppxPackage -Path $MsixPath -ErrorAction Stop | Out-String | Write-Host
} catch {
  Write-Host "Add-AppxPackage failed: $($_.Exception.Message)"
  exit 1
}
'@

  Set-Content -Path $scriptPath -Value $userScript -Force

  $action = New-ScheduledTaskAction -Execute "powershell.exe" `
    -Argument "-NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -File `"$scriptPath`" `"$stagedMsix`""
  $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
  $principal = New-ScheduledTaskPrincipal -UserId $userName -RunLevel Highest
  $task = New-ScheduledTask -Action $action -Settings $settings -Principal $principal
  Register-ScheduledTask -TaskName $taskName -InputObject $task -User $userName -Force | Out-Null

  $startDate = Get-Date
  $lastRun = (Get-ScheduledTaskInfo -TaskName $taskName -ErrorAction Stop).LastRunTime
  Start-ScheduledTask -TaskName $taskName

  # Wait for a result rather than for the "Running" state, which a fast task can
  # enter and leave between polls. A task that's still queued hasn't updated
  # LastRunTime yet.
  $code = $null
  Start-Sleep -Seconds 2
  while ($true) {
    $info = Get-ScheduledTaskInfo -TaskName $taskName -ErrorAction Stop
    $state = (Get-ScheduledTask -TaskName $taskName -ErrorAction Stop).State
    if ($info.LastRunTime -ne $lastRun -and $state -ne "Running" -and $info.LastTaskResult -ne $taskRunning) {
      $code = $info.LastTaskResult
      break
    }
    if ((New-TimeSpan -Start $startDate).TotalSeconds -gt 90) {
      Stop-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue
      Write-Host "Per-user registration task did not complete within 90s; provisioned install is still valid."
      break
    }
    Start-Sleep -Seconds 2
  }

  if ($code -eq 0) {
    Write-Host "Per-user registration completed for '$userName'."
  } elseif ($null -ne $code) {
    Write-Host "Per-user registration did not complete cleanly (exit code: $code). Provisioned install is still valid."
  }

  Start-Sleep -Seconds 5
  Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1
} finally {
  Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue | Out-Null
  if ($stageDir -and (Test-Path -LiteralPath $stageDir)) {
    Remove-Item -LiteralPath $stageDir -Recurse -Force -ErrorAction SilentlyContinue
  }
}
