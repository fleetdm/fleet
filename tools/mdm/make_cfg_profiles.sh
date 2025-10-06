#!/usr/bin/env bash

# ----------------------------------------------------
# Build a JSON payload for configuration profiles batch request payload.
# ----------------------------------------------------

set -euo pipefail

# Repeating block:
#   --file <path> --name <display> [--labels-type include_any|include_all|exclude_any] [--label <str> ...] --next
# Finish with --next or just end of args.

usage() {
  echo "Usage: $0 (--file F --name N [--labels-type include_any|include_all|exclude_any] [--label L ...] --next)..." >&2
  exit 1
}

profiles='[]'

cur_file=""
cur_name=""
cur_ltype=""
cur_labels=()

flush_item() {
  [[ -n "${cur_file:-}" || -n "${cur_name:-}" || ${#cur_labels[@]} -gt 0 || -n "${cur_ltype:-}" ]] || return 0
  [[ -n "${cur_file:-}" ]] || { echo "Missing --file before --next/end." >&2; exit 1; }
  [[ -f "$cur_file" ]] || { echo "No such file: $cur_file" >&2; exit 1; }

  b64="$(base64 < "$cur_file" | tr -d '\n')"

  # labels array -> JSON
  labels_json="$(printf '%s\n' "${cur_labels[@]:-}" | jq -R . | jq -s .)"

  # choose labels key
  lkey=""
  case "${cur_ltype:-}" in
    include_any)  lkey="labels_include_any" ;;
    include_all)  lkey="labels_include_all" ;;
    exclude_any)  lkey="labels_exclude_any" ;;
    "" )          lkey="" ;; # omit labels entirely if not provided
    * ) echo "Invalid --labels-type: $cur_ltype" >&2; exit 1 ;;
  esac

  # base object
  item="$(jq -n --arg p "$b64" '{profile:$p}')"

  # add display_name only if provided
  if [[ -n "${cur_name:-}" ]]; then
    item="$(jq --arg n "$cur_name" '. + {display_name:$n}' <<<"$item")"
  fi

  # add labels if provided
  if [[ -n "$lkey" ]]; then
    item="$(jq --arg lk "$lkey" --argjson lv "$labels_json" '. + {($lk):$lv}' <<<"$item")"
  fi

  profiles="$(jq --argjson it "$item" '. + [$it]' <<<"$profiles")"

  # reset block
  cur_file=""; cur_name=""; cur_ltype=""; cur_labels=()
}

[[ $# -gt 0 ]] || usage
while [[ $# -gt 0 ]]; do
  case "$1" in
    --file)        shift; [[ $# -gt 0 ]] || usage; cur_file="$1"; shift ;;
    --name|--display-name) shift; [[ $# -gt 0 ]] || usage; cur_name="$1"; shift ;;
    --labels-type) shift; [[ $# -gt 0 ]] || usage; cur_ltype="$1"; shift ;;
    --label)       shift; [[ $# -gt 0 ]] || usage; cur_labels+=("$1"); shift ;;
    --next)        shift; flush_item ;;
    -h|--help)     usage ;;
    *) echo "Unknown arg: $1" >&2; usage ;;
  esac
done
flush_item

jq -n --argjson arr "$profiles" '{configuration_profiles: $arr}'
