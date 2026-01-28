# Please don't delete. This script is used in the guide here: https://fleetdm.com/guides/scripts

# Error if not run as admin
if (-not ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Error "This script must be run as an administrator."
    exit 1
}   
# Get the BinaryPathName using Get-WmiObject
$service = Get-WmiObject -Class Win32_Service -Filter "Name='Fleet osquery'"
if (-not $service) {
    Write-Error "Service '$serviceName' not found."
    exit 1
}
$binaryPath = $service.PathName
# Replace any existing --enable-scripts flag with --enable-scripts="True"
$modifiedPath = $binaryPath -replace '--enable-scripts(=".*?")?', '--enable-scripts="True"'
# Update the service configuration
$setServiceCmd = "sc.exe config `"$serviceName`" binPath= `"$modifiedPath`""
Invoke-Expression $setServiceCmd
# Restart the service
Restart-Service -Name $serviceName
Write-Host "Fleet Desktop feature enabled and service restarted."
