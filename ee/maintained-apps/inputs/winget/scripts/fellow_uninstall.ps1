# Attempts to locate Fellow's Squirrel.Windows (electron) uninstaller and run it silently.
# Fellow is user-scope and installs under %LocalAppData%\Fellow. Squirrel registers an
# UninstallString like "...\Update.exe --uninstall"; we run it verbatim with -s appended
# for a silent uninstall (NSIS-style /S _?= args do NOT apply to Squirrel).

$displayName = "Fellow"
$publisher   = "Fellow Insights"

$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName *") -and
    ($publisher -eq "" -or $_.Publisher -like "*$publisher*")
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

Stop-Process -Name "Fellow" -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

$uninstallString = $uninstall.UninstallString
$exePath = ""
$exeArgs = ""
if ($uninstallString -match '^"([^"]+)"(.*)') {
    $exePath = $matches[1]
    $exeArgs = $matches[2].Trim()
} elseif ($uninstallString -match '^(.+?\.exe)(.*)$') {
    $exePath = $matches[1]
    $exeArgs = $matches[2].Trim()
} else {
    Write-Host "Error: Could not parse uninstall string: $uninstallString"
    Exit 1
}

# Ensure the silent flag is present for Squirrel's Update.exe --uninstall.
if ($exeArgs -notmatch '(^|\s)-s(\s|$)') {
    $exeArgs = ("$exeArgs -s").Trim()
}

$installDir = if ($uninstall.InstallLocation -and (Test-Path -LiteralPath $uninstall.InstallLocation)) {
    $uninstall.InstallLocation.TrimEnd('\')
} else {
    (Split-Path -Parent $exePath).TrimEnd('\')
}

Write-Host "Uninstall executable: $exePath"
Write-Host "Uninstall arguments: $exeArgs"

try {
    $processOptions = @{
        FilePath     = $exePath
        ArgumentList = $exeArgs
        NoNewWindow  = $true
        PassThru     = $true
        Wait         = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    # Squirrel leaves the install dir in place after --uninstall; clean it up.
    Start-Sleep -Seconds 3
    # Only sweep leftovers on a successful uninstall, and never a root/short path
    if ($exitCode -eq 0 -and $installDir) {
        $resolvedDir = $null
        try { $resolvedDir = (Resolve-Path -LiteralPath $installDir -ErrorAction Stop).Path } catch { $resolvedDir = $null }
        if ($resolvedDir -and ($resolvedDir -match '^[A-Za-z]:\\') -and ((($resolvedDir.TrimEnd('\')) -split '\\').Count -ge 3) -and (Test-Path -LiteralPath $resolvedDir)) {
            Remove-Item -LiteralPath $resolvedDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
