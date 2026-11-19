#!/bin/bash
# Schedules the latest macOS minor update for Software Update's overnight install
# (AutoInstallProductKeys, the "Update Tonight" list). Never queues a major upgrade,
# but keeps one the user scheduled. Requires macOS 15 or later.

PREFS="/Library/Preferences/com.apple.SoftwareUpdate"

if (( $(/usr/bin/sw_vers -productVersion | /usr/bin/cut -d. -f1) < 15 )); then
    echo "AutoInstallProductKeys requires macOS 15 or later."
    exit 0
fi

read_list() {
    local count i value
    count=$(/usr/bin/plutil -extract "$1" raw -o - "$PREFS.plist" 2>/dev/null) || return 0
    for (( i = 0; i < count; i++ )); do
        value=$(/usr/bin/plutil -extract "$1.$i$2" raw -o - "$PREFS.plist" 2>/dev/null) && echo "$value"
    done
}

all_scheduled() {
    local existing key
    existing=$(read_list AutoInstallProductKeys)
    for key in "${keys[@]}"; do
        /usr/bin/grep -qxF -- "$key" <<< "$existing" || return 1
    done
}

keys=()
builds=" "
while IFS= read -r key; do
    [[ "$key" =~ ^MSU_UPDATE_([^_]+)_ ]] && builds+="${BASH_REMATCH[1]} "
    [[ "$key" == MSU_UPDATE_*_minor ]] || continue
    # Both payload keys are queued; Software Update ignores the one that doesn't apply.
    # A key in any other format is queued as Apple offers it.
    if [[ "$key" =~ ^MSU_UPDATE_([^_]+)_(patch|full)_([^_]+)_minor$ ]]; then
        keys+=("MSU_UPDATE_${BASH_REMATCH[1]}_patch_${BASH_REMATCH[3]}_minor" "MSU_UPDATE_${BASH_REMATCH[1]}_full_${BASH_REMATCH[3]}_minor")
    else
        keys+=("$key")
    fi
done < <(read_list RecommendedUpdates ".Product Key")

if (( ${#keys[@]} == 0 )); then
    echo "No macOS minor update is available."
    exit 0
fi

if all_scheduled; then
    echo "Already scheduled: ${keys[*]}"
    exit 0
fi

while IFS= read -r key; do
    if [[ "$key" =~ ^MSU_UPDATE_([^_]+)_.*_major$ && "$builds" == *" ${BASH_REMATCH[1]} "* ]]; then
        keys+=("$key")
    fi
done < <(read_list AutoInstallProductKeys)

# defaults parses -array elements as plist values; quoting keeps any key text intact.
quoted=()
for key in "${keys[@]}"; do
    quoted+=("\"$key\"")
done
/usr/bin/defaults write "$PREFS" AutoInstallProductKeys -array "${quoted[@]}" || exit 1
if ! all_scheduled; then
    echo "Couldn't schedule: ${keys[*]}"
    exit 1
fi
/usr/bin/notifyutil -p com.apple.SoftwareUpdate.TriggerBackgroundCheck
echo "Scheduled for overnight install: ${keys[*]}"
