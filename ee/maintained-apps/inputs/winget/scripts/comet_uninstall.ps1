# Fleet extracts name from installer (EXE) and saves it to PACKAGE_ID
# variable
$softwareName = "Comet"

# It is recommended to use exact software name here if possible to avoid
# uninstalling unintended software.
$softwareNameLike = "*$softwareName*"

# Comet uses a Chromium/Omaha uninstaller. Its UninstallString already
# contains "--uninstall --system-level"; "--force-uninstall" runs it silently
# without a confirmation prompt.
$uninstallArgs = "--force-uninstall"

# Comet installs machine-wide, so look in the per-machine uninstall keys.
$machineKey = `
 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = `
 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

# Define acceptable/expected exit codes (19 = uninstall requires reboot)
$ExpectedExitCodes = @(0, 19)

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

# Treat acceptable exit codes as success
if ($ExpectedExitCodes -contains $exitCode) {
    Exit 0
} else {
    Exit $exitCode
}
