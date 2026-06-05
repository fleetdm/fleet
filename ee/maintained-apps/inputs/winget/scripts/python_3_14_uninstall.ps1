# Locates Python 3.14's WiX "Burn" bundle registration from the registry and
# runs its uninstaller silently. python.org's per-machine x64 installer registers
# a bundle DisplayName of exactly "Python 3.14.<patch> (64-bit)" with Publisher
# "Python Software Foundation", PLUS a set of component MSIs with names like
# "Python 3.14.<patch> Executables (64-bit)" / "...Core Interpreter (64-bit)".
# We must uninstall the BUNDLE (an exe with "/uninstall"), which removes all of
# its components; uninstalling a component MSI directly leaves the rest behind.
# Hence the strict anchored match below that excludes the component entries.

$publisher = "Python Software Foundation"
# Matches only the bundle, e.g. "Python 3.14.5 (64-bit)" — not components such as
# "Python 3.14.5 Executables (64-bit)".
$bundleNamePattern = '^Python 3\.14\.\d+ \(64-bit\)$'

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

# 0 = success, 3010 = success (reboot required), 1641 = success (reboot
# initiated), 1605 = product already uninstalled.
$ExpectedExitCodes = @(0, 1605, 1641, 3010)
$exitCode = 0

function Wait-ForProcessExit {
    param([string[]]$Names, [int]$TimeoutSeconds = 240)
    $elapsed = 0
    while ($elapsed -lt $TimeoutSeconds) {
        $running = $Names | Where-Object { Get-Process -Name $_ -ErrorAction SilentlyContinue }
        if (-not $running) { break }
        Start-Sleep -Seconds 3
        $elapsed += 3
    }
}

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$selected = $null
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -and `
        $key.DisplayName -match $bundleNamePattern -and `
        $key.Publisher -eq $publisher) {
        $selected = $key
        break
    }
}

if (-not $selected) {
    Write-Host "No Python 3.14 (64-bit) bundle entry found (already removed)."
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

# Burn relaunches a cached copy of itself and drives msiexec to remove the
# component MSIs asynchronously, so the launched process can return before the
# work is done. Wait for those to finish before we report success, otherwise a
# follow-up inventory check may still see lingering component entries.
$exeName = [System.IO.Path]::GetFileNameWithoutExtension($exePath)
Wait-ForProcessExit -Names @($exeName, "msiexec") -TimeoutSeconds 240

if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
