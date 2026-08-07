#!/usr/bin/env bash
#
# collect-metrics.sh - Collect AWS CloudWatch metrics for a Fleet load test environment.
#
# Usage:
#   ./collect-metrics.sh --workspace <name> [options]
#
# Options:
#   -w, --workspace NAME  Terraform workspace name (required)
#   -i, --interval RANGE  Lookback interval. Accepts <N>h, <N>m, or a bare integer (treated as hours). Default: 3h
#   -c, --category CAT    Run category for filing output: baseline | migration | mdm.
#                         Files output under runs/<category>/<workspace>/. Omit to use runs/<workspace>/.
#   -o, --output FILE     Output file path (default: runs/[<category>/]<workspace>/<workspace>-<date>Z-<interval>.json)
#   -r, --region REGION   AWS region (default: us-east-2)
#   -h, --help            Show this help message
#
# The script discovers AWS resources by naming convention from the Terraform workspace name,
# then collects CloudWatch metrics averaged over the specified interval ending at the current time.
#
# Required: aws cli v2, jq

set -euo pipefail

# ---------------------------------------------------------------------------
# Dependency check
# ---------------------------------------------------------------------------
for cmd in aws jq; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "Error: '$cmd' is required but not found in PATH." >&2; exit 1; }
done

usage() {
  sed -n '3,15p' "$0" | sed 's/^# \?//'
  exit "${1:-0}"
}

# ---------------------------------------------------------------------------
# Parse arguments
# ---------------------------------------------------------------------------
WORKSPACE=""
OUTPUT=""
REGION="us-east-2"
INTERVAL_INPUT="3h"
CATEGORY=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -w|--workspace)  WORKSPACE="$2"; shift 2 ;;
    -i|--interval)   INTERVAL_INPUT="$2"; shift 2 ;;
    -c|--category)   CATEGORY="$2"; shift 2 ;;
    -o|--output)     OUTPUT="$2"; shift 2 ;;
    -r|--region)     REGION="$2"; shift 2 ;;
    -h|--help)       usage 0 ;;
    -*)              echo "Unknown option: $1" >&2; usage 1 ;;
    *)               echo "Unknown argument: $1. Use --workspace to specify the workspace name." >&2; usage 1 ;;
  esac
done

if [[ -z "$WORKSPACE" ]]; then
  echo "Error: --workspace is required." >&2
  usage 1
fi

# The workspace name is interpolated into output file paths, so constrain it to a
# conservative slug. This rejects path separators / "../" (no writes outside runs/)
# and matches the Terraform workspace naming conventions documented in the README.
if [[ ! "$WORKSPACE" =~ ^[A-Za-z0-9][A-Za-z0-9_-]*$ ]]; then
  echo "Error: --workspace must match ^[A-Za-z0-9][A-Za-z0-9_-]*$ (letters, digits, '-', '_'). Got: '$WORKSPACE'" >&2
  exit 1
fi

if [[ -n "$CATEGORY" ]]; then
  case "$CATEGORY" in
    baseline|migration|mdm) ;;
    *) echo "Error: --category must be one of: baseline, migration, mdm. Got: '$CATEGORY'" >&2; exit 1 ;;
  esac
fi

# Parse interval: accept "<N>h", "<N>m", or a bare integer (interpreted as hours).
if [[ "$INTERVAL_INPUT" =~ ^([0-9]+)([hm]?)$ ]]; then
  INTERVAL_NUM="${BASH_REMATCH[1]}"
  INTERVAL_UNIT="${BASH_REMATCH[2]:-h}"
else
  echo "Error: --interval must be of the form '3h', '30m', or a bare positive integer (hours). Got: '$INTERVAL_INPUT'" >&2
  exit 1
fi

if [[ "$INTERVAL_NUM" -lt 1 ]]; then
  echo "Error: --interval must be a positive integer." >&2
  exit 1
fi

case "$INTERVAL_UNIT" in
  h) INTERVAL_SECONDS=$((INTERVAL_NUM * 3600)) ;;
  m) INTERVAL_SECONDS=$((INTERVAL_NUM * 60)) ;;
esac
INTERVAL_LABEL="${INTERVAL_NUM}${INTERVAL_UNIT}"

TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# Runs are filed under runs/[<category>/]<workspace>/. compare-metrics.sh discovers
# them recursively, so the category subfolder is purely for human organization.
METRICS_DIR="${SCRIPT_DIR}/runs${CATEGORY:+/$CATEGORY}/${WORKSPACE}"
mkdir -p "$METRICS_DIR"
OUTPUT="${OUTPUT:-${METRICS_DIR}/${WORKSPACE}-$(date -u +%Y-%m-%d-%H%M%SZ)-${INTERVAL_LABEL}.json}"
# Ensure the parent dir exists even when a custom --output path is given.
mkdir -p "$(dirname "$OUTPUT")"

# ---------------------------------------------------------------------------
# Time window: <interval> ending now
# ---------------------------------------------------------------------------
END_TIME="$TIMESTAMP"
if [[ "$(uname)" == "Darwin" ]]; then
  case "$INTERVAL_UNIT" in
    h) START_TIME=$(date -u -v-${INTERVAL_NUM}H +"%Y-%m-%dT%H:%M:%SZ") ;;
    m) START_TIME=$(date -u -v-${INTERVAL_NUM}M +"%Y-%m-%dT%H:%M:%SZ") ;;
  esac
else
  case "$INTERVAL_UNIT" in
    h) START_TIME=$(date -u -d "${INTERVAL_NUM} hours ago" +"%Y-%m-%dT%H:%M:%SZ") ;;
    m) START_TIME=$(date -u -d "${INTERVAL_NUM} minutes ago" +"%Y-%m-%dT%H:%M:%SZ") ;;
  esac
fi

# Max 5-min data points expected in this window (minimum 1 to avoid div-by-zero
# on sub-5-minute intervals).
MAX_POINTS=$(( INTERVAL_SECONDS / 300 ))
[[ $MAX_POINTS -lt 1 ]] && MAX_POINTS=1

echo "Collecting metrics for workspace: $WORKSPACE"
echo "Region: $REGION"
echo "Interval: ${INTERVAL_LABEL}"
echo "Time range: $START_TIME → $END_TIME"
echo ""

# ---------------------------------------------------------------------------
# Resource name derivation
# ---------------------------------------------------------------------------
# Two possible naming schemes:
#   Root config:  cluster = fleet-<ws>-backend, rds = fleetdm-<ws>-mysql, redis = fleet-<ws>-redis
#   Infra module: cluster = fleet-<ws>,         rds = fleet-<ws>,         redis = fleet-<ws>

PREFIX="fleet-${WORKSPACE}"

# ---------------------------------------------------------------------------
# Discovery: find actual resource names
# ---------------------------------------------------------------------------
echo "Discovering resources..."

# ECS clusters — try both naming patterns
ECS_CLUSTER=""
for candidate in "${PREFIX}-backend" "${PREFIX}"; do
  if aws ecs describe-clusters --clusters "$candidate" --region "$REGION" \
       --query "clusters[?status=='ACTIVE'].clusterName" --output text 2>/dev/null | grep -q .; then
    ECS_CLUSTER="$candidate"
    break
  fi
done

if [[ -z "$ECS_CLUSTER" ]]; then
  echo "Warning: Could not find ECS cluster for workspace '$WORKSPACE'. Trying tag-based discovery..." >&2
  ECS_CLUSTER=$(aws ecs list-clusters --region "$REGION" --output json \
    | jq -r --arg ws "$WORKSPACE" '.clusterArns[] | select(contains($ws))' \
    | head -1 | xargs -I{} basename {})
fi

if [[ -z "$ECS_CLUSTER" ]]; then
  echo "Error: No ECS cluster found for workspace '$WORKSPACE'" >&2
  exit 1
fi
echo "  ECS Cluster: $ECS_CLUSTER"

# ECS services in the cluster
ECS_SERVICES_JSON=$(aws ecs list-services --cluster "$ECS_CLUSTER" --region "$REGION" --output json 2>/dev/null || echo '{"serviceArns":[]}')
ECS_SERVICE_ARNS=$(echo "$ECS_SERVICES_JSON" | jq -r '.serviceArns[]')

FLEET_SERVICE=""
OSQUERY_PERF_SERVICE=""
LOADTEST_SERVICES=()

for arn in $ECS_SERVICE_ARNS; do
  svc=$(basename "$arn")
  case "$svc" in
    fleet)          FLEET_SERVICE="$svc" ;;
    osquery_perf)   OSQUERY_PERF_SERVICE="$svc" ;;
    loadtest-*)     LOADTEST_SERVICES+=("$svc") ;;
  esac
done

echo "  Fleet Service: ${FLEET_SERVICE:-<not found>}"
echo "  osquery-perf Service: ${OSQUERY_PERF_SERVICE:-<not found>}"
echo "  Loadtest Services: ${#LOADTEST_SERVICES[@]} found"

# RDS — try both naming patterns
RDS_CLUSTER_ID=""
for candidate in "fleetdm-${WORKSPACE}-mysql" "${PREFIX}"; do
  if aws rds describe-db-clusters --db-cluster-identifier "$candidate" --region "$REGION" \
       --query "DBClusters[0].DBClusterIdentifier" --output text 2>/dev/null | grep -q .; then
    RDS_CLUSTER_ID="$candidate"
    break
  fi
done
echo "  RDS Cluster: ${RDS_CLUSTER_ID:-<not found>}"

