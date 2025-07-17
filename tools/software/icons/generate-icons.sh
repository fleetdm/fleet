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

# Defaults
SLUG=""
APP_PATH=""

# Parse required arguments
while getopts ":s:a:" opt; do
  case "$opt" in
    s) SLUG=$(echo "$OPTARG" | cut -d'/' -f1) ;;
    a) APP_PATH="$OPTARG" ;;
    \?) echo "Invalid option: -$OPTARG" >&2; exit 1 ;;
    :) echo "Option -$OPTARG requires an argument." >&2; exit 1 ;;
  esac
done

if ! command -v svgr &> /dev/null; then
  echo "Error: 'svgr' is not installed. Install it with 'npm install -g @svgr/cli'"
  exit 1
fi

# Validate
if [[ -z "$SLUG" || -z "$APP_PATH" ]]; then
  echo "Usage: $0 -s slug-name -a /path/to/App.app"
  exit 1
fi

# Find the first .icns file in the app bundle
ICNS_PATH=$(find "$APP_PATH/Contents/Resources" -maxdepth 1 -name "*.icns" | head -n 1 || true)
if [[ ! -f "$ICNS_PATH" ]]; then
  echo "Error: .icns file not found in $APP_PATH"
  exit 1
fi

# Extract iconset
TMP_DIR=$(mktemp -d)
ICON_DIR="$TMP_DIR/icons.iconset"
mkdir -p "$ICON_DIR"
iconutil -c iconset "$ICNS_PATH" -o "$ICON_DIR"

# Find the 128x128 PNG
ICON_128=$(find "$ICON_DIR" -name "*128x128.png" | head -n 1)
if [[ -z "$ICON_128" ]]; then
  echo "Error: 128x128 icon not found."
  exit 1
fi
echo "Using icon for SVG and PNG: $ICON_128"

# Generate SVG from 128 PNG
WIDTH=32
HEIGHT=32
BASE64_DATA=$(base64 -i "$ICON_128" | tr -d '\n')
OUTPUT_SVG="$TMP_DIR/$(basename "$APP_PATH" .app).svg"

cat > "$OUTPUT_SVG" <<EOF
<svg xmlns="http://www.w3.org/2000/svg" width="$WIDTH" height="$HEIGHT" viewBox="0 0 $WIDTH $HEIGHT" version="1.1">
  <image width="$WIDTH" height="$HEIGHT" href="data:image/png;base64,$BASE64_DATA"/>
</svg>
EOF

echo "SVG saved to: $OUTPUT_SVG"

# Generate TSX component
svgr "$OUTPUT_SVG" --typescript --ext tsx --out-dir frontend/pages/SoftwarePage/components/icons/

# Determine component and file names
APP_NAME=$(basename "$APP_PATH" .app)
COMPONENT_NAME="${APP_NAME//[^a-zA-Z0-9]/}"
TSX_FILE="frontend/pages/SoftwarePage/components/icons/${COMPONENT_NAME}.tsx"

echo "Component name: $COMPONENT_NAME"

# Fix import spacing
fix_import_spacing "$TSX_FILE"

# Adjust component name (remove Svg prefix)
SVG_COMPONENT_NAME=$(grep -oE '^const Svg[A-Za-z0-9_]+' "$TSX_FILE" | awk '{print $2}')
if [[ -z "$SVG_COMPONENT_NAME" ]]; then
  echo "Error: could not find Svg component name in $TSX_FILE"
  exit 1
fi

NEW_COMPONENT_NAME="${SVG_COMPONENT_NAME#Svg}"
sed -i '' "s/$SVG_COMPONENT_NAME/$NEW_COMPONENT_NAME/g" "$TSX_FILE"

######################################
# Copy 128x128 PNG to asset location #
######################################

OUTPUT_IMAGE_DIR="website/assets/images"
OUTPUT_PNG="$OUTPUT_IMAGE_DIR/app-icon-${SLUG}-60x60@2x.png"
mkdir -p "$OUTPUT_IMAGE_DIR"
cp "$ICON_128" "$OUTPUT_PNG"

echo "Copied 128x128 PNG to: $OUTPUT_PNG"
