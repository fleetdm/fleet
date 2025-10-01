# GitOps Migration Tool

A utility tool for Fleet that migrates specific configuration keys from software YAML files to team YAML files.

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
    └── gitops-migrate/
        ├── migrate.sh
        └── README.md
```

## Usage

### Basic Usage

```bash
# From the fleet repository root
./tools/gitops-migrate/migrate.sh <teams_directory_path>

# Example:
./tools/gitops-migrate/migrate.sh it-and-security/teams
```

The script will:
1. Automatically discover all `.yml` files in the specified teams directory
2. For each team file, process all packages listed in `software.packages[]`
3. Extract the target keys from each referenced software file (Pass 1)
4. Move those keys to the corresponding package entry in the team file (Pass 1)
5. Remove the keys from the original software files after all teams are processed (Pass 2)

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
GitOps Migration Tool
Moving keys from software files to team files
Teams directory: it-and-security/teams

Finding team files...
Found 3 team files

=== PASS 1: UPDATING TEAM FILES ===
Processing team file: it-and-security/teams/workstations.yml
  Found 2 packages
  Processing package 1/2
    Package path: ../lib/macos/software/mozilla-firefox.yml
    Processing: it-and-security/lib/macos/software/mozilla-firefox.yml
    Adding keys to team file at package index 0
    Added self_service
    Added categories
    ✓ Package processed successfully

=== PASS 2: CLEANING UP SOFTWARE FILES ===
Removing keys from 15 unique software files
  Removing keys from: mozilla-firefox.yml
✓ Software file cleanup complete

=== PROCESSING COMPLETE ===
Teams processed: 3
Packages processed: 8
✓ All files processed successfully!
```

## Two-Pass Processing

The tool uses a two-pass approach to handle multiple teams referencing the same software files:

1. **Pass 1**: Extract and add keys to ALL team files (without removing keys from software files)
2. **Pass 2**: Remove keys from software files only after all teams have been processed

This ensures that all teams receive the appropriate keys, even when multiple teams reference the same software file.

## Error Recovery

If something goes wrong during processing:

1. **Individual File Errors**: The script continues processing other files
2. **YAML Validation Failures**: Reports errors but continues with other files
3. **Git Recovery**: Use git to restore files if needed:
   ```bash
   git checkout -- it-and-security/
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

3. **"Teams directory not found"**
   - Verify the directory path argument is correct
   - Ensure you're running from the correct location

4. **"Software file not found"**
   - Check that the `path` in the team file is correct relative to the team file location

### Debug Mode

For troubleshooting, you can add debug output by modifying the script temporarily:
```bash
# Add this after the shebang line
set -euxo pipefail  # Adds debug output
```

## Contributing

When modifying this tool:
1. Test on a small subset of files first
2. Ensure shellcheck passes: `shellcheck migrate.sh`
3. Verify YAML syntax validation works correctly
4. Test the two-pass processing logic thoroughly