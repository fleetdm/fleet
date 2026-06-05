# Uninstalls the Microsoft .NET Runtime WiX "burn" bundle.
#
# The runtime installs as a burn bootstrapper that registers a *bundle* ARP entry
# (keyed by the bundle ProductCode) alongside several MSI component entries that
# share the same DisplayName. Only the bundle entry removes the whole runtime, and
# it uninstalls by running its cached bootstrapper .exe with /uninstall -- never via
# msiexec (see https://silentinstallhq.com/net-runtime-8-0-silent-uninstall-powershell/).
# We target the bundle by its ProductCode (injected by the ingester) and fall back
# to the cached bootstrapper in the Package Cache.

$productCode = ${PACKAGE_ID}

function Invoke-Uninstaller {
    param([string]$exe, [string]$exeArgs)
    if ($exeArgs -notmatch '/uninstall') { $exeArgs = "/uninstall $exeArgs" }
    if ($exeArgs -notmatch '/quiet')     { $exeArgs = "$exeArgs /quiet" }
    if ($exeArgs -notmatch '/norestart') { $exeArgs = "$exeArgs /norestart" }
    $exeArgs = $exeArgs.Trim()
    Write-Host "Uninstall command: $exe"
    Write-Host "Uninstall args: $exeArgs"
    $process = Start-Process -FilePath $exe -ArgumentList $exeArgs -NoNewWindow -PassThru -Wait
    return $process.ExitCode
}

$exitCode = $null

# 1) Preferred: the bundle ARP entry, looked up by the bundle ProductCode. Its
#    UninstallString/QuietUninstallString points to the cached bootstrapper .exe.
$keys = @(
  "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\$productCode",
  "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\$productCode"
)

foreach ($key in $keys) {
  if (-not (Test-Path $key)) { continue }
  $entry = Get-ItemProperty $key -ErrorAction SilentlyContinue
  if (-not $entry) { continue }

  $raw = $entry.QuietUninstallString
  if (-not $raw) { $raw = $entry.UninstallString }
  if (-not $raw) { continue }

  # Parse into executable + args, handling quoted/unquoted/bare shapes.
  if ($raw -match '^\s*"([^"]+)"\s*(.*)$') {
      $exe = $matches[1]; $exeArgs = $matches[2].Trim()
  } elseif ($raw -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
      $exe = $matches[1]; $exeArgs = $matches[2].Trim()
  } else {
      $exe = $raw; $exeArgs = ""
  }

  $exitCode = Invoke-Uninstaller -exe $exe -exeArgs $exeArgs
  break
}

# 2) Fallback: run the cached bootstrapper directly from the Package Cache, which
#    burn names after the bundle ProductCode.
if ($null -eq $exitCode) {
  $cached = Get-ChildItem -Path "C:\ProgramData\Package Cache\$productCode" -Filter *.exe -ErrorAction SilentlyContinue | Select-Object -First 1
  if ($cached) {
    $exitCode = Invoke-Uninstaller -exe $cached.FullName -exeArgs ""
  }
}

if ($null -eq $exitCode) {
  Write-Host "Uninstall entry not found for product code: $productCode"
  Exit 0
}

Write-Host "Uninstall exit code: $exitCode"
# 0 = success, 3010 = success but reboot required, 1641 = reboot initiated
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode
