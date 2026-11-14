# Fleet extracts name from installer (EXE) and saves it to PACKAGE_ID
# variable
# Match Firefox ESR only (e.g. "Mozilla Firefox 140.7.1 ESR (x64 en-US)"), not regular Firefox
$softwareNameLike = "*Firefox*ESR*"

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

# Try x64 entries first, so an ARM64 or 32-bit Firefox ESR on the same host is
# only removed when no x64 Firefox ESR is installed.
$uninstallKeys = @($uninstallKeys | Sort-Object { $_.DisplayName -notlike "*(x64 *" })

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
        $process.WaitForExit()
        $exitCode = $process.ExitCode
        Write-Host "Uninstall exit code: $exitCode"

        # helper.exe hands off to Un_A.exe and exits. Un_A's Default Browser
        # Agent cleanup (firefox.exe --backgroundtask defaultagent) can hang
        # when run as SYSTEM, so stop it after a grace period to let Un_A finish.
        $uninstallDir = Split-Path $uninstallCommand
        $graceEnd = (Get-Date).AddSeconds(30)
        $deadline = (Get-Date).AddSeconds(300)
        while ($true) {
            $unA = Get-CimInstance Win32_Process -Filter "Name='Un_A.exe'" -ErrorAction SilentlyContinue |
                Where-Object { $_.CommandLine -and $_.CommandLine.Contains("_?=$uninstallDir") }
            if (-not $unA) { break }
            if ((Get-Date) -gt $deadline) {
                Write-Host "Timed out waiting for the Firefox ESR uninstaller to finish."
                $exitCode = 1
                break
            }
            if ((Get-Date) -gt $graceEnd) {
                Get-CimInstance Win32_Process -Filter "Name='default-browser-agent.exe' OR Name='firefox.exe'" -ErrorAction SilentlyContinue |
                    Where-Object { $_.CommandLine -match 'defaultagent|default-browser-agent\.exe"?\s+uninstall' } |
                    ForEach-Object {
                        Write-Host "Stopping hung Default Browser Agent cleanup (PID $($_.ProcessId))"
                        Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue
                    }
            }
            Start-Sleep -Seconds 2
        }
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
