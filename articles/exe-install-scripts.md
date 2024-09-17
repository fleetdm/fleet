# Windows EXE install scripts

## What are EXE install scripts?

EXE install scripts are a way to install software on Windows. EXE installers, such as `Figma-124.3.2.exe`, are self-contained packages that include all the files and instructions needed to install software on a Windows device. EXE installers are fully customizable and do not follow the same installation process as MSI installers.

For EXE installers, there is no unique script or command that will work for all installers. Because of that, we advise using MSI installers whenever possible.

Some EXE installers and uninstallers require additional switches or flags to run silently. Common flags include `/S`, `/q`, `/quiet`, `/silent`, or `--silent`.

## Device-scoped install scripts

The recommended way to install software on Windows devices is to use device-scoped install scripts. The software is installed for all users of the device, and the installation process is run with administrator privileges.

Fleet defaults to a device-scoped install scripts when you add software using an EXE installer.

## User-scoped install scripts

Some software can only be installed for a specific user. In this case, you can use user-scoped install scripts. The software is installed only for the user who is currently logged in, and the installation process is run with the user's privileges.

### Example user-scoped install script

The install script creates a scheduled task that will automatically be run as the current (logged-in) user. The EXE installer is copied to a public directory accessible by the user. After the task is done, the installer and the task are both deleted.

Since the installation is run by the current user, the script does not output the messages from the installer to the console. If you need to see the output, you can modify the script to redirect the output to a file and append it to the script output.

```powershell
# Some installers require a flag to run silently.
# Each installer might use different argument (usually it's "/S" or "/s")
$installArgs = "/S"

$exeFilePath = "${env:INSTALLER_PATH}"

$exitCode = 0

try {

# Copy the installer to public folder so that it can be accessed by all
# users
$exeFilename = Split-Path $exeFilePath -leaf
Copy-Item -Path $exeFilePath -Destination "${env:PUBLIC}" -Force
$exeFilePath = "${env:PUBLIC}\$exeFilename"

# Task properties. Task will be started by logged in user
$action = New-ScheduledTaskAction -Execute "$exeFilePath" `
    -Argument "$installArgs"
$trigger = New-ScheduledTaskTrigger -AtLogOn
$userName = Get-CimInstance -ClassName Win32_ComputerSystem |
        Select-Object -expand UserName
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries

# Create task object with properties defined above
$task = New-ScheduledTask -Action $action -Trigger $trigger `
    -Settings $settings

# Register the task
$taskName = "fleet-install-$exeFilename"
Register-ScheduledTask "$taskName" -InputObject $task -User "$userName"

# keep track of start time, to cancel if taking too long to start
$startDate = Get-Date

# Start the task now that it is ready
Start-ScheduledTask -TaskName "$taskName" -TaskPath "\"

# Wait for the task to be running
$state = (Get-ScheduledTask -TaskName "$taskName").State
Write-Host "ScheduledTask is '$state'"

while ($state  -ne "Running") {
    Write-Host "ScheduledTask is '$state'. Waiting to run .exe..."

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

# Remove task
Write-Host "Removing ScheduledTask: $taskName."
Unregister-ScheduledTask -TaskName "$taskName" -Confirm:$false

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    # Remove installer
    Remove-Item -Path $exeFilePath -Force
}

Exit $exitCode
```

### Example user-scoped uninstall script

The uninstall script creates a scheduled task that will automatically be run as the current (logged-in) user. The uninstaller creates a separate PowerShell script for the user. After the task is done, the script and the task are both deleted.

Since the uninstall script is run by the current user, that script does not output its messages to the console. If you need to see the output, you can modify the main script to redirect the output to a file and append it to the output.

```powershell
# Fleet extracts name from installer (EXE) and saves it to PACKAGE_ID
# variable
$softwareName = $PACKAGE_ID

# Script to uninstall software as the current logged in user.
$userScript = @'
$softwareName = $PACKAGE_ID

# It is recommended to use exact software name here if possible to avoid
# uninstalling unintended software.
$softwareNameLike = "*$softwareName*"

# Some uninstallers require additional flags to run silently.
# Each uninstaller might use different argument (usually it's "/S" or "/s")
$uninstallArgs = "/S"

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

        # Start process and track exit code
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

Exit $exitCode
'@

$exitCode = 0

# Create script in public folder so that it can be accessed by all users.
$uninstallScriptPath = "${env:PUBLIC}/uninstall-$softwareName.ps1"
$taskName = "fleet-uninstall-$softwareName"
try {
    Set-Content -Path $uninstallScriptPath -Value $userScript -Force

    # Task properties. Task will be started by logged in user
    $action = New-ScheduledTaskAction -Execute "PowerShell.exe" `
        -Argument "$uninstallScriptPath"
    $trigger = New-ScheduledTaskTrigger -AtLogOn
    $userName = Get-CimInstance -ClassName Win32_ComputerSystem |
            Select-Object -expand UserName
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries

    # Create task object with properties defined above
    $task = New-ScheduledTask -Action $action -Trigger $trigger `
        -Settings $settings

    # Register the task
    Register-ScheduledTask "$taskName" -InputObject $task -User "$userName"

    # keep track of start time, to cancel if taking too long to start
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
```

## Install script for raw executables

If you have a raw executable that does not come with an installer, you can use the following script to install it. This script copies the executable to Program Files accessible by all users.

```powershell
$exeFilePath = "${env:INSTALLER_PATH}"

try {

# extract the name of the executable to use as the sub-directory name
$exeName = [System.IO.Path]::GetFileName($exeFilePath)
$subDir = [System.IO.Path]::GetFileNameWithoutExtension($exeFilePath)

$destinationPath = Join-Path -Path $env:ProgramFiles -ChildPath $subDir

# check if the directory does not exist, and create it if necessary
if (-not (Test-Path -Path $destinationPath)) {
    New-Item -ItemType Directory -Path $destinationPath
}

# copy the .exe file to the new sub-directory
$destinationExePath = Join-Path -Path $destinationPath -ChildPath $exeName
Copy-Item -Path $exeFilePath -Destination $destinationExePath
Exit $LASTEXITCODE

} catch {
    Write-Host "Error: $_"
    Exit 1
}
```
