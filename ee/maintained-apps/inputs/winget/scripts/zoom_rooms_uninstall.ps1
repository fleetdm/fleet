# Zoom Rooms registers TWO ARP entries:
#   1. "Zoom Rooms Installer" — the MSI bootstrapper itself (ProductCode
#      {C54339B7-44E0-45D6-8BD4-DCDAC57A7267}).
#   2. "Zoom Rooms" (with or without a trailing version) — the real app the
#      bootstrapper installs.
#
# Uninstalling the bootstrapper MSI via its UpgradeCode chains into the real
# app's uninstaller (the MSI's CustomAction.Uninstall execs the embedded EXE
# with `--uninstall_path=[ProgramFiles64Folder]ZoomRooms\uninstall`). We
# therefore prefer to uninstall the bootstrapper first; failing that we fall
# back to running the real app's registry UninstallString directly.

$bootstrapNameLike = "Zoom Rooms Installer*"
$appNameLike       = "Zoom Rooms*"
$publisherLike     = "Zoom*Communications*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

# Acceptable exit codes: 0 = success, 1605 = product not installed (already
# gone), 1641 = reboot initiated, 3010 = reboot required.
$ExpectedExitCodes = @(0, 1605, 1641, 3010)
$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

# Prefer the bootstrapper (it cascades into the real app's uninstall).
$selected = $null
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -and $key.DisplayName -like $bootstrapNameLike -and $key.Publisher -like $publisherLike) {
        $selected = $key
        break
    }
}

# Fall back to the real app if the bootstrapper entry is gone.
if (-not $selected) {
    foreach ($key in $uninstallKeys) {
        if ($key.DisplayName -and $key.DisplayName -like $appNameLike `
                              -and $key.DisplayName -notlike "*Installer*" `
                              -and $key.Publisher -like $publisherLike) {
            $selected = $key
            break
        }
    }
}

if (-not $selected -or (-not $selected.UninstallString -and -not $selected.QuietUninstallString)) {
    Write-Host "Uninstall entry not found for Zoom Rooms"
    Exit 0
}

# Stop running Zoom Rooms processes so the uninstaller doesn't fail on locked
# files.
foreach ($proc in @("ZoomRooms", "ZRClient", "ZRController", "Zoom")) {
    Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
}
Start-Sleep -Seconds 2

# Prefer QuietUninstallString when present; otherwise rebuild a quiet msiexec
# invocation from the registry's MSI product code.
$uninstallCommand = if ($selected.QuietUninstallString) {
    $selected.QuietUninstallString
} else {
    $selected.UninstallString
}

# If we got an MsiExec.exe-style command, force quiet + norestart and (when
# not already present) treat as /X.
if ($uninstallCommand -match '(?i)MsiExec\.exe\s+/[IX]\s*(\{[A-F0-9-]+\})') {
    $msiCode = $Matches[1]
    $argumentList = @("/X", $msiCode, "/qn", "/norestart")
    $exePath = "MsiExec.exe"
} elseif ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $extraArgs = $matches[2].Trim()
    $argumentList = @()
    if ($extraArgs) { $argumentList += ($extraArgs -split '\s+') }
    foreach ($s in @("/qn", "/norestart")) {
        if ($argumentList -notcontains $s) { $argumentList += $s }
    }
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]
    $extraArgs = $matches[2].Trim()
    $argumentList = @()
    if ($extraArgs) { $argumentList += ($extraArgs -split '\s+') }
} else {
    Throw "Could not parse uninstall command: $uninstallCommand"
}

Write-Host "Selected entry DisplayName: $($selected.DisplayName)"
Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $($argumentList -join ' ')"

$processOptions = @{
    FilePath     = $exePath
    ArgumentList = $argumentList
    PassThru     = $true
    Wait         = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
