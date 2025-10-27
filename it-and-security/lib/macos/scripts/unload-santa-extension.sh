#!/usr/bin/env bash
#
# Unload (stop) an osquery extension and prevent it from reloading
# Usage: sudo ./unload_osquery_extension.sh /path/to/extension.ext

EXT_PATH="$1"

if [[ -z "$EXT_PATH" ]]; then
  echo "Usage: $0 /var/fleet/extensions/santa.ex"
  exit 1
fi

# Common autoload locations
AUTOLOAD_FILES=(
  "/etc/osquery/extensions.load"
  "/usr/local/osquery/extensions.load"
  "/var/osquery/extensions.load"
)

echo "🧩 Unloading osquery extension: $EXT_PATH"

# 1️⃣ Stop any running extension process
PID=$(pgrep -f "$EXT_PATH")
if [[ -n "$PID" ]]; then
  echo "Stopping running extension process (PID $PID)..."
  kill "$PID" 2>/dev/null
  sleep 1
  if ps -p "$PID" >/dev/null; then
    echo "Force killing extension (PID $PID)..."
    kill -9 "$PID" 2>/dev/null
  fi
else
  echo "No running process found for $EXT_PATH."
fi

# 2️⃣ Remove from autoload files (so it doesn’t restart)
for file in "${AUTOLOAD_FILES[@]}"; do
  if [[ -f "$file" ]]; then
    if grep -q "$EXT_PATH" "$file"; then
      echo "Removing from autoload file: $file"
      sudo sed -i.bak "\|$EXT_PATH|d" "$file"
    fi
  fi
done

# 3️⃣ Optionally delete the binary
read -p "Do you want to delete the extension binary (y/N)? " confirm
if [[ "$confirm" =~ ^[Yy]$ ]]; then
  echo "Deleting $EXT_PATH..."
  sudo rm -f "$EXT_PATH"
fi

# 4️⃣ Restart osqueryd (optional)
read -p "Restart osqueryd service to ensure clean state (y/N)? " restart
if [[ "$restart" =~ ^[Yy]$ ]]; then
  if systemctl list-units --type=service | grep -q osqueryd; then
    echo "Restarting osqueryd..."
    sudo systemctl restart osqueryd
  elif launchctl list | grep -q osqueryd; then
    echo "Restarting osqueryd (macOS)..."
    sudo launchctl kickstart -k system/com.facebook.osqueryd
  else
    echo "Could not detect osqueryd service manager."
  fi
fi

echo "✅ Extension unloaded and autoload entry removed."
