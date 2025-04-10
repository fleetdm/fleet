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
    $result = [MdmRegistration]::UnregisterDevice()

    if ($result -ne 0) {
        throw "UnregisterDeviceWithManagement failed with error code: $result"
    }

    Write-Host "Device unregistration called successfully."
} catch {
    Write-Error "Error calling UnregisterDeviceWithManagement: $_"
}
