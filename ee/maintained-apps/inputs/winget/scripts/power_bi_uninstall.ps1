```powershell
# It is recommended to use the exact software name to avoid uninstalling
# unintended software. The Power BI Desktop EXE installer registers an
# MSI-backed uninstall entry named "Microsoft Power BI Desktop (x64)".
$softwareNameLike = "Microsoft Power BI Desktop*"

# Silent flag used only if the uninstaller turns out to be a plain EXE.
$uninstallArgs = "/S"

$machineKey = `
 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = `
 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

# 0 = success; 3010/1641 = success but reboot required (common for MSI).
$ExpectedExitCodes = @(0, 3010, 1641)

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

        # Best-effort: stop running Power BI processes so the uninstaller
        # doesn't fail on locked files.
        foreach ($proc in @("PBIDesktop", "msmdsrv", "Microsoft.Mashup.Container")) {
            Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
        }

        # Get the uninstall command. Some uninstallers do not include
        # 'QuietUninstallString' and require a flag to run silently.
        $uninstallCommand = if ($key.QuietUninstallString) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

        # Power BI's EXE installer is an MSI wrapper, so the UninstallString is
        # usually "MsiExec.exe /X{GUID}" with no quotes. Detect that case and
        # call msiexec correctly (/qn) instead of the EXE /S flag.
        if ($uninstallCommand -match "(?i)MsiExec\.exe\s+/[IX]\s*(\{[A-F0-9-]+\})") {
            $uninstallCommand = "MsiExec.exe"
            $uninstallArgs = "/X $($Matches[1]) /qn /norestart"
        } else {
            # Plain EXE uninstaller. Split the quoted command from its args.
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

        # msiexec can return before the uninstall fully completes; wait it out.
        $timeout = 120
        $elapsed = 0
        while ((Get-Process -Name "msiexec" -ErrorAction SilentlyContinue) -and ($elapsed -lt $timeout)) {
            Start-Sleep -Seconds 2
            $elapsed += 2
        }

        # Exit the loop once the software is found and uninstalled.
        break
    }
}

if (-not $foundUninstaller) {
    Write-Host "Uninstaller for '$softwareNameLike' not found."
    # Change exit code to 0 if you don't want to fail when the uninstaller is
    # not found (e.g. already uninstalled).
    $exitCode = 1
}

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

# Treat acceptable exit codes as success.
if ($ExpectedExitCodes -contains $exitCode) {
    Exit 0
} else {
    Exit $exitCode
}
```

This is the confirmed-working version for your registry entry (`MsiExec.exe /X{c7d2053f-a89b-41bf-9f74-d1c640ef1f33}`). Copy away.
