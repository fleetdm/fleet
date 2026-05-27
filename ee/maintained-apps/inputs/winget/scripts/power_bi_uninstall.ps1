# Power BI Desktop registers MORE THAN ONE uninstall entry, and the spacing of
# the DisplayName varies ("Microsoft Power BI Desktop (x64)" vs.
# "Microsoft PowerBI Desktop (x64)"). We normalize the DisplayName (strip
# spaces, lowercase) so every variant matches, and we uninstall EVERY matching
# entry instead of stopping at the first one.
$normalizedTarget = "microsoftpowerbidesktop"

# Silent flag used only if an uninstaller turns out to be a plain EXE.
$defaultExeArgs = "/S"

$machineKey = `
 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = `
 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

# 0 = success; 1605 = product not installed (already gone); 3010/1641 = success
# but reboot required.
$ExpectedExitCodes = @(0, 1605, 1641, 3010)

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

# Stop running Power BI processes up front so locked files don't block any of
# the uninstalls.
foreach ($proc in @("PBIDesktop", "msmdsrv", "Microsoft.Mashup.Container")) {
    Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
}

$foundUninstaller = $false
foreach ($key in $uninstallKeys) {
    if (-not $key.DisplayName) { continue }

    $normalized = ($key.DisplayName -replace '\s', '').ToLower()
    if (-not $normalized.StartsWith($normalizedTarget)) { continue }

    $foundUninstaller = $true
    Write-Host "Uninstalling entry: $($key.DisplayName)"

    $uninstallCommand = if ($key.QuietUninstallString) {
        $key.QuietUninstallString
    } else {
        $key.UninstallString
    }

    if (-not $uninstallCommand) {
        Write-Host "  No uninstall string for '$($key.DisplayName)', skipping."
        continue
    }

    $uninstallArgs = $defaultExeArgs

    if ($uninstallCommand -match "(?i)MsiExec\.exe\s+/[IX]\s*(\{[A-F0-9-]+\})") {
        # MSI-backed entry (the Power BI EXE installer registers this form).
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

    Write-Host "  Uninstall command: $uninstallCommand"
    Write-Host "  Uninstall args: $uninstallArgs"

    $processOptions = @{
        FilePath = $uninstallCommand
        PassThru = $true
        Wait     = $true
    }
    if ($uninstallArgs -ne '') {
        $processOptions.ArgumentList = "$uninstallArgs"
    }

    $process = Start-Process @processOptions
    $entryExit = $process.ExitCode
    Write-Host "  Uninstall exit code: $entryExit"

    # msiexec can return before the uninstall fully completes; wait it out.
    $timeout = 120
    $elapsed = 0
    while ((Get-Process -Name "msiexec" -ErrorAction SilentlyContinue) -and ($elapsed -lt $timeout)) {
        Start-Sleep -Seconds 2
        $elapsed += 2
    }

    # Record the first failing (non-expected) exit code; keep going so every
    # matching entry gets removed.
    if (($ExpectedExitCodes -notcontains $entryExit) -and ($exitCode -eq 0)) {
        $exitCode = $entryExit
    }
}

if (-not $foundUninstaller) {
    Write-Host "No Power BI Desktop uninstall entries found (already removed)."
    $exitCode = 0
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
