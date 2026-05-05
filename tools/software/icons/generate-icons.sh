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
  ' "$file" > "${file}.tmp"
  
  if [ $? -eq 0 ]; then
    mv "${file}.tmp" "$file"
  else
    rm -f "${file}.tmp"
    echo "Error: Failed to update $file" >&2
    return 1
  fi
}

# Check if a string is a valid JavaScript identifier
# Valid identifiers: start with letter/underscore/dollar, contain only letters/digits/underscore/dollar
is_valid_js_identifier() {
  local str="$1"
  # Check if it matches the pattern: starts with letter/underscore/dollar, followed by letters/digits/underscore/dollar only
  if [[ "$str" =~ ^[a-zA-Z_$][a-zA-Z0-9_$]*$ ]]; then
    return 0  # valid identifier
  else
    return 1  # not a valid identifier (contains spaces, special chars, or starts with number)
  fi
}

# Compare two strings alphabetically (case-insensitive)
# Returns: 0 if str1 < str2, 1 if str1 >= str2
compare_alphabetically() {
  local str1="$1"
  local str2="$2"
  # Convert to lowercase for comparison
  local lower1=$(echo "$str1" | tr '[:upper:]' '[:lower:]')
  local lower2=$(echo "$str2" | tr '[:upper:]' '[:lower:]')
  
  if [[ "$lower1" < "$lower2" ]]; then
    return 0
  else
    return 1
  fi
}

