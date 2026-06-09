$softwareNameLike = "Antigravity*"
$publisherLike    = "*Google*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

$ExpectedExitCodes = @(0, 3010, 1641)
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

# Stop Antigravity and any helpers running out of the install dir so the
# uninstaller doesn't fail on locked files.
Stop-Process -Name "Antigravity" -Force -ErrorAction SilentlyContinue
if ($selected.InstallLocation -and (Test-Path -LiteralPath $selected.InstallLocation)) {
    $loc = $selected.InstallLocation.TrimEnd('\')
    Get-Process | Where-Object { $_.Path -and $_.Path -like "$loc\*" } |
        ForEach-Object { Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue }
}
Start-Sleep -Seconds 2

# Prefer QuietUninstallString -- Inno populates it with /VERYSILENT
# /SUPPRESSMSGBOXES already, so we just add /NORESTART if missing.
$uninstallCommand = if ($selected.QuietUninstallString) {
    $selected.QuietUninstallString
} else {
    $selected.UninstallString
}

# Parse the uninstaller exe path. Inno usually quotes it.
$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

$argumentList = @()
if ($existingArgs) { $argumentList += ($existingArgs -split '\s+') }
foreach ($s in @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART")) {
    if ($argumentList -notcontains $s) { $argumentList += $s }
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
