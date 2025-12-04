# Set message and MDM Window variables
$message = @'
IT requires you to migrate this PC to Fleet.
Please sign in using your work credentials when prompted.
After you have finished, open Fleet Desktop from your system tray and select Refetch on your My device page to tell your organization that MDM is on.
'@

$uri = "ms-device-enrollment:?mode=mdm"

# Send the message first
msg.exe * $message

# Wait a few seconds before opening the "Set up a work or school account" window to give the end user a chance to read the message
Start-Sleep -Seconds 10
Start-Process $uri -ErrorAction Stop
