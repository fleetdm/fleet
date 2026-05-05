# Attempts to locate VLC's product code from registry and uninstall it using msiexec

$displayName = "VLC media player"
$publisher = "VideoLAN"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$productCode = $null
foreach ($p in $paths) {
  $items = Get-ChildItem -Path $p -ErrorAction SilentlyContinue | ForEach-Object {
    Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue
  } | Where-Object {
    $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $_.Publisher -eq $publisher) -and $_.PSChildName -match '^{[A-F0-9]{8}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{4}-[A-F0-9]{12}}$'
  }
  if ($items) {
    $productCode = ($items | Select-Object -First 1).PSChildName
    break
  }
}

if (-not $productCode) {
  Write-Host "Product code not found for $displayName"
  Exit 1
}

Write-Host "Found product code: $productCode"
Write-Host "Attempting to uninstall using msiexec..."

$timeoutSeconds = 300  # 5 minute timeout

try {
  $process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $productCode, "/norestart") -PassThru -NoNewWindow
  
  # Wait for process with timeout
  $completed = $process.WaitForExit($timeoutSeconds * 1000)
  
  if (-not $completed) {
    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
    Write-Host "Uninstall timed out after $timeoutSeconds seconds"
    Exit 1603  # ERROR_UNINSTALL_FAILURE
  }
  
  # Check exit code and output result
  if ($process.ExitCode -eq 0) {
    Write-Host "Uninstall successful"
    Exit 0
  } else {
    Write-Host "Uninstall failed with exit code: $($process.ExitCode)"
    Exit $process.ExitCode
  }
} catch {
  Write-Host "Error running uninstaller: $_"
  Exit 1
}

