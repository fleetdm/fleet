# Uninstalls every SuperCollider version on the host, since each version
# installs side by side in its own folder. Running copies are closed first.

$arpPaths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

function Get-SuperColliderEntries {
  @(Get-ItemProperty -Path $arpPaths -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -like 'SuperCollider Version *' -and $_.Publisher -eq 'SuperCollider Community'
  })
}

function Uninstall-SuperCollider {
  $exitCode = 0
  foreach ($entry in Get-SuperColliderEntries) {
    Write-Host "Uninstalling $($entry.DisplayName)"
    $uninstallExe = ([string]$entry.UninstallString).Trim().Trim('"')
    if ($uninstallExe -notlike '*.exe' -or -not (Test-Path -LiteralPath $uninstallExe)) {
      Write-Host "Uninstaller not found: $($entry.UninstallString)"
      $exitCode = 1
      continue
    }
    $installDir = Split-Path $uninstallExe -Parent

    $running = @(Get-Process -ErrorAction SilentlyContinue |
      Where-Object { $_.Path -and $_.Path.StartsWith("$installDir\", [System.StringComparison]::OrdinalIgnoreCase) })
    if ($running) {
      $running | Stop-Process -Force -ErrorAction SilentlyContinue
      $running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue
    }

    try {
      $process = Start-Process -FilePath $uninstallExe -ArgumentList @('/S', "_?=$installDir") `
        -PassThru -Wait -NoNewWindow -ErrorAction Stop
    } catch {
      Write-Host "Error running uninstaller: $_"
      $exitCode = 1
      continue
    }
    Write-Host "Uninstall exit code: $($process.ExitCode)"
    if ($process.ExitCode -ne 0) {
      $exitCode = $process.ExitCode
      continue
    }

    if ((Split-Path $installDir -Leaf) -like 'SuperCollider*') {
      for ($i = 0; $i -lt 10 -and (Test-Path -LiteralPath $installDir); $i++) {
        Remove-Item -LiteralPath $installDir -Recurse -Force -ErrorAction SilentlyContinue
        if (Test-Path -LiteralPath $installDir) { Start-Sleep -Seconds 2 }
      }
      if (Test-Path -LiteralPath $installDir) { Write-Host "Could not remove $installDir" }
    }
  }

  $remaining = @(Get-SuperColliderEntries)
  if ($exitCode -eq 0 -and $remaining.Count -gt 0) {
    Write-Host "Still installed: $(($remaining | ForEach-Object { $_.DisplayName }) -join ', ')"
    $exitCode = 1
  }
  return $exitCode
}

try {
  Exit (Uninstall-SuperCollider)
} catch {
  Write-Host "Error: $_"
  Exit 1
}
