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

# Stop anything Logitech left running: it holds file locks, and because
# "Start-Process -Wait" waits for descendants as well as the process itself, a
# resident helper would block this script.
foreach ($name in @("LogiUnify", "Unifying", "UnifyingUnInstaller", "DJCUHost")) {
    Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
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

    # UnifyingUnInstaller.exe is an NSIS uninstaller, but it is a vendor-built one
    # that rejects NSIS's in-place "_?=<dir>" switch with exit code 10, so pass the
    # plain silent switch. It returns within a couple of seconds while the real
    # removal continues in a detached child, which is why the app was still
    # registered when inventory was re-queried; the poll at the end of this script
    # is what waits for the removal to actually land.
    $uninstallArgs = "/S"

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
while ((Get-UnifyingUninstallKey) -and ($elapsed -lt 240)) {
    Start-Sleep -Seconds 5
    $elapsed += 5
    Write-Host "Waiting for the uninstall to finish... ($elapsed seconds)"
}

if (Get-UnifyingUninstallKey) {
    Write-Host "'$softwareName' is still registered after the uninstall."
    Exit 1
}

Exit $exitCode
