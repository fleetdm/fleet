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
    $thread = [Threading.Thread]::CurrentThread.GetApartmentState().ToString()
    $result = [MdmRegistration]::UnregisterDevice()

    Write-Host "Running in thread mode: $thread"

    if ($result -ne 0) {
        throw "UnregisterDeviceWithManagement failed with error code: $result"
    }

    Write-Host "Device unregistration called successfully."
} catch {
    Write-Error "Error calling UnregisterDeviceWithManagement: $_"
}
