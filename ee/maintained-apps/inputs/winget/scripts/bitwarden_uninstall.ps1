# Attempts to locate Bitwarden's uninstaller from registry and execute it silently

$displayName = "Bitwarden"
$publisher = "8bit Solutions LLC"

# Some uninstallers require a flag to run silently.
# NSIS installers typically use "/S" for silent uninstall
$uninstallArgs = "/S"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$exitCode = 0

try {
    # Kill any running Bitwarden processes before uninstalling
    Stop-Process -Name "Bitwarden" -Force -ErrorAction SilentlyContinue

    [array]$uninstallKeys = Get-ChildItem `
        -Path $paths `
        -ErrorAction SilentlyContinue |
            ForEach-Object { Get-ItemProperty $_.PSPath }

    $foundUninstaller = $false
    foreach ($key in $uninstallKeys) {
        if ($key.DisplayName -and ($key.DisplayName -eq $displayName -or $key.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $key.Publisher -eq $publisher)) {
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
            
            # Wait a moment for registry to update after uninstall completes
            # NSIS uninstallers may take a moment to clean up registry entries
            if ($exitCode -eq 0) {
                Start-Sleep -Seconds 3
            }
            
            # Exit the loop once the software is found and uninstalled.
            break
        }
    }

    if (-not $foundUninstaller) {
        Write-Host "Uninstaller for '$displayName' not found."
        # Change exit code to 0 if you don't want to fail if uninstaller is not
        # found. This could happen if program was already uninstalled.
        $exitCode = 0
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode

