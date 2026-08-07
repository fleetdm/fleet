# Fleet extracts name from installer (EXE) and saves it to PACKAGE_ID
# variable
# Evernote (electron-builder NSIS) registers a versioned DisplayName in HKLM
# when installed with /allusers, e.g. "Evernote 11.24.3 (All Users)". The
# "(All Users)" suffix is locale-dependent, so match on the "Evernote" prefix
# only. Its QuietUninstallString runs "Uninstall Evernote.exe" /allusers /S,
# which also closes the running tray app.
$softwareName = "Evernote"

$softwareNameLike = "Evernote*"

# electron-builder NSIS uninstaller; /allusers matches the machine-wide install
$uninstallArgs = "/allusers /S"

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
        # "C:\Program Files\Software\uninstall.exe" /SILENT
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

        # Without an "_?=<installdir>" argument, an NSIS uninstaller copies
        # itself to %TEMP%, relaunches the copy detached, and the process we
        # waited on exits 0 immediately -- so the removal is still in flight
        # here. Poll the uninstall registry key until the entry disappears so
        # this script only reports success once the app is really gone.
        if ($exitCode -eq 0) {
            $deadline = (Get-Date).AddSeconds(300)
            do {
                Start-Sleep -Seconds 5
                # Fail closed: only an enumeration that actually succeeds may
                # be read as proof of removal. If the query itself fails, the
                # empty result says nothing about whether Evernote is gone,
                # so keep the previous "still installed" answer rather than
                # reporting a successful uninstall. Individual keys are still
                # read leniently: the uninstaller deletes keys while we walk
                # them, and those transient misses resolve on the next pass.
                $stillInstalled = $true
                try {
                    $stillInstalled = [bool](Get-ChildItem `
                        -Path @($machineKey, $machineKey32on64) `
                        -ErrorAction Stop |
                            ForEach-Object {
                                Get-ItemProperty $_.PSPath `
                                    -ErrorAction SilentlyContinue
                            } |
                            Where-Object {
                                $_.DisplayName -like $softwareNameLike
                            })
                } catch {
                    Write-Host "Could not query the uninstall registry: $_"
                }
            } while ($stillInstalled -and (Get-Date) -lt $deadline)

            if ($stillInstalled) {
                Write-Host "Could not confirm '$softwareName' was removed."
                $exitCode = 1
            } else {
                Write-Host "'$softwareName' was removed."
            }
        }

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
