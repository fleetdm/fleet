$softwareName = "Brave"

# Script to uninstall software as the current logged-in user.
$userScript = @'

# Define acceptable/expected exit codes
$ExpectedExitCodes = @(0, 19)

$softwareName = "Brave"

# Using the exact software name here is recommended to avoid
# uninstalling unintended software.
$softwareNameLike = "*$softwareName*"

# Some uninstallers require additional flags to run silently.
# Each uninstaller might use a different argument (usually it's "/S" or "/s")
$uninstallArgs = "--force-uninstall"

$uninstallCommand = ""
$exitCode = 0

try {

$userKey = `
 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*'
[array]$uninstallKeys = Get-ChildItem `
    -Path @($userKey) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

$foundUninstaller = $false
foreach ($key in $uninstallKeys) {
    # If needed, add -notlike to the comparison to exclude certain similar
    # software
    if ($key.DisplayName -like $softwareNameLike) {
        $foundUninstaller = $true
        # Get the uninstall command. Some uninstallers do not include
        # 'QuietUninstallString' and require a flag to run silently.
        $uninstallCommand = if ($key.QuietUninstallString) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

        # The uninstall command may contain command and args, like:
        # "C:\Program Files\Software\uninstall.exe" --uninstall --silent
        # Split the command and args
        $splitArgs = $uninstallCommand.Split('"')
        if ($splitArgs.Length -gt 1) {
            if ($splitArgs.Length -eq 3) {
                $uninstallArgs = "$( $splitArgs[2] ) $uninstallArgs".Trim()
            } elseif ($splitArgs.Length -gt 3) {
                Throw `
                    "Uninstall command contains multiple quoted strings. " +
                        "Please update the uninstall script.`n" +
                        "Uninstall command: $uninstallCommand"
            }
            $uninstallCommand = $splitArgs[1]
        }
        Write-Host "Uninstall command: $uninstallCommand"
        Write-Host "Uninstall args: $uninstallArgs"

        $processOptions = @{
            FilePath = $uninstallCommand
            PassThru = $true
            Wait = $true
        }
        if ($uninstallArgs -ne '') {
            $processOptions.ArgumentList = "$uninstallArgs"
        }

        # Start the process and track the exit code
        $process = Start-Process @processOptions
        $exitCode = $process.ExitCode

        # Prints the exit code
        Write-Host "Uninstall exit code: $exitCode"
        # Exit the loop once the software is found and uninstalled.
        break
    }
}

if (-not $foundUninstaller) {
    Write-Host "Uninstaller for '$softwareName' not found."
    $exitCode = 1
}

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

# Treat acceptable exit codes as success
if ($ExpectedExitCodes -contains $exitCode) {
    Exit 0
} else {
    Exit $exitCode
}
'@

$exitCode = 0

# Create a script in a public folder so that it can be accessed by all users.
$uninstallScriptPath = "${env:PUBLIC}/uninstall-$softwareName.ps1"
$taskName = "fleet-uninstall-$softwareName"
try {
    Set-Content -Path $uninstallScriptPath -Value $userScript -Force

    # Task properties. The task will be started by the logged in user
    $action = New-ScheduledTaskAction -Execute "PowerShell.exe" `
        -Argument "$uninstallScriptPath"
    $trigger = New-ScheduledTaskTrigger -AtLogOn
    $userName = (Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' | Invoke-CimMethod -MethodName getowner).User
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries

    # Create a task object with the properties defined above
    $task = New-ScheduledTask -Action $action -Trigger $trigger `
        -Settings $settings

    # Register the task
    Register-ScheduledTask "$taskName" -InputObject $task -User "$userName"

    # keep track of the start time to cancel if taking too long to start
    $startDate = Get-Date

    # Start the task now that it is ready
    Start-ScheduledTask -TaskName "$taskName" -TaskPath "\"

    # Wait for the task to be running
    $state = (Get-ScheduledTask -TaskName "$taskName").State
    Write-Host "ScheduledTask is '$state'"

    while ($state  -ne "Running") {
        Write-Host "ScheduledTask is '$state'. Waiting to uninstall..."

        $endDate = Get-Date
        $elapsedTime = New-Timespan -Start $startDate -End $endDate
        if ($elapsedTime.TotalSeconds -gt 120) {
            Throw "Timed-out waiting for scheduled task state."
        }

        Start-Sleep -Seconds 1
        $state = (Get-ScheduledTask -TaskName "$taskName").State
    }

    # Wait for the task to be done
    $state = (Get-ScheduledTask -TaskName "$taskName").State
    while ($state  -eq "Running") {
        Write-Host "ScheduledTask is '$state'. Waiting for .exe to complete..."

        $endDate = Get-Date
        $elapsedTime = New-Timespan -Start $startDate -End $endDate
        if ($elapsedTime.TotalSeconds -gt 120) {
            Throw "Timed-out waiting for scheduled task state."
        }

        Start-Sleep -Seconds 10
        $state = (Get-ScheduledTask -TaskName "$taskName").State
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    # Remove task
    Write-Host "Removing ScheduledTask: $taskName."
    Unregister-ScheduledTask -TaskName "$taskName" -Confirm:$false

    # Remove user script
    Remove-Item -Path $uninstallScriptPath -Force
}

Exit $exitCode