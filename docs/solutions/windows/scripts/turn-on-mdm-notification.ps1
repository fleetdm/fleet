# Set message and MDM Window variables
$message = @'
Mobile device management is off. MDM lets your org change settings & install software.

A 'Setup work or school' prompt will appear soon. Enter your work credentials.

Click Fleet Desktop from your system tray, My device, and Refetch to confirm MDM is on.
'@

$uri = "ms-device-enrollment:?mode=mdm"


# Send the message first
msg.exe * $message

# Wait a few seconds before opening the "Set up a work or school account" window to give the end user a chance to read the message
Start-Sleep -Seconds 5
Start-Process $uri -ErrorAction Stop
