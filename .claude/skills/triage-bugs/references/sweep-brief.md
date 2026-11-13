# Fixed-by-PR sweep


## Gather

Run from the scratchpad. `$FLEET` is the fleet repo checkout.

```bash
git -C "$FLEET" fetch origin main --quiet

# Merged PRs since the oldest Inbox bug. gh caps a search at 1,000 results,
# so query in date ranges and split any range that comes back at 1000.
for r in 2026-06-01..2026-06-30 2026-07-01..2026-07-31; do   # extend to today
  gh pr list -R fleetdm/fleet --state merged --search "merged:$r" --limit 1000 \
    --json number,title,mergedAt,baseRefName,body,author > "prs_${r%%..*}.json" &
done; wait
for f in prs_*.json; do echo "$f $(jq length "$f")"; done
jq -s 'add | unique_by(.number) | sort_by(.mergedAt)' prs_*.json > prs_raw.json
jq -r '.[] | "#\(.number)\t\(.mergedAt[0:10])\t\(.baseRefName)\t\(.author.login)\t\(.title)"' prs_raw.json > prs_titles.tsv
jq -r '.[] | "=== #\(.number) \(.mergedAt[0:10]) base=\(.baseRefName) \(.title)\n\((.body // "")[0:1500])\n"' prs_raw.json > prs_bodies.txt

# Timeline cross-references per bug
for n in $(cat inbox.txt); do
  gh api graphql -f query="query{repository(owner:\"fleetdm\",name:\"fleet\"){issue(number:$n){number title state timelineItems(first:100,itemTypes:[CROSS_REFERENCED_EVENT,CONNECTED_EVENT,CLOSED_EVENT]){nodes{__typename ... on CrossReferencedEvent{source{__typename ... on PullRequest{number title state mergedAt baseRefName} ... on Issue{number title state}}} ... on ConnectedEvent{subject{__typename ... on PullRequest{number title state mergedAt}}} ... on ClosedEvent{closer{__typename ... on PullRequest{number title}}}}}}}}" > "issue_$n.json"
done
```

Zsh doesn't word-split `$var`. Use an array or `${=var}` when looping over a space-separated list.


## Subagent brief

Split the Inbox into groups of about 10 bugs, one read-only agent each, all launched in one turn. Fill in the paths and the bug list.

```text
Task: has a merged or open PR addressed each of these open bug issues?

READ-ONLY. Don't comment, label, move, assign, or edit anything on GitHub. Don't check out branches or edit repo files.
Repo: <FLEET>. origin/main is freshly fetched. Inspect code with `git show origin/main:<path>`, `git grep <pat> origin/main`, and `git log origin/main`.

Scratchpad: <SCRATCHPAD>
- prs_titles.tsv: every PR merged in the window (number, date, base branch, author, title)
- prs_bodies.txt: the same PRs with the first 1,500 characters of each body
- issue_<N>.json: timeline cross-references for each issue

For each issue:
1. Read it: `gh issue view N -R fleetdm/fleet --json title,body,comments`.
2. Check its cross-references, then search prs_titles.tsv and prs_bodies.txt for its number, key error strings, and feature names.
3. For a candidate PR, confirm the fix is on origin/main. A PR merged to a feature branch or a release branch alone doesn't count as fixed on main. Read the code path the bug describes. A PR that fixes a neighboring bug doesn't count.
4. Check whether the PR covers every part of the bug, or only some of it.

Return one block per issue:
#N <title>
VERDICT: FIXED | IN FLIGHT | NOT ADDRESSED [confidence high|medium|low]
PR: #NNNNN (merged YYYY-MM-DD to <base>) or none
EVIDENCE: one or two lines, with the file and function on origin/main
GAPS: any part of the bug the PR doesn't cover
```


## Verdicts

- **FIXED, high or medium, no gaps:** draft "I think this is fixed in #NNNNN. Need to confirm."
- **FIXED with gaps:** don't draft the fixed comment. Classify the rest of the bug normally, and name the PR in whatever you draft.
- **IN FLIGHT:** list it with its open PR in the report.
- **Low confidence:** treat it as NOT ADDRESSED and mention the candidate PR in one line.

A live check on the local Fleet instance is optional. Do it only when the user asks, and restore the instance to its starting state afterward.
