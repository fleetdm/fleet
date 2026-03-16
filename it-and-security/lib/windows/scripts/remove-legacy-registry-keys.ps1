# Windows enrollment script: Remove legacy registry keys
#
# This script runs on Windows devices when they are enrolled into Fleet.
# It cleans up legacy registry keys left behind by previous management tools
# or configurations, ensuring a clean state for the newly enrolled device.
#
# Usage: This script is intended to be used as a Windows enrollment script
# in Fleet (i.e., it runs automatically when a Windows device is added/enrolled
# to Fleet). It requires administrator privileges.

# List of legacy registry key paths to remove.
#
# TODO: Replace these placeholder paths with the actual registry key paths
# that need to be cleaned up for your environment. Common examples include
# keys left behind by previous MDM solutions, old management agents, or
# outdated configuration policies.
$legacyRegistryKeys = @(
    # TODO: Add real registry key paths below. Examples of what these might look like:
    # "HKLM:\SOFTWARE\Policies\PreviousMDM"
    # "HKLM:\SOFTWARE\PreviousManagementTool\AgentConfig"
    # "HKCU:\SOFTWARE\OldVendor\DeprecatedSettings"
    # "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\OldAgent"

    "HKLM:\SOFTWARE\TODO\LegacyKey1"   # TODO: Replace with actual registry key path
    "HKLM:\SOFTWARE\TODO\LegacyKey2"   # TODO: Replace with actual registry key path
    "HKLM:\SOFTWARE\TODO\LegacyKey3"   # TODO: Replace with actual registry key path
)

$totalKeys = $legacyRegistryKeys.Count
$removedCount = 0
$skippedCount = 0
$failedCount = 0

Write-Host "Starting legacy registry key cleanup ($totalKeys keys to process)..."

for ($i = 0; $i -lt $totalKeys; $i++) {
    $keyPath = $legacyRegistryKeys[$i]
    $step = $i + 1

    if (-not (Test-Path $keyPath)) {
        Write-Host "[$step/$totalKeys] Key not found (skipped): $keyPath"
        $skippedCount++
        continue
    }

    try {
        Remove-Item -Path $keyPath -Recurse -Force -ErrorAction Stop
        Write-Host "[$step/$totalKeys] Removed: $keyPath"
        $removedCount++
    } catch {
        Write-Host "[$step/$totalKeys] Failed to remove: $keyPath - Error: $_"
        $failedCount++
    }
}

Write-Host ""
Write-Host "Legacy registry key cleanup complete."
Write-Host "  Removed: $removedCount"
Write-Host "  Skipped (not found): $skippedCount"
Write-Host "  Failed: $failedCount"

if ($failedCount -gt 0) {
    Write-Host "WARNING: $failedCount key(s) could not be removed. Review the errors above."
    exit 1
}

exit 0