# Add import and map entry to index.ts
add_icon_to_index() {
  local component_name="$1"
  local app_name="$2"
  
  # Ensure we're in the repo root (in case function is called from different context)
  local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  local repo_root="$(cd "$script_dir/../../.." && pwd)"
  
  if [[ ! -d "$repo_root" ]]; then
    echo "Error: Repository root not found at $repo_root" >&2
    return 1
  fi
  
  cd "$repo_root" || {
    echo "Error: Failed to change to repository root directory" >&2
    return 1
  }
  
  # Use absolute path to avoid any directory issues
  local index_file="$repo_root/frontend/pages/SoftwarePage/components/icons/index.ts"
  
  # Normalize the path (resolve any .. or . components)
  index_file="$(cd "$(dirname "$index_file")" && pwd)/$(basename "$index_file")"
  
  if [[ ! -f "$index_file" ]]; then
    echo "Error: index.ts not found at $index_file" >&2
    echo "Current directory: $(pwd)" >&2
    echo "Repository root: $repo_root" >&2
    return 1
  fi
  
  if [[ ! -r "$index_file" ]]; then
    echo "Error: Cannot read index.ts at $index_file" >&2
    return 1
  fi
  
  # Check if import already exists
  if grep -q "import ${component_name} from" "$index_file"; then
    echo "Import for ${component_name} already exists in index.ts"
  else
    # Insert import in alphabetical order
    local import_line="import ${component_name} from \"./${component_name}\";"
    
    # Verify index_file is set and exists before running awk
    if [[ -z "$index_file" ]]; then
      echo "Error: index_file variable is empty" >&2
      return 1
    fi
    
    # Ensure we have an absolute path
    if [[ "$index_file" != /* ]]; then
      index_file="$(cd "$(dirname "$index_file")" && pwd)/$(basename "$index_file")"
    fi
    
    # Create temporary awk script file to avoid heredoc issues
    local awk_script_file=$(mktemp)
    cat > "$awk_script_file" << 'AWK_EOF'
BEGIN {
  inserted = 0
  in_icon_imports = 0
}
# Print interface imports as-is
/^import \{/ || /^import ISoftware/ {
  print
  next
}
# Detect start of icon imports (after interface imports)
# Icon imports are component imports that use "./" path (not interface imports)
/^import [A-Za-z]/ && !in_icon_imports && /from "\.\// {
  in_icon_imports = 1
  # Check if we should insert before this first import
  # Use field-based parsing to avoid "from" being a reserved word
  if ($1 == "import" && $3 == "from") {
    current_component = $2
    new_lower = tolower(new_component)
    current_lower = tolower(current_component)
    if (new_lower < current_lower) {
      print new_import
      inserted = 1
    }
  }
}
# If we're in icon imports section and haven't inserted yet
in_icon_imports && !inserted {
  # Extract component name from current import line using field-based parsing
  if ($1 == "import" && $3 == "from") {
    current_component = $2
    # Compare alphabetically (case-insensitive)
    new_lower = tolower(new_component)
    current_lower = tolower(current_component)
    if (new_lower < current_lower) {
      print new_import
      inserted = 1
    }
  }
}
# Stop processing imports when we hit the SOFTWARE_NAME_TO_ICON_MAP comment
/^\/\/ SOFTWARE_NAME_TO_ICON_MAP/ {
  if (!inserted) {
    print new_import
    inserted = 1
  }
  in_icon_imports = 0
}
{ print }
AWK_EOF
    
    # Use awk with script file and pipe input
    if ! cat "$index_file" | awk -v new_import="$import_line" -v new_component="$component_name" -f "$awk_script_file" > "${index_file}.tmp" 2>&1; then
      local awk_error=$(cat "${index_file}.tmp" 2>&1)
      rm -f "${index_file}.tmp" "$awk_script_file"
      echo "Error: Failed to add import for ${component_name}" >&2
      echo "Awk error: $awk_error" >&2
      echo "Index file path: $index_file" >&2
      echo "Current directory: $(pwd)" >&2
      return 1
    fi
    
    rm -f "$awk_script_file"
    
    if [ -f "${index_file}.tmp" ]; then
      mv "${index_file}.tmp" "$index_file"
      echo "Added import for ${component_name} to index.ts (in alphabetical order)"
    else
      echo "Error: awk succeeded but temp file was not created" >&2
      return 1
    fi
  fi
  
  # Add to SOFTWARE_NAME_TO_ICON_MAP if not already present
  # Create map key from app name (lowercase)
  local map_key=$(echo "$app_name" | tr '[:upper:]' '[:lower:]' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
  
  # Check if map entry already exists (check for both quoted and unquoted keys)
  if grep -qE "[\"']${map_key}[\"']:" "$index_file" || grep -qE "^[[:space:]]*${map_key}:" "$index_file"; then
    echo "Map entry for ${map_key} already exists in index.ts"
  else
    # Determine if we need to quote the key (only quote if it's not a valid JS identifier)
    local quoted_key
    if is_valid_js_identifier "$map_key"; then
      quoted_key="$map_key"
    else
      quoted_key="\"${map_key}\""
    fi
    
    # Insert map entry in alphabetical order
    local map_entry="  ${quoted_key}: ${component_name},"
    
    # Create temporary awk script file to avoid heredoc issues
    local awk_script_file=$(mktemp)
    cat > "$awk_script_file" << 'AWK_EOF'
BEGIN {
  inserted = 0
  in_map = 0
  first_map_entry = 1
}
/^export const SOFTWARE_NAME_TO_ICON_MAP = \{/ {
  in_map = 1
  print
  next
}
# If we're in the map and haven't inserted yet
in_map && !inserted {
  # Check if this is a map entry (has a colon)
  if (match($0, /:/)) {
    # Extract everything before the colon
    line_before_colon = substr($0, 1, RSTART - 1)
    # Remove leading/trailing whitespace
    gsub(/^[[:space:]]+|[[:space:]]+$/, "", line_before_colon)
    # Remove quotes if present
    if (substr(line_before_colon, 1, 1) == "\"" && substr(line_before_colon, length(line_before_colon), 1) == "\"") {
      current_key = substr(line_before_colon, 2, length(line_before_colon) - 2)
    } else {
      current_key = line_before_colon
    }
    # Compare alphabetically (case-insensitive)
    new_lower = tolower(new_key)
    current_lower = tolower(current_key)
    if (new_lower < current_lower) {
      print new_entry
      inserted = 1
      first_map_entry = 0
    }
    first_map_entry = 0
  } else if (first_map_entry && /^[[:space:]]*$/) {
    # Skip blank lines at the start of the map
    next
  }
}
# Stop at closing brace
in_map && /^\} as const;/ {
  if (!inserted) {
    print new_entry
    inserted = 1
  }
  print
  in_map = 0
  next
}
{ print }
AWK_EOF
    
    # Use awk with script file and pipe input
    if ! cat "$index_file" | awk -v new_entry="$map_entry" -v new_key="$map_key" -f "$awk_script_file" > "${index_file}.tmp" 2>&1; then
      local awk_error=$(cat "${index_file}.tmp" 2>&1)
      rm -f "${index_file}.tmp" "$awk_script_file"
      echo "Error: Failed to add map entry for ${map_key}" >&2
      echo "Awk error: $awk_error" >&2
      echo "Index file path: $index_file" >&2
      echo "Current directory: $(pwd)" >&2
      return 1
    fi
    
    rm -f "$awk_script_file"
    
    if [ -f "${index_file}.tmp" ]; then
      mv "${index_file}.tmp" "$index_file"
      echo "Added map entry ${quoted_key}: ${component_name} to SOFTWARE_NAME_TO_ICON_MAP (in alphabetical order)"
    else
      echo "Error: awk succeeded but temp file was not created" >&2
      return 1
    fi
  fi
}

# Change to repository root directory (where this script is located)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
cd "$REPO_ROOT" || {
  echo "Error: Failed to change to repository root directory" >&2
  exit 1
}

# Defaults
SLUG=""
APP_PATH=""
PNG_PATH=""

# Parse required arguments
while getopts ":s:a:i:" opt; do
  case "$opt" in
    s) SLUG=$(echo "$OPTARG" | cut -d'/' -f1) ;;
    a) APP_PATH="$OPTARG" ;;
    i) PNG_PATH="$OPTARG" ;;
    \?) echo "Invalid option: -$OPTARG" >&2; exit 1 ;;
    :) echo "Option -$OPTARG requires an argument." >&2; exit 1 ;;
  esac
done

if ! command -v svgr &> /dev/null; then
  echo "Error: 'svgr' is not installed. Install it with 'npm install -g @svgr/cli'"
  exit 1
fi

# Validate
if [[ -z "$SLUG" ]]; then
  echo "Usage: $0 -s slug-name [-a /path/to/App.app | -i /path/to/icon.png]"
  echo "  -s: Slug name (required)"
  echo "  -a: Path to .app bundle (required if -i not provided)"
  echo "  -i: Path to PNG icon file (required if -a not provided)"
  exit 1
fi

if [[ -z "$APP_PATH" && -z "$PNG_PATH" ]]; then
  echo "Error: Either -a (app path) or -i (PNG path) must be provided"
  echo "Usage: $0 -s slug-name [-a /path/to/App.app | -i /path/to/icon.png]"
  exit 1
fi

if [[ -n "$APP_PATH" && -n "$PNG_PATH" ]]; then
  echo "Error: Cannot specify both -a and -i options"
  exit 1
fi

TMP_DIR=$(mktemp -d)

# Handle PNG file input directly
if [[ -n "$PNG_PATH" ]]; then
  if [[ ! -f "$PNG_PATH" ]]; then
    echo "Error: PNG file not found at $PNG_PATH"
    exit 1
  fi
  
  # Verify it's a PNG file
  if ! file "$PNG_PATH" | grep -q "PNG"; then
    echo "Error: File does not appear to be a PNG image: $PNG_PATH"
    exit 1
  fi
  
  echo "Using PNG file directly: $PNG_PATH"
  ICON_128="$PNG_PATH"
  
  # Derive component name from slug
  # Convert slug to PascalCase (e.g., "company-portal" -> "CompanyPortal")
  # Split by hyphens/underscores, capitalize first letter of each word, join
  COMPONENT_NAME=$(echo "$SLUG" | awk -F'[_-]' '{
    result = ""
    for (i=1; i<=NF; i++) {
      word = $i
      if (length(word) > 0) {
        first = toupper(substr(word, 1, 1))
        rest = substr(word, 2)
        result = result first rest
      }
    }
    print result
  }')
  
  # Derive app display name from slug (convert hyphens/underscores to spaces and capitalize)
  APP_DISPLAY_NAME=$(echo "$SLUG" | awk -F'[_-]' '{
    result = ""
    for (i=1; i<=NF; i++) {
      if (i > 1) result = result " "
      word = $i
      if (length(word) > 0) {
        first = toupper(substr(word, 1, 1))
        rest = substr(word, 2)
        result = result first rest
      }
    }
    print result
  }')
else
  # Read Info.plist to get the icon file name
  INFO_PLIST="$APP_PATH/Contents/Info.plist"
  if [[ ! -f "$INFO_PLIST" ]]; then
    echo "Error: Info.plist not found at $INFO_PLIST"
    exit 1
  fi

  # Extract CFBundleIconFile from Info.plist
  ICON_FILE=$(defaults read "$INFO_PLIST" CFBundleIconFile 2>/dev/null || plutil -extract CFBundleIconFile raw "$INFO_PLIST" 2>/dev/null || echo "")

  # If CFBundleIconFile not found, try modern approach with CFBundleIconName (Asset Catalog)
  if [[ -z "$ICON_FILE" ]]; then
    ICON_NAME=$(defaults read "$INFO_PLIST" CFBundleIconName 2>/dev/null || plutil -extract CFBundleIconName raw "$INFO_PLIST" 2>/dev/null || echo "")

    if [[ -n "$ICON_NAME" ]]; then
      echo "CFBundleIconFile not found, but CFBundleIconName found: $ICON_NAME"
      echo "Extracting icon from Asset Catalog using macOS Workspace API..."

      # Create temporary directory for extraction
      TMP_EXTRACT_DIR=$(mktemp -d)
      EXTRACTED_ICON="$TMP_EXTRACT_DIR/app-icon.png"

      # Use osascript to extract icon via NSWorkspace
      osascript <<EOF
use framework "Foundation"
use framework "AppKit"

set appPath to "$APP_PATH"
set outputPath to "$EXTRACTED_ICON"

set workspace to current application's NSWorkspace's sharedWorkspace()
set appIcon to workspace's iconForFile:appPath

set imageData to appIcon's TIFFRepresentation()
set imageRep to (current application's NSBitmapImageRep's imageRepWithData:imageData)
set pngData to (imageRep's representationUsingType:(current application's NSPNGFileType) |properties|:(missing value))

pngData's writeToFile:outputPath atomically:true
EOF

      if [[ ! -f "$EXTRACTED_ICON" ]]; then
        echo "Error: Failed to extract icon from Asset Catalog"
        exit 1
      fi

      # Create a temporary icns file from the extracted PNG for compatibility with rest of script
      TMP_ICNS="$TMP_EXTRACT_DIR/app-icon.icns"

      # Create iconset directory
      ICONSET_DIR="$TMP_EXTRACT_DIR/AppIcon.iconset"
      mkdir -p "$ICONSET_DIR"

      # Generate multiple sizes for iconset
      sips -z 16 16 "$EXTRACTED_ICON" --out "$ICONSET_DIR/icon_16x16.png" > /dev/null 2>&1
      sips -z 32 32 "$EXTRACTED_ICON" --out "$ICONSET_DIR/icon_16x16@2x.png" > /dev/null 2>&1
      sips -z 32 32 "$EXTRACTED_ICON" --out "$ICONSET_DIR/icon_32x32.png" > /dev/null 2>&1
      sips -z 64 64 "$EXTRACTED_ICON" --out "$ICONSET_DIR/icon_32x32@2x.png" > /dev/null 2>&1
      sips -z 128 128 "$EXTRACTED_ICON" --out "$ICONSET_DIR/icon_128x128.png" > /dev/null 2>&1
      sips -z 256 256 "$EXTRACTED_ICON" --out "$ICONSET_DIR/icon_128x128@2x.png" > /dev/null 2>&1
      sips -z 256 256 "$EXTRACTED_ICON" --out "$ICONSET_DIR/icon_256x256.png" > /dev/null 2>&1
      sips -z 512 512 "$EXTRACTED_ICON" --out "$ICONSET_DIR/icon_256x256@2x.png" > /dev/null 2>&1
      sips -z 512 512 "$EXTRACTED_ICON" --out "$ICONSET_DIR/icon_512x512.png" > /dev/null 2>&1
      sips -z 1024 1024 "$EXTRACTED_ICON" --out "$ICONSET_DIR/icon_512x512@2x.png" > /dev/null 2>&1

      # Create icns from iconset
      iconutil -c icns "$ICONSET_DIR" -o "$TMP_ICNS"

      if [[ ! -f "$TMP_ICNS" ]]; then
        echo "Error: Failed to create icns file from extracted icon"
        exit 1
      fi

      ICNS_PATH="$TMP_ICNS"
      echo "Successfully extracted icon from Asset Catalog"
    else
      echo "Error: Neither CFBundleIconFile nor CFBundleIconName found in Info.plist"
      exit 1
    fi
  else
    # Handle case where extension might or might not be included
    if [[ "$ICON_FILE" != *.icns ]]; then
      ICON_FILE="${ICON_FILE}.icns"
    fi

    # Find the exact icon file in Resources folder
    ICNS_PATH="$APP_PATH/Contents/Resources/$ICON_FILE"
    if [[ ! -f "$ICNS_PATH" ]]; then
      echo "Error: Icon file '$ICON_FILE' not found in $APP_PATH/Contents/Resources"
      exit 1
    fi

    echo "Using icon file: $ICON_FILE"
  fi

  # Extract iconset
  ICON_DIR="$TMP_DIR/icons.iconset"
  mkdir -p "$ICON_DIR"
  iconutil -c iconset "$ICNS_PATH" -o "$ICON_DIR"

  # Debug: list all PNG files in iconset
  echo "Available icon files in iconset:"
  find "$ICON_DIR" -name "*.png" | sort || true

  # Find the 128x128 PNG (prefer exact 128x128, not @2x)
  ICON_128=$(find "$ICON_DIR" -name "*128x128.png" ! -name "*@2x*" | head -n 1)
  if [[ -z "$ICON_128" ]]; then
    # Fallback: try 128x128@2x (which is 256x256)
    ICON_128=$(find "$ICON_DIR" -name "*128x128@2x.png" | head -n 1)
  fi
  if [[ -z "$ICON_128" ]]; then
    # Fallback: try any icon with 128 in the name
    ICON_128=$(find "$ICON_DIR" -name "*128*.png" | head -n 1)
  fi
  if [[ -z "$ICON_128" ]]; then
    # Last resort: find the largest PNG (prefer larger sizes)
    ICON_128=$(find "$ICON_DIR" -name "*.png" | sort -V | tail -n 1)
  fi

  if [[ -z "$ICON_128" ]]; then
    echo "Error: No icon PNG files found in extracted iconset."
    echo "ICON_DIR: $ICON_DIR"
    exit 1
  fi

  echo "Using icon for SVG and PNG: $ICON_128"
  
  # Determine component and file names from app bundle
  APP_NAME=$(basename "$APP_PATH" .app)
  COMPONENT_NAME="${APP_NAME//[^a-zA-Z0-9]/}"
  
  # Extract app name from Info.plist for map key
  APP_DISPLAY_NAME=$(defaults read "$INFO_PLIST" CFBundleName 2>/dev/null || \
                     plutil -extract CFBundleName raw "$INFO_PLIST" 2>/dev/null || \
                     defaults read "$INFO_PLIST" CFBundleDisplayName 2>/dev/null || \
                     plutil -extract CFBundleDisplayName raw "$INFO_PLIST" 2>/dev/null || \
                     echo "$APP_NAME")
fi

# Check dimensions and resize to 128x128 if larger
ICON_WIDTH=$(sips -g pixelWidth "$ICON_128" | grep pixelWidth | awk '{print $2}')
ICON_HEIGHT=$(sips -g pixelHeight "$ICON_128" | grep pixelHeight | awk '{print $2}')
echo "Icon dimensions: ${ICON_WIDTH}x${ICON_HEIGHT}"

if [[ $ICON_WIDTH -gt 128 || $ICON_HEIGHT -gt 128 ]]; then
  echo "Resizing icon from ${ICON_WIDTH}x${ICON_HEIGHT} to 128x128..."
  RESIZED_ICON="$TMP_DIR/icon_128x128.png"
  sips -z 128 128 "$ICON_128" --out "$RESIZED_ICON" > /dev/null 2>&1
  if [[ -f "$RESIZED_ICON" ]]; then
    ICON_128="$RESIZED_ICON"
    echo "Resized icon saved to: $ICON_128"
  else
    echo "Warning: Failed to resize icon, using original"
  fi
fi

# Generate SVG from 128 PNG
WIDTH=32
HEIGHT=32
BASE64_DATA=$(base64 -i "$ICON_128" | tr -d '\n')

# Determine SVG filename based on input method
if [[ -n "$PNG_PATH" ]]; then
  SVG_BASENAME="$SLUG"
else
  SVG_BASENAME=$(basename "$APP_PATH" .app)
fi
OUTPUT_SVG="$TMP_DIR/${SVG_BASENAME}.svg"

cat > "$OUTPUT_SVG" <<EOF
<svg xmlns="http://www.w3.org/2000/svg" width="$WIDTH" height="$HEIGHT" viewBox="0 0 $WIDTH $HEIGHT" version="1.1">
  <image width="$WIDTH" height="$HEIGHT" href="data:image/png;base64,$BASE64_DATA"/>
</svg>
EOF

echo "SVG saved to: $OUTPUT_SVG"

## Generate TSX component
# TODO: we could just have a template file that we populated with the base64 PNG and the component name,
# rather than needing a separate tool. Particularly since we have to do tweaks on the output of the
# svgr tool to make the result pass linting.
svgr "$OUTPUT_SVG" --typescript --ext tsx --out-dir frontend/pages/SoftwarePage/components/icons/

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

# Update index.ts with import and map entry
echo "Updating index.ts with new icon..."
add_icon_to_index "$NEW_COMPONENT_NAME" "$APP_DISPLAY_NAME"

######################################
# Copy 128x128 PNG to asset location #
######################################

OUTPUT_IMAGE_DIR="website/assets/images"
OUTPUT_PNG="$OUTPUT_IMAGE_DIR/app-icon-${SLUG}-60x60@2x.png"
mkdir -p "$OUTPUT_IMAGE_DIR"
cp "$ICON_128" "$OUTPUT_PNG"

echo "Copied 128x128 PNG to: $OUTPUT_PNG"
