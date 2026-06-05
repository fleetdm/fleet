# Locates Python 3.14's WiX "Burn" bundle registration from the registry and
# runs its uninstaller silently. python.org's per-machine x64 installer registers
# a DisplayName of "Python 3.14.<patch> (64-bit)" with Publisher
# "Python Software Foundation". The bundle's uninstaller removes the bundle's own
# registration along with the component MSIs it installed.

$publisher = "Python Software Foundation"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

# 0 = success, 3010 = success (reboot required), 1641 = success (reboot
# initiated), 1605 = product already uninstalled.
$ExpectedExitCodes = @(0, 1605, 1641, 3010)
$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$selected = $null
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -and `
        $key.DisplayName -like "Python 3.14.* (64-bit)" -and `
        $key.Publisher -eq $publisher) {
        $selected = $key
        break
    }
}

if (-not $selected) {
    Write-Host "No Python 3.14 (64-bit) entry found (already removed)."
    Exit 0
}

# Burn bundles expose a QuietUninstallString that already includes the silent
# flags; fall back to UninstallString otherwise.
$uninstallCommand = if ($selected.QuietUninstallString) {
    $selected.QuietUninstallString
} else {
    $selected.UninstallString
}

if (-not $uninstallCommand) {
    Write-Host "Uninstall string not found for '$($selected.DisplayName)'"
    Exit 1
}

# Split the uninstall string into exe + args. Handle quoted paths, unquoted paths
# that may contain spaces, and bare tokens.
$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

# Ensure the bundle runs in silent uninstall mode without prompting for a reboot.
if ($existingArgs -notmatch '(?i)/uninstall') { $existingArgs = ("/uninstall $existingArgs").Trim() }
if ($existingArgs -notmatch '(?i)/quiet')     { $existingArgs = ("$existingArgs /quiet").Trim() }
if ($existingArgs -notmatch '(?i)/norestart') { $existingArgs = ("$existingArgs /norestart").Trim() }

Write-Host "Selected entry DisplayName: $($selected.DisplayName)"
Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $existingArgs"

$processOptions = @{
    FilePath = $exePath
    ArgumentList = $existingArgs
    PassThru = $true
    Wait = $true
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
