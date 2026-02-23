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
    # Step 1: Check for DiscoveryServiceFullURL values before unregistering
    # This helps us provide clearer output about what happened
    $enrollmentsPath = "HKLM:\SOFTWARE\Microsoft\Enrollments"
    $foundBeforeUnregister = $false
    
    if (Test-Path $enrollmentsPath) {
        $enrollmentKeys = Get-ChildItem -Path $enrollmentsPath -ErrorAction SilentlyContinue
        
        foreach ($key in $enrollmentKeys) {
            $upnPath = Join-Path $key.PSPath "UPN"
            $discoveryUrlPath = Join-Path $key.PSPath "DiscoveryServiceFullURL"
            
            if (Test-Path $upnPath) {
                if (Test-Path $discoveryUrlPath) {
                    $foundBeforeUnregister = $true
                    break
                }
            }
        }
    }

    # Step 2: Unregister the device from MDM using the Windows API
    $result = [MdmRegistration]::UnregisterDevice()

    if ($result -ne 0) {
        throw "UnregisterDeviceWithManagement failed with error code: $result"
    }

    Write-Host "Device unregistration called successfully."

    # Step 3: Clear any remaining DiscoveryServiceFullURL registry values to ensure Fleet detects
    # the device as unenrolled on the next refetch. The UnregisterDeviceWithManagement API
    # may have already cleared these values, but we check and clear any remaining ones to be safe.
    $clearedCount = 0
    
    if (Test-Path $enrollmentsPath) {
        $enrollmentKeys = Get-ChildItem -Path $enrollmentsPath -ErrorAction SilentlyContinue
        
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
    }
    
    # Provide clearer output based on what we found
    if ($clearedCount -gt 0) {
        Write-Host "Cleared DiscoveryServiceFullURL from $clearedCount enrollment key(s). Fleet will detect the device as unenrolled on the next refetch."
    } elseif ($foundBeforeUnregister) {
        Write-Host "MDM unregistration completed. The UnregisterDeviceWithManagement API automatically cleared the registry values."
        Write-Host "Fleet will detect the device as unenrolled on the next refetch."
    } else {
        Write-Host "MDM unregistration completed. No DiscoveryServiceFullURL registry values were found (device was not enrolled or values were already cleared)."
    }
} catch {
    Write-Error "Error calling UnregisterDeviceWithManagement: $_"
    exit 1
}