# Get RDS instance identifiers (writer + readers)
RDS_WRITER_INSTANCE=""
RDS_READER_INSTANCES=()
if [[ -n "$RDS_CLUSTER_ID" ]]; then
  RDS_MEMBERS_JSON=$(aws rds describe-db-clusters --db-cluster-identifier "$RDS_CLUSTER_ID" --region "$REGION" \
    --query "DBClusters[0].DBClusterMembers" --output json 2>/dev/null || echo "[]")
  RDS_WRITER_INSTANCE=$(echo "$RDS_MEMBERS_JSON" | jq -r '.[] | select(.IsClusterWriter==true) | .DBInstanceIdentifier' | head -1)
  while IFS= read -r inst; do
    [[ -n "$inst" ]] && RDS_READER_INSTANCES+=("$inst")
  done < <(echo "$RDS_MEMBERS_JSON" | jq -r '.[] | select(.IsClusterWriter==false) | .DBInstanceIdentifier')
  echo "  RDS Writer: ${RDS_WRITER_INSTANCE:-<not found>}"
  echo "  RDS Readers: ${RDS_READER_INSTANCES[*]:-<none>}"
fi

# RDS instance class (for IOPS utilization calculation)
RDS_WRITER_INSTANCE_CLASS=""
if [[ -n "$RDS_WRITER_INSTANCE" ]]; then
  RDS_WRITER_INSTANCE_CLASS=$(aws rds describe-db-instances --db-instance-identifier "$RDS_WRITER_INSTANCE" --region "$REGION" \
    --query "DBInstances[0].DBInstanceClass" --output text 2>/dev/null || echo "")
fi

# RDS DbiResourceId for Performance Insights
RDS_WRITER_DBI_RESOURCE_ID=""
if [[ -n "$RDS_WRITER_INSTANCE" ]]; then
  RDS_WRITER_DBI_RESOURCE_ID=$(aws rds describe-db-instances --db-instance-identifier "$RDS_WRITER_INSTANCE" --region "$REGION" \
    --query "DBInstances[0].DbiResourceId" --output text 2>/dev/null || echo "")
fi

RDS_READER_DBI_RESOURCE_IDS=()
for reader in "${RDS_READER_INSTANCES[@]}"; do
  rid=$(aws rds describe-db-instances --db-instance-identifier "$reader" --region "$REGION" \
    --query "DBInstances[0].DbiResourceId" --output text 2>/dev/null || echo "")
  [[ -n "$rid" ]] && RDS_READER_DBI_RESOURCE_IDS+=("$rid")
done

# ElastiCache Redis — try both naming patterns
REDIS_REPLICATION_GROUP=""
for candidate in "${PREFIX}-redis" "${PREFIX}"; do
  if aws elasticache describe-replication-groups --replication-group-id "$candidate" --region "$REGION" \
       --query "ReplicationGroups[0].ReplicationGroupId" --output text 2>/dev/null | grep -q .; then
    REDIS_REPLICATION_GROUP="$candidate"
    break
  fi
done
echo "  Redis Replication Group: ${REDIS_REPLICATION_GROUP:-<not found>}"

REDIS_NODE_IDS=()
if [[ -n "$REDIS_REPLICATION_GROUP" ]]; then
  while IFS= read -r nid; do
    [[ -n "$nid" ]] && REDIS_NODE_IDS+=("$nid")
  done < <(aws elasticache describe-replication-groups --replication-group-id "$REDIS_REPLICATION_GROUP" --region "$REGION" \
    --query "ReplicationGroups[0].MemberClusters[]" --output json 2>/dev/null | jq -r '.[]')
  echo "  Redis Nodes: ${REDIS_NODE_IDS[*]:-<none>}"
fi

# ALB discovery
ALB_ARN_SUFFIX=""
ALB_TG_ARN_SUFFIX=""
ALB_ARN=$(aws elbv2 describe-load-balancers --region "$REGION" --output json 2>/dev/null \
  | jq -r --arg ws "$WORKSPACE" '.LoadBalancers[] | select(.LoadBalancerName | contains($ws)) | .LoadBalancerArn' \
  | head -1)
if [[ -n "$ALB_ARN" ]]; then
  # Extract the suffix after "app/" for CloudWatch dimensions
  ALB_ARN_SUFFIX=$(echo "$ALB_ARN" | grep -o 'app/.*')
  echo "  ALB: $ALB_ARN_SUFFIX"

  # Find the target group for the fleet service
  ALB_TG_ARN=$(aws elbv2 describe-target-groups --load-balancer-arn "$ALB_ARN" --region "$REGION" \
    --query "TargetGroups[0].TargetGroupArn" --output text 2>/dev/null || echo "")
  if [[ -n "$ALB_TG_ARN" ]]; then
    ALB_TG_ARN_SUFFIX=$(echo "$ALB_TG_ARN" | grep -o 'targetgroup/.*')
    echo "  ALB Target Group: $ALB_TG_ARN_SUFFIX"
  fi
else
  echo "  ALB: <not found>"
fi

# Discover CloudWatch log group for Fleet server
# Try exact name matches first, then fall back to a broad search.
# Patterns: /ecs/<prefix>-backend/fleet, /ecs/<prefix>/fleet, <prefix> (bare)
FLEET_LOG_GROUP=""
for candidate in "/ecs/${PREFIX}-backend/fleet" "/ecs/${PREFIX}/fleet" "${PREFIX}"; do
  if aws logs describe-log-groups --log-group-name-prefix "$candidate" --region "$REGION" \
       --query "logGroups[0].logGroupName" --output text 2>/dev/null | grep -q "^${candidate}"; then
    FLEET_LOG_GROUP="$candidate"
    break
  fi
done
if [[ -z "$FLEET_LOG_GROUP" ]]; then
  # Broad search: find log groups containing the workspace name and "fleet",
  # excluding Container Insights groups (which are metrics, not application logs).
  FLEET_LOG_GROUP=$(aws logs describe-log-groups --region "$REGION" --output json 2>/dev/null \
    | jq -r --arg ws "$WORKSPACE" '.logGroups[].logGroupName | select(contains($ws)) | select(contains("fleet")) | select(contains("containerinsights") | not)' \
    | head -1)
fi
echo "  Fleet Log Group: ${FLEET_LOG_GROUP:-<not found>}"

echo ""
echo "Collecting CloudWatch metrics..."

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# get_metric <namespace> <metric_name> <dimensions> <period> <statistics...>
# Returns JSON with Datapoints
get_metric() {
  local ns="$1" metric="$2" dims="$3" period="$4"
  shift 4
  local stats=("$@")

  aws cloudwatch get-metric-statistics \
    --namespace "$ns" \
    --metric-name "$metric" \
    --dimensions $dims \
    --start-time "$START_TIME" \
    --end-time "$END_TIME" \
    --period "$period" \
    --statistics "${stats[@]}" \
    --region "$REGION" \
    --output json 2>/dev/null || echo '{"Datapoints":[]}'
}

# collect_metric <namespace> <metric_name> <dimensions> <statistics...>
# Returns JSON object with the metric values and data_coverage info.
# Uses the full interval as one period for the aggregate, plus 5-min periods
# to count actual data points for coverage reporting.
# Note: data_coverage is a 0-1 ratio (1.0 == full coverage), not a percentage.
collect_metric() {
  local ns="$1" metric="$2" dims="$3"
  shift 3
  local stats=("$@")

  local result fine
  result=$(get_metric "$ns" "$metric" "$dims" "$INTERVAL_SECONDS" "${stats[@]}")
  fine=$(get_metric "$ns" "$metric" "$dims" 300 SampleCount)

  local dp_count
  dp_count=$(echo "$fine" | jq '[.Datapoints[]] | length')

  # Extract the single datapoint, rounding numbers to 2 decimal places
  local datapoint
  datapoint=$(echo "$result" | jq '.Datapoints[0] // null | if . then with_entries(if .value | type == "number" then .value = (.value * 100 | round / 100) else . end) else null end')

  jq -n \
    --argjson dp "$datapoint" \
    --argjson dp_count "$dp_count" \
    --argjson max_points "$MAX_POINTS" \
    'if $dp then $dp + {"data_points": $dp_count, "max_points": $max_points, "data_coverage": (if $dp_count > 0 then ($dp_count / $max_points * 100 | round / 100) else 0 end)} else null end'
}

# collect_ecs_utilization <cluster> <service> <utilized_metric> <reserved_metric>
# Computes utilization as Sum(Utilized) / Sum(Reserved) * 100 using Container Insights.
# This matches the ECS Performance Dashboard formula and gives accurate service-wide
# percentages. Returns JSON with Average, Minimum, Maximum as percentages, plus
# data_coverage info.
collect_ecs_utilization() {
  local cluster="$1" service="$2" util_metric="$3" resv_metric="$4"
  local dims="Name=ClusterName,Value=$cluster Name=ServiceName,Value=$service"

  # Get utilized and reserved at 5-min granularity using Sum stat
  local util_raw resv_raw
  util_raw=$(get_metric "ECS/ContainerInsights" "$util_metric" "$dims" 300 Sum)
  resv_raw=$(get_metric "ECS/ContainerInsights" "$resv_metric" "$dims" 300 Sum)

  # Compute utilization percentage per 5-min period, then derive avg/min/max.
  # As above, data_coverage is a 0-1 ratio (1.0 == full coverage), not a percentage.
  jq -n \
    --argjson util "$util_raw" \
    --argjson resv "$resv_raw" \
    --argjson max_points "$MAX_POINTS" \
    '
    # Build lookup of reserved by timestamp
    ($resv.Datapoints | map({(.Timestamp): .Sum}) | add // {}) as $resv_map |
    # Calculate percentage for each utilized datapoint where reserved exists
    [($util.Datapoints // [])[] |
      . as $dp |
      ($resv_map[$dp.Timestamp] // null) as $r |
      select($r != null and $r > 0) |
      ($dp.Sum / $r * 100)
    ] |
    if length > 0 then
      {
        Average: (add / length * 100 | round / 100),
        Minimum: (min * 100 | round / 100),
        Maximum: (max * 100 | round / 100),
        Unit: "Percent",
        data_points: length,
        max_points: $max_points,
        data_coverage: (length / $max_points * 100 | round / 100)
      }
    else null end'
}

# ---------------------------------------------------------------------------
# Collect ECS Fleet Server metrics
# max_iops_for_class <instance_class>
# Returns the max IOPS for a given Aurora instance class. Echoes 0 if unknown.
# Source: https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/AuroraMySQL.Managing.Performance.html
max_iops_for_class() {
  case "$1" in
    db.r6g.large)     echo 10000 ;;
    db.r6g.xlarge)    echo 15000 ;;
    db.r6g.2xlarge)   echo 20000 ;;
    db.r6g.4xlarge)   echo 20000 ;;
    db.r6g.8xlarge)   echo 40000 ;;
    db.r6g.12xlarge)  echo 40000 ;;
    db.r6g.16xlarge)  echo 80000 ;;
    db.r7g.large)     echo 10000 ;;
    db.r7g.xlarge)    echo 15000 ;;
    db.r7g.2xlarge)   echo 20000 ;;
    db.r7g.4xlarge)   echo 20000 ;;
    db.r7g.8xlarge)   echo 40000 ;;
    db.r7g.12xlarge)  echo 40000 ;;
    db.r7g.16xlarge)  echo 80000 ;;
    *)                echo 0 ;;
  esac
}

