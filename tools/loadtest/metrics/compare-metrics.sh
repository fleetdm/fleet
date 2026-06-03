#!/usr/bin/env bash
#
# compare-metrics.sh - Compare metrics across Fleet load test runs to detect regressions.
#
# Usage:
#   ./compare-metrics.sh [options]
#   ./compare-metrics.sh <file1.json> <file2.json>
#
# Options:
#   -f, --filter PATTERN   Only include runs whose workspace name contains PATTERN.
#                          Run-naming conventions make this a category selector, e.g.
#                          "loadtest" → baselines, "mig" → migrations.
#   -d, --depth N          Number of most recent runs to compare (default: 2)
#   -u, --unique           Deduplicate by workspace — keep only the most recent run
#                          per workspace. Useful for release-over-release comparison.
#   -m, --metrics-dir DIR  Path to runs directory (default: runs/ next to this script;
#                          searched recursively, so category subfolders are included)
#   -h, --help             Show this help message
#
# Examples:
#   ./compare-metrics.sh                                       # compare 2 most recent runs
#   ./compare-metrics.sh --filter loadtest --depth 4 --unique  # last 4 baseline releases
#   ./compare-metrics.sh file-a.json file-b.json               # compare two specific files
#
# Required: jq

set -euo pipefail

command -v jq >/dev/null 2>&1 || { echo "Error: 'jq' is required but not found in PATH." >&2; exit 1; }

usage() {
  sed -n '3,23p' "$0" | sed 's/^# \?//'
  exit "${1:-0}"
}

# ---------------------------------------------------------------------------
# Parse arguments
# ---------------------------------------------------------------------------
FILTER=""
DEPTH=2
UNIQUE=false
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
METRICS_DIR="${SCRIPT_DIR}/runs"
EXPLICIT_FILES=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    -f|--filter)      [[ $# -ge 2 ]] || { echo "Error: $1 requires a value." >&2; exit 1; }; FILTER="$2"; shift 2 ;;
    -d|--depth)       [[ $# -ge 2 ]] || { echo "Error: $1 requires a value." >&2; exit 1; }; DEPTH="$2"; shift 2 ;;
    -u|--unique)      UNIQUE=true; shift ;;
    -m|--metrics-dir) [[ $# -ge 2 ]] || { echo "Error: $1 requires a value." >&2; exit 1; }; METRICS_DIR="$2"; shift 2 ;;
    -h|--help)        usage 0 ;;
    -*)               echo "Unknown option: $1" >&2; usage 1 ;;
    *.json)           EXPLICIT_FILES+=("$1"); shift ;;
    *)                echo "Unknown argument: $1" >&2; usage 1 ;;
  esac
done

if ! [[ "$DEPTH" =~ ^[0-9]+$ ]] || [[ "$DEPTH" -lt 2 ]]; then
  echo "Error: --depth must be an integer >= 2." >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# Discover and select JSON files to compare
# ---------------------------------------------------------------------------

# Collect all candidate files, sorted by collected_at timestamp (newest first)
gather_files() {
  local dir="$1"
  if [[ ! -d "$dir" ]]; then
    echo "Error: Metrics directory '$dir' does not exist." >&2
    exit 1
  fi

  # Find all .json files, extract collected_at, sort descending by timestamp
  local tmpfile
  tmpfile=$(mktemp)
  trap 'rm -f "$tmpfile"' EXIT

  while IFS= read -r f; do
    # Extract collected_at from metadata; skip files that aren't valid metric files
    local ts workspace
    ts=$(jq -r '.metadata.collected_at // empty' "$f" 2>/dev/null) || continue
    workspace=$(jq -r '.metadata.workspace // empty' "$f" 2>/dev/null) || continue
    [[ -z "$ts" || -z "$workspace" ]] && continue

    # Apply filter if specified
    if [[ -n "$FILTER" ]] && [[ "$workspace" != *"$FILTER"* ]]; then
      continue
    fi

    echo "${ts}|${workspace}|${f}"
  done < <(find "$dir" -name '*.json' -type f 2>/dev/null) | sort -t'|' -k1,1 -r > "$tmpfile"

  if [[ "$UNIQUE" == true ]]; then
    # Keep only the most recent file per workspace (first occurrence since sorted desc)
    awk -F'|' '!seen[$2]++' "$tmpfile"
  else
    cat "$tmpfile"
  fi
}

