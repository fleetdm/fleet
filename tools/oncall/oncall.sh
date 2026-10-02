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
    prs [-v|-s] List open prs from outside contributors.
                -v also lists the linked issue's labels, the assignee and the pr link.
                -s prints one Slack mrkdwn line per pr, with links and short dates.
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
	slack=""
	if [ "${1:-}" = "-v" ] || [ "${1:-}" = "--verbose" ]; then
		verbose="yes"
	elif [ "${1:-}" = "-s" ] || [ "${1:-}" = "--slack" ]; then
		slack="yes"
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

	if [ -n "$slack" ]; then
		# titles are untrusted. escaping &, < and > stops them from injecting mentions
		# or links, and replacing backticks stops them from opening a code block.
		jq -r --argjson members "$members" "$defs"'
			def slack_escape: gsub("&"; "&amp;") | gsub("<"; "&lt;") | gsub(">"; "&gt;") | gsub("`"; "'"'"'");
			def short_date:
				(now | gmtime | .[0]) as $year
				| .createdAt | fromdateiso8601 | gmtime
				| (["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"][.[1]] + " " + (.[2] | tostring))
				+ (if .[0] == $year then "" else ", " + (.[0] | tostring) end);
			open_prs
			| .[]
			| linked_issue as $issue
			| [
				"<\(.url)|#\(.url | split("/") | last)>",
				short_date,
				(if $issue == "" then "no issue" else "issue <https://github.com/fleetdm/fleet/issues/\($issue)|#\($issue)>" end),
				(if manually_tested == "" then "not tested" else "tested" end),
				.author.login,
				(.title | gsub("[\\r\\n\\t]"; " ") | slack_escape)
			]
			| join(" · ")' <<<"$prs_json"
		return
	fi

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
