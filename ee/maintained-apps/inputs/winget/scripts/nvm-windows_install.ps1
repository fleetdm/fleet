# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# NVM for Windows installs per user, so this script runs setup as the signed-in user.
# A standard user sees a UAC prompt; canceling it still completes the install.

$exeFilePath = "${env:INSTALLER_PATH}"
$taskName = "fleet-install-nvm-windows"
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$exitCode = 0
$stagedInstaller = $null

try {
    $owners = @(Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' -ErrorAction SilentlyContinue |
        Invoke-CimMethod -MethodName GetOwner -ErrorAction SilentlyContinue |
        Where-Object { $_.User })
    if ($owners.Count -eq 0) {
        Throw "NVM for Windows installs per user and no user is signed in to this host. Sign in and try again."
    }

    # With several users signed in, upgrade whoever has the oldest copy, since that's
    # the copy the patch policy flags.
    $owner = $owners[0]
    $oldest = $null
    foreach ($candidate in $owners) {
        try {
            $sid = (New-Object System.Security.Principal.NTAccount("$($candidate.Domain)\$($candidate.User)")).Translate(
                [System.Security.Principal.SecurityIdentifier]).Value
        } catch { continue }
        foreach ($sub in (Get-ChildItem "Registry::HKEY_USERS\$sid\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall" -ErrorAction SilentlyContinue)) {
            $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
            if ($key.DisplayName -ne "NVM for Windows" -or $key.Publisher -ne "Author Software Inc.") { continue }
            $version = $null
            if (-not [version]::TryParse([string]$key.DisplayVersion, [ref]$version)) { $version = [version]"0.0" }
            if ($null -eq $oldest -or $version -lt $oldest) {
                $oldest = $version
                $owner = $candidate
            }
        }
    }
    $userAccount = "$($owner.Domain)\$($owner.User)"
    Write-Host "Installing NVM for Windows for $userAccount."

    # Fleet's installer directory is not readable by that user.
    $stagedInstaller = Join-Path $env:PUBLIC (Split-Path $exeFilePath -Leaf)
    Copy-Item -Path $exeFilePath -Destination $stagedInstaller -Force -ErrorAction Stop

    $installArgs = "/SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /CLOSEAPPLICATIONS"
    if ($installArgs) {
        $action = New-ScheduledTaskAction -Execute $stagedInstaller -Argument $installArgs
    } else {
        $action = New-ScheduledTaskAction -Execute $stagedInstaller
    }
    $trigger = New-ScheduledTaskTrigger -AtLogOn
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
    $principal = New-ScheduledTaskPrincipal -UserId $userAccount -RunLevel Highest
    $task = New-ScheduledTask -Action $action -Trigger $trigger -Settings $settings -Principal $principal
    Register-ScheduledTask -TaskName $taskName -InputObject $task -Force | Out-Null

    $startDate = Get-Date
    Start-ScheduledTask -TaskName $taskName

    Start-Sleep -Seconds 2
    while ($true) {
        $info = Get-ScheduledTaskInfo -TaskName $taskName
        $state = (Get-ScheduledTask -TaskName $taskName).State
        if ($state -ne "Running" -and $info.LastTaskResult -ne $taskRunning) {
            $exitCode = $info.LastTaskResult
            break
        }
        if ((New-TimeSpan -Start $startDate).TotalSeconds -gt 900) {
            Throw "Timed out waiting for the install task to finish. If $userAccount is not a local admin, setup may be waiting on a UAC prompt."
        }
        Start-Sleep -Seconds 5
    }
    Write-Host "Install exit code: $exitCode"

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    if (Get-ScheduledTask -TaskName $taskName -ErrorAction SilentlyContinue) {
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    }
    if ($stagedInstaller -and (Test-Path -LiteralPath $stagedInstaller)) {
        Remove-Item -LiteralPath $stagedInstaller -Force -ErrorAction SilentlyContinue
    }
}

# Exit turns a code above Int32.MaxValue, such as a task's HRESULT, into 0.
if ($exitCode -gt [int]::MaxValue) { $exitCode = 1 }
Exit $exitCode