FILES=()
EXPLICIT_MODE=false
if [[ ${#EXPLICIT_FILES[@]} -gt 0 ]]; then
  EXPLICIT_MODE=true
  # Explicit files provided — use them directly (in the order given)
  for f in "${EXPLICIT_FILES[@]}"; do
    if [[ ! -f "$f" ]]; then
      echo "Error: File not found: $f" >&2
      exit 1
    fi
    FILES+=("$f")
  done
  if [[ ${#FILES[@]} -lt 2 ]]; then
    echo "Error: Need at least 2 files to compare." >&2
    exit 1
  fi
else
  # Auto-discover from metrics directory
  while IFS='|' read -r _ts _ws filepath; do
    FILES+=("$filepath")
    [[ ${#FILES[@]} -ge $DEPTH ]] && break
  done < <(gather_files "$METRICS_DIR")

  if [[ ${#FILES[@]} -lt 2 ]]; then
    echo "Error: Found ${#FILES[@]} metric file(s) in '$METRICS_DIR' (need at least 2)." >&2
    [[ -n "$FILTER" ]] && echo "  Filter: '$FILTER'" >&2
    [[ "$UNIQUE" == true ]] && echo "  Unique: enabled" >&2
    exit 1
  fi
fi

# For auto-discovered files, reverse so oldest is first (they come sorted newest-first).
# For explicit files, keep the user-provided order (assumed oldest → newest).
ORDERED_FILES=()
if [[ "$EXPLICIT_MODE" == true ]]; then
  ORDERED_FILES=("${FILES[@]}")
else
  for (( i=${#FILES[@]}-1; i>=0; i-- )); do
    ORDERED_FILES+=("${FILES[$i]}")
  done
fi

echo "Comparing ${#ORDERED_FILES[@]} runs (oldest → newest):"
echo ""
for f in "${ORDERED_FILES[@]}"; do
  ws=$(jq -r '.metadata.workspace' "$f")
  ts=$(jq -r '.metadata.collected_at' "$f")
  interval=$(jq -r '.metadata.interval' "$f")
  printf "  %s  (%s, %s)\n" "$(basename "$f")" "$ws" "$interval"
done
echo ""

# ---------------------------------------------------------------------------
# Metric definitions: JSON path, display name, format, thresholds
#
# Format: path|name|unit|warn_pct|alert_pct|zero_alert|abs_op|abs_threshold
#   - warn_pct:       percentage change (delta) that triggers a ⚠️  warning
#   - alert_pct:      percentage change (delta) that triggers a 🔴 alert
#   - zero_alert:     "true" if going from 0 to non-zero is always an alert
#   - abs_op:         absolute threshold operator: "lt" (value must be < threshold),
#                     "eq" (value must equal threshold), or "" for none
#   - abs_threshold:  the absolute threshold value. Each individual column value
#                     is checked against this — a 🔴 appears inline next to any
#                     value that violates. Delta/Status only reflects relative change.
# ---------------------------------------------------------------------------
METRIC_DEFS=(
  # Fleet Server
  ".fleet_server.cpu_utilization.Average|Fleet CPU|%|15|30|false|lt|80"
  ".fleet_server.cpu_utilization.Maximum|Fleet CPU (max)|%|20|40|false||"
  ".fleet_server.memory_utilization.Average|Fleet Memory|%|15|30|false|lt|80"

  # RDS Writer
  ".rds_writer.cpu_utilization.Average|RDS Writer CPU|%|15|30|false|lt|80"
  ".rds_writer.database_connections.Average|RDS Writer Conns||50|100|false||"
  ".rds_writer.read_iops.Average|RDS Writer Read IOPS||50|100|false||"
  ".rds_writer.write_iops.Average|RDS Writer Write IOPS||50|100|false||"
  ".rds_writer.deadlocks.Sum|RDS Deadlocks||0|0|true|eq|0"

  # Redis (first node)
  ".redis[0].cpu_utilization.Average|Redis CPU|%|20|40|false|lt|80"
  ".redis[0].memory_utilization.Average|Redis Memory|%|15|30|false|lt|70"
)

# ALB metrics
ALB_DEFS=(
  ".alb.target_response_time.Average|ALB Latency|s|50|100|false||"
  ".alb.target_response_time.Maximum|ALB Latency (max)|s|50|100|false||"
  ".alb.http_5xx_count.Sum|ALB 5xx Errors||0|0|true|eq|0"
  ".alb.request_count.Sum|ALB Requests||0|0|false||"
)

# RDS Writer extended
RDS_WRITER_EXT_DEFS=(
  ".rds_writer_extended.buffer_cache_hit_ratio.Average|RDS Cache Hit Ratio|%|1|3|false||"
  ".rds_writer_extended.freeable_memory.Average|RDS Freeable Memory|bytes|15|30|false||"
  ".rds_writer_extended.select_latency.Average|RDS Select Latency|ms|50|100|false||"
  ".rds_writer_extended.insert_latency.Average|RDS Insert Latency|ms|50|100|false||"
  ".rds_writer_extended.dml_latency.Average|RDS DML Latency|ms|50|100|false||"
  ".rds_writer_extended.iops_utilization.utilization_pct|IOPS Utilization|%|15|30|false|lt|80"
)

# RDS Reader extended (first reader)
RDS_READER_EXT_DEFS=(
  ".rds_readers_extended[0].aurora_replica_lag.Average|Aurora Replica Lag|ms|50|100|false||"
  ".rds_readers_extended[0].buffer_cache_hit_ratio.Average|Reader Cache Hit Ratio|%|1|3|false||"
  ".rds_readers_extended[0].iops_utilization.utilization_pct|Reader IOPS Utilization|%|15|30|false|lt|80"
)

# Redis extended (first node)
REDIS_EXT_DEFS=(
  ".redis_extended[0].curr_connections.Average|Redis Connections||50|100|false||"
  ".redis_extended[0].evictions.Sum|Redis Evictions||0|0|true|eq|0"
  ".redis_extended[0].cache_hit_rate.Average|Redis Cache Hit Rate|%|5|15|false||"
)

# Other
OTHER_DEFS=(
  ".fleet_server_errors.error_count|Fleet Server Errors||0|0|true|eq|0"
  ".container_health.abnormal_stops|Container Stops||0|0|true|eq|0"
  ".container_health.start_spread_min|Container Start Spread|min|5|10|false|lt|10"
)

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# Portable last/second-to-last element accessors (macOS bash 3 lacks [-1])
arr_last()  { local arr=("$@"); echo "${arr[$((${#arr[@]}-1))]}"; }
arr_prev()  { local arr=("$@"); echo "${arr[$((${#arr[@]}-2))]}"; }

# Extract a metric value from a file, returning "null" if not present
get_val() {
  local file="$1" path="$2"
  jq -r "$path // \"null\"" "$file" 2>/dev/null || echo "null"
}

# Format a number for display
fmt_val() {
  local val="$1" unit="$2"
  if [[ "$val" == "null" || -z "$val" ]]; then
    echo "N/A"
    return
  fi
  case "$unit" in
    bytes)
      # Convert to human-readable
      echo "$val" | awk '{
        if ($1 >= 1073741824) printf "%.1fGB", $1/1073741824
        else if ($1 >= 1048576) printf "%.1fMB", $1/1048576
        else if ($1 >= 1024) printf "%.1fKB", $1/1024
        else printf "%d B", $1
      }'
      ;;
    %)  printf "%.1f%%" "$val" ;;
    s)  printf "%.3fs" "$val" ;;
    ms) printf "%.1fms" "$val" ;;
    min) printf "%.1fmin" "$val" ;;
    *)  printf "%.1f" "$val" ;;
  esac
}

# Calculate percentage change between two values
# For "inverse" metrics (cache hit ratio) where lower is worse,
# the caller handles interpretation.
pct_change() {
  local old="$1" new="$2"
  if [[ "$old" == "null" || "$new" == "null" ]]; then
    echo "null"
    return
  fi
  awk -v o="$old" -v n="$new" 'BEGIN {
    if (o == 0 && n == 0) { print 0; exit }
    if (o == 0) { print 999; exit }
    printf "%.1f", ((n - o) / (o < 0 ? -o : o)) * 100
  }'
}

# Determine status icon based on change and thresholds
# For metrics where decrease is bad (cache hit ratio), pass "invert"
status_icon() {
  local change="$1" warn="$2" alert="$3" zero_alert="$4" old_val="$5" new_val="$6" direction="${7:-normal}"

  if [[ "$change" == "null" ]]; then
    echo "  —"
    return
  fi

  # Informational metrics (e.g. request counts) are reported but never graded.
  if [[ "$direction" == "info" ]]; then
    echo "  —"
    return
  fi

  # Zero-alert: value went from 0 to non-zero
  if [[ "$zero_alert" == "true" && "$old_val" != "null" && "$new_val" != "null" ]]; then
    local is_old_zero is_new_nonzero
    is_old_zero=$(awk -v v="$old_val" 'BEGIN { print (v == 0) ? "true" : "false" }')
    is_new_nonzero=$(awk -v v="$new_val" 'BEGIN { print (v != 0) ? "true" : "false" }')
    if [[ "$is_old_zero" == "true" && "$is_new_nonzero" == "true" ]]; then
      echo "  ALERT"
      return
    fi
  fi

  # For inverted metrics (higher is better, like cache hit ratio),
  # flip the sign for threshold comparison
  local abs_change
  if [[ "$direction" == "invert" ]]; then
    abs_change=$(awk -v c="$change" 'BEGIN { print -c }')
  else
    abs_change="$change"
  fi

  # Only flag increases (or decreases for inverted metrics)
  local is_regression
  is_regression=$(awk -v c="$abs_change" 'BEGIN { print (c > 0) ? "true" : "false" }')

  if [[ "$is_regression" == "true" ]]; then
    local exceeds_alert exceeds_warn
    exceeds_alert=$(awk -v c="$abs_change" -v t="$alert" 'BEGIN { print (c >= t) ? "true" : "false" }')
    exceeds_warn=$(awk -v c="$abs_change" -v t="$warn" 'BEGIN { print (c >= t) ? "true" : "false" }')

    if [[ "$exceeds_alert" == "true" ]]; then
      echo "  ALERT"
    elif [[ "$exceeds_warn" == "true" ]]; then
      echo "  WARN"
    else
      echo "  ok"
    fi
  else
    echo "  ok"
  fi
}

# Check a value against an absolute threshold, return icon suffix or empty string
# Usage: abs_icon=$(check_abs_threshold "$val" "$abs_op" "$abs_threshold")
check_abs_threshold() {
  local val="$1" op="$2" threshold="$3"
  if [[ -z "$op" || -z "$threshold" || "$val" == "null" ]]; then
    echo ""
    return
  fi
  local violated=false
  case "$op" in
    lt) violated=$(awk -v v="$val" -v t="$threshold" 'BEGIN { print (v >= t) ? "true" : "false" }') ;;
    eq) violated=$(awk -v v="$val" -v t="$threshold" 'BEGIN { print (v != t) ? "true" : "false" }') ;;
    gt) violated=$(awk -v v="$val" -v t="$threshold" 'BEGIN { print (v > t) ? "true" : "false" }') ;;
  esac
  if [[ "$violated" == "true" ]]; then
    echo " 🔴"
  else
    echo ""
  fi
}

# ---------------------------------------------------------------------------
# Compare metrics across all files
# ---------------------------------------------------------------------------

# Track totals and individual findings for synopsis
TOTAL_ALERTS=0
TOTAL_WARNINGS=0
# Each entry: "severity|metric_name|detail"
# severity: 1=ALERT, 2=WARN (for sort: 1 sorts first = highest severity)
ALERT_LOG=()

log_alert() {
  local severity="$1" name="$2" detail="$3"
  ALERT_LOG+=("${severity}|${name}|${detail}")
  if [[ "$severity" == "1" ]]; then
    TOTAL_ALERTS=$((TOTAL_ALERTS + 1))
  else
    TOTAL_WARNINGS=$((TOTAL_WARNINGS + 1))
  fi
}

compare_metric_set() {
  local label="$1"
  shift
  local defs=("$@")
  local has_data=false
  local section_output=""

  for def in "${defs[@]}"; do
    IFS='|' read -r path name unit warn alert zero_alert abs_op abs_threshold <<< "$def"

    # Check if this metric exists in at least the last two files
    local last_file
    last_file=$(arr_last "${ORDERED_FILES[@]}")
    local prev_file
    prev_file=$(arr_prev "${ORDERED_FILES[@]}")
    local last_val prev_val
    last_val=$(get_val "$last_file" "$path")
    prev_val=$(get_val "$prev_file" "$path")

    if [[ "$last_val" == "null" && "$prev_val" == "null" ]]; then
      continue
    fi
    has_data=true

    # Determine grading direction:
    #   invert — decrease is the regression (higher is better)
    #   info   — throughput counters that grow run-to-run; show the delta but never grade it
    local direction="normal"
    case "$name" in
      *"Cache Hit"*|*"Freeable Memory"*) direction="invert" ;;
      *"Requests"*) direction="info" ;;
    esac

    # Build the row: name, then value for each file, then delta + status
    local row
    row=$(printf "  %-28s" "$name")

    # Values for each file — check each against absolute threshold
    for f in "${ORDERED_FILES[@]}"; do
      local val ws_name
      val=$(get_val "$f" "$path")
      ws_name=$(jq -r '.metadata.workspace' "$f")
      local abs_icon
      abs_icon=$(check_abs_threshold "$val" "$abs_op" "$abs_threshold")
      if [[ -n "$abs_icon" ]]; then
        row+=$(printf "  %9s%s" "$(fmt_val "$val" "$unit")" "$abs_icon")
        log_alert 1 "$name" "$(fmt_val "$val" "$unit") in $ws_name (threshold: ${abs_op} ${abs_threshold})"
      else
        row+=$(printf "  %12s" "$(fmt_val "$val" "$unit")")
      fi
    done

    # Delta between last two
    local change
    change=$(pct_change "$prev_val" "$last_val")
    if [[ "$change" != "null" ]]; then
      local sign=""
      local is_positive
      is_positive=$(awk -v c="$change" 'BEGIN { print (c > 0) ? "true" : "false" }')
      [[ "$is_positive" == "true" ]] && sign="+"
      row+=$(printf "  %8s%%" "${sign}${change}")
    else
      row+=$(printf "  %9s" "—")
    fi

    # Status (relative change)
    local icon
    icon=$(status_icon "$change" "$warn" "$alert" "$zero_alert" "$prev_val" "$last_val" "$direction")
    row+="$icon"

    case "$icon" in
      *ALERT*) log_alert 1 "$name" "delta ${change}% (threshold: ${alert}%)" ;;
      *WARN*)  log_alert 2 "$name" "delta ${change}% (threshold: ${warn}%)" ;;
    esac

    section_output+="${row}"$'\n'
  done

  if [[ "$has_data" == true ]]; then
    echo "$label"
    echo "$section_output"
  fi
}

