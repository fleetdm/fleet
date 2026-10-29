# Fleet extracts name from installer (EXE) and saves it to PACKAGE_ID
# variable
# Match the ARM64 ESR build only (e.g. "Mozilla Firefox ESR (AArch64 en-US)"),
# leaving an x64 ESR or regular Firefox on the same host alone
$softwareNameLike = "Mozilla Firefox*ESR (AArch64 *"

# NSIS installers require /S flag for silent uninstall
$uninstallArgs = "/S"

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
    if ($key.DisplayName -like $softwareNameLike) {
        $foundUninstaller = $true
        $uninstallCommand = if ($key.QuietUninstallString) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

        # The uninstall command may contain command and args, like:
        # "C:\Program Files\Mozilla Firefox ESR\uninstall\helper.exe" /S
        $splitArgs = $uninstallCommand.Split('"')
        if ($splitArgs.Length -gt 1) {
            if ($splitArgs.Length -eq 3) {
                $existingArgs = $splitArgs[2].Trim()
                if ($existingArgs -notmatch '\b/S\b') {
                    $uninstallArgs = "$existingArgs /S".Trim()
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
            if ($uninstallCommand -notmatch '\b/S\b') {
                $uninstallArgs = "/S"
            } else {
                $uninstallArgs = ""
            }
        }
        Write-Host "Uninstall command: $uninstallCommand"
        Write-Host "Uninstall args: $uninstallArgs"

        $processOptions = @{
            FilePath = $uninstallCommand
            PassThru = $true
        }

        if ($uninstallArgs -ne '') {
            $processOptions.ArgumentList = $uninstallArgs
        }

        $process = Start-Process @processOptions
        $null = $process.Handle
        # The uninstaller hands off to a copy of itself, which waits on the Default Browser
        # Agent's cleanup; as SYSTEM that cleanup can hang. Wait for the uninstall entry to go
        # away, ending the agent's cleanup if it's still running after a minute.
        $installDir = Split-Path (Split-Path $uninstallCommand)
        $agentDeadline = (Get-Date).AddSeconds(60)
        $deadline = (Get-Date).AddSeconds(300)
        while ($true) {
            $agent = @(Get-CimInstance Win32_Process | Where-Object {
                $_.ExecutablePath -like "$installDir\*" -and
                ($_.Name -eq 'default-browser-agent.exe' -or $_.CommandLine -like '*--backgroundtask*')
            })
            $registered = (Get-ItemProperty -Path $key.PSPath -ErrorAction SilentlyContinue).DisplayName -like $softwareNameLike
            if (-not $registered -and $agent.Count -eq 0) {
                break
            }
            if ((Get-Date) -gt $deadline) {
                Throw "Firefox is still installed after 300 seconds."
            }
            if ($agent.Count -gt 0 -and (Get-Date) -gt $agentDeadline) {
                $agent | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
            }
            Start-Sleep -Seconds 2
        }
        $exitCode = $process.ExitCode
        Write-Host "Uninstall exit code: $exitCode"
        break
    }
}

if (-not $foundUninstaller) {
    Write-Host "Uninstaller for Firefox ESR not found."
    $exitCode = 1
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
