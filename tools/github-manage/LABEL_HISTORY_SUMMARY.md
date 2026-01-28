# Label History Tool Summary

## Overview

I've successfully built a command-line tool that finds GitHub issues that have ever had a specific label applied to them, including labels that may have been removed. This tool integrates with the existing `gm` (GitHub Management) tool in the Fleet repository.

## What Was Created

### 1. Core Implementation Files

**`fleet/tools/github-manage/cmd/gm/label_history.go`**
- New command implementation that searches for issues with a historical label
- Supports both text and JSON output formats
- Efficiently checks current labels first, then timeline events for historical data
- Includes comprehensive logging and error handling

**`fleet/tools/github-manage/pkg/ghapi/issues.go`** (Modified)
- Added `TimelineEvent` struct to represent issue timeline events
- Added `GetIssueTimelineEvents()` function to fetch label history for issues
- Integrates with existing GitHub CLI infrastructure

**`fleet/tools/github-manage/cmd/gm/main.go`** (Modified)
- Registered `labelHistoryCmd` as a new subcommand

### 2. Documentation Files

**`LABEL_HISTORY_README.md`**
- Comprehensive user documentation
- Usage examples and troubleshooting guide
- Performance considerations and limitations

**`demo-label-history.sh`**
- Executable demo script showcasing various use cases
- Illustrates different output formats and label types

**`LABEL_HISTORY_SUMMARY.md`** (This file)
- Project summary and technical overview

## Key Features

✅ **Historical Label Tracking**
- Checks both current labels and full timeline history
- Handles labeled/unlabeled events from GitHub's API
- Finds issues even if the label was removed

✅ **Date Filtering**
- Only searches issues created on or after specified date (YYYY-MM-DD)
- Reduces search scope for better performance

✅ **Private Repository Support**
- Uses GitHub CLI authentication
- Works seamlessly with private repos
- Access tokens managed via `gh auth login`

✅ **Flexible Output**
- Human-readable text format by default
- JSON format for programmatic use
- Sorted issue numbers

✅ **Robust Error Handling**
- Graceful handling of API errors
- Detailed logging to `dgm.log`
- Informative error messages

## Usage

### Basic Command

```bash
gm label-history <owner/repo> --start-date <YYYY-MM-DD> --label <label-name>
```

### Examples

```bash
# Find all issues ever labeled as "bug" since 2024-06-01
gm label-history fleetdm/fleet --start-date 2024-06-01 --label "bug"

# Output as JSON for programmatic use
gm label-history fleetdm/fleet --start-date 2024-06-01 --label "bug" --json

# Find issues with special character labels
gm label-history fleetdm/fleet --start-date 2024-01-01 --label ":product"
gm label-history fleetdm/fleet --start-date 2024-01-01 --label "#g-software"

# Private repository
gm label-history myorg/private-repo --start-date 2023-01-01 --label "critical"
```

### Output Examples

**Text Format:**
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

**JSON Format:**
```json
{
  "repository": "fleetdm/fleet",
  "start_date": "2024-06-01",
  "label": "bug",
  "count": 7,
  "issues": [38809, 38816, 38833, 38844, 38882, 38884, 38898]
}
```

## Technical Implementation

### Algorithm

1. **Search Phase**: Use GitHub search API with `repo:owner/repo is:issue created:>=$START_DATE`
2. **Quick Check**: For each issue, check current labels first (no API call needed)
3. **Deep Check**: For issues without current label, fetch timeline events
4. **Historical Scan**: Look for `event: "labeled"` with matching label name in timeline
5. **Collect Results**: Aggregate all matching issue numbers
6. **Sort and Output**: Return results sorted numerically

### API Calls

- Search API: `gh issue list --json` - fetches initial issue list
- Timeline API: `gh api repos/$REPO/issues/$NUMBER/timeline` - per issue for historical check

### Performance Considerations

- O(n) search complexity where n = number of issues in date range
- Timeline API calls only for issues without current label
- GitHub CLI handles pagination automatically
- Rate limits apply (5,000 requests/hour for authenticated users)

## Dependencies

- GitHub CLI (`gh`) with authentication
- Go 1.25.5+
- Existing Fleet `gm` tool infrastructure (Cobra, logger, etc.)

## Building

```bash
cd fleet/tools/github-manage
go build -o gm cmd/gm/*.go
# or
make
```

## Testing

The tool has been tested with:
- Public repository (fleetdm/fleet)
- Real labels including those with special characters
- Both text and JSON output formats
- Date filtering with various date ranges

Test results showed:
- ✅ Successfully identifies currently labeled issues
- ✅ Successfully finds historically labeled issues
- ✅ Correctly handles special characters in label names
- ✅ JSON output is valid and machine-readable
- ✅ Logging provides useful debugging information

## Future Enhancements (Optional)

Potential improvements that could be added:
1. **Pagination Control**: Add `--limit` flag to cap number of results
2. **Multiple Labels**: Support searching with `--label` flag multiple times (OR logic)
3. **Output Formats**: Add CSV or markdown table formats
4. **Detailed Output**: Option to include issue titles and URLs
5. **Exclude Current**: Option to only find issues where label was removed
6. **Date Range**: Support both start and end dates
7. **Batch Processing**: Process multiple labels in one command
8. **Parallel Processing**: Fetch timeline events concurrently for better performance

## Conclusion

The label-history tool successfully addresses the requirement to find GitHub issues that had a specific label at any point in their history. It integrates cleanly with the existing `gm` tool infrastructure, supports private repositories, provides flexible output options, and handles edge cases gracefully. The implementation is straightforward, well-documented, and ready for production use.