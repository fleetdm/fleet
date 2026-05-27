# Define acceptable/expected exit codes (0 = success, 3010/1641 = success, reboot required)
$ExpectedExitCodes = @(0, 3010, 1641)

# Power BI Desktop registers in Programs and Features. Match on DisplayName since the
# MSI product code (GUID) changes between versions.
$softwareNameLike = "Microsoft Power BI Desktop*"
$publisher        = "Microsoft Corporation"

# Silent flag used only if the uninstaller turns out to be a plain EXE (not MsiExec)
# and does not provide its own QuietUninstallString.
$exeSilentArgs = "-q -norestart"

$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

# Initialize exit code
$exitCode = 0

try {
    # Locate the uninstall entry
    $selected = $null
    foreach ($p in $paths) {
        $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
            $_.DisplayName -and ($_.DisplayName -like $softwareNameLike) -and
            ($publisher -eq "" -or $_.Publisher -eq $publisher)
        }
        if ($items) { $selected = $items | Select-Object -First 1; break }
    }

    if (-not $selected -or -not $selected.UninstallString) {
        Write-Host "Uninstall entry not found for $softwareNameLike"
        Exit 0
    }

    # Best-effort: stop running Power BI processes so the uninstaller doesn't fail on locked files
    foreach ($proc in @("PBIDesktop", "msmdsrv", "Microsoft.Mashup.Container")) {
        Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
    }

    # Prefer QuietUninstallString (already includes silent switches) when present.
    $uninstallCommand = if ($selected.QuietUninstallString) {
        $selected.QuietUninstallString
    } else {
        $selected.UninstallString
    }

    $uninstallArgs = ""

    if ($uninstallCommand -match "MsiExec\.exe\s+/[IX]\s*(\{[A-F0-9-]+\})") {
        # MSI-backed uninstall (the common case for the Power BI EXE installer)
        $productCode = $Matches[1]
        $uninstallArgs = "/X $productCode /qn /norestart"
        $uninstallCommand = "MsiExec.exe"
    } else {
        # Plain EXE uninstaller. Split the quoted command from its args.
        $splitArgs = $uninstallCommand.Split('"')
        if ($splitArgs.Length -gt 1) {
            if ($splitArgs.Length -eq 3) {
                $uninstallArgs = $splitArgs[2].Trim()
            } elseif ($splitArgs.Length -gt 3) {
                Throw "Uninstall command contains multiple quoted strings. Please update the uninstall script.`nUninstall command: $uninstallCommand"
            }
            $uninstallCommand = $splitArgs[1]
        }
        # If the registry didn't supply silent switches, add ours.
        if (-not $selected.QuietUninstallString) {
            $uninstallArgs = "$uninstallArgs $exeSilentArgs".Trim()
        }
    }

    Write-Host "Uninstall command: $uninstallCommand"
    Write-Host "Uninstall args: $uninstallArgs"

    $processOptions = @{
        FilePath    = $uninstallCommand
        NoNewWindow = $true
        PassThru    = $true
        Wait        = $true
    }
    if ($uninstallArgs -ne '') {
        $processOptions.ArgumentList = "$uninstallArgs"
    }

    # Start uninstall process
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    # msiexec can return before the uninstall is fully complete; wait it out
    $timeout = 120
    $elapsed = 0
    while ((Get-Process -Name "msiexec" -ErrorAction SilentlyContinue) -and ($elapsed -lt $timeout)) {
        Start-Sleep -Seconds 2
        $elapsed += 2
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
