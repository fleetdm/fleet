#!/bin/bash

set -euo pipefail

fix_import_spacing() {
  local file="$1"
  awk '
    /^import \* as React from / {
      print;
      print "";
      next;
    }
    /^import type / {
      print;
      print "";
      next;
    }
    { print }
  ' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
}

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 /path/to/App.app"
  exit 1
fi

APP_PATH="$1"

# Find the first .icns file in the app bundle
ICNS_PATH=$(find "$APP_PATH/Contents/Resources" -maxdepth 1 -name "*.icns" | head -n 1 || true)

if [[ ! -f "$ICNS_PATH" ]]; then
  echo "Error: .icns file not found at $ICNS_PATH"
  exit 1
fi

TMP_DIR=$(mktemp -d)
echo "Extracting icons to: $TMP_DIR"

# Convert .icns to iconset (PNG files in multiple sizes)
ICON_DIR="$TMP_DIR/icons.iconset"
mkdir -p "$ICON_DIR"
iconutil -c iconset "$ICNS_PATH" -o "$ICON_DIR"

# Find the 128x128 PNG
ICON_32=$(find "$ICON_DIR" -name "*128x128*.png" | head -n 1)
if [[ -z "$ICON_32" ]]; then
  echo "Error: 128x128 icon not found."
  exit 1
fi
echo "Using 128x128 icon: $ICON_32"

# Fixed width and height
WIDTH=32
HEIGHT=32

# Encode PNG to base64 and wrap it in an SVG
BASE64_DATA=$(base64 -i "$ICON_32" | tr -d '\n')
OUTPUT_SVG="$TMP_DIR/$(basename "$APP_PATH" .app).svg"

cat > "$OUTPUT_SVG" <<EOF
<svg xmlns="http://www.w3.org/2000/svg" width="$WIDTH" height="$HEIGHT" viewBox="0 0 $WIDTH $HEIGHT" version="1.1">
  <image width="$WIDTH" height="$HEIGHT" href="data:image/png;base64,$BASE64_DATA"/>
</svg>
EOF

echo "SVG saved to: $OUTPUT_SVG"

svgr "$OUTPUT_SVG" --typescript --ext tsx --out-dir frontend/pages/SoftwarePage/components/icons/

# Extract the base name without .app and with PascalCase for component name
APP_NAME=$(basename "$APP_PATH" .app)
COMPONENT_NAME="${APP_NAME//[^a-zA-Z0-9]/}"  # Optional cleanup if needed
TSX_FILE="frontend/pages/SoftwarePage/components/icons/${COMPONENT_NAME}.tsx"

echo "Component name: $COMPONENT_NAME"

# Fix import spacing in the generated TSX file
# 1. Add blank line before `import type`
# 2. Add blank line after `import type`
fix_import_spacing "$TSX_FILE"

# Dynamically find the actual component name in the TSX file (e.g., SvgITerm)
SVG_COMPONENT_NAME=$(grep -oE '^const Svg[A-Za-z0-9_]+' "$TSX_FILE" | awk '{print $2}')
if [[ -z "$SVG_COMPONENT_NAME" ]]; then
  echo "Error: could not find Svg component name in $TSX_FILE"
  exit 1
fi

# Strip the 'Svg' prefix (e.g., SvgITerm -> ITerm)
NEW_COMPONENT_NAME="${SVG_COMPONENT_NAME#Svg}"

# Replace all occurrences in the file
sed -i '' "s/$SVG_COMPONENT_NAME/$NEW_COMPONENT_NAME/g" "$TSX_FILE"
