# The ARP DisplayName carries a version suffix ("Logitech Unifying Software 2.52"),
# so match on a prefix, plus the publisher to avoid other products sharing it.
$softwareName = "Logitech Unifying Software"
$softwareNameLike = "$softwareName*"
$softwarePublisher = "Logitech"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$exitCode = 0
$timeoutSeconds = 300

# Logs matching entries and their publishers to diagnose a name/publisher miss.
function Write-UnifyingCandidates {
    Write-Host "Registry entries matching '$softwareNameLike':"
    $found = Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like $softwareNameLike }
    if (-not $found) { Write-Host "  (none)" ; return }
    foreach ($f in $found) {
        Write-Host "  DisplayName='$($f.DisplayName)' Publisher='$($f.Publisher)' Version='$($f.DisplayVersion)'"
    }
}

function Get-UnifyingUninstallKey {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like $softwareNameLike -and $_.Publisher -eq $softwarePublisher } |
        Select-Object -First 1
}

# Stop leftovers: they hold file locks, and -Wait would block on them.
foreach ($name in @("LogiUnify", "Unifying", "UnifyingUnInstaller", "DJCUHost")) {
    Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
}

try {
    $key = Get-UnifyingUninstallKey
    if (-not $key) {
        Write-UnifyingCandidates
        Write-Host "Uninstall entry not found for '$softwareName' with publisher '$softwarePublisher'."
        Exit 0
    }

    $uninstallString = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
    Write-Host "Uninstall string: $uninstallString"

    # Handles quoted paths, unquoted paths with spaces, and bare tokens.
    $uninstallCommand = $uninstallString
    $existingArgs = ""
    if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
        $uninstallCommand = $Matches[1]; $existingArgs = $Matches[2]
    } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        $uninstallCommand = $Matches[1]; $existingArgs = $Matches[2]
    } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
        $uninstallCommand = $Matches[1]; $existingArgs = $Matches[2]
    }

    # This vendor NSIS uninstaller rejects the in-place "_?=<dir>" switch with
    # exit code 10, so use plain /S and poll for removal at the end instead.
    # Keep any registry arguments rather than dropping them.
    $uninstallArgs = ("$existingArgs /S").Trim()

    Write-Host "Uninstall command: $uninstallCommand"
    Write-Host "Uninstall args: $uninstallArgs"

    # No -NoNewWindow: a leftover child would hold this script's pipes open.
    $process = Start-Process -FilePath $uninstallCommand -ArgumentList $uninstallArgs -PassThru
    # Keeps .ExitCode readable after the process ends.
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

# The uninstaller hands off, so wait for the ARP entry to disappear.
$elapsed = 0
while ((Get-UnifyingUninstallKey) -and ($elapsed -lt 240)) {
    Start-Sleep -Seconds 5
    $elapsed += 5
    Write-Host "Waiting for the uninstall to finish... ($elapsed seconds)"
}

if (Get-UnifyingUninstallKey) {
    Write-UnifyingCandidates
    Write-Host "'$softwareName' is still registered after the uninstall."
    Exit 1
}

Exit $exitCode
