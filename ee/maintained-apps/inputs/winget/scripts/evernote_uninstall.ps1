# The "(All Users)" suffix in Evernote's DisplayName is locale-dependent, so
# match on the "Evernote" prefix only.
$softwareName = "Evernote"

$softwareNameLike = "Evernote*"

# /allusers matches the machine-wide install
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
        $uninstallCommand = if ($key.QuietUninstallString) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

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

        $process = Start-Process @processOptions
        $exitCode = $process.ExitCode

        Write-Host "Uninstall exit code: $exitCode"

        # Without an "_?=<installdir>" argument an NSIS uninstaller relaunches
        # itself from %TEMP% and exits 0 before the removal finishes, so wait
        # for the registry entry to clear.
        if ($exitCode -eq 0) {
            $deadline = (Get-Date).AddSeconds(300)
            do {
                Start-Sleep -Seconds 5
                # Fail closed: a failed query is not proof of removal. Keys are
                # read leniently since the uninstaller deletes them as we walk.
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
