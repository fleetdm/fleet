# Windows EXE install scripts

## What are EXE install scripts?

EXE install scripts are a way to install software on Windows. EXE installers, such as `Figma-124.3.2.exe`, are self-contained packages with all the files and instructions needed to install software on a Windows host. EXE installers are fully customizable and do not follow the same installation process as MSI installers.

For EXE installers, there is no unique script or command that will work for all installers. MSI installers are typically preferred over EXE installers because they provide a standardized installation process, easier silent deployment, and better integration with Windows Installer Service. If available, MSI installers offer more predictable results in enterprise environments.

Some EXE installers and uninstallers require additional switches or flags to run silently. Common flags include `/S`, `/q`, `/quiet`, `/silent`, or `--silent`.

## Machine-scoped installs

The recommended way to install software on Windows hosts is to use machine-scoped install scripts. These scripts install the software for all local users on the computer and run the installation process with administrator privileges.

Here's an example **install** script:

```PowerShell
$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Add argument to install silently
# Argument to make install silent depends on installer,
# each installer might use different argument (usually it's "/S" or "/s")
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S"
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
```

Here's an example **uninstall** script:

```PowerShell
# Fleet extracts name from installer (EXE) and saves it to PACKAGE_ID
# variable
$softwareName = $PACKAGE_ID

# It is recommended to use exact software name here if possible to avoid
# uninstalling unintended software.
$softwareNameLike = "*$softwareName*"

# Some uninstallers require a flag to run silently.
# Each uninstaller might use different argument (usually it's "/S" or "/s")
$uninstallArgs = "/S"

$machineKey = `
 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = `
 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
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
    # Change exit code to 0 if you don't want to fail if uninstaller is not
    # found. This could happen if program was already uninstalled.
    $exitCode = 1
}

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode
```

## User-scoped installs

Some software can only be installed for a specific user. In this case, you can use user-scoped install scripts. The software is installed only for the user currently logged in, and the installation process is run with the user's privileges.

Here's an example **install** script:

```PowerShell
# Some installers require a flag to run silently.
# Each installer might use a different argument (usually it's "/S" or "/s")
$installArgs = "/S"

$exeFilePath = "${env:INSTALLER_PATH}"
$exeFilename = Split-Path $exeFilePath -Leaf
$taskName = "fleet-install-$exeFilename"
$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$taskHasNotRun = 267011  # SCHED_S_TASK_HAS_NOT_RUN
$exitCode = 0
$stagedInstaller = $null

try {

# Find the signed-in user. Include the domain: on Entra ID (Azure AD) joined
# hosts, the user name alone does not resolve to the user's account.
$owner = Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' |
    Invoke-CimMethod -MethodName GetOwner |
    Where-Object { $_.User } |
    Select-Object -First 1
if (-not $owner) {
    Throw "No user is signed in to this host. Sign in and try again."
}
$userAccount = "$($owner.Domain)\$($owner.User)"
Write-Host "Installing for $userAccount."

# Copy the installer to a public folder so that the user can access it
$stagedInstaller = "${env:PUBLIC}\$exeFilename"
Copy-Item -Path $exeFilePath -Destination $stagedInstaller -Force -ErrorAction Stop

# The task runs the installer as the signed-in user
$action = New-ScheduledTaskAction -Execute $stagedInstaller -Argument $installArgs
$principal = New-ScheduledTaskPrincipal -UserId $userAccount -LogonType Interactive
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
$task = New-ScheduledTask -Action $action -Principal $principal -Settings $settings
Register-ScheduledTask -TaskName $taskName -InputObject $task -Force -ErrorAction Stop | Out-Null

$startDate = Get-Date
Start-ScheduledTask -TaskName $taskName -ErrorAction Stop

# Wait for the task's result rather than for the "Running" state, which a fast
# installer can enter and leave between checks.
while ($true) {
    Start-Sleep -Seconds 5
    $state = (Get-ScheduledTask -TaskName $taskName).State
    $lastResult = (Get-ScheduledTaskInfo -TaskName $taskName).LastTaskResult
    if ($state -ne "Running" -and $lastResult -ne $taskRunning -and $lastResult -ne $taskHasNotRun) {
        $exitCode = $lastResult
        break
    }
    if ((New-TimeSpan -Start $startDate).TotalSeconds -gt 900) {
        Throw "Timed out waiting for the install task. State: $state, last result: 0x$('{0:X}' -f $lastResult)"
    }
}
Write-Host "Install exit code: $exitCode"

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    if ($stagedInstaller) {
        Remove-Item -Path $stagedInstaller -Force -ErrorAction SilentlyContinue
    }
}

# Exit codes above Int32.MaxValue, such as an HRESULT from the task, become 0
if ($exitCode -gt [int]::MaxValue) { $exitCode = 1 }
Exit $exitCode
```

The install script creates a scheduled task that runs the installer as the signed-in user. A user must be signed in to the host, or the script fails. The script identifies the user by domain and user name (for example, `AzureAD\JaneDoe`). The EXE installer is copied to a public directory so that even non-administrator users can run it. After the task finishes, the script reports the installer's exit code and deletes the installer and the task.

The use of scheduled tasks allows the installer to run with user-level permissions, which is especially useful when installing software for non-admin users without requiring administrator credentials at the time of execution.

Since the installation is run by the signed-in user, the script does not output the installer's messages to the console. If you need to see the output, you can modify the script to redirect it to a file and append it to the script output.

