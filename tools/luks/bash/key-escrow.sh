#!/bin/bash

# This is a script that adds a luks key in almost the same way orbit does
# and sends a request to /fleet/orbit/luks_data to store it on fleet

# Install dependencies
# Source ID from os-release
if [[ -f /etc/os-release ]]; then
  . /etc/os-release
else
  echo "Cannot detect OS"
  exit 1
fi

install_deps() {
  case "$ID" in
    ubuntu|debian|kali)
      apt-get install --assume-yes -f curl jq ;;
    fedora|rhel|centos)
      dnf install --assumeyes curl jq ;;
    *)
      echo "Could not find distribution"
      exit 1
      ;;
  esac
}

# Detect active GUI user

ACTIVE_USER="$(loginctl list-sessions --no-legend | awk '$2 != "root" && $3 == "seat0" {print $2; exit}')"

if [[ -z "$ACTIVE_USER" ]]; then
  ACTIVE_USER="$(loginctl list-sessions --no-legend | awk '$2 != "root" {print $2; exit}')"
fi

if [[ -z "$ACTIVE_USER" ]]; then
  echo "ERROR: No active GUI user found"
  exit 1
fi

USER_ID="$(id -u "$ACTIVE_USER")"
USER_NAME="$(id -nu "$USER_ID")"
SESSION_ID="$(loginctl list-sessions --no-legend | awk -v u="$ACTIVE_USER" '$2==u {print $1; exit}')"
SESSION_TYPE="$(loginctl show-session "$SESSION_ID" -p Type --value)"

export XDG_RUNTIME_DIR="/run/user/$USER_ID"
export DBUS_SESSION_BUS_ADDRESS="unix:path=$XDG_RUNTIME_DIR/bus"

if [[ "$SESSION_TYPE" == "x11" ]]; then
  DISPLAY_VALUE="$(loginctl show-session "$SESSION_ID" -p Display --value)"
  if [[ -z "$DISPLAY_VALUE" ]]; then
    DISPLAY_VALUE=":0"
  fi
  export DISPLAY="$DISPLAY_VALUE"

elif [[ "$SESSION_TYPE" == "wayland" ]]; then
  WAYLAND_PATH="$(ls "$XDG_RUNTIME_DIR"/wayland-* 2>/dev/null | grep -v '\.lock$' | head -n1)"
  if [[ -z "$WAYLAND_PATH" ]]; then
    echo "ERROR: Wayland session detected but no wayland display found"
    exit 1
  fi
  export WAYLAND_DISPLAY="$(basename "$WAYLAND_PATH")"

else
  echo "ERROR: No graphical session detected (type=$SESSION_TYPE)"
  exit 1
fi