# calc_iops_util <riops_json> <wiops_json> <instance_class>
# Returns a JSON object with IOPS utilization details.
calc_iops_util() {
  local riops="$1" wiops="$2" inst_class="$3"
  local max_iops
  max_iops=$(max_iops_for_class "$inst_class")
  jq -n \
    --argjson riops "$riops" \
    --argjson wiops "$wiops" \
    --argjson max_iops "$max_iops" \
    --arg instance_class "${inst_class:-unknown}" \
    '{
      instance_class: $instance_class,
      max_iops: $max_iops,
      read_iops_avg: ($riops.Average // null),
      write_iops_avg: ($wiops.Average // null),
      total_iops_avg: (if ($riops.Average // null) and ($wiops.Average // null) then (($riops.Average + $wiops.Average) * 100 | round / 100) else null end),
      utilization_pct: (if $max_iops > 0 and ($riops.Average // null) and ($wiops.Average // null) then ((($riops.Average + $wiops.Average) / $max_iops * 100) * 100 | round / 100) else null end)
    }'
}

# vcpus_for_class <instance_class>
# Returns the vCPU count for a given Aurora instance class. Echoes 0 if unknown.
# Source: https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/Concepts.DBInstanceClass.html
vcpus_for_class() {
  case "$1" in
    db.r6g.large)     echo 2 ;;
    db.r6g.xlarge)    echo 4 ;;
    db.r6g.2xlarge)   echo 8 ;;
    db.r6g.4xlarge)   echo 16 ;;
    db.r6g.8xlarge)   echo 32 ;;
    db.r6g.12xlarge)  echo 48 ;;
    db.r6g.16xlarge)  echo 64 ;;
    db.r7g.large)     echo 2 ;;
    db.r7g.xlarge)    echo 4 ;;
    db.r7g.2xlarge)   echo 8 ;;
    db.r7g.4xlarge)   echo 16 ;;
    db.r7g.8xlarge)   echo 32 ;;
    db.r7g.12xlarge)  echo 48 ;;
    db.r7g.16xlarge)  echo 64 ;;
    *)                echo 0 ;;
  esac
}

# ---------------------------------------------------------------------------
FLEET_SERVER_METRICS="{}"
if [[ -n "$FLEET_SERVICE" ]]; then
  echo "  Fleet Server: CPU Utilization (Container Insights)..."
  fleet_cpu=$(collect_ecs_utilization "$ECS_CLUSTER" "$FLEET_SERVICE" "CpuUtilized" "CpuReserved")

  echo "  Fleet Server: Memory Utilization (Container Insights)..."
  fleet_mem=$(collect_ecs_utilization "$ECS_CLUSTER" "$FLEET_SERVICE" "MemoryUtilized" "MemoryReserved")

  # Get running task count from describe-services (always available, no Container Insights needed)
  fleet_tasks=$(aws ecs describe-services --cluster "$ECS_CLUSTER" --services "$FLEET_SERVICE" --region "$REGION" \
    --query "services[0].{runningCount:runningCount,desiredCount:desiredCount}" --output json 2>/dev/null || echo '{}')

  FLEET_SERVER_METRICS=$(jq -n \
    --argjson cpu "$fleet_cpu" \
    --argjson mem "$fleet_mem" \
    --argjson tasks "$fleet_tasks" \
    '{cpu_utilization: $cpu, memory_utilization: $mem, task_counts: $tasks}')
fi

# ---------------------------------------------------------------------------
# Collect ECS osquery-perf / loadtest container metrics
# ---------------------------------------------------------------------------
LOADTEST_METRICS="{}"

# osquery-perf service (infra module path)
if [[ -n "$OSQUERY_PERF_SERVICE" ]]; then
  echo "  osquery-perf: CPU Utilization (Container Insights)..."
  oqp_cpu=$(collect_ecs_utilization "$ECS_CLUSTER" "$OSQUERY_PERF_SERVICE" "CpuUtilized" "CpuReserved")

  echo "  osquery-perf: Memory Utilization (Container Insights)..."
  oqp_mem=$(collect_ecs_utilization "$ECS_CLUSTER" "$OSQUERY_PERF_SERVICE" "MemoryUtilized" "MemoryReserved")

  oqp_tasks=$(aws ecs describe-services --cluster "$ECS_CLUSTER" --services "$OSQUERY_PERF_SERVICE" --region "$REGION" \
    --query "services[0].{runningCount:runningCount,desiredCount:desiredCount}" --output json 2>/dev/null || echo '{}')

  LOADTEST_METRICS=$(jq -n \
    --argjson cpu "$oqp_cpu" \
    --argjson mem "$oqp_mem" \
    --argjson tasks "$oqp_tasks" \
    '{cpu_utilization: $cpu, memory_utilization: $mem, task_counts: $tasks}')

