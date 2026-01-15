# Please don't delete. This script is referenced in the guide here: https://fleetdm.com/guides/windows-mdm-setup#turn-off-windows-mdm

Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;

public class MdmRegistration
{
    [DllImport("mdmregistration.dll", SetLastError = true)]
    public static extern int UnregisterDeviceWithManagement(IntPtr pDeviceID);

    public static int UnregisterDevice()
    {
        return UnregisterDeviceWithManagement(IntPtr.Zero);
    }
}
"@ -Language CSharp

try {
    # Step 1: Unregister the device from MDM using the Windows API
    $result = [MdmRegistration]::UnregisterDevice()

    if ($result -ne 0) {
        throw "UnregisterDeviceWithManagement failed with error code: $result"
    }

    Write-Host "Device unregistration called successfully."

    # Step 2: Clear the DiscoveryServiceFullURL registry values to ensure Fleet detects
    # the device as unenrolled on the next detail query. This addresses the issue where
    # the registry may still contain enrollment data even after unregistering, preventing
    # Fleet from being able to re-enabled MDM.
    $enrollmentsPath = "HKLM:\SOFTWARE\Microsoft\Enrollments"
    
    if (Test-Path $enrollmentsPath) {
        $enrollmentKeys = Get-ChildItem -Path $enrollmentsPath -ErrorAction SilentlyContinue
        
        $clearedCount = 0
        foreach ($key in $enrollmentKeys) {
            # Only clear DiscoveryServiceFullURL from enrollment keys that have a UPN
            # (these are the ones Fleet's query checks). This matches Fleet's query logic
            # which filters by entries with UPN values.
            $upnPath = Join-Path $key.PSPath "UPN"
            $discoveryUrlPath = Join-Path $key.PSPath "DiscoveryServiceFullURL"
            
            if (Test-Path $upnPath) {
                if (Test-Path $discoveryUrlPath) {
                    try {
                        Remove-ItemProperty -Path $key.PSPath -Name "DiscoveryServiceFullURL" -ErrorAction Stop
                        $clearedCount++
                        Write-Host "Cleared DiscoveryServiceFullURL from enrollment key: $($key.PSChildName)"
                    } catch {
                        Write-Warning "Failed to clear DiscoveryServiceFullURL from $($key.PSChildName): $_"
                    }
                }
            }
        }
        
        if ($clearedCount -gt 0) {
            Write-Host "Cleared DiscoveryServiceFullURL from $clearedCount enrollment key(s). Fleet will detect the device as unenrolled on the next detail query."
        } else {
            Write-Host "No DiscoveryServiceFullURL values found to clear (device may already be unenrolled)."
        }
    } else {
        Write-Host "Enrollments registry path not found. Device may already be unenrolled."
    }
} catch {
    Write-Error "Error calling UnregisterDeviceWithManagement: $_"
    exit 1
}