Here's an example **uninstall** script:

```PowerShell
# Fleet extracts the name from the installer (EXE) and saves it to PACKAGE_ID
# variable
$softwareName = $PACKAGE_ID

# Script to uninstall software as the current logged-in user.
$userScript = @'
$softwareName = $PACKAGE_ID

# Using the exact software name here is recommended to avoid
# uninstalling unintended software.
$softwareNameLike = "*$softwareName*"

# Some uninstallers require additional flags to run silently.
# Each uninstaller might use a different argument (usually it's "/S" or "/s")
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

Exit $exitCode
'@

$taskRunning = 267009  # SCHED_S_TASK_RUNNING
$taskHasNotRun = 267011  # SCHED_S_TASK_HAS_NOT_RUN
$exitCode = 0

# Create a script in a public folder so that it can be accessed by all users.
$uninstallScriptPath = "${env:PUBLIC}\uninstall-$softwareName.ps1"
$taskName = "fleet-uninstall-$softwareName"
try {
    # Find the signed-in user. Include the domain: on Entra ID (Azure AD) joined
    # hosts, the user name alone does not resolve to the user's account.
    $owner = Get-CimInstance Win32_Process -Filter 'name = "explorer.exe"' |
        Invoke-CimMethod -MethodName GetOwner |
        Where-Object { $_.User } |
        Select-Object -First 1
    if (-not $owner) {
        Throw "No user is signed in to this host. Sign in and try again."
    }
    $userAccount = "$($owner.Domain)\$($owner.User)"
    Write-Host "Uninstalling for $userAccount."

    Set-Content -Path $uninstallScriptPath -Value $userScript -Force -ErrorAction Stop

    # The task runs the uninstall script as the signed-in user
    $action = New-ScheduledTaskAction -Execute "PowerShell.exe" `
        -Argument "-NoProfile -ExecutionPolicy Bypass -File `"$uninstallScriptPath`""
    $principal = New-ScheduledTaskPrincipal -UserId $userAccount -LogonType Interactive
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
    $task = New-ScheduledTask -Action $action -Principal $principal -Settings $settings
    Register-ScheduledTask -TaskName $taskName -InputObject $task -Force -ErrorAction Stop | Out-Null

    $startDate = Get-Date
    Start-ScheduledTask -TaskName $taskName -ErrorAction Stop

    # Wait for the task's result rather than for the "Running" state, which a
    # fast uninstaller can enter and leave between checks.
    while ($true) {
        Start-Sleep -Seconds 5
        $state = (Get-ScheduledTask -TaskName $taskName).State
        $lastResult = (Get-ScheduledTaskInfo -TaskName $taskName).LastTaskResult
        if ($state -ne "Running" -and $lastResult -ne $taskRunning -and $lastResult -ne $taskHasNotRun) {
            $exitCode = $lastResult
            break
        }
        if ((New-TimeSpan -Start $startDate).TotalSeconds -gt 900) {
            Throw "Timed out waiting for the uninstall task. State: $state, last result: 0x$('{0:X}' -f $lastResult)"
        }
    }
    Write-Host "Uninstall exit code: $exitCode"

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Remove-Item -Path $uninstallScriptPath -Force -ErrorAction SilentlyContinue
}

# Exit codes above Int32.MaxValue, such as an HRESULT from the task, become 0
if ($exitCode -gt [int]::MaxValue) { $exitCode = 1 }
Exit $exitCode
```

The uninstall script writes a separate PowerShell script to a public directory and creates a scheduled task that runs it as the signed-in user. Like the install script, it needs a signed-in user. After the task finishes, the script reports the uninstall exit code and deletes the user script and the task.

Since the uninstall script is run by the signed-in user, it does not output its messages to the console. If you need to see the output, you can modify the main script to redirect it to a file and append it to the output.

## Install script for raw executables

Raw executables without installers are less common but may be used in specific scenarios, such as when a vendor provides a standalone binary file for a lightweight application. In these cases, ensuring all necessary dependencies are in place is important. Additionally, consider cleaning up the source executable after installation to avoid leaving unnecessary files on the system. If you have a raw executable that does not come with an installer, you can use the following script to install it. This script copies the executable to Program Files, which are accessible by all users.

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

## Conclusion

EXE install scripts provide a flexible solution for installing software on Windows hosts when MSI installers are unavailable. By leveraging the power of PowerShell and scheduled tasks, IT administrators can easily automate both machine-scoped and user-scoped installations. Whether you're deploying software for all users on a computer or targeting a specific logged-in user, the provided scripts offer a robust starting point for handling EXE installations.

Always verify the EXE installer's specific flags for silent installation for smoother operations, ensure proper permissions are in place, and consider implementing logging for troubleshooting. While MSI installers are generally preferred for their standardized behavior, these scripts allow you to manage even the most customized EXE installs in enterprise environments.

Following this guide will enable you to manage software deployments using EXE install scripts, improving efficiency and ensuring a seamless installation experience across your Windows devices.

<meta name="category" value="guides">
<meta name="authorFullName" value="Victor Lyuboslavsky">
<meta name="authorGitHubUsername" value="getvictor">
<meta name="publishedOn" value="2024-09-20">
<meta name="articleTitle" value="Windows EXE install scripts">
<meta name="description" value="This guide will walk you through adding software to Fleet using EXE installers.">
