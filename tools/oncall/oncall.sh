#!/usr/bin/env bash
# Script for on-call use.
# Formatted with shfmt. See https://github.com/mvdan/sh

set -euo pipefail

usage() {
	cat <<EOF
Contains useful commands for on-call.

Usage:
    $(basename "$0") <command>

Commands:
    issues  List open issues from outside contributors.
    prs     List open prs from outside contributors.
EOF
}

require() {
	type "$1" >/dev/null 2>&1 || {
		echo "$1 is required but not installed. Aborting." >&2
		exit 1
	}
}

issues() {
	require gh
	require jq

	auth_status="$(gh auth status -t 2>&1)"
	username="$(echo "${auth_status}" | sed -n -r 's/^.* Logged in to github.com account ([^[:space:]]+).*/\1/p')"
	token="$(echo "${auth_status}" | sed -n -r 's/^.*Token: ([a-zA-Z0-9_]*)/\1/p')"

	members="$(curl -s -u "${username}:${token}" https://api.github.com/orgs/fleetdm/members?per_page=100 | jq -r 'map(.login)')"

	gh issue list --repo fleetdm/fleet --json id,title,author,url,createdAt,labels --limit 100 |
		jq -r --argjson members "$members" \
			'map(select(.author.login as $in | $members | index($in) | not)) | sort_by(.createdAt) | reverse'
}

prs() {
	require gh
	require jq

	auth_status="$(gh auth status -t 2>&1)"
	username="$(echo "${auth_status}" | sed -n -r 's/^.* Logged in to github.com account ([^[:space:]]+).*/\1/p')"
	token="$(echo "${auth_status}" | sed -n -r 's/^.*Token: ([a-zA-Z0-9_]*)/\1/p')"

	members="$(curl -s -u "${username}:${token}" https://api.github.com/orgs/fleetdm/members?per_page=100 | jq -r 'map(.login)' | jq '. += ["app/dependabot"]')"

	# defaults to listing open prs
	gh pr list --repo fleetdm/fleet --json id,title,author,url,createdAt |
		jq -r --argjson members "$members" \
			'map(select(.author.login as $in | $members | index($in) | not)) | sort_by(.createdAt) | reverse'
}

# main script
case "$1" in
issues)
	issues
	;;
prs)
	prs
	;;
-h | --help)
	usage
	exit 0
	;;
*)
	echo "Invalid argument: $1"
	usage
	exit 1
	;;
esac
