# Locates Tableau Desktop's uninstaller from the registry and runs it silently.
# Tableau Desktop is a WiX Burn bundle; its registered UninstallString points
# to the bundle EXE in the Package Cache.

# Tableau Desktop registers in Add/Remove Programs as
# "Tableau YYYY.X (BUILD)" (e.g. "Tableau 2024.3 (20243.25.0208.0338)"),
# NOT "Tableau Desktop ...". The "Tableau 20*" pattern matches Desktop while
# excluding sibling products (Tableau Prep Builder, Tableau Reader, Tableau Public)
# whose DisplayNames continue with a letter, not the year.
$softwareNameLike = "Tableau 20*"
$publisherLike = "*Tableau*"

$exitCode = 0

try {

# Match osquery's `programs` table: HKLM (both views) plus every user hive
# under HKEY_USERS. Per-user installs land in HKCU/HKU and would otherwise be
# invisible to a HKLM-only search even though detection still flags them.
$paths = [System.Collections.Generic.List[string]]::new()
$paths.Add('HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall')
$paths.Add('HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall')
foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
    if ($hive.Name -match '_Classes$') { continue }
    $paths.Add("Registry::$($hive.Name)\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall")
    $paths.Add("Registry::$($hive.Name)\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall")
}

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

if (-not $selected -or -not $selected.UninstallString) {
    Write-Host "Uninstall entry not found for $softwareNameLike"
    Exit 1
}

$uninstallCommand = if ($selected.QuietUninstallString) {
    $selected.QuietUninstallString
} else {
    $selected.UninstallString
}

# Defensive UninstallString parser: handle quoted, unquoted-with-spaces (Burn
# bundles often live under "C:\ProgramData\Package Cache\..."), or bare token.
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

# Burn bundles: append /uninstall if missing (UninstallString doesn't always
# include the verb), then /quiet /norestart.
if ($exePath -notmatch '(?i)msiexec' -and $existingArgs -notmatch '/uninstall') {
    $existingArgs = ("/uninstall $existingArgs").Trim()
}
if ($existingArgs -notmatch '/quiet' -and $existingArgs -notmatch '/qn') {
    $existingArgs = ("$existingArgs /quiet /norestart").Trim()
}

Write-Host "Selected entry DisplayName: $($selected.DisplayName)"
Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $existingArgs"

$processOptions = @{
    FilePath = $exePath
    PassThru = $true
    Wait = $true
}

if ($existingArgs -ne '') {
    $processOptions.ArgumentList = $existingArgs
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
    Exit 0
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
