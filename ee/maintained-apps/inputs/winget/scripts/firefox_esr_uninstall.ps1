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

        # TEMP DIAGNOSTICS (do not merge): find what hangs in CI. Dumps Mozilla
        # scheduled-task and process state around the uninstall, and bounds the
        # wait at 300s instead of relying on the validator's 10-minute kill.
        Write-Host "DIAG: user=$(whoami)"
        $sched = New-Object -ComObject Schedule.Service
        $sched.Connect()
        try {
            $mozFolder = $sched.GetFolder("\Mozilla")
            Write-Host "DIAG: \Mozilla task folder EXISTS; tasks:"
            @($mozFolder.GetTasks(1)) | ForEach-Object {
                Write-Host "DIAG:   task '$($_.Name)' enabled=$($_.Enabled)"
            }
        } catch {
            Write-Host "DIAG: no \Mozilla task folder"
        }

        $processOptions = @{
            FilePath = $uninstallCommand
            PassThru = $true
        }

        if ($uninstallArgs -ne '') {
            $processOptions.ArgumentList = $uninstallArgs
        }

        $process = Start-Process @processOptions
        $t0 = Get-Date
        $treeDone = $false
        while (((Get-Date) - $t0).TotalSeconds -lt 300) {
            Start-Sleep -Seconds 15
            $moz = Get-CimInstance Win32_Process -ErrorAction SilentlyContinue |
                Where-Object { $_.Name -match "^(helper|Un_A|firefox|default-browser-agent|private_browsing|pingsender)\.exe$" }
            if (-not $moz) {
                $treeDone = $true
                break
            }
            Write-Host ("DIAG: t+" + [int]((Get-Date) - $t0).TotalSeconds + "s still running:")
            $moz | ForEach-Object {
                Write-Host ("DIAG:   pid=" + $_.ProcessId + " ppid=" + $_.ParentProcessId + " " + $_.Name + " :: " + $_.CommandLine)
            }
        }

        if ($treeDone) {
            $exitCode = if ($process.HasExited) { $process.ExitCode } else { 0 }
            Write-Host "Uninstall exit code: $exitCode"
        } else {
            Write-Host "DIAG: uninstall still running after 300s, giving up"
            Get-WinEvent -FilterHashtable @{LogName="Application"; StartTime=$t0} -ErrorAction SilentlyContinue |
                Where-Object { $_.ProviderName -like "*Firefox*" -or $_.ProviderName -like "*Mozilla*" -or $_.ProviderName -like "*Default Browser Agent*" } |
                ForEach-Object { Write-Host ("DIAG: event [" + $_.ProviderName + "] id=" + $_.Id + " :: " + (($_.Message -split "\r?\n")[0])) }
            $exitCode = 99
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
