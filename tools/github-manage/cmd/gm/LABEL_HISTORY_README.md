# Label History Command

Find issue numbers for all issues in a GitHub repository that had a specific label applied to them at any point in their history, filtered by creation date.

## Overview

The `label-history` command searches through issues in a repository and identifies which ones have ever had a specific label, including labels that may have been removed. This is useful for:

- Analyzing historical labeling patterns
- Finding all issues that were ever marked with a certain label (e.g., "bug", "critical")
- Tracking issue categorization changes over time
- Generating reports based on label history

## Features

- **Historical label tracking**: Checks both current labels and historical timeline events
- **Date filtering**: Only searches issues created on or after a specified date
- **Private repository support**: Works with private repositories via GitHub CLI authentication
- **Multiple output formats**: Text format for human reading, JSON format for programmatic use
- **Efficient search**: Uses GitHub's search API to find issues, then checks timeline events for historical labels

## Prerequisites

- GitHub CLI (`gh`) installed and authenticated
- For private repositories, ensure your GitHub token has appropriate repository access

```bash
gh auth login
gh auth refresh -s repo  # Ensure repo access for private repositories
```

## Usage

```bash
gm label-history <repo> --start-date <date> --label <label> [--json]
```

### Arguments

- `repo`: Repository in the format `owner/repo` (required)

### Flags

- `--start-date`: Start date in YYYY-MM-DD format (issues created on or after this date) [required]
- `--label`: Label name to search for in issue history [required]
- `--json`: Output results in JSON format for programmatic use [optional]

### Examples

**Find all issues that were ever labeled as "bug" since 2024-01-01:**
```bash
gm label-history fleetdm/fleet --start-date 2024-01-01 --label "bug"
```

**Find issues that had the "critical" label since a specific date, output as JSON:**
```bash
gm label-history myorg/myrepo --start-date 2024-06-01 --label "critical" --json
```

**Find issues that were ever labeled "good first issue" for a private repository:**
```bash
gm label-history myorg/private-repo --start-date 2023-01-01 --label "~good first issue"
```

**Find issues that had a label with special characters:**
```bash
gm label-history fleetdm/fleet --start-date 2024-01-01 --label ":product"
```

## Output

### Text Format (Default)

The text format provides a human-readable list:

```
Found 7 issue(s):

- #38809
- #38816
- #38833
- #38844
- #38882
- #38884
- #38898

Full list: 38809, 38816, 38833, 38844, 38882, 38884, 38898
```

### JSON Format

The JSON format provides structured data for programmatic use:

```json
{
  "repository": "fleetdm/fleet",
  "start_date": "2024-06-01",
  "label": "bug",
  "count": 7,
  "issues": [
    38809,
    38816,
    38833,
    38844,
    38882,
    38884,
    38898
  ]
}
```

## How It Works

1. **Search for Issues**: Uses GitHub's search API to find all issues in the repository created on or after the specified start date
2. **Check Current Labels**: Quickly checks if the issue currently has the target label
3. **Check Timeline History**: For issues that don't currently have the label, fetches the issue's timeline events to see if the label was ever applied in the past
4. **Return Results**: Collects and returns all issue numbers that have ever had the label, sorted numerically

## Performance Considerations

- The command makes one API call to search for issues, then makes additional API calls to fetch timeline events for issues that don't currently have the label
- For large repositories, this may take some time depending on the number of issues
- The command includes logging that can help track progress (check `dgm.log` in the same directory)

## Limitations

- GitHub API rate limits apply: approximately 5,000 requests per hour for authenticated users
- Pagination is automatically handled by the GitHub CLI
- The issue timeline API has some limitations on very old events (prior to late 2016)

## Troubleshooting

**"failed to fetch issues" error:**
- Ensure your GitHub CLI is authenticated: `gh auth status`
- For private repositories, ensure you have appropriate permissions

**Command runs slowly:**
- This is expected for repositories with many issues, as timeline events are fetched individually
- To reduce the number of issues checked, use a more recent start date

**No issues found:**
- Verify the label name exactly matches the label in the repository (including special characters like `:` or `~`)
- Try a broader date range
- Check that issues exist in the repository for the specified date range

## Building from Source

```bash
cd tools/github-manage
go build -o gm cmd/gm/*.go
```

## License

This tool is part of the Fleet repository and follows the same licensing terms.