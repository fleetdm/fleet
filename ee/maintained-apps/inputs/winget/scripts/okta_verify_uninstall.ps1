# Fleet extracts name from installer (EXE) and saves it to PACKAGE_ID
# variable
$softwareName = $PACKAGE_ID

# It is recommended to use exact software name here if possible to avoid
# uninstalling unintended software.
$softwareNameLike = "*Okta Verify*"

# WiX Burn bootstrapper uses /quiet for silent uninstall
$uninstallArgs = "/quiet /norestart"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
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
        # "C:\Program Files\Software\uninstall.exe" /quiet
        # Split the command and args
        $splitArgs = $uninstallCommand.Split('"')
        if ($splitArgs.Length -gt 1) {
            if ($splitArgs.Length -eq 3) {
                $existingArgs = $splitArgs[2].Trim()
                if ($existingArgs -notmatch '/quiet') {
                    $uninstallArgs = "$existingArgs /quiet /norestart".Trim()
                } else {
                    $uninstallArgs = $existingArgs
                }
            } elseif ($splitArgs.Length -gt 3) {
                Throw `
                    "Uninstall command contains multiple quoted strings. " +
                        "Please update the uninstall script.`n" +
                        "Uninstall command: $uninstallCommand"
            }
            $uninstallCommand = $splitArgs[1]
        } else {
            if ($uninstallCommand -notmatch '/quiet') {
                $uninstallArgs = "/quiet /norestart"
            } else {
                $uninstallArgs = ""
            }
        }
        Write-Host "Uninstall command: $uninstallCommand"
        Write-Host "Uninstall args: $uninstallArgs"

        $processOptions = @{
            FilePath = $uninstallCommand
            PassThru = $true
            Wait = $true
        }

        if ($uninstallArgs -ne '') {
            $processOptions.ArgumentList = $uninstallArgs
        }

        $process = Start-Process @processOptions
        $exitCode = $process.ExitCode
        Write-Host "Uninstall exit code: $exitCode"
        break
    }
}

if (-not $foundUninstaller) {
    Write-Host "Uninstall entry not found for $softwareNameLike"
    Exit 0
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