# Find the root partition
luks_device=$(lsblk --json | jq -r '
def search_root(devs; path):
  (devs // [])[] as $dev |
  if ((($dev.mountpoints? // []) | index("/")) != null) then
    {node: $dev, path: path}
  else
    search_root($dev.children? // []; path + [$dev])
  end;

search_root(.blockdevices; []) as $root |
($root.path? // [] | map(select(type=="object")) | reverse[] | select(.type=="part") | .name) as $parent_partition |
"/dev/" + $parent_partition
')

echo "Detected LUKS device: $luks_device"

if [[ ! -b "$luks_device" ]]; then
  echo "ERROR: Resolved LUKS device is not a block device: $luks_device"
  exit 1
fi

if ! cryptsetup isLuks "$luks_device"; then
  echo "ERROR: Root filesystem is not LUKS-encrypted"
  exit 1
fi

# Prompt for passphrase (GUI user)
is_passphrase_valid() {
  device="$1"
  slot="$2"
  passphrase="$3"

  if [ -z "$passphrase" ]; then
    return 1
  fi

  printf '%s' "$passphrase" | cryptsetup luksOpen \
    --test-passphrase \
    --key-slot "$slot" \
    "$device" \
    --key-file=- \
    >/dev/null 2>&1
}

user_key_slot=0

while true; do
  passphrase="$(sudo -u "$USER_NAME" \
    DISPLAY="$DISPLAY" \
    WAYLAND_DISPLAY="${WAYLAND_DISPLAY:-}" \
    XDG_RUNTIME_DIR="/run/user/$USER_ID" \
    DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$USER_ID/bus" \
    zenity --password \
      --title="Enter disk encryption passphrase" \
      --text="Passphrase:" \
      --timeout=60
  )"

  if [ -z "$passphrase" ]; then
    echo "ERROR: Passphrase entry cancelled or timed out"
    exit 1
  fi

  if is_passphrase_valid "$luks_device" "$user_key_slot" "$passphrase"; then
    break
  fi

  sudo -u "$USER_NAME" \
    DISPLAY="$DISPLAY" \
    WAYLAND_DISPLAY="${WAYLAND_DISPLAY:-}" \
    XDG_RUNTIME_DIR="/run/user/$USER_ID" \
    DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/$USER_ID/bus" \
    zenity --error --text="Incorrect passphrase. Please try again."
done

# Find free LUKS keyslot
luks_json="$(cryptsetup luksDump --dump-json-metadata "$luks_device")"

if [[ -z "$luks_json" ]]; then
  echo "ERROR: Failed to retrieve LUKS metadata"
  exit 1
fi

# Get the next available keyslot
get_next_available_keyslot() {
  device="$1"

  # Consistent with orbit code
  MAX_KEYSLOTS=8

  # Get occupied keyslots as sorted integers
  keys_taken="$(
    cryptsetup luksDump --dump-json-metadata "$device" 2> /dev/null \
      | jq -r '.keyslots | keys[]' \
      | sort -n
  )"

  unused_key=0

  for key in $keys_taken; do
    if [ "$unused_key" -eq "$key" ]; then
      unused_key=$((unused_key + 1))
    fi
  done

  if [ "$unused_key" -ge "$MAX_KEYSLOTS" ]; then
    echo "ERROR: no empty key slots available ($unused_key)" >&2
    return 1
  fi

  echo "$unused_key"
}

free_slot="$(get_next_available_keyslot "$luks_device")"

if [[ -z "$free_slot" ]]; then
  echo "ERROR: No free LUKS keyslots available"
  exit 1
fi

generate_random_passphrase() {
  tr -dc 'A-Za-z0-9' < /dev/urandom |
    head -c 16 |
    sed 's/.\{4\}/&-/g; s/-$//'
}

new_key="$(generate_random_passphrase)"

if [[ -z "$new_key" ]]; then
  echo "ERROR: Failed to generate recovery passphrase"
  exit 1
fi

# Add recovery key to LUKS
old_len=${#passphrase}
buffer="$passphrase$new_key"

# Note: orbit uses --key-slot instead of --new-key-slot
printf '%s' "$buffer" | \
  cryptsetup luksAddKey "$luks_device" \
    --key-file=- \
    --keyfile-size="$old_len" \
    --new-key-slot="$free_slot" \
    --key-size=512

if ! is_passphrase_valid "$luks_device" "$free_slot" "$new_key"; then
  echo "ERROR: Failed to validate escrow key" >&2
  exit 1
fi

unset buffer
unset passphrase

# Extract the salt for our free_slot 
luks_json="$(cryptsetup luksDump --dump-json-metadata "$luks_device")"
salt="$(echo "$luks_json" | jq -r --arg slot "$free_slot" '.keyslots[$slot].kdf.salt')"

if [[ -z "$salt" || "$salt" == "null" ]]; then
  echo "ERROR: Could not retrieve salt for keyslot $free_slot"
  exit 1
fi

# Send the new encryption key to Fleet
FLEET_URL="$(awk -F= '/^ORBIT_FLEET_URL=/ {print $2}' /etc/default/orbit)"
ORBIT_NODE_KEY="$(cat /opt/orbit/secret-orbit-node-key.txt)"

luks_response_json=$(jq -n \
  --arg orbit_node_key "$ORBIT_NODE_KEY" \
  --arg passphrase "$new_key" \
  --arg salt "$salt" \
  --argjson keyslot "$free_slot" \
  --arg client_error "" \
  '{
    orbit_node_key: $orbit_node_key,
    passphrase: $passphrase,
    salt: $salt,
    key_slot: $keyslot,
    client_error: $client_error
  }'
)

curl -sS -X POST "$FLEET_URL/api/fleet/orbit/luks_data" \
  -H "Content-Type: application/json" \
  -d "$luks_response_json"

curl_code=$?
if [ $curl_code -ne 0 ]; then
  echo "ERROR: Failed to send LUKS escrow to Fleet (curl exit code $curl_code)" >&2
  exit 1
fi

unset luks_json
unset ORBIT_NODE_KEY
unset FLEET_URL
