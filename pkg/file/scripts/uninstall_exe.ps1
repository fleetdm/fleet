# Fleet extracts name from installer (EXE) and saves it to package ID variable
$softwareName = $PACKAGE_ID

# Get the list of subkeys under the Uninstall registry path
$uninstallKeys = Get-ChildItem "HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall" | ForEach-Object { Get-ItemProperty $_.PSPath }

# Loop through each registry key to find the one containing "$softwareName" in DisplayName and run uninstall command from UninstallString
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -like "*$softwareName*") {
        # Get the uninstall command
        $uninstallCommand = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
        
        # Run the uninstall command with arguments using the call operator &
        & $uninstallCommand
        break  # Exit the loop once the software is found and uninstalled
    }
}