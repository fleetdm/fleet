# The registry DisplayName carries a version suffix ("Logitech Unifying Software
# 2.52"), so match on a prefix rather than an exact string.
$softwareName = "Logitech Unifying Software"
$softwareNameLike = "$softwareName*"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$exitCode = 0
$timeoutSeconds = 300

function Get-UnifyingUninstallKey {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like $softwareNameLike } |
        Select-Object -First 1
}

try {
    $key = Get-UnifyingUninstallKey
    if (-not $key) {
        Write-Host "Uninstaller for '$softwareName' not found."
        Exit 1
    }

    $uninstallString = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
    Write-Host "Uninstall string: $uninstallString"

    # Parse the executable path out of the uninstall string, handling quoted paths,
    # unquoted paths containing spaces, and bare tokens.
    $uninstallCommand = $uninstallString
    if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
        $uninstallCommand = $Matches[1]
    } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        $uninstallCommand = $Matches[1]
    } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
        $uninstallCommand = $Matches[1]
    }

    # This is an NSIS uninstaller. By default it copies itself to %TEMP% and
    # relaunches, so the process we start exits within a second or two while the
    # real uninstall is still running -- the app is still registered when software
    # inventory is re-queried. NSIS's undocumented "_?=<dir>" switch runs the
    # uninstaller in place instead, which makes it synchronous. It must be the last
    # argument and must not be quoted, so build one argument string rather than an
    # array (PowerShell would quote an element containing spaces).
    $installDir = Split-Path -Parent $uninstallCommand
    $uninstallArgs = "/S _?=$installDir"

    Write-Host "Uninstall command: $uninstallCommand"
    Write-Host "Uninstall args: $uninstallArgs"

    # No -NoNewWindow: that would make the uninstaller inherit this script's stdout
    # and stderr handles, and any process it leaves behind would hold those pipes
    # open after the script exits.
    $process = Start-Process -FilePath $uninstallCommand -ArgumentList $uninstallArgs -PassThru
    # Touch .Handle so the exit code is still readable after the process ends:
    # Start-Process -PassThru otherwise returns $null for .ExitCode.
    $null = $process.Handle

    if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Write-Host "Uninstall timed out after $timeoutSeconds seconds"
        Exit 1603
    }

    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"
} catch {
    Write-Host "Error: $_"
    Exit 1
}

# Belt and braces: if the uninstaller still detached despite "_?=", wait for the
# ARP entry to disappear before returning so inventory sees a clean state.
$elapsed = 0
while ((Get-UnifyingUninstallKey) -and ($elapsed -lt 120)) {
    Start-Sleep -Seconds 5
    $elapsed += 5
    Write-Host "Waiting for the uninstall to finish... ($elapsed seconds)"
}

if (Get-UnifyingUninstallKey) {
    Write-Host "'$softwareName' is still registered after the uninstall."
    Exit 1
}

Exit $exitCode
