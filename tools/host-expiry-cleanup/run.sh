#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 <ssh-target> <absolute-remote-config-path> [--sudo] [--estimate] [--apply] [--serial-only] [--max-rounds N] [--workers N] [--bucket-days N]" >&2
  exit 2
}

[[ $# -ge 2 ]] || usage
ssh_target=$1
remote_config=$2
shift 2
use_sudo=false
apply_arg=
max_rounds_arg=
estimate_arg=
serial_only_arg=
workers_arg=
bucket_days_arg=
while [[ $# -gt 0 ]]; do
  case "$1" in
    --sudo) use_sudo=true; shift ;;
    --estimate) [[ -z "$estimate_arg" ]] || usage; estimate_arg=--estimate; shift ;;
    --serial-only) [[ -z "$serial_only_arg" ]] || usage; serial_only_arg=--serial-only; shift ;;
    --apply) [[ -z "$apply_arg" ]] || usage; apply_arg=--apply; shift ;;
    --max-rounds)
      [[ $# -ge 2 && "$2" =~ ^[1-9][0-9]*$ && -z "$max_rounds_arg" ]] || usage
      max_rounds_arg="--max-rounds $2"
      shift 2
      ;;
    --workers)
      [[ $# -ge 2 && "$2" =~ ^[1-9][0-9]*$ && -z "$workers_arg" ]] || usage
      workers_arg="--workers $2"
      shift 2
      ;;
    --bucket-days)
      [[ $# -ge 2 && "$2" =~ ^[1-9][0-9]*$ && -z "$bucket_days_arg" ]] || usage
      bucket_days_arg="--bucket-days $2"
      shift 2
      ;;
    *) usage ;;
  esac
done
[[ "$ssh_target" != -* ]] || {
  echo "SSH target must not start with '-'." >&2
  exit 2
}
[[ "$remote_config" =~ ^/[A-Za-z0-9_./-]+$ ]] || {
  echo "Remote config path must be absolute and contain only letters, digits, /, ., _, or -." >&2
  exit 2
}

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd "$script_dir/../.." && pwd)
local_binary="${TMPDIR:-/tmp}/host-expiry-cleanup-linux-amd64-$$"
remote_binary=/tmp/host-expiry-cleanup
trap 'rm -f "$local_binary"' EXIT

cd "$repo_root"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "$local_binary" ./tools/host-expiry-cleanup
scp "$local_binary" "$ssh_target:$remote_binary"
remote_runner=$remote_binary
if [[ "$use_sudo" == true ]]; then
  remote_runner="sudo -n $remote_binary"
fi
ssh "$ssh_target" "chmod 700 $remote_binary && exec $remote_runner --config $remote_config $estimate_arg $serial_only_arg $apply_arg $max_rounds_arg $workers_arg $bucket_days_arg"
