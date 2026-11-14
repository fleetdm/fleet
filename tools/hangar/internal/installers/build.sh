#!/usr/bin/env bash
# Builds a server's fleetd installers for Hangar's Installers tab: one
# `fleetctl package` per job, all jobs at once, continuing past failures.
# Hangar writes this script to its data dir and runs it through the process
# engine, passing:
#
#   HANGAR_FLEETCTL     the server's own fleetctl binary
#   HANGAR_JOBS         one job per line: key<TAB>type<TAB>arch<TAB>outfile
#                       (arch "-" = none, for the universal macOS pkg)
#   HANGAR_COMMON_ARGS  fleetctl package flags shared by every job, one per line
#
# The "==> [key] building|ok|failed" lines are parsed by the tab for per-platform
# status; keep their format in sync with lib/installers.ts. A job's own output
# is prefixed "[key] " so the interleaved log stays readable.
#
# Runs under macOS's bash 3.2: no `wait -n`, no associative arrays.
set -uo pipefail

# Jobs build in a hidden folder next to the packages rather than in $TMPDIR, so
# moving a finished package into place is a rename: Stop can't cut it short the
# way it could a copy from another volume.
out_dir=$(dirname "$(cut -f 4 <<< "$HANGAR_JOBS" | head -n 1)")
# Left by a build that was killed before it could clean up.
rm -rf "$out_dir"/.hangar-build.*
work_root=$(mktemp -d "$out_dir/.hangar-build.XXXXXX") || exit 1
# Stop sends SIGTERM to the whole group, every job's fleetctl included; exit
# through the EXIT trap so the jobs' work dirs are removed.
trap 'rm -rf "$work_root"' EXIT
trap 'exit 143' TERM INT

common=()
while IFS= read -r a; do
  [ -n "$a" ] && common+=("$a")
done <<< "$HANGAR_COMMON_ARGS"

tag() {
  local line
  while IFS= read -r line || [ -n "$line" ]; do
    printf '[%s] %s\n' "$1" "$line"
  done
}

# build runs one job into its own empty dir and only replaces $out on success,
# so a failed run never leaves (or is mistaken for) an old file. The dir is also
# the cwd: some fleetctl versions ignore --outfile for arm64 .msi and write
# their default name there instead.
build() {
  local key=$1 type=$2 arch=$3 out=$4 work built
  local arch_flag=()
  [ "$arch" != "-" ] && arch_flag=(--arch="$arch")
  echo "==> [$key] building $(basename "$out")"
  if ! work=$(mktemp -d "$work_root/$key.XXXXXX"); then
    echo "==> [$key] failed"
    return 1
  fi
  if ! (cd "$work" && "$HANGAR_FLEETCTL" package --type="$type" "${arch_flag[@]+"${arch_flag[@]}"}" --outfile="$work/$(basename "$out")" "${common[@]}" </dev/null) 2>&1 | tag "$key"; then
    echo "==> [$key] failed"
    return 1
  fi
  built=$(find "$work" -maxdepth 1 -type f -name "*.$type" | head -n 1)
  if [ -z "$built" ] || ! mv -f "$built" "$out"; then
    echo "[$key] fleetctl reported success but wrote no $type file"
    echo "==> [$key] failed"
    return 1
  fi
  # fleetctl writes 0600; make it readable for sharing.
  chmod 644 "$out"
  echo "==> [$key] ok"
}

pids=()
# Jobs are read on fd 3 so nothing a job runs can swallow them from stdin.
while IFS=$'\t' read -r key type arch out <&3; do
  [ -z "$key" ] && continue
  build "$key" "$type" "$arch" "$out" 3<&- &
  pids+=("$!")
done 3<<< "$HANGAR_JOBS"

failed=0
for pid in "${pids[@]+"${pids[@]}"}"; do
  wait "$pid" || failed=$((failed + 1))
done
echo "==> done: $((${#pids[@]} - failed)) of ${#pids[@]} built"
[ "$failed" -eq 0 ]
