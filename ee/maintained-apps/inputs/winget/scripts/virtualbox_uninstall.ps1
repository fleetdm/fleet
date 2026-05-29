# Locates Oracle VirtualBox's MSI uninstaller from the registry and runs it
# silently. VirtualBox installs as a single MSI under HKLM (machine scope) --
# no Burn bundle, no per-user entries -- with DisplayName "Oracle VirtualBox
# <version>" and Publisher "Oracle Corporation". The MSI product code GUID
# changes per version, so parse it out of the UninstallString.

$softwareNameLike = "Oracle VirtualBox*"
$publisherLike    = "*Oracle*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

# 0 = success; 1605 = product not installed (already gone); 3010/1641 = reboot
# requested but uninstall succeeded (VBox kernel drivers often need a reboot
# to fully unload).
$ExpectedExitCodes = @(0, 1605, 1641, 3010)
$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$selected = $null
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -and $key.DisplayName -like $softwareNameLike -and $key.Publisher -like $publisherLike) {
        $selected = $key
        break
    }
}

if (-not $selected -or (-not $selected.UninstallString -and -not $selected.QuietUninstallString)) {
    Write-Host "Uninstall entry not found for $softwareNameLike"
    Exit 0
}

# Stop running VirtualBox processes so the uninstaller doesn't fail on locked
# files. NOTE: this force-kills any running VMs (VBoxHeadless / VirtualBoxVM)
# -- expected for an uninstall.
foreach ($proc in @("VirtualBox", "VirtualBoxVM", "VBoxSVC", "VBoxSDS",
                    "VBoxHeadless", "VBoxManage", "VBoxNetDHCP", "VBoxNetNAT")) {
    Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
}
# Stop the VBoxSDS Windows service if present (uninstaller can stall otherwise).
Stop-Service -Name "VBoxSDS" -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

# Parse the MSI product code from the UninstallString
# ("MsiExec.exe /I{GUID}" or "MsiExec.exe /X{GUID}"). Fall back to the
# registry key name if it's itself a GUID, which is how MSI products usually
# register.
$uninstallCommand = if ($selected.QuietUninstallString) {
    $selected.QuietUninstallString
} else {
    $selected.UninstallString
}

$msiCode = $null
if ($uninstallCommand -match "(?i)MsiExec\.exe\s+/[IX]\s*(\{[A-F0-9-]+\})") {
    $msiCode = $Matches[1]
} elseif ($selected.PSChildName -match "(?i)^\{[A-F0-9-]+\}$") {
    $msiCode = $selected.PSChildName
} else {
    Throw "Could not parse MSI product code from uninstall command: $uninstallCommand"
}

Write-Host "Selected entry DisplayName: $($selected.DisplayName)"
Write-Host "Uninstall command: MsiExec.exe"
Write-Host "Uninstall args: /X $msiCode /qn /norestart"

$processOptions = @{
    FilePath     = "MsiExec.exe"
    ArgumentList = "/X", $msiCode, "/qn", "/norestart"
    PassThru     = $true
    Wait         = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

# msiexec can return before the uninstall fully finishes -- especially when
# the kernel-driver cleanup helpers (drvinst.exe) are still running. Wait
# for them to drain so detection sees a settled state.
$elapsed = 0
while ((Get-Process -Name "msiexec","drvinst" -ErrorAction SilentlyContinue) -and ($elapsed -lt 120)) {
    Start-Sleep -Seconds 3
    $elapsed += 3
}

# Treat acceptable exit codes (including reboot-required) as success.
if ($ExpectedExitCodes -contains $exitCode) {
    Exit 0
} else {
    Exit $exitCode
}

} catch {
    Write-Host "Error: $_"
    Exit 1
}
