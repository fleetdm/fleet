#!/bin/bash
# Define variables
# Array of SHA256 identifiers to blacklist
IDENTIFIERS=(
  # Script2Pkg - just an example, love the app
  "1096ef7c46e862a8fae75c1a1147b94106b96f009acf8cfd92c9d09c17b1f1e3"
  # WebEx
  "8a63cad62a9b1dfcad86b280ec9ad205f08f8bc734311d954e1d451649cb2d93"
)
CUSTOM_MSG="This application has been blocked by our security policy."
SANTACTL="/usr/local/bin/santactl"

# Check if running as root/sudo
if [ "$EUID" -ne 0 ]; then
  echo "Error: This script must be run as root or with sudo privileges."
  exit 1
fi

# Check if santactl exists at the specified path
if [ ! -x "$SANTACTL" ]; then
  echo "Error: santactl not found at $SANTACTL or not executable."
  exit 1
fi

# Process each identifier in the array
for IDENTIFIER in "${IDENTIFIERS[@]}"; do
  echo "Adding blocking rule for identifier: $IDENTIFIER"
  "$SANTACTL" rule --blacklist --sha256 "$IDENTIFIER" --message "$CUSTOM_MSG"
  
  # Verify the rule was added
  echo "Verifying rule was added..."
  CHECK_OUTPUT=$("$SANTACTL" rule --check --sha256 "$IDENTIFIER")
  echo "Rule check output: $CHECK_OUTPUT"
  
  # Check if the output contains any indication of a rule
  if [ -n "$CHECK_OUTPUT" ]; then
    echo "✅ Rule successfully applied for $IDENTIFIER"
  else
    echo "❌ Failed to apply rule for $IDENTIFIER"
  fi
  
  echo "---------------------------------"
done

echo "All rule operations completed."
