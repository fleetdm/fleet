# YAML Key Processing Script

A utility script for Fleet that moves specific configuration keys from software YAML files to team YAML files.

## Overview

This script automates the migration of software configuration keys from individual software packages to team-level configurations. It processes YAML files in the `it-and-security/teams/` directory and moves the following keys from referenced software files to the team files:

- `self_service`
- `categories` 
- `labels_include_any`
- `labels_exclude_any`

## Prerequisites

### Required Dependencies

- **yq** (version 4 or higher)
  ```bash
  # Install on macOS
  brew install yq
  
  # Install on Ubuntu/Debian
  sudo apt install yq
  
  # Install on other systems - see https://github.com/mikefarah/yq
  ```

### Directory Structure

The script must be run from the Fleet repository root directory. It expects:

```
fleet/
├── it-and-security/
│   └── teams/
│       ├── team1.yml
│       ├── team2.yml
│       └── ...
└── tools/
    └── yaml-key-processor/
        ├── process_yaml_keys.sh
        └── README.md
```

## Usage

### Basic Usage

```bash
# From the fleet repository root
./tools/yaml-key-processor/process_yaml_keys.sh
```

The script will:
1. Automatically discover all `.yml` files in the `it-and-security/teams/` directory
2. For each team file, process all packages listed in `software.packages[]`
3. Extract the target keys from each referenced software file
4. Move those keys to the corresponding package entry in the team file
5. Remove the keys from the original software files
6. Create `.bak` backup files for all modified files

### What the Script Does

#### Before Migration
**Team File (`it-and-security/teams/example.yml`):**
```yaml
apiVersion: v1
kind: team
spec:
  name: Example Team
  software:
    packages:
      - path: ../lib/macos/software/firefox.yml
```

**Software File (`it-and-security/lib/macos/software/firefox.yml`):**
```yaml
name: Mozilla Firefox
url: https://download.mozilla.org/...
self_service: true
categories:
  - "Web Browser"
labels_include_any:
  - "Department:Engineering"
labels_exclude_any:
  - "OS:Windows"
```

#### After Migration
**Team File (`it-and-security/teams/example.yml`):**
```yaml
apiVersion: v1
kind: team
spec:
  name: Example Team
  software:
    packages:
      - path: ../lib/macos/software/firefox.yml
        self_service: true
        categories:
          - "Web Browser"
        labels_include_any:
          - "Department:Engineering"
        labels_exclude_any:
          - "OS:Windows"
```

**Software File (`it-and-security/lib/macos/software/firefox.yml`):**
```yaml
name: Mozilla Firefox
url: https://download.mozilla.org/...
```

## Features

- **Automatic Discovery**: Finds all team YAML files automatically
- **Backup Creation**: Creates `.bak` files before making any changes
- **YAML Validation**: Validates syntax before and after processing
- **Error Handling**: Graceful error handling with detailed reporting
- **Path Resolution**: Handles relative paths correctly
- **Colorized Output**: Easy-to-read colored terminal output

## Output

The script provides detailed, colorized output showing:
- Files being processed
- Keys being moved
- Success/error status for each operation
- Final summary with counts of processed teams and packages

Example output:
```
YAML Key Processing Script
Moving keys from software files to team files

✓ yq version 4.35.2 found
Finding team files...
Found 3 team files

Processing team file: it-and-security/teams/workstations.yml
  Found 2 packages
  Processing package 1/2
    Package path: ../lib/macos/software/mozilla-firefox.yml
    Processing: it-and-security/lib/macos/software/mozilla-firefox.yml
    Created backup: it-and-security/lib/macos/software/mozilla-firefox.yml.bak
    Adding keys to team file at package index 0
    Added self_service
    Added categories
    ✓ Package processed successfully

=== PROCESSING COMPLETE ===
Teams processed: 3
Packages processed: 8
✓ All files processed successfully!

Backup files created with .bak extension
To restore from backups if needed: find . -name '*.bak' -exec bash -c 'mv "$1" "${1%.bak}"' _ {} \;
```

## Error Recovery

If something goes wrong during processing:

1. **Individual File Errors**: The script continues processing other files
2. **YAML Validation Failures**: Automatically restores from backup
3. **Manual Recovery**: Use the restore command shown in the output:
   ```bash
   find . -name '*.bak' -exec bash -c 'mv "$1" "${1%.bak}"' _ {} \;
   ```

## Limitations

- Only processes `.yml` files (not `.yaml`)
- Requires team files to have `software.packages[]` structure
- Software file paths must be relative to the team file location
- Requires yq v4+ for advanced YAML manipulation

## Troubleshooting

### Common Issues

1. **"yq is required but not installed"**
   - Install yq using the instructions in Prerequisites

2. **"yq version 4 or higher is required"**
   - Upgrade yq: `brew upgrade yq`

3. **"it-and-security/teams directory not found"**
   - Run the script from the Fleet repository root directory

4. **"Software file not found"**
   - Check that the `path` in the team file is correct relative to the team file location

### Debug Mode

For troubleshooting, you can add debug output by modifying the script temporarily:
```bash
# Add this after the shebang line
set -euxo pipefail  # Adds debug output
```

## Contributing

When modifying this script:
1. Test on a small subset of files first
2. Ensure shellcheck passes: `shellcheck process_yaml_keys.sh`
3. Verify YAML syntax validation works correctly
4. Test error recovery scenarios