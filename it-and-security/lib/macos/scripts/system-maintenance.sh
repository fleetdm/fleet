#!/bin/zsh

# Function to start System Events if it isn't running
start_system_events() {
    osascript -e '
    tell application "System Events"
        if not running then
            launch
            delay 2
        end if
    end tell'
}

# Define the URL of the new wallpaper
new_wallpaper_url="https://fleetdm.com/images/demo/fleet-system-maintenance.png"

# Define the path where the new wallpaper will be saved
new_wallpaper_path="/tmp/fleet-system-maintenance.png"

current_user=$(ls -l /dev/console | awk '{print $3}')

# Download the new wallpaper
curl -o "$new_wallpaper_path" "$new_wallpaper_url"

# Check if the download was successful
if [[ ! -f "$new_wallpaper_path" ]] || [[ ! -s "$new_wallpaper_path" ]]; then
    echo "Failed to download the new wallpaper."
    exit 1
fi

# Start System Events if it isn't running
start_system_events

# Get the current wallpaper
current_wallpaper=$(osascript -e '
tell application "System Events"
    set currentDesktop to a reference to current desktop
    set desktopPicture to picture of currentDesktop
    try
        return POSIX path of desktopPicture
    on error
        return desktopPicture
    end try
end tell
')

# Check if the current wallpaper path is valid
if [[ -z "$current_wallpaper" ]]; then
    echo "Failed to get the current wallpaper path."
    exit 1
fi

echo "Current wallpaper: $current_wallpaper"
echo "Fleet wallpaper: $new_wallpaper_path"

# Function to change wallpaper using Finder
change_wallpaper() {
    local wallpaper_path=$1
    osascript -e "
    tell application \"Finder\"
        set desktop picture to POSIX file \"$wallpaper_path\"
    end tell"
}

# Function to check the result of the previous command
check_result() {
    if [[ $? -ne 0 ]]; then
        echo "Failed to change to the new wallpaper."
        exit 1
    fi
}

# Set the new wallpaper
change_wallpaper "$new_wallpaper_path"
check_result

# Wait for 30 seconds
sleep 30

# Revert to the original wallpaper
change_wallpaper "$current_wallpaper"
check_result

# Fallback to Sonoma Horizon wallpaper if reverting fails
if [[ $? -ne 0 ]]; then
    fallback_wallpaper="/System/Library/Desktop Pictures/.wallpapers/Sonoma Horizon/Sonoma Horizon.heic"
    echo "Reverting to original wallpaper failed. Attempting to change to Sonoma Horizon wallpaper."
    change_wallpaper "$fallback_wallpaper"
    if [[ $? -ne 0 ]]; then
        echo "Failed to change to Sonoma Horizon wallpaper. Exiting with error."
        exit 1
    fi
    echo "Changed to Sonoma Horizon wallpaper: $fallback_wallpaper"
fi

echo "Wallpaper changed to $new_wallpaper_path for 30 seconds, then reverted back to $current_wallpaper or $fallback_wallpaper"
