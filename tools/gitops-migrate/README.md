# GitOps migration tool

Fleet 4.74.0 includes [breaking changes](https://github.com/fleetdm/fleet/pull/30837/files#r2205252594) to the [experimental](https://fleetdm.com/handbook/company/product-groups#experimental-features) software YAML files. This tool automatically migrates your YAML to the new YAML format Fleet 4.74.0 expects.

## Overview

This script automates the migration of software configuration keys from individual software packages to team-level configurations. It processes YAML files in the `it-and-security/teams/` directory and moves the following keys from referenced software files to the team files:

- `self_service`
- `categories` 
- `labels_include_any`
- `labels_exclude_any`

## Prerequisites

**yq** is required (version 4 or higher)

```bash
# Install on macOS
brew install yq

# Install on Ubuntu/Debian
# yq installed from apt is NOT supported
sudo snap install yq

# Install on other systems - see https://github.com/mikefarah/yq
```

## Usage

### Basic usage

```bash
./tools/gitops-migrate/migrate.sh <teams_directory_path>
```

The script will:
1. Automatically discover all `.yml` files in the specified teams directory
2. For each team file, process all packages listed in `software.packages[]`
3. Extract the target keys from each referenced software file (Pass 1)
4. Move those keys to the corresponding package entry in the team file (Pass 1)
5. Remove the keys from the original software files after all teams are processed (Pass 2)

### What the script does

#### Before migration

**Team file (`it-and-security/teams/example.yml`):**
```yaml
name: Example Team
software:
  packages:
    - path: ../lib/macos/software/firefox.yml
```

**Software file (`it-and-security/lib/macos/software/firefox.yml`):**

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

#### After migration

**Team file (`it-and-security/teams/example.yml`):**
```yaml
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

**Software file (`it-and-security/lib/macos/software/firefox.yml`):**

```yaml
name: Mozilla Firefox
url: https://download.mozilla.org/...
```

## Features

- **Automatic discovery**: Finds all team YAML files automatically
- **Backup creation**: Creates `.bak` files before making any changes
- **YAML validation**: Validates syntax before and after processing
- **Error handling**: Graceful error handling with detailed reporting
- **Path resolution**: Handles relative paths correctly
- **Colorized output**: Easy-to-read colored terminal output

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

## Two-pass processing

The tool uses a two-pass approach to handle multiple teams referencing the same software files:

1. **Pass 1**: Extract and add keys to ALL team files (without removing keys from software files)
2. **Pass 2**: Remove keys from software files only after all teams have been processed

This ensures that all teams receive the appropriate keys, even when multiple teams reference the same software file.

## Error recovery

If something goes wrong during processing:

1. **Individual file errors**: The script continues processing other files
2. **YAML validation failures**: Reports errors but continues with other files
3. **Git recovery**: Use git to restore files if needed:
   ```bash
   git checkout -- it-and-security/
   ```

## Limitations

- Only processes `.yml` files (not `.yaml`)
- Requires team files to have `software.packages[]` structure
- Software file paths must be relative to the team file location
- Requires yq v4+ for advanced YAML manipulation

## Troubleshooting

### Common issues

1. **"yq is required but not installed"**
   - Install yq using the instructions in Prerequisites

2. **"yq version 4 or higher is required"**
   - Upgrade yq: `brew upgrade yq`

3. **"Teams directory not found"**
   - Verify the directory path argument is correct
   - Ensure you're running from the correct location

4. **"Software file not found"**
   - Check that the `path` in the team file is correct relative to the team file location

### Debug mode

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
