#!/bin/zsh

################################################################################################
# Homebrew Update Script
################################################################################################

# Logging function
logging() {
    local log_level=$(printf "%s" "$1" | /usr/bin/tr '[:lower:]' '[:upper:]')
    local log_statement="$2"
    local script_name="$(/usr/bin/basename "$0")"
    local prefix=$(/bin/date +"[%b %d, %Y %Z %T $log_level]:")

    # Log file path
    local LOG_PATH="/var/log/homebrew_update.log"

    # Default log level to INFO if not provided
    [[ -z $log_level ]] && log_level="INFO"

    # Log to stdout and file
    /bin/echo "$prefix $log_statement"
    printf "%s %s\n" "$prefix" "$log_statement" >>"$LOG_PATH"
}

# Check if Homebrew is installed and update if it is
check_and_update_brew() {
    local brew_path="$(/usr/bin/find /usr/local/bin /opt -maxdepth 3 -name brew 2>/dev/null)"

    if [[ -n $brew_path ]]; then
        logging "info" "Homebrew already installed at $brew_path..."
        logging "info" "Updating homebrew ..."
        /usr/bin/su - "$current_user" -c "$brew_path update --force" | /usr/bin/tee -a "/Library/Logs/homebrew_update.log"
        logging "info" "Done ..."
    else
        logging "info" "Homebrew is not installed..."
    fi
}

# Get the current logged-in user excluding loginwindow, _mbsetupuser, and root
current_user=$(/usr/sbin/scutil <<<"show State:/Users/ConsoleUser" | /usr/bin/awk '/Name :/ && ! /loginwindow/ && ! /root/ && ! /_mbsetupuser/ { print $3 }' | /usr/bin/awk -F '@' '{print $1}')

# Verify the current_user is valid
if ! /usr/bin/dscl . -read "/Users/$current_user" >/dev/null 2>&1; then
    logging "error" "Specified user \"$current_user\" is invalid"
    exit 1
fi

logging "info" "--- Start homebrew update log ---"
check_and_update_brew
logging "info" "--- End homebrew update log ---"
