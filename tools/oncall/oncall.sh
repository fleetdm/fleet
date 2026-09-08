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
    issues      List open issues from outside contributors.
    prs [-v]    List open prs from outside contributors.
                -v also lists the linked issue's labels, the assignee and the pr link.
EOF
}

require() {
	type "$1" >/dev/null 2>&1 || {
		echo "$1 is required but not installed. Aborting." >&2
		exit 1
	}
}

ensure_gh_auth() {
    auth_status="$(gh auth status -t 2>&1 || true)"
    if echo "$auth_status" | grep -q "You are not logged into any GitHub hosts."; then
        echo "$auth_status" >&2
        exit 1
    fi
    username="$(echo "${auth_status}" | sed -n -r 's/^.* Logged in to github.com account ([^[:space:]]+).*/\1/p')"
    token="$(echo "${auth_status}" | sed -n -r 's/^.*Token: ([a-zA-Z0-9_]*)/\1/p')"
    if [ -z "${username}" ] || [ -z "${token}" ]; then
        echo "Failed to parse GitHub auth status. Try: gh auth login" >&2
        exit 1
    fi
}

issues() {
	require gh
	require jq

	ensure_gh_auth

	members="$(curl -s -u "${username}:${token}" https://api.github.com/orgs/fleetdm/members?per_page=100 | jq -r 'map(.login)')"

	gh issue list --repo fleetdm/fleet --json id,title,author,url,createdAt,labels --limit 100 |
		jq -r --argjson members "$members" \
			'map(select(.author.login as $in | $members | index($in) | not)) | sort_by(.createdAt) | reverse | .[] | [(.url | split("/") | last), .createdAt, .author.login, .title] | @tsv'
}

prs() {
	require gh
	require jq

	ensure_gh_auth

	verbose=""
	if [ "${1:-}" = "-v" ] || [ "${1:-}" = "--verbose" ]; then
		verbose="yes"
	elif [ -n "${1:-}" ]; then
		echo "Invalid argument for prs: $1"
		usage
		exit 1
	fi

	members="$(curl -s -u "${username}:${token}" https://api.github.com/orgs/fleetdm/members?per_page=100 | jq -r 'map(.login) + ["app/dependabot", "app/kiloconnect", "app/kilo-code-bot"]')"

	# the issue column comes from GitHub's linking keywords, the tested column from
	# the manual QA checkboxes. html comments are stripped so the hints left in the
	# pr template do not count as either.
	defs='def pad($n): . + ((" " * ($n - length)) // "");
		def body_text: (.body // "") | gsub("<!--.*?-->"; ""; "m");
		def linked_issue:
			[body_text | scan("(?i)\\b(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\\s*:?\\s*(?:https://github\\.com/fleetdm/fleet/issues/|fleetdm/fleet#|#)([0-9]+)\\b")]
			| if length > 0 then .[0][0] else "" end;
		def manually_tested:
			if body_text | test("(?i)[-*]\\s*\\[x\\][^\\n]*(?:QA.?d all new/changed functionality|Attached a screenshot or screen recording)")
			then "tested"
			else ""
			end;
		def open_prs:
			map(select((.author.login as $login | ($members | index($login)) == null) and .isDraft == false))
			| sort_by(.createdAt)
			| reverse;'

	# defaults to listing open prs
	prs_json="$(gh pr list --limit 1000 --repo fleetdm/fleet --json id,title,author,url,createdAt,isDraft,body,assignees)"

	# labels are not part of the pr, so verbose mode looks up each linked issue
	issue_labels="{}"
	if [ -n "$verbose" ]; then
		for issue in $(jq -r --argjson members "$members" "$defs"'open_prs | map(linked_issue) | unique | .[] | select(. != "")' <<<"$prs_json"); do
			labels="$(gh issue view "$issue" --repo fleetdm/fleet --json labels -q '[.labels[].name] | join(", ")' || true)"
			issue_labels="$(jq --arg issue "$issue" --arg labels "$labels" '. + {($issue): $labels}' <<<"$issue_labels")"
		done
	fi

	jq -r --argjson members "$members" --argjson issue_labels "$issue_labels" --arg verbose "$verbose" "$defs"'
		[open_prs
		| .[]
		| [(.url | split("/") | last), .createdAt, linked_issue, manually_tested, .author.login]
		+ (if $verbose == "" then [] else [([.assignees[].login] | join(", ")), .url] end)
		+ [(.title | gsub("[\\r\\n\\t]"; " "))]
		+ (if $verbose == "" then [] else [($issue_labels[linked_issue] // "")] end)]
		| (transpose | map(map(length) | max)) as $widths
		| .[]
		| [range(0; length) as $i | (.[$i] | pad($widths[$i]))]
		| join("  ")
		| sub(" +$"; "")' <<<"$prs_json"
}

# check for at least one argument
if [ "$#" -lt 1 ]; then
	echo -e "No command provided.\n"
	usage
	exit 1
fi

# main script
case "$1" in
issues)
	issues
	;;
prs)
	prs "${2:-}"
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
