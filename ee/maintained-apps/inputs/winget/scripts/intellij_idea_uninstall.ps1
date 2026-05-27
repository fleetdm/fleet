# Locates IntelliJ IDEA Ultimate's NSIS uninstaller from the registry and runs
# it silently. Ultimate's DisplayName is "IntelliJ IDEA <version>" with no
# edition marker, so exclude the Community and Educational editions (whose
# DisplayNames also start with "IntelliJ IDEA ") to avoid uninstalling the
# wrong product.

$publisherLike = "*JetBrains*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$selected = $null
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -and `
        $key.DisplayName -like "IntelliJ IDEA *" -and `
        $key.DisplayName -notlike "*Community*" -and `
        $key.DisplayName -notlike "*Educational*" -and `
        $key.Publisher -like $publisherLike) {
        $selected = $key
        break
    }
}

if (-not $selected -or -not $selected.UninstallString) {
    Write-Host "Uninstall entry not found for IntelliJ IDEA Ultimate"
    Exit 1
}

# Best-effort: stop the IDE so the uninstaller doesn't fail on locked files.
Stop-Process -Name "idea64" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "idea" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "fsnotifier" -Force -ErrorAction SilentlyContinue

$uninstallCommand = $selected.UninstallString

# Split the uninstall string into exe + args. Handle both quoted and unquoted
# exe paths.
$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

# NSIS uninstallers require /S for silent uninstall.
if ($existingArgs -notmatch '\b/S\b') {
    $existingArgs = ("$existingArgs /S").Trim()
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

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
