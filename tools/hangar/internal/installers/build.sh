#!/usr/bin/env bash
# Builds a server's fleetd installers for Hangar's Installers tab: one
# `fleetctl package` per job, continuing past failures. Hangar writes this
# script to its data dir and runs it through the process engine, passing:
#
#   HANGAR_FLEETCTL     the server's own fleetctl binary
#   HANGAR_JOBS         one job per line: key<TAB>type<TAB>arch<TAB>outfile
#                       (arch "-" = none, for the universal macOS pkg)
#   HANGAR_COMMON_ARGS  fleetctl package flags shared by every job, one per line
#
# The "==> [key] building|ok|failed" lines are parsed by the tab for per-platform
# status; keep their format in sync with lib/installers.ts.
set -u

work=""
# Stop sends SIGTERM to the group; exit through the EXIT trap so the
# in-progress job's temp dir is removed.
trap 'rm -rf "${work:-}"' EXIT
trap 'exit 143' TERM INT

common=()
while IFS= read -r a; do
  [ -n "$a" ] && common+=("$a")
done <<< "$HANGAR_COMMON_ARGS"

total=0
failed=0
# Jobs are read on fd 3 so fleetctl can't swallow them from stdin.
while IFS=$'\t' read -r key type arch out <&3; do
  [ -z "$key" ] && continue
  total=$((total + 1))
  echo "==> [$key] building $(basename "$out")"
  arch_flag=()
  [ "$arch" != "-" ] && arch_flag=(--arch="$arch")
  # Each job builds into its own empty dir and only replaces $out on success,
  # so a failed run never leaves (or is mistaken for) an old file. The dir is
  # also the cwd: some fleetctl versions ignore --outfile for arm64 .msi and
  # write their default name there instead.
  work=$(mktemp -d)
  if (cd "$work" && "$HANGAR_FLEETCTL" package --type="$type" "${arch_flag[@]+"${arch_flag[@]}"}" --outfile="$work/$(basename "$out")" "${common[@]}" </dev/null); then
    built=$(find "$work" -maxdepth 1 -type f -name "*.$type" | head -n 1)
    if [ -n "$built" ] && mv -f "$built" "$out"; then
      # fleetctl writes 0600; make it readable for sharing.
      chmod 644 "$out"
      echo "==> [$key] ok"
    else
      echo "fleetctl reported success but wrote no $type file"
      echo "==> [$key] failed"
      failed=$((failed + 1))
    fi
  else
    echo "==> [$key] failed"
    failed=$((failed + 1))
  fi
  rm -rf "$work"
  work=""
done 3<<< "$HANGAR_JOBS"

echo "==> done: $((total - failed)) of $total built"
[ "$failed" -eq 0 ]
