# The registry DisplayName carries a version suffix ("Logitech Unifying Software
# 2.52"), so match on a prefix rather than an exact string.
$softwareName = "Logitech Unifying Software"
$softwareNameLike = "$softwareName*"

# Require the publisher too, mirroring the manifest's exists query
# (name LIKE 'Logitech Unifying Software %' AND publisher = 'Logitech'). A prefix
# match alone could select a different product sharing the prefix. This also puts
# the publisher under test: the validator's appExists looks up by name only, so a
# wrong exists-query publisher would otherwise ship undetected (cf. the Spyder
# finding in #50016). That matters here in particular -- unlike the other apps in
# this batch, this value could not be verified statically, because the installer's
# PE version resource carries an unexpanded NSIS variable ("$Co_Name Inc.")
# instead of a literal company name. If 'Logitech' is wrong, this uninstall fails
# in CI rather than shipping a manifest that can never match an install.
$softwarePublisher = "Logitech"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$exitCode = 0
$timeoutSeconds = 300

# Print every matching entry with its publisher, so a publisher mismatch names the
# correct value instead of just failing.
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

# Stop anything Logitech left running: it holds file locks, and because
# "Start-Process -Wait" waits for descendants as well as the process itself, a
# resident helper would block this script.
foreach ($name in @("LogiUnify", "Unifying", "UnifyingUnInstaller", "DJCUHost")) {
    Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
}

try {
    $key = Get-UnifyingUninstallKey
    if (-not $key) {
        # Nothing to remove is not a failure: uninstall scripts are idempotent here,
        # as in nordpass_uninstall.ps1 and windsurf_uninstall.ps1.
        Write-UnifyingCandidates
        Write-Host "Uninstall entry not found for '$softwareName' with publisher '$softwarePublisher'."
        Exit 0
    }

    $uninstallString = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
    Write-Host "Uninstall string: $uninstallString"

    # Parse the executable path out of the uninstall string, handling quoted paths,
    # unquoted paths containing spaces, and bare tokens.
    $uninstallCommand = $uninstallString
    $existingArgs = ""
    if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
        $uninstallCommand = $Matches[1]; $existingArgs = $Matches[2]
    } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        $uninstallCommand = $Matches[1]; $existingArgs = $Matches[2]
    } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
        $uninstallCommand = $Matches[1]; $existingArgs = $Matches[2]
    }

    # UnifyingUnInstaller.exe is an NSIS uninstaller, but it is a vendor-built one
    # that rejects NSIS's in-place "_?=<dir>" switch with exit code 10, so pass the
    # plain silent switch. It returns within a couple of seconds while the real
    # removal continues in a detached child, which is why the app was still
    # registered when inventory was re-queried; the poll at the end of this script
    # is what waits for the removal to actually land.
    # Today the registry string is a bare path with no arguments, but keep any that
    # appear in future versions rather than dropping them.
    $uninstallArgs = ("$existingArgs /S").Trim()

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

# The uninstaller returns as soon as it has handed off, so wait for the ARP entry
# to disappear before returning and let inventory see a clean state.
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
