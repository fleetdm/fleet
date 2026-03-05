# Please don't delete. This script is referenced in the guide here: https://fleetdm.com/guides/windows-mdm-setup#migrating-from-another-mdm-solution
# Detects and removes unreachable WSUS server configurations that can break Windows Update
# after migrating from another MDM solution. Only removes WSUS config if the server cannot
# be reached at all (HTTP error responses like 403 are treated as reachable).
# Reboot the device after running this script.

$WUPath = "HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate"
if (-not (Test-Path $WUPath)) {
  Write-Host "Windows Update policy path not found - no action needed"
  exit 0
}

$wuServer = (Get-ItemProperty -Path $WUPath -Name "WUServer" -ErrorAction SilentlyContinue).WUServer
if (-not $wuServer) {
  Write-Host "No WSUS server configured - no action needed"
  exit 0
}

$reachable = $false
try {
  $null = Invoke-WebRequest -Uri $wuServer -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
  $reachable = $true
} catch [System.Net.WebException] {
  if ($_.Exception.Response) {
    # Server responded with an HTTP error (e.g., 403) - it is still reachable
    $reachable = $true
  }
} catch {
  # Connection failed entirely
}

if ($reachable) {
  Write-Host "WSUS server $wuServer is reachable - no action taken"
} else {
  Write-Host "WSUS server $wuServer is unreachable - removing configuration"
  Remove-ItemProperty -Path $WUPath -Name "WUServer" -ErrorAction SilentlyContinue
  Remove-ItemProperty -Path $WUPath -Name "WUStatusServer" -ErrorAction SilentlyContinue
  Restart-Service wuauserv -Force
  Write-Host "Windows Update service restarted"
}