# Loadtest services (root config path — services named loadtest-0, loadtest-1, etc.)
elif [[ ${#LOADTEST_SERVICES[@]} -gt 0 ]]; then
  first_svc="${LOADTEST_SERVICES[0]}"
  echo "  loadtest containers: CPU Utilization (${#LOADTEST_SERVICES[@]} services)..."

  lt_running=0
  lt_desired=0
  for svc in "${LOADTEST_SERVICES[@]}"; do
    counts=$(aws ecs describe-services --cluster "$ECS_CLUSTER" --services "$svc" --region "$REGION" \
      --query "services[0].{r:runningCount,d:desiredCount}" --output json 2>/dev/null || echo '{"r":0,"d":0}')
    lt_running=$((lt_running + $(echo "$counts" | jq '.r // 0')))
    lt_desired=$((lt_desired + $(echo "$counts" | jq '.d // 0')))
  done

  lt_cpu=$(collect_ecs_utilization "$ECS_CLUSTER" "$first_svc" "CpuUtilized" "CpuReserved")

  lt_mem=$(collect_ecs_utilization "$ECS_CLUSTER" "$first_svc" "MemoryUtilized" "MemoryReserved")

  LOADTEST_METRICS=$(jq -n \
    --argjson cpu "$lt_cpu" \
    --argjson mem "$lt_mem" \
    --arg running "$lt_running" \
    --arg desired "$lt_desired" \
    --arg count "${#LOADTEST_SERVICES[@]}" \
    '{cpu_utilization: $cpu, memory_utilization: $mem, task_counts: {runningCount: ($running|tonumber), desiredCount: ($desired|tonumber), serviceCount: ($count|tonumber)}}')
fi

# ---------------------------------------------------------------------------
# Collect RDS Writer metrics
# ---------------------------------------------------------------------------
RDS_WRITER_METRICS="{}"
if [[ -n "$RDS_WRITER_INSTANCE" ]]; then
  echo "  RDS Writer: CPU Utilization..."
  rds_w_cpu=$(collect_metric "AWS/RDS" "CPUUtilization" \
    "Name=DBInstanceIdentifier,Value=$RDS_WRITER_INSTANCE" \
    Average Maximum Minimum)

  echo "  RDS Writer: Database Connections..."
  rds_w_conns=$(collect_metric "AWS/RDS" "DatabaseConnections" \
    "Name=DBInstanceIdentifier,Value=$RDS_WRITER_INSTANCE" \
    Average Maximum Minimum)

  echo "  RDS Writer: Read IOPS..."
  rds_w_riops=$(collect_metric "AWS/RDS" "ReadIOPS" \
    "Name=DBInstanceIdentifier,Value=$RDS_WRITER_INSTANCE" \
    Average Maximum)

  echo "  RDS Writer: Write IOPS..."
  rds_w_wiops=$(collect_metric "AWS/RDS" "WriteIOPS" \
    "Name=DBInstanceIdentifier,Value=$RDS_WRITER_INSTANCE" \
    Average Maximum)

  echo "  RDS Writer: Deadlocks..."
  rds_w_deadlocks=$(collect_metric "AWS/RDS" "Deadlocks" \
    "Name=DBInstanceIdentifier,Value=$RDS_WRITER_INSTANCE" \
    Sum Average Maximum)

  RDS_WRITER_METRICS=$(jq -n \
    --argjson cpu "$rds_w_cpu" \
    --argjson conns "$rds_w_conns" \
    --argjson riops "$rds_w_riops" \
    --argjson wiops "$rds_w_wiops" \
    --argjson deadlocks "$rds_w_deadlocks" \
    '{instance: "writer", cpu_utilization: $cpu, database_connections: $conns, read_iops: $riops, write_iops: $wiops, deadlocks: $deadlocks}')
fi

# ---------------------------------------------------------------------------
# Collect RDS Reader metrics (standard + extended in one pass)
# ---------------------------------------------------------------------------
RDS_READER_METRICS="[]"
RDS_READERS_EXT="[]"
if [[ ${#RDS_READER_INSTANCES[@]} -gt 0 ]]; then
  reader_arr="[]"
  readers_ext_arr="[]"
  reader_idx=1
  for reader in "${RDS_READER_INSTANCES[@]}"; do
    reader_label="reader-${reader_idx}"
    echo "  RDS $reader_label: CPU, Connections, IOPS..."

    rds_r_cpu=$(collect_metric "AWS/RDS" "CPUUtilization" \
      "Name=DBInstanceIdentifier,Value=$reader" \
      Average Maximum Minimum)

    rds_r_conns=$(collect_metric "AWS/RDS" "DatabaseConnections" \
      "Name=DBInstanceIdentifier,Value=$reader" \
      Average Maximum Minimum)

    rds_r_riops=$(collect_metric "AWS/RDS" "ReadIOPS" \
      "Name=DBInstanceIdentifier,Value=$reader" \
      Average Maximum)

    rds_r_wiops=$(collect_metric "AWS/RDS" "WriteIOPS" \
      "Name=DBInstanceIdentifier,Value=$reader" \
      Average Maximum)

    reader_obj=$(jq -n \
      --arg instance "$reader_label" \
      --argjson cpu "$rds_r_cpu" \
      --argjson conns "$rds_r_conns" \
      --argjson riops "$rds_r_riops" \
      --argjson wiops "$rds_r_wiops" \
      '{instance: $instance, cpu_utilization: $cpu, database_connections: $conns, read_iops: $riops, write_iops: $wiops}')
    reader_arr=$(echo "$reader_arr" | jq --argjson obj "$reader_obj" '. + [$obj]')

    # Extended metrics for this reader
    echo "  RDS $reader_label: AuroraReplicaLag, FreeableMemory, BufferCacheHitRatio, SelectLatency..."

    # AuroraReplicaLag: Replication delay (milliseconds) between the writer
    # and this reader. Values above 20-50ms under load may cause stale reads.
    # A rising trend across releases indicates the reader can't keep up with
    # write volume.
    rds_r_lag=$(collect_metric "AWS/RDS" "AuroraReplicaLag" \
      "Name=DBInstanceIdentifier,Value=$reader" \
      Average Maximum)

    # FreeableMemory: Tracks available RAM on the reader.
    rds_r_freemem=$(collect_metric "AWS/RDS" "FreeableMemory" \
      "Name=DBInstanceIdentifier,Value=$reader" \
      Average Minimum)

    # BufferCacheHitRatio: Readers often have different cache profiles since
    # they serve read-heavy query patterns. Investigate if below 99%.
    rds_r_cache=$(collect_metric "AWS/RDS" "BufferCacheHitRatio" \
      "Name=DBInstanceIdentifier,Value=$reader" \
      Average Minimum)

    # SelectLatency: Readers only serve SELECT queries, so this is their
    # primary latency indicator.
    rds_r_select_lat=$(collect_metric "AWS/RDS" "SelectLatency" \
      "Name=DBInstanceIdentifier,Value=$reader" \
      Average Maximum)

    # IOPS utilization: Combined read+write IOPS as a percentage of the
    # reader's instance class maximum.
    echo "  RDS $reader_label: IOPS Utilization..."
    rds_r_inst_class=$(aws rds describe-db-instances --db-instance-identifier "$reader" --region "$REGION" \
      --query "DBInstances[0].DBInstanceClass" --output text 2>/dev/null || echo "")
    rds_r_iops_util=$(calc_iops_util "$rds_r_riops" "$rds_r_wiops" "$rds_r_inst_class")

    # Threads Running (active sessions via PI): Same as writer — average
    # active sessions on this reader. db.load.avg without GroupBy returns
    # the total Average Active Sessions (AAS).
    rds_r_threads="null"
    reader_dbi_idx=$((reader_idx - 1))
    if [[ $reader_dbi_idx -lt ${#RDS_READER_DBI_RESOURCE_IDS[@]} ]] && [[ -n "${RDS_READER_DBI_RESOURCE_IDS[$reader_dbi_idx]:-}" ]]; then
      echo "  RDS $reader_label: Threads Running (active sessions via PI)..."
      pi_r_threads_raw=$(aws pi get-resource-metrics \
        --service-type RDS \
        --identifier "${RDS_READER_DBI_RESOURCE_IDS[$reader_dbi_idx]}" \
        --start-time "$START_TIME" \
        --end-time "$END_TIME" \
        --period-in-seconds 300 \
        --metric-queries '[{"Metric":"db.load.avg"}]' \
        --region "$REGION" \
        --output json 2>/dev/null || echo '{}')
      rds_r_vcpus=$(vcpus_for_class "$rds_r_inst_class")
      rds_r_threads=$(echo "$pi_r_threads_raw" | jq --argjson vcpus "$rds_r_vcpus" '
        (.MetricList // [])[0].DataPoints
        | if . and length > 0 then
            [.[].Value | select(. != null)] |
            if length > 0 then
              { average: (add / length * 100 | round / 100),
                maximum: (max * 100 | round / 100),
                vcpus: $vcpus }
            else null end
          else null end')
    fi

    reader_ext=$(jq -n \
      --arg instance "$reader_label" \
      --argjson lag "$rds_r_lag" \
      --argjson freemem "$rds_r_freemem" \
      --argjson cache "$rds_r_cache" \
      --argjson select_lat "$rds_r_select_lat" \
      --argjson iops_util "$rds_r_iops_util" \
      --argjson threads "$rds_r_threads" \
      '{
        instance: $instance,
        aurora_replica_lag: $lag,
        freeable_memory: $freemem,
        buffer_cache_hit_ratio: $cache,
        select_latency: $select_lat,
        iops_utilization: $iops_util,
        threads_running: $threads
      }')
    readers_ext_arr=$(echo "$readers_ext_arr" | jq --argjson obj "$reader_ext" '. + [$obj]')

    reader_idx=$((reader_idx + 1))
  done
  RDS_READER_METRICS="$reader_arr"
  RDS_READERS_EXT="$readers_ext_arr"
fi

# ---------------------------------------------------------------------------
# Helper: extract flattened top SQL from a raw PI get-resource-metrics response.
# Input: raw JSON from aws pi get-resource-metrics
# Output: array of {rank, sql, load_avg} sorted by load descending
#
# PI response structure: MetricList[] where [0] is the aggregate and [1..N]
# are per-SQL entries. Each entry has .Key.Dimensions["db.sql_tokenized.statement"]
# and .DataPoints[].Value. We average the DataPoints for the load value.
# ---------------------------------------------------------------------------
flatten_top_sql() {
  local raw="$1"
  echo "$raw" | jq '
    [(.MetricList // [])[1:] | .[] |
      select(.Key.Dimensions["db.sql_tokenized.statement"] | . and (. | length > 0)) |
      # Filter to only datapoints that have a Value field (PI omits Value when load is zero)
      (.DataPoints | [.[] | select(.Value)]) as $valid_points |
      select($valid_points | length > 0) |
      {
        sql: .Key.Dimensions["db.sql_tokenized.statement"],
        load_avg: ([$valid_points[].Value] | add / length * 100 | round / 100)
      }
    ] | sort_by(-.load_avg) | to_entries | map({rank: (.key + 1)} + .value)
  ' 2>/dev/null || echo '[]'
}

# ---------------------------------------------------------------------------
# Collect RDS Performance Insights — Top SQL by Load (writer)
# ---------------------------------------------------------------------------
RDS_PI_WRITER="[]"
if [[ -n "$RDS_WRITER_DBI_RESOURCE_ID" ]]; then
  echo "  RDS Performance Insights: Top SQL (writer)..."
  pi_raw=$(aws pi get-resource-metrics \
    --service-type RDS \
    --identifier "$RDS_WRITER_DBI_RESOURCE_ID" \
    --start-time "$START_TIME" \
    --end-time "$END_TIME" \
    --period-in-seconds 3600 \
    --metric-queries '[{"Metric":"db.load.avg","GroupBy":{"Group":"db.sql_tokenized","Limit":5}}]' \
    --region "$REGION" \
    --output json 2>/dev/null || echo '{}')

  RDS_PI_WRITER=$(flatten_top_sql "$pi_raw")
fi

# Performance Insights for reader(s)
RDS_PI_READERS="[]"
if [[ ${#RDS_READER_DBI_RESOURCE_IDS[@]} -gt 0 ]]; then
  pi_reader_arr="[]"
  idx=0
  for rid in "${RDS_READER_DBI_RESOURCE_IDS[@]}"; do
    reader_label="reader-$((idx + 1))"
    echo "  RDS Performance Insights: Top SQL ($reader_label)..."

    pi_raw=$(aws pi get-resource-metrics \
      --service-type RDS \
      --identifier "$rid" \
      --start-time "$START_TIME" \
      --end-time "$END_TIME" \
      --period-in-seconds 3600 \
      --metric-queries '[{"Metric":"db.load.avg","GroupBy":{"Group":"db.sql_tokenized","Limit":5}}]' \
      --region "$REGION" \
      --output json 2>/dev/null || echo '{}')

    pi_flat=$(flatten_top_sql "$pi_raw")
    pi_reader_obj=$(jq -n --arg inst "$reader_label" --argjson sql "$pi_flat" '{instance: $inst, top_sql: $sql}')
    pi_reader_arr=$(echo "$pi_reader_arr" | jq --argjson obj "$pi_reader_obj" '. + [$obj]')
    idx=$((idx + 1))
  done
  RDS_PI_READERS="$pi_reader_arr"
fi

# ---------------------------------------------------------------------------
# Collect ElastiCache Redis metrics (standard + extended in one pass)
# ---------------------------------------------------------------------------
REDIS_METRICS="[]"
REDIS_EXT="[]"
if [[ ${#REDIS_NODE_IDS[@]} -gt 0 ]]; then
  redis_arr="[]"
  redis_ext_arr="[]"
  redis_idx=1
  for node_id in "${REDIS_NODE_IDS[@]}"; do
    redis_label="redis-${redis_idx}"
    echo "  $redis_label: CPU, Memory, Connections, Evictions, CacheHitRate..."

    redis_cpu=$(collect_metric "AWS/ElastiCache" "EngineCPUUtilization" \
      "Name=CacheClusterId,Value=$node_id" \
      Average Maximum Minimum)

    redis_mem=$(collect_metric "AWS/ElastiCache" "DatabaseMemoryUsagePercentage" \
      "Name=CacheClusterId,Value=$node_id" \
      Average Maximum Minimum)

    node_obj=$(jq -n \
      --arg node "$redis_label" \
      --argjson cpu "$redis_cpu" \
      --argjson mem "$redis_mem" \
      '{node: $node, cpu_utilization: $cpu, memory_utilization: $mem}')
    redis_arr=$(echo "$redis_arr" | jq --argjson obj "$node_obj" '. + [$obj]')

    # Extended metrics for this node

    # CurrConnections: Number of active client connections. A rising trend
    # across runs with the same container count may indicate connection leaks.
    redis_conns=$(collect_metric "AWS/ElastiCache" "CurrConnections" \
      "Name=CacheClusterId,Value=$node_id" \
      Average Maximum)

    # Evictions: Keys evicted due to memory pressure. Any non-zero value means
    # Redis is dropping data to stay within its memory limit.
    redis_evictions=$(collect_metric "AWS/ElastiCache" "Evictions" \
      "Name=CacheClusterId,Value=$node_id" \
      Sum Maximum)

    # CacheHitRate: Percentage of key lookups that found data. A declining
    # trend means more cache misses pushing load to the database.
    redis_hitrate=$(collect_metric "AWS/ElastiCache" "CacheHitRate" \
      "Name=CacheClusterId,Value=$node_id" \
      Average Minimum)

    node_ext=$(jq -n \
      --arg node "$redis_label" \
      --argjson conns "$redis_conns" \
      --argjson evictions "$redis_evictions" \
      --argjson hitrate "$redis_hitrate" \
      '{
        node: $node,
        curr_connections: $conns,
        evictions: $evictions,
        cache_hit_rate: $hitrate
      }')
    redis_ext_arr=$(echo "$redis_ext_arr" | jq --argjson obj "$node_ext" '. + [$obj]')

    redis_idx=$((redis_idx + 1))
  done
  REDIS_METRICS="$redis_arr"
  REDIS_EXT="$redis_ext_arr"
fi

# ---------------------------------------------------------------------------
# ALB metrics
# Provides API-level latency and error rates from the load balancer's
# perspective — the closest proxy to end-user experience.
# -------------------------------------------------------------------------
ALB_METRICS="{}"
if [[ -n "$ALB_ARN_SUFFIX" ]]; then
  # TargetResponseTime: Average time (seconds) for the target (Fleet server)
  # to respond to a request. Tracks API latency as seen by the ALB. Rising
  # values across releases indicate server-side performance degradation.
  echo "  ALB: TargetResponseTime (API latency)..."
  alb_latency=$(collect_metric "AWS/ApplicationELB" "TargetResponseTime" \
    "Name=LoadBalancer,Value=$ALB_ARN_SUFFIX" \
    Average Maximum Minimum)

  # HTTPCode_Target_5XX_Count: Total number of HTTP 5xx responses from Fleet
  # server containers. Non-zero values indicate server errors. Track over time
  # to catch regressions that introduce error-producing code paths.
  echo "  ALB: HTTPCode_Target_5XX_Count (server errors)..."
  alb_5xx=$(collect_metric "AWS/ApplicationELB" "HTTPCode_Target_5XX_Count" \
    "Name=LoadBalancer,Value=$ALB_ARN_SUFFIX" \
    Sum)

  # RequestCount: Total HTTP requests handled by the ALB during the interval.
  # Useful for normalizing other metrics (e.g. errors per request) and verifying
  # the load test is generating expected traffic volume.
  echo "  ALB: RequestCount (total throughput)..."
  alb_requests=$(collect_metric "AWS/ApplicationELB" "RequestCount" \
    "Name=LoadBalancer,Value=$ALB_ARN_SUFFIX" \
    Sum)

  # ProcessedBytes: Total bytes processed by the ALB (request + response).
  # Tracks the volume of data flowing through the load balancer. Useful for
  # detecting unexpected payload size changes across releases (e.g. a new
  # API response that's much larger than before).
  echo "  ALB: ProcessedBytes (traffic volume)..."
  alb_bytes=$(collect_metric "AWS/ApplicationELB" "ProcessedBytes" \
    "Name=LoadBalancer,Value=$ALB_ARN_SUFFIX" \
    Sum)

  ALB_METRICS=$(jq -n \
    --argjson latency "$alb_latency" \
    --argjson errors_5xx "$alb_5xx" \
    --argjson requests "$alb_requests" \
    --argjson bytes "$alb_bytes" \
    '{
      target_response_time: $latency,
      http_5xx_count: $errors_5xx,
      request_count: $requests,
      processed_bytes: $bytes
    }')
fi

# -------------------------------------------------------------------------
# CloudWatch Logs Insights — Fleet server error query
# Queries the Fleet server log group for error-level log entries, filtering
# out known noise:
#   - fleet_detail_query_software errors (osquery-perf has a default 50%
#     failure rate for software queries)
#   - "context canceled" errors (normal during container deployments)
#
# Returns the total error count and up to 10 sample error messages for
# manual review. The expected value is 0 errors over at least 1 hour.
# -------------------------------------------------------------------------
LOGS_ERRORS="{}"
if [[ -n "$FLEET_LOG_GROUP" ]]; then
  echo "  CloudWatch Logs: Querying Fleet server errors..."
  # Start the Logs Insights query (async)
  LOGS_QUERY_ID=$(aws logs start-query \
    --log-group-name "$FLEET_LOG_GROUP" \
    --start-time "$(date -d "$START_TIME" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%SZ" "$START_TIME" +%s)" \
    --end-time "$(date -d "$END_TIME" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%SZ" "$END_TIME" +%s)" \
    --query-string 'fields @timestamp, @message
| filter ispresent(error) or ispresent(err) or level = "error"
| filter @message not like /fleet_detail_query_software/
| filter @message not like /context canceled/
| filter level != "debug"
| filter level != "info"
| sort @timestamp desc
| limit 10000' \
    --region "$REGION" \
    --output text --query 'queryId' 2>/dev/null || echo "")

  if [[ -n "$LOGS_QUERY_ID" ]]; then
    # Poll for query completion (typically takes 5-15 seconds)
    echo "  CloudWatch Logs: Waiting for query results..."
    logs_status="Running"
    logs_attempts=0
    while [[ "$logs_status" == "Running" || "$logs_status" == "Scheduled" ]] && [[ $logs_attempts -lt 30 ]]; do
      sleep 2
      logs_result=$(aws logs get-query-results --query-id "$LOGS_QUERY_ID" --region "$REGION" --output json 2>/dev/null || echo '{"status":"Failed"}')
      logs_status=$(echo "$logs_result" | jq -r '.status')
      logs_attempts=$((logs_attempts + 1))
    done

    if [[ "$logs_status" == "Complete" ]]; then
      # Count total matched records and extract sample messages
      logs_total=$(echo "$logs_result" | jq '.statistics.recordsMatched // 0')
      logs_samples=$(echo "$logs_result" | jq '[.results[:10][] | [.[] | select(.field == "@message") | .value] | first // empty]')

      LOGS_ERRORS=$(jq -n \
        --argjson total "$logs_total" \
        --argjson samples "$logs_samples" \
        '{
          error_count: $total,
          sample_messages: $samples
        }')
      echo "  CloudWatch Logs: Found $logs_total errors"
    else
      echo "  CloudWatch Logs: Query did not complete (status: $logs_status)" >&2
      LOGS_ERRORS='{"error_count": null, "query_status": "'"$logs_status"'"}'
    fi
  else
    echo "  CloudWatch Logs: Failed to start query" >&2
  fi
fi

# -------------------------------------------------------------------------
# Aurora MySQL extended metrics (writer)
# These track database health indicators that degrade slowly over time
# and are easy to miss in a single load test run.
# -------------------------------------------------------------------------
RDS_WRITER_EXT="{}"
if [[ -n "$RDS_WRITER_INSTANCE" ]]; then
  # FreeableMemory: Available RAM (bytes) on the DB instance. A downward
  # trend across releases can indicate memory leaks in queries, growing
  # working sets, or the need to upsize the instance class.
  echo "  RDS Writer: FreeableMemory (available RAM)..."
  rds_w_freemem=$(collect_metric "AWS/RDS" "FreeableMemory" \
    "Name=DBInstanceIdentifier,Value=$RDS_WRITER_INSTANCE" \
    Average Minimum)

  # BufferCacheHitRatio: Percentage of requests served from the Aurora buffer
  # cache vs reading from disk. Values near 100% are ideal. A decline over
  # time means more queries are hitting disk, increasing latency. Investigate
  # if this drops below 99%.
  echo "  RDS Writer: BufferCacheHitRatio (cache efficiency)..."
  rds_w_cache=$(collect_metric "AWS/RDS" "BufferCacheHitRatio" \
    "Name=DBInstanceIdentifier,Value=$RDS_WRITER_INSTANCE" \
    Average Minimum)

  # VolumeBytesUsed: Total storage consumed by the Aurora cluster (bytes).
  # Track over time for capacity planning. Sharp increases may indicate
  # unexpected data growth from new features or logging changes.
  echo "  RDS Writer: VolumeBytesUsed (disk usage)..."
  rds_w_disk=$(collect_metric "AWS/RDS" "VolumeBytesUsed" \
    "Name=DBClusterIdentifier,Value=$RDS_CLUSTER_ID" \
    Average Maximum)

  # SelectLatency: Average latency (milliseconds) of SELECT queries on the writer.
  # Rising values indicate query performance degradation, possibly due to
  # missing indexes, growing table sizes, or lock contention.
  echo "  RDS Writer: SelectLatency..."
  rds_w_select_lat=$(collect_metric "AWS/RDS" "SelectLatency" \
    "Name=DBInstanceIdentifier,Value=$RDS_WRITER_INSTANCE" \
    Average Maximum)

  # InsertLatency: Average latency (milliseconds) of INSERT queries on the writer.
  # Fleet's workload is insert-heavy (host checkins, software inventory).
  # Rising values may indicate index bloat or lock contention.
  echo "  RDS Writer: InsertLatency..."
  rds_w_insert_lat=$(collect_metric "AWS/RDS" "InsertLatency" \
    "Name=DBInstanceIdentifier,Value=$RDS_WRITER_INSTANCE" \
    Average Maximum)

  # DMLLatency: Average latency (milliseconds) of all DML operations (INSERT,
  # UPDATE, DELETE). Provides a single overall write-path health indicator.
  echo "  RDS Writer: DMLLatency..."
  rds_w_dml_lat=$(collect_metric "AWS/RDS" "DMLLatency" \
    "Name=DBInstanceIdentifier,Value=$RDS_WRITER_INSTANCE" \
    Average Maximum)

  # DatabaseConnections via Performance Insights — "threads running" metric.
  # This is distinct from CloudWatch DatabaseConnections (which counts all
  # connections including idle ones). Threads running counts only actively
  # executing queries. Expected: <= vCPUs on the instance (e.g.
  # db.r6g.4xlarge has 16 vCPUs). Sustained values above vCPU count indicate
  # CPU saturation from query concurrency.
  echo "  RDS Writer: Threads Running (active sessions via PI)..."
  rds_w_threads="null"
  if [[ -n "$RDS_WRITER_DBI_RESOURCE_ID" ]]; then
    pi_threads_raw=$(aws pi get-resource-metrics \
      --service-type RDS \
      --identifier "$RDS_WRITER_DBI_RESOURCE_ID" \
      --start-time "$START_TIME" \
      --end-time "$END_TIME" \
      --period-in-seconds 300 \
      --metric-queries '[{"Metric":"db.load.avg"}]' \
      --region "$REGION" \
      --output json 2>/dev/null || echo '{}')
    # db.load.avg without GroupBy returns the total Average Active Sessions
    # (AAS), which equals the average number of threads running.
    # Expected: average <= vCPUs on the instance. Sustained values above
    # vCPU count indicate CPU saturation from query concurrency.
    rds_w_vcpus=$(vcpus_for_class "$RDS_WRITER_INSTANCE_CLASS")
    rds_w_threads=$(echo "$pi_threads_raw" | jq --argjson vcpus "$rds_w_vcpus" '
      (.MetricList // [])[0].DataPoints
      | if . and length > 0 then
          [.[].Value | select(. != null)] |
          if length > 0 then
            { average: (add / length * 100 | round / 100),
              maximum: (max * 100 | round / 100),
              vcpus: $vcpus }
          else null end
        else null end')
  fi

  # IOPS utilization: Combined read+write IOPS as a percentage of the instance
  # class maximum. Values above 80% indicate I/O saturation risk.
  echo "  RDS Writer: IOPS Utilization..."
  rds_w_iops_util=$(calc_iops_util "$rds_w_riops" "$rds_w_wiops" "$RDS_WRITER_INSTANCE_CLASS")

  RDS_WRITER_EXT=$(jq -n \
    --argjson freemem "$rds_w_freemem" \
    --argjson cache "$rds_w_cache" \
    --argjson disk "$rds_w_disk" \
    --argjson select_lat "$rds_w_select_lat" \
    --argjson insert_lat "$rds_w_insert_lat" \
    --argjson dml_lat "$rds_w_dml_lat" \
    --argjson threads "$rds_w_threads" \
    --argjson iops_util "$rds_w_iops_util" \
    '{
      freeable_memory: $freemem,
      buffer_cache_hit_ratio: $cache,
      volume_bytes_used: $disk,
      select_latency: $select_lat,
      insert_latency: $insert_lat,
      dml_latency: $dml_lat,
      threads_running: $threads,
      iops_utilization: $iops_util
    }')
fi



# -------------------------------------------------------------------------
# ECS Network traffic (Container Insights)
# NetworkRxBytes / NetworkTxBytes measure the total bytes received/sent by
# Fleet server containers. Useful for detecting unexpected traffic changes
# (e.g. a new feature that increases payload sizes). Requires Container
# Insights to be enabled on the ECS cluster.
# -------------------------------------------------------------------------
NETWORK_METRICS="{}"
if [[ -n "$FLEET_SERVICE" ]]; then
  echo "  Fleet Server: Network RX (Container Insights)..."
  net_rx=$(collect_metric "ECS/ContainerInsights" "NetworkRxBytes" \
    "Name=ClusterName,Value=$ECS_CLUSTER Name=ServiceName,Value=$FLEET_SERVICE" \
    Sum Average)

  echo "  Fleet Server: Network TX (Container Insights)..."
  net_tx=$(collect_metric "ECS/ContainerInsights" "NetworkTxBytes" \
    "Name=ClusterName,Value=$ECS_CLUSTER Name=ServiceName,Value=$FLEET_SERVICE" \
    Sum Average)

  NETWORK_METRICS=$(jq -n \
    --argjson rx "$net_rx" \
    --argjson tx "$net_tx" \
    '{
      network_rx_bytes: $rx,
      network_tx_bytes: $tx
    }')
fi

# -------------------------------------------------------------------------
# osquery-perf / loadtest container health
#
# Checks two things:
#   1. Stopped tasks — any tasks with non-zero exit codes during the interval
#      indicate crashes or OOM kills. Expected: 0.
#   2. Running task uptime — queries all running tasks for the loadtest service,
#      captures each task's startedAt time and calculates uptime. If any
#      container started significantly later than the others (>10 min spread),
#      it likely restarted. All containers should start within a few minutes
#      of each other.
#
# Output JSON:
#   abnormal_stops:    count of tasks that stopped with non-zero exit code
#   running_tasks:     number of currently running tasks
#   oldest_start:      ISO timestamp of the earliest startedAt
#   newest_start:      ISO timestamp of the latest startedAt
#   start_spread_min:  difference in minutes between oldest and newest start
#   start_spread_alert: true if spread > 10 minutes (indicates a restart)
#   tasks:             array of {task_id, started_at, uptime_min} per task
# -------------------------------------------------------------------------
CONTAINER_HEALTH="{}"
LOADTEST_SVC="${OSQUERY_PERF_SERVICE:-${LOADTEST_SERVICES[0]:-}}"
if [[ -n "$LOADTEST_SVC" ]]; then
  echo "  Loadtest: Checking for container restarts..."
  # List stopped tasks in the cluster from the interval and count non-zero exit codes
  stopped_tasks=$(aws ecs list-tasks --cluster "$ECS_CLUSTER" --desired-status STOPPED \
    --region "$REGION" --output json 2>/dev/null || echo '{"taskArns":[]}')
  stopped_arns=$(echo "$stopped_tasks" | jq -r '.taskArns[]' | head -50)

  restart_count=0
  if [[ -n "$stopped_arns" ]]; then
    task_details=$(aws ecs describe-tasks --cluster "$ECS_CLUSTER" \
      --tasks $stopped_arns \
      --region "$REGION" --output json 2>/dev/null || echo '{"tasks":[]}')

    restart_count=$(echo "$task_details" | jq \
      --arg starttime "$START_TIME" --arg endtime "$END_TIME" \
      '[.tasks[] | select(.stoppedAt >= $starttime) | select(.stoppedAt <= $endtime) | select([.containers[]? | select(.exitCode > 0)] | length > 0)] | length')
  fi
  echo "  Loadtest: $restart_count abnormal container stops detected"

  # Query running tasks for uptime analysis
  echo "  Loadtest: Checking container uptime..."
  running_tasks=$(aws ecs list-tasks --cluster "$ECS_CLUSTER" --service-name "$LOADTEST_SVC" \
    --desired-status RUNNING --region "$REGION" --output json 2>/dev/null || echo '{"taskArns":[]}')
  running_arns=$(echo "$running_tasks" | jq -r '.taskArns[]')
  running_count=$(echo "$running_tasks" | jq '.taskArns | length')

  uptime_json="[]"
  oldest_start=""
  newest_start=""
  if [[ -n "$running_arns" ]] && [[ "$running_count" -gt 0 ]]; then
    running_details=$(aws ecs describe-tasks --cluster "$ECS_CLUSTER" \
      --tasks $running_arns \
      --region "$REGION" --output json 2>/dev/null || echo '{"tasks":[]}')

    # Extract task start times and compute uptime in minutes
    uptime_json=$(echo "$running_details" | jq --arg now "$TIMESTAMP" '
      [.tasks[] | select(.startedAt) | {
        task_id: (.taskArn | split("/") | last),
        started_at: .startedAt,
        uptime_min: (
          (($now | strptime("%Y-%m-%dT%H:%M:%SZ") | mktime) -
           (.startedAt | strptime("%Y-%m-%dT%H:%M:%S") | mktime)) / 60
          | . * 100 | round / 100
        )
      }] | sort_by(.started_at)
    ' 2>/dev/null || echo '[]')

    oldest_start=$(echo "$uptime_json" | jq -r 'if length > 0 then first.started_at else null end')
    newest_start=$(echo "$uptime_json" | jq -r 'if length > 0 then last.started_at else null end')
  fi

  # Calculate the spread between oldest and newest start time
  start_spread_min="null"
  start_spread_alert="false"
  if [[ -n "$oldest_start" && "$oldest_start" != "null" && -n "$newest_start" && "$newest_start" != "null" ]]; then
    if [[ "$(uname)" == "Darwin" ]]; then
      oldest_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%S" "${oldest_start%%.*}" +%s 2>/dev/null || echo 0)
      newest_epoch=$(date -j -f "%Y-%m-%dT%H:%M:%S" "${newest_start%%.*}" +%s 2>/dev/null || echo 0)
    else
      oldest_epoch=$(date -d "${oldest_start}" +%s 2>/dev/null || echo 0)
      newest_epoch=$(date -d "${newest_start}" +%s 2>/dev/null || echo 0)
    fi
    if [[ "$oldest_epoch" -gt 0 && "$newest_epoch" -gt 0 ]]; then
      spread_seconds=$(( newest_epoch - oldest_epoch ))
      start_spread_min=$(awk "BEGIN { printf \"%.1f\", $spread_seconds / 60 }")
      # Alert if any container started >10 minutes after the first one
      if (( spread_seconds > 600 )); then
        start_spread_alert="true"
      fi
    fi
  fi

  echo "  Loadtest: $running_count running tasks, start spread=${start_spread_min}min"

  CONTAINER_HEALTH=$(jq -n \
    --argjson abnormal_stops "$restart_count" \
    --argjson running_tasks "$running_count" \
    --arg oldest_start "${oldest_start:-null}" \
    --arg newest_start "${newest_start:-null}" \
    --argjson start_spread_min "${start_spread_min:-null}" \
    --argjson start_spread_alert "$start_spread_alert" \
    --argjson tasks "$uptime_json" \
    '{
      abnormal_stops: $abnormal_stops,
      running_tasks: $running_tasks,
      oldest_start: (if $oldest_start == "null" then null else $oldest_start end),
      newest_start: (if $newest_start == "null" then null else $newest_start end),
      start_spread_min: $start_spread_min,
      start_spread_alert: $start_spread_alert,
      tasks: $tasks
    }')
fi


# ---------------------------------------------------------------------------
# Assemble final JSON output
# ---------------------------------------------------------------------------
echo ""
echo "Assembling results..."

jq -n \
  --arg workspace "$WORKSPACE" \
  --arg collected_at "$TIMESTAMP" \
  --arg region "$REGION" \
  --arg interval "${INTERVAL_LABEL}" \
  --arg start_time "$START_TIME" \
  --arg end_time "$END_TIME" \
  --argjson fleet_server "$FLEET_SERVER_METRICS" \
  --argjson loadtest_containers "$LOADTEST_METRICS" \
  --argjson rds_writer "$RDS_WRITER_METRICS" \
  --argjson rds_readers "$RDS_READER_METRICS" \
  --argjson rds_pi_writer "$RDS_PI_WRITER" \
  --argjson rds_pi_readers "$RDS_PI_READERS" \
  --argjson redis "$REDIS_METRICS" \
  --argjson alb "$ALB_METRICS" \
  --argjson fleet_server_errors "$LOGS_ERRORS" \
  --argjson rds_writer_ext "$RDS_WRITER_EXT" \
  --argjson rds_readers_ext "$RDS_READERS_EXT" \
  --argjson redis_ext "$REDIS_EXT" \
  --argjson network "$NETWORK_METRICS" \
  --argjson container_health "$CONTAINER_HEALTH" \
  '{
    metadata: {
      workspace: $workspace,
      collected_at: $collected_at,
      region: $region,
      interval: $interval,
      time_window: {start: $start_time, end: $end_time}
    },
    fleet_server: $fleet_server,
    loadtest_containers: $loadtest_containers,
    rds_writer: $rds_writer,
    rds_readers: $rds_readers,
    rds_performance_insights: {
      writer: $rds_pi_writer,
      readers: $rds_pi_readers
    },
    redis: $redis,
    alb: $alb,
    fleet_server_errors: $fleet_server_errors,
    rds_writer_extended: $rds_writer_ext,
    rds_readers_extended: $rds_readers_ext,
    redis_extended: $redis_ext,
    network: $network,
    container_health: $container_health
  }' > "$OUTPUT"

echo "Done! Metrics written to: $OUTPUT"

# Markdown synopsis file — same name as JSON but .md extension
MD_OUTPUT="${OUTPUT%.json}.md"

# Start building synopsis (goes to both stdout and markdown file)
{
echo "# Metrics Synopsis: ${WORKSPACE}"
echo ""
echo "- **Collected:** ${TIMESTAMP}"
echo "- **Interval:** ${INTERVAL_LABEL}"
echo "- **Window:** ${START_TIME} to ${END_TIME}"
echo ""
echo "## Summary (${INTERVAL_LABEL} averages)"
echo ""
echo '```'

# Print a quick summary table to stdout
if jq -e '.fleet_server.cpu_utilization' "$OUTPUT" >/dev/null 2>&1; then
  fleet_cpu_avg=$(jq -r '.fleet_server.cpu_utilization.Average // "N/A"' "$OUTPUT")
  fleet_mem_avg=$(jq -r '.fleet_server.memory_utilization.Average // "N/A"' "$OUTPUT")
  fleet_running=$(jq -r '.fleet_server.task_counts.runningCount // "N/A"' "$OUTPUT")
  printf "Fleet Server:  CPU=%s%%  Mem=%s%%  Containers=%s\n" "$fleet_cpu_avg" "$fleet_mem_avg" "$fleet_running"
fi

if jq -e '.loadtest_containers.task_counts' "$OUTPUT" >/dev/null 2>&1; then
  lt_cpu_avg=$(jq -r '.loadtest_containers.cpu_utilization.Average // "N/A"' "$OUTPUT")
  lt_mem_avg=$(jq -r '.loadtest_containers.memory_utilization.Average // "N/A"' "$OUTPUT")
  lt_running=$(jq -r '.loadtest_containers.task_counts.runningCount // "N/A"' "$OUTPUT")
  printf "Loadtest:      CPU=%s%%  Mem=%s%%  Containers=%s\n" "$lt_cpu_avg" "$lt_mem_avg" "$lt_running"
fi

if jq -e '.rds_writer.cpu_utilization' "$OUTPUT" >/dev/null 2>&1; then
  rds_w_cpu_avg=$(jq -r '.rds_writer.cpu_utilization.Average // "N/A"' "$OUTPUT")
  rds_w_conns_avg=$(jq -r '.rds_writer.database_connections.Average // "N/A"' "$OUTPUT")
  rds_w_deadlocks=$(jq -r '.rds_writer.deadlocks.Sum // "N/A"' "$OUTPUT")
  printf "RDS Writer:    CPU=%s%%  Connections=%s  Deadlocks=%s\n" "$rds_w_cpu_avg" "$rds_w_conns_avg" "$rds_w_deadlocks"
fi

for reader_label in $(jq -r '.rds_readers[].instance // empty' "$OUTPUT" 2>/dev/null); do
  rds_r_cpu_avg=$(jq -r --arg inst "$reader_label" '.rds_readers[] | select(.instance==$inst) | .cpu_utilization.Average // "N/A"' "$OUTPUT")
  rds_r_conns_avg=$(jq -r --arg inst "$reader_label" '.rds_readers[] | select(.instance==$inst) | .database_connections.Average // "N/A"' "$OUTPUT")
  printf "RDS %s:  CPU=%s%%  Connections=%s\n" "$reader_label" "$rds_r_cpu_avg" "$rds_r_conns_avg"
done

if jq -e '.redis[0].cpu_utilization' "$OUTPUT" >/dev/null 2>&1; then
  redis_cpu_avg=$(jq -r '.redis[0].cpu_utilization.Average // "N/A"' "$OUTPUT")
  redis_mem_avg=$(jq -r '.redis[0].memory_utilization.Average // "N/A"' "$OUTPUT")
  printf "Redis:         CPU=%s%%  Mem=%s%%\n" "$redis_cpu_avg" "$redis_mem_avg"
fi

# Print top SQL for writer
if jq -e '.rds_performance_insights.writer | length > 0' "$OUTPUT" >/dev/null 2>&1; then
  printf "\nTop SQL (writer):\n"
  jq -r '.rds_performance_insights.writer[] | "  #\(.rank) load=\(.load_avg)  \(.sql[:80])"' "$OUTPUT"
fi

# Print top SQL for each reader
for reader_label in $(jq -r '.rds_performance_insights.readers[].instance // empty' "$OUTPUT" 2>/dev/null); do
  printf "\nTop SQL (%s):\n" "$reader_label"
  jq -r --arg inst "$reader_label" '.rds_performance_insights.readers[] | select(.instance==$inst) | .top_sql[] | "  #\(.rank) load=\(.load_avg)  \(.sql[:80])"' "$OUTPUT"
done

if jq -e '.alb.target_response_time' "$OUTPUT" >/dev/null 2>&1; then
  alb_lat=$(jq -r '.alb.target_response_time.Average // "N/A"' "$OUTPUT")
  alb_5xx=$(jq -r '.alb.http_5xx_count.Sum // 0' "$OUTPUT")
  alb_reqs=$(jq -r '.alb.request_count.Sum // "N/A"' "$OUTPUT")
  alb_bytes=$(jq -r '.alb.processed_bytes.Sum // "N/A" | if type == "number" then (. / 1048576 * 100 | round / 100 | tostring) + "MB" else . end' "$OUTPUT")
  printf "ALB:           Latency=%ss  5xx=%s  Requests=%s  Traffic=%s\n" "$alb_lat" "$alb_5xx" "$alb_reqs" "$alb_bytes"
fi

if jq -e '.fleet_server_errors.error_count' "$OUTPUT" >/dev/null 2>&1; then
  err_count=$(jq -r '.fleet_server_errors.error_count // "N/A"' "$OUTPUT")
  printf "Fleet Errors:  Count=%s\n" "$err_count"
fi

if jq -e '.rds_writer_extended.freeable_memory' "$OUTPUT" >/dev/null 2>&1; then
  rds_freemem=$(jq -r '.rds_writer_extended.freeable_memory.Average // "N/A" | if type == "number" then (. / 1073741824 * 100 | round / 100 | tostring) + "GB" else . end' "$OUTPUT")
  rds_cache=$(jq -r '.rds_writer_extended.buffer_cache_hit_ratio.Average // "N/A"' "$OUTPUT")
  rds_disk=$(jq -r '.rds_writer_extended.volume_bytes_used.Average // "N/A" | if type == "number" then (. / 1073741824 * 100 | round / 100 | tostring) + "GB" else . end' "$OUTPUT")
  rds_sel_lat=$(jq -r '.rds_writer_extended.select_latency.Average // "N/A"' "$OUTPUT")
  rds_ins_lat=$(jq -r '.rds_writer_extended.insert_latency.Average // "N/A"' "$OUTPUT")
  rds_threads=$(jq -r '.rds_writer_extended.threads_running.average // "N/A"' "$OUTPUT")
  rds_iops_pct=$(jq -r '.rds_writer_extended.iops_utilization.utilization_pct // "N/A"' "$OUTPUT")
  printf "RDS Writer:    FreeMem=%s  CacheHit=%s%%  Disk=%s  Threads=%s  IOPS=%s%%\n" \
    "$rds_freemem" "$rds_cache" "$rds_disk" "$rds_threads" "$rds_iops_pct"
  printf "               SelectLat=%sms  InsertLat=%sms\n" "$rds_sel_lat" "$rds_ins_lat"
fi

for reader_label in $(jq -r '.rds_readers_extended[].instance // empty' "$OUTPUT" 2>/dev/null); do
  rlag=$(jq -r --arg i "$reader_label" '.rds_readers_extended[] | select(.instance==$i) | .aurora_replica_lag.Average // "N/A"' "$OUTPUT")
  rcache=$(jq -r --arg i "$reader_label" '.rds_readers_extended[] | select(.instance==$i) | .buffer_cache_hit_ratio.Average // "N/A"' "$OUTPUT")
  rfreemem=$(jq -r --arg i "$reader_label" '.rds_readers_extended[] | select(.instance==$i) | .freeable_memory.Average // "N/A" | if type == "number" then (. / 1073741824 * 100 | round / 100 | tostring) + "GB" else . end' "$OUTPUT")
  rthreads=$(jq -r --arg i "$reader_label" '.rds_readers_extended[] | select(.instance==$i) | .threads_running.average // "N/A"' "$OUTPUT")
  rsel_lat=$(jq -r --arg i "$reader_label" '.rds_readers_extended[] | select(.instance==$i) | .select_latency.Average // "N/A"' "$OUTPUT")
  riops_pct=$(jq -r --arg i "$reader_label" '.rds_readers_extended[] | select(.instance==$i) | .iops_utilization.utilization_pct // "N/A"' "$OUTPUT")
  printf "RDS %s: FreeMem=%s  CacheHit=%s%%  ReplicaLag=%sms  Threads=%s  IOPS=%s%%\n" \
    "$reader_label" "$rfreemem" "$rcache" "$rlag" "$rthreads" "$riops_pct"
  printf "               SelectLat=%sms\n" "$rsel_lat"
done

if jq -e '.redis_extended[0].curr_connections' "$OUTPUT" >/dev/null 2>&1; then
  r_conns=$(jq -r '.redis_extended[0].curr_connections.Average // "N/A"' "$OUTPUT")
  r_evict=$(jq -r '.redis_extended[0].evictions.Sum // 0' "$OUTPUT")
  r_hit=$(jq -r '.redis_extended[0].cache_hit_rate.Average // "N/A"' "$OUTPUT")
  printf "Redis:         Connections=%s  Evictions=%s  CacheHit=%s%%\n" "$r_conns" "$r_evict" "$r_hit"
fi

if jq -e '.network.network_rx_bytes' "$OUTPUT" >/dev/null 2>&1; then
  net_rx_mb=$(jq -r '.network.network_rx_bytes.Sum // "N/A" | if type == "number" then (. / 1048576 * 100 | round / 100 | tostring) + "MB" else . end' "$OUTPUT")
  net_tx_mb=$(jq -r '.network.network_tx_bytes.Sum // "N/A" | if type == "number" then (. / 1048576 * 100 | round / 100 | tostring) + "MB" else . end' "$OUTPUT")
  printf "Network:       RX=%s  TX=%s\n" "$net_rx_mb" "$net_tx_mb"
fi

if jq -e '.container_health' "$OUTPUT" >/dev/null 2>&1; then
  ch_stops=$(jq -r '.container_health.abnormal_stops // 0' "$OUTPUT")
  ch_running=$(jq -r '.container_health.running_tasks // 0' "$OUTPUT")
  ch_spread=$(jq -r '.container_health.start_spread_min // "N/A"' "$OUTPUT")
  ch_alert=$(jq -r '.container_health.start_spread_alert // false' "$OUTPUT")
  printf "Containers:    Running=%s  AbnormalStops=%s  StartSpread=%smin" "$ch_running" "$ch_stops" "$ch_spread"
  if [[ "$ch_alert" == "true" ]]; then
    printf "  ⚠ STAGGERED STARTS"
  fi
  printf "\n"
fi

echo '```'
echo ""
echo "## Threshold Checks"
echo ""
echo '```'

# ---------------------------------------------------------------------------
# Threshold alerts
#
# Based on expected values from the load test key metrics document.
# Thresholds are checked against the collected data and violations are
# printed as alerts.
#
# Thresholds:
#   Fleet Server CPU          < 80% avg
#   Fleet Server Memory       < 80% avg
#   RDS Writer CPU            < 80% avg
#   RDS Reader CPU            < 90% avg
#   RDS Writer Deadlocks      == 0
#   Redis CPU                 < 80% avg
#   Redis Memory              < 70% avg
#   Loadtest CPU              < 90% avg
#   Loadtest Memory           < 90% avg
#   Fleet Server Errors       == 0
#   IOPS Utilization          < 80% avg
#   Container Abnormal Stops   == 0
#   Container Start Spread    < 10 min
# ---------------------------------------------------------------------------
ALERT_COUNT=0

check_threshold() {
  local label="$1" jq_path="$2" op="$3" threshold="$4"
  local value
  value=$(jq -r "$jq_path" "$OUTPUT" 2>/dev/null)

  # Skip if value is null, N/A, or empty
  if [[ -z "$value" || "$value" == "null" || "$value" == "N/A" ]]; then
    return
  fi

  local failed=false
  case "$op" in
    lt)  failed=$(echo "$value $threshold" | awk '{print ($1 >= $2) ? "true" : "false"}') ;;
    eq)  failed=$(echo "$value $threshold" | awk '{print ($1 != $2) ? "true" : "false"}') ;;
    gt)  failed=$(echo "$value $threshold" | awk '{print ($1 > $2) ? "true" : "false"}') ;;
  esac

  if [[ "$failed" == "true" ]]; then
    case "$op" in
      lt) printf "  🔴 ALERT: %s = %s (expected < %s)\n" "$label" "$value" "$threshold" ;;
      eq) printf "  🔴 ALERT: %s = %s (expected %s)\n" "$label" "$value" "$threshold" ;;
      gt) printf "  🔴 ALERT: %s = %s (expected <= %s)\n" "$label" "$value" "$threshold" ;;
    esac
    ALERT_COUNT=$((ALERT_COUNT + 1))
  fi
}

# Standard thresholds
check_threshold "Fleet Server CPU avg"     '.fleet_server.cpu_utilization.Average'        lt 80
check_threshold "Fleet Server Memory avg"  '.fleet_server.memory_utilization.Average'     lt 80
check_threshold "RDS Writer CPU avg"       '.rds_writer.cpu_utilization.Average'          lt 80
check_threshold "RDS Writer Deadlocks"     '.rds_writer.deadlocks.Sum'                   eq 0
check_threshold "Redis CPU avg"            '.redis[0].cpu_utilization.Average'            lt 80
check_threshold "Redis Memory avg"         '.redis[0].memory_utilization.Average'         lt 70
check_threshold "Loadtest CPU avg"         '.loadtest_containers.cpu_utilization.Average'  lt 90
check_threshold "Loadtest Memory avg"      '.loadtest_containers.memory_utilization.Average' lt 90

# Check each reader
for reader_label in $(jq -r '.rds_readers[].instance // empty' "$OUTPUT" 2>/dev/null); do
  check_threshold "RDS $reader_label CPU avg" \
    ".rds_readers[] | select(.instance==\"$reader_label\") | .cpu_utilization.Average" lt 90
  check_threshold "RDS $reader_label IOPS Utilization" \
    ".rds_readers_extended[] | select(.instance==\"$reader_label\") | .iops_utilization.utilization_pct" lt 80
done

check_threshold "Fleet Server Errors"      '.fleet_server_errors.error_count' eq 0
check_threshold "IOPS Utilization"         '.rds_writer_extended.iops_utilization.utilization_pct' lt 80
check_threshold "Container Abnormal Stops"  '.container_health.abnormal_stops' eq 0
check_threshold "Container Start Spread (min)" '.container_health.start_spread_min' lt 10

# Threads running vs vCPU count — dynamic threshold per instance
# Average active sessions should be <= vCPUs. Above that means CPU saturation.
writer_vcpus=$(jq -r '.rds_writer_extended.threads_running.vcpus // 0' "$OUTPUT")
if [[ "$writer_vcpus" -gt 0 ]]; then
  check_threshold "RDS Writer Threads Running (vCPUs=$writer_vcpus)" \
    '.rds_writer_extended.threads_running.average' lt "$writer_vcpus"
fi

for reader_label in $(jq -r '.rds_readers_extended[].instance // empty' "$OUTPUT" 2>/dev/null); do
  reader_vcpus=$(jq -r --arg i "$reader_label" '.rds_readers_extended[] | select(.instance==$i) | .threads_running.vcpus // 0' "$OUTPUT")
  if [[ "$reader_vcpus" -gt 0 ]]; then
    check_threshold "RDS $reader_label Threads Running (vCPUs=$reader_vcpus)" \
      ".rds_readers_extended[] | select(.instance==\"$reader_label\") | .threads_running.average" lt "$reader_vcpus"
  fi
done

if [[ $ALERT_COUNT -eq 0 ]]; then
  echo "  ✅ All metrics within expected thresholds"
else
  echo ""
  printf "  ⚠ %d alert(s) detected\n" "$ALERT_COUNT"
fi
echo '```'
} | tee "$MD_OUTPUT"

echo ""
echo "Synopsis written to: $MD_OUTPUT"
