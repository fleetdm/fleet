#!/bin/bash

set -euo pipefail
shopt -s nullglob

usage() {
  echo "Usage: $0 -u FLEET_URL -t TEAM_ID -k API_TOKEN -f FOLDERPATH"
  exit 1
}

while getopts "u:t:k:f:" opt; do
  case ${opt} in
    u ) API_URL="$OPTARG" ;;
    t ) TEAM_ID="$OPTARG" ;;
    k ) API_TOKEN="$OPTARG" ;;
    f ) FOLDER="$OPTARG" ;;
    * ) usage ;;
  esac
done

if [[ -z "${API_URL:-}" || -z "${TEAM_ID:-}" || -z "${API_TOKEN:-}" || -z "${FOLDER:-}" ]]; then
  usage
fi

ENDPOINT="$API_URL/api/v1/fleet/software/package"
found_files=false

files=("$FOLDER"/*)
[ ${#files[@]} -eq 0 ] && echo "⚠️ No files found in '$FOLDER'" && exit 1

for file in "${files[@]}"; do
  [ -f "$file" ] || continue  # Skip directories, symlinks, etc.
  found_files=true
  echo "🔼 Uploading: $file"

  tmp_body=$(mktemp)
  tmp_err=$(mktemp)

  ext="${file##*.}"

  CURL_ARGS=(
    -s -k -w "%{http_code}" -o "$tmp_body"
    -X POST "$ENDPOINT"
    -H "Authorization: Bearer $API_TOKEN"
    -F "software=@${file}"
    -F "team_id=$TEAM_ID"
  )

  if [[ "$ext" == "exe" ]]; then
    CURL_ARGS+=(
      -F "install_script=exit 0"
      -F "uninstall_script=exit 0"
    )
  fi

  http_status=$(curl "${CURL_ARGS[@]}" 2>"$tmp_err")
  curl_exit=$?

  if [[ $curl_exit -ne 0 ]]; then
    echo "❌ curl transport error (exit code $curl_exit)"
    echo "stderr:"
    cat "$tmp_err"
  elif [[ "$http_status" =~ ^2 ]]; then
    echo "✅ Success ($http_status)"
  else
    echo "❌ Upload failed for $file (HTTP $http_status)"
    echo "Response body:"
    cat "$tmp_body"
  fi

  rm -f "$tmp_body" "$tmp_err"
done

if ! $found_files; then
  echo "⚠️ No supported installer files found in '$FOLDER'"
fi

