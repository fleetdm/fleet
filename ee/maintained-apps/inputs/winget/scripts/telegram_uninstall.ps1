# Attempts to locate Telegram Desktop's uninstaller from registry and execute it silently

$displayName = "Telegram Desktop"
$publisher = "Telegram FZ-LLC"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $_.Publisher -eq $publisher)
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

# Kill any running Telegram processes before uninstalling
Stop-Process -Name "Telegram" -Force -ErrorAction SilentlyContinue

$uninstallString = $uninstall.UninstallString
$exePath = ""
$arguments = ""

# Parse the uninstall string to extract executable path and existing arguments
# Handles both quoted and unquoted paths
if ($uninstallString -match '^"([^"]+)"(.*)') {
    $exePath = $matches[1]
    $arguments = $matches[2].Trim()
} elseif ($uninstallString -match '^([^\s]+)(.*)') {
    $exePath = $matches[1]
    $arguments = $matches[2].Trim()
} else {
    Write-Host "Error: Could not parse uninstall string: $uninstallString"
    Exit 1
}

# Build argument list array, preserving existing arguments and adding /S for silent (Inno Setup)
$baseArgumentList = @()
if ($arguments -ne '') {
    # Split existing arguments and add them
    $baseArgumentList += $arguments -split '\s+'
}

function Invoke-Uninstall {
    param(
        [string]$Executable,
        [array]$BaseArgs,
        [array]$ExtraArgs
    )

    $finalArgs = @()
    if ($BaseArgs) {
        $finalArgs += $BaseArgs
    }
    if ($ExtraArgs) {
        $finalArgs += $ExtraArgs
    }

    Write-Host "Uninstall executable: $Executable"
    Write-Host "Uninstall arguments: $($finalArgs -join ' ')"

    try {
        $processOptions = @{
            FilePath = $Executable
            ArgumentList = $finalArgs
            NoNewWindow = $true
            PassThru = $true
            Wait = $true
            WorkingDirectory = (Split-Path -Path $Executable -Parent)
        }

        $process = Start-Process @processOptions
        return $process.ExitCode
    } catch {
        Write-Host "Error running uninstaller: $_"
        return 1
    }
}

$preferredSilentArgs = @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART")
$exitCode = Invoke-Uninstall -Executable $exePath -BaseArgs $baseArgumentList -ExtraArgs $preferredSilentArgs

if ($exitCode -ne 0) {
    Write-Host "Preferred silent uninstall failed with exit code $exitCode. Retrying with /S."
    $exitCode = Invoke-Uninstall -Executable $exePath -BaseArgs $baseArgumentList -ExtraArgs @("/S")
}

Exit $exitCode