# Print header
header=$(printf "  %-28s" "Metric")
for f in "${ORDERED_FILES[@]}"; do
  ws=$(jq -r '.metadata.workspace' "$f")
  header+=$(printf "  %12s" "$ws")
done
header+=$(printf "  %9s" "Delta")
header+="  Status"
echo "$header"

separator=""
for (( i=0; i < ${#header}; i++ )); do separator+="-"; done
echo "$separator"

compare_metric_set "Fleet Server" "${METRIC_DEFS[@]:0:3}"
compare_metric_set "RDS Writer" "${METRIC_DEFS[@]:3:5}"
compare_metric_set "Redis" "${METRIC_DEFS[@]:8:2}"
compare_metric_set "ALB" "${ALB_DEFS[@]}"
compare_metric_set "RDS Writer (extended)" "${RDS_WRITER_EXT_DEFS[@]}"
compare_metric_set "Aurora Replication" "${RDS_READER_EXT_DEFS[@]}"
compare_metric_set "Redis (extended)" "${REDIS_EXT_DEFS[@]}"
compare_metric_set "Errors & Restarts" "${OTHER_DEFS[@]}"

# ---------------------------------------------------------------------------
# RDS Reader comparison (dynamic — one row per reader across files)
# ---------------------------------------------------------------------------
# Readers are trickier since different workspaces may have different reader
# counts. We compare by position (first reader vs first reader).
has_readers=false
for f in "${ORDERED_FILES[@]}"; do
  if jq -e '.rds_readers | length > 0' "$f" >/dev/null 2>&1; then
    has_readers=true
    break
  fi
done

if [[ "$has_readers" == true ]]; then
  echo "RDS Readers"
  for idx in 0 1 2; do
    last_file=$(arr_last "${ORDERED_FILES[@]}")
    if ! jq -e ".rds_readers[$idx]" "$last_file" >/dev/null 2>&1; then
      continue
    fi

    for metric_pair in \
      "cpu_utilization.Average|Reader $((idx+1)) CPU|%|15|30|lt|90" \
      "database_connections.Average|Reader $((idx+1)) Conns||50|100||" \
      "read_iops.Average|Reader $((idx+1)) Read IOPS||50|100||"; do
      IFS='|' read -r mpath mname munit mwarn malert mabs_op mabs_threshold <<< "$metric_pair"
      full_path=".rds_readers[$idx].$mpath"

      row=$(printf "  %-28s" "$mname")
      local_vals=()
      for f in "${ORDERED_FILES[@]}"; do
        val=$(get_val "$f" "$full_path")
        local_vals+=("$val")
        ws_name=$(jq -r '.metadata.workspace' "$f")
        abs_icon=$(check_abs_threshold "$val" "$mabs_op" "$mabs_threshold")
        if [[ -n "$abs_icon" ]]; then
          row+=$(printf "  %9s%s" "$(fmt_val "$val" "$munit")" "$abs_icon")
          log_alert 1 "$mname" "$(fmt_val "$val" "$munit") in $ws_name (threshold: ${mabs_op} ${mabs_threshold})"
        else
          row+=$(printf "  %12s" "$(fmt_val "$val" "$munit")")
        fi
      done

      prev_val=$(arr_prev "${local_vals[@]}")
      last_val=$(arr_last "${local_vals[@]}")
      change=$(pct_change "$prev_val" "$last_val")
      if [[ "$change" != "null" ]]; then
        sign=""
        is_positive=$(awk -v c="$change" 'BEGIN { print (c > 0) ? "true" : "false" }')
        [[ "$is_positive" == "true" ]] && sign="+"
        row+=$(printf "  %8s%%" "${sign}${change}")
      else
        row+=$(printf "  %9s" "—")
      fi

      icon=$(status_icon "$change" "$mwarn" "$malert" "false" "$prev_val" "$last_val")
      row+="$icon"
      case "$icon" in
        *ALERT*) log_alert 1 "$mname" "delta ${change}% (threshold: ${malert}%)" ;;
        *WARN*)  log_alert 2 "$mname" "delta ${change}% (threshold: ${mwarn}%)" ;;
      esac
      echo "$row"
    done
  done
  echo ""
fi


# ---------------------------------------------------------------------------
# Synopsis — all alerts and warnings sorted by severity (highest first)
# ---------------------------------------------------------------------------
echo "==============================="
if [[ ${#ALERT_LOG[@]} -gt 0 ]]; then
  printf "Synopsis: %d alert(s), %d warning(s)\n\n" "$TOTAL_ALERTS" "$TOTAL_WARNINGS"
  # Sort by severity field (1=ALERT before 2=WARN)
  printf '%s\n' "${ALERT_LOG[@]}" | sort -t'|' -k1,1n | while IFS='|' read -r sev name detail; do
    if [[ "$sev" == "1" ]]; then
      printf "  🔴 ALERT: %s — %s\n" "$name" "$detail"
    else
      printf "  ⚠️  WARN:  %s — %s\n" "$name" "$detail"
    fi
  done
else
  echo "✅ All metrics within thresholds"
fi
