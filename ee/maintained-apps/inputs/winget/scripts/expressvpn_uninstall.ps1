$bundleProductCode = '{86881f66-59ab-4941-94e4-a647e519e053}'
$displayNameLike   = "ExpressVPN*"
$publisherLike     = "*ExpressVPN*"

$installDir = (Join-Path ${env:ProgramFiles(x86)} 'ExpressVPN')

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

function Get-ExePath {
    param([string]$Command)
    if ($Command -match '"([^"]+\.exe)"') { return $Matches[1] }
    if ($Command -match '(?i)([A-Z]:\\[^"]+?\.exe)') { return $Matches[1] }
    return $null
}

try {

$paths = @(
  "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\$bundleProductCode",
  "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\$bundleProductCode",
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

$entry = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty $p -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -like $displayNameLike -and ($_.Publisher -like $publisherLike -or $p -like "*$bundleProductCode")
  }
  if ($items) { $entry = $items | Select-Object -First 1; break }
}

if (-not $entry -or (-not $entry.UninstallString -and -not $entry.QuietUninstallString)) {
  Write-Host "Uninstall entry not found (already removed)."
  Exit 0
}

# Stop the client/services so the uninstaller doesn't fail on locked files.
foreach ($proc in @("ExpressVPN", "expressvpnd", "ExpressVPN.UI", "ExpressVPN.BrowserHelper")) {
  Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
}

$uninstallCommand = if ($entry.QuietUninstallString) { $entry.QuietUninstallString } else { $entry.UninstallString }

$exePath = Get-ExePath $uninstallCommand
if (-not $exePath) { Throw "Could not parse uninstaller exe from: $uninstallCommand" }
if (-not (Test-Path -LiteralPath $exePath)) { Throw "Bundle exe missing: $exePath" }

# Ensure silent uninstall switches are present.
$existingArgs = ($uninstallCommand -replace [regex]::Escape("`"$exePath`""), "").Trim()
$existingArgs = ($existingArgs -replace [regex]::Escape($exePath), "").Trim()
foreach ($s in @("/uninstall", "/quiet", "/norestart")) {
    if ($existingArgs -notmatch [regex]::Escape($s)) { $existingArgs = ("$existingArgs $s").Trim() }
}

Write-Host "Selected entry DisplayName: $($entry.DisplayName)"
Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $existingArgs"

$process = Start-Process -FilePath $exePath -ArgumentList $existingArgs -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

Wait-ForProcessExit -Names @("ExpressVPN", "msiexec") -TimeoutSeconds 240

} catch {
  Write-Host "Error: $_"
  $exitCode = 1
}

if ($ExpectedExitCodes -contains $exitCode) { Exit 0 } else { Exit $exitCode }
