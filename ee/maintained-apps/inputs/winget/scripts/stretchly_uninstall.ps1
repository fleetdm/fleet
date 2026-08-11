# Locate the uninstall entry in the registry and run it silently.
# Stretchly 1.22.0's NSIS uninstaller runs a custom PATH-cleanup step (EnVar
# plugin) before any removal work and can crash with 0xc0000005, leaving the
# app fully installed. Don't trust its exit code: verify the registry entry is
# gone and fall back to removing the app manually.

$displayNameLike = "Stretchly*"
$publisher = "Jan Hovancik"

$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

function Find-UninstallEntry {
  foreach ($p in $paths) {
    $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
      $_.DisplayName -like $displayNameLike -and ($publisher -eq "" -or $_.Publisher -like "*$publisher*")
    }
    if ($items) { return $items | Select-Object -First 1 }
  }
  return $null
}

function Remove-StretchlyFromPath {
  # The vendor uninstaller's EnVar step removes "<install dir>\bin" from PATH;
  # do it ourselves since that step is what crashes. Use the raw registry API
  # to preserve unexpanded REG_EXPAND_SZ entries like %SystemRoot%.
  $envKeys = @(
    @{ Hive = [Microsoft.Win32.Registry]::CurrentUser; SubKey = 'Environment' },
    @{ Hive = [Microsoft.Win32.Registry]::LocalMachine; SubKey = 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment' }
  )
  foreach ($envKey in $envKeys) {
    try {
      $key = $envKey.Hive.OpenSubKey($envKey.SubKey, $true)
      if (-not $key) { continue }
      $current = $key.GetValue('Path', $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
      if ($current) {
        $kind = $key.GetValueKind('Path')
        $updated = ($current -split ';' | Where-Object { $_ -and $_ -notlike '*\Stretchly\bin' }) -join ';'
        if ($updated -ne $current) {
          $key.SetValue('Path', $updated, $kind)
          Write-Host "Removed Stretchly from PATH in $($envKey.SubKey)"
        }
      }
      $key.Close()
    } catch {
      Write-Host "PATH cleanup skipped for $($envKey.SubKey): $_"
    }
  }
}

$uninstall = Find-UninstallEntry
if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

try {
    $uninstallString = $uninstall.UninstallString

    # Parse the uninstaller executable path from the UninstallString.
    if ($uninstallString -match '^"([^"]+)"') {
        $uninstallExe = $matches[1]
    } elseif ($uninstallString -match '^(.+?\.exe)') {
        $uninstallExe = $matches[1]
    } else {
        $uninstallExe = $uninstallString
    }

    # Determine the install directory for cleanup.
    $installDir = $uninstall.InstallLocation
    if (-not $installDir -or -not (Test-Path $installDir)) {
        $installDir = Split-Path $uninstallExe -Parent
    }

    Stop-Process -Name "stretchly" -Force -ErrorAction SilentlyContinue

    $uninstallArgs = @("/S", "_?=$installDir")

    Write-Host "Uninstall command: $uninstallExe"
    Write-Host "Uninstall args: $uninstallArgs"

    $processOptions = @{
        FilePath = $uninstallExe
        ArgumentList = $uninstallArgs
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    if ($exitCode -ne 0) {
        $remaining = Find-UninstallEntry
        if ($remaining) {
            # The vendor uninstaller crashed before removing anything; finish
            # the job manually.
            Write-Host "Uninstaller failed and app is still registered; removing manually"
            Remove-Item -Path $remaining.PSPath -Recurse -Force -ErrorAction SilentlyContinue
            $shortcuts = @(
                "$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Stretchly.lnk",
                "$env:ProgramData\Microsoft\Windows\Start Menu\Programs\Stretchly.lnk",
                "$env:USERPROFILE\Desktop\Stretchly.lnk",
                "$env:PUBLIC\Desktop\Stretchly.lnk"
            )
            foreach ($shortcut in $shortcuts) {
                Remove-Item $shortcut -Force -ErrorAction SilentlyContinue
            }
        } else {
            Write-Host "App is no longer registered despite exit code $exitCode; treating as success"
        }
    }

    Remove-StretchlyFromPath

    if ($installDir -and (Test-Path $installDir)) {
        Remove-Item $installDir -Recurse -Force -ErrorAction SilentlyContinue
    }

    if (Find-UninstallEntry) {
        Write-Host "Stretchly is still present after removal attempts"
        Exit 1
    }

    Write-Host "Stretchly is no longer present"
    Exit 0
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
