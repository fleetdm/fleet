# Commands and mechanics

Everything here is `gh` or the Bugbop plugin script. Writes that notify a reporter (Bugbop comments and status changes, advisory comments) are listed last on purpose: draft them, read them once more, then send.

## Dedup sweep

`gh search issues` only matches titles reliably. Dump every issue and grep the bodies.

```bash
# Private trackers (needs maintainer access to both repos)
gh issue list --repo fleetdm/security --state all --limit 500 \
  --json number,title,state,createdAt,closedAt,labels,milestone,body,comments > /tmp/sec.json
# confidential has more than 1000 issues: dump the recent 1000 (any label) plus every `~security`-labeled issue
gh issue list --repo fleetdm/confidential --state all --limit 1000 \
  --json number,title,state,createdAt,closedAt,labels,body > /tmp/conf.json
gh issue list --repo fleetdm/confidential --state all --limit 1000 --label "~security" \
  --json number,title,state,createdAt,closedAt,labels,body > /tmp/conf_sec.json

# Grep bodies and comments for exact names from the report: endpoint path, query parameter, Go function
python3 - "<term1>" "<term2>" <<'EOF'
import json,re,sys
terms=[re.escape(t) for t in sys.argv[1:]]
seen=set()
for f,repo in (("/tmp/sec.json","security"),("/tmp/conf.json","confidential"),("/tmp/conf_sec.json","confidential")):
    for i in json.load(open(f)):
        if (repo,i["number"]) in seen: continue
        seen.add((repo,i["number"]))
        hay=i["title"]+"\n"+(i.get("body") or "")+"\n"+"\n".join(c["body"] for c in i.get("comments",[]))
        if any(re.search(t,hay,re.I) for t in terms):
            ms=(i.get("milestone") or {}).get("title") or ""
            print(f"{repo}#{i['number']} [{i['state']}] {i['title'][:90]} | {i['createdAt'][:10]} {ms}")
EOF

# Advisories in every state
for s in triage draft published closed; do
  gh api "/repos/fleetdm/fleet/security-advisories?state=$s&per_page=100" \
    -q ".[] | \"$s \(.ghsa_id) \(.severity) \(.created_at[:10]) \(.author.login // \"fleet\") | \(.summary)\""
done | grep -iE "<term1>|<term2>"

# Bugbop, both programs, all statuses
python3 "$BUGBOP" GET /reports --all --summary | grep -iE "<term1>|<term2>"

# Advisory running list (shipped fixes waiting on an advisory). `gh search` doesn't find it; use the number.
gh issue view 17937 --repo fleetdm/confidential --json title,state,body,comments | grep -iE "<term1>|<term2>"
```

The dedup rule: same root cause and same sink means duplicate, even with a different identifier or endpoint. Same symptom through a different code path is a new issue that cross-references the old one.

## Verify against main

For a server report, read the handler, the authz policy, and the datastore query. For a UI report (XSS, injection into rendered content), `git grep` the JSON key under `frontend/` and check each render site: JSX text or attribute is escaped by React; `dangerouslySetInnerHTML` and server-side `text/template` are not (`html/template` is). Note that the session token is a non-HttpOnly cookie and CSP is opt-in, so a real unescaped sink is a session-theft primitive.

```bash
git fetch -q origin main
git grep -n "<function or route>" origin/main -- server/ ee/
git log --format='%h %ad %s' --date=short -S "<guard or check text>" origin/main -- <file>   # when was it added or removed
# First tag whose file contains the fix. Backports are cherry-picks with new SHAs, so check file contents, not ancestry, and include patch tags.
git fetch -q --tags origin
for t in $(git tag -l 'fleet-v4.*' | sort -V); do git show "${t}:<file>" 2>/dev/null | grep -q '<marker from the fix>' && echo "in $t"; done | head -5
```

Write `${t}:path`, not `$t:path`; zsh reads `$t:s` as a history modifier.

Verified-on-main means you read the current code at a named commit. Running a local server is better when you can.

## File the issue

`gh issue create` can't fill an issue form, so render the form's sections yourself. Start from `issue-template.md`, keep every `###` heading, then:

```bash
gh issue create --repo fleetdm/security \
  --title "Report [High]: <what the attacker gets, via what>" \
  --label security --label p2 --label "#g-orchestration" \
  --assignee <em-github-login> \
  --body-file /tmp/report.md
```

Then the boards:

```bash
# Group board (number from the product groups handbook page)
gh project item-add <board-number> --owner fleetdm --url https://github.com/fleetdm/security/issues/<n>

# :help-security board (113): the `security` label adds the card; set it to Ready
gh project item-list 113 --owner fleetdm --format json --limit 500 \
  | python3 -c 'import sys,json; [print(i["id"]) for i in json.load(sys.stdin)["items"] if i.get("content",{}).get("number")==<n> and i["content"].get("repository","").endswith("/security")]'
gh project item-edit --project-id PVT_kwDOBDAnic4BgFfY --id <item-id> \
  --field-id "$(gh project field-list 113 --owner fleetdm --format json | python3 -c 'import sys,json; print(next(f["id"] for f in json.load(sys.stdin)["fields"] if f["name"]=="Status"))')" \
  --single-select-option-id 64ac26b5
```

If `gh project` fails with a scope error: `gh auth refresh -s project -h github.com`.

Product group board numbers and EMs are on the [product groups page](https://fleetdm.com/handbook/company/product-groups); look them up rather than trusting a copy. If the page doesn't say which group owns the affected surface, use the `CODEOWNERS` entry for the files involved. If that is also silent, ask the CTO rather than guessing.

## GitHub security advisory submissions

```bash
# Read one
gh api /repos/fleetdm/fleet/security-advisories/<ghsa> -q '{summary,severity,cvss,state,credits,vulnerabilities,description}'

# Accept it and set everything in one PATCH (severity OR cvss_vector_string, never both)
cat > /tmp/adv.json <<'EOF'
{"state":"draft",
 "cvss_vector_string":"CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:L/I:N/A:N",
 "cwe_ids":["CWE-200"],
 "credits":[{"login":"<reporter-login>","type":"reporter"}],
 "vulnerabilities":[{"package":{"ecosystem":"go","name":"github.com/fleetdm/fleet"},
                     "vulnerable_version_range":"<= 4.94.0","patched_versions":""}],
 "description":"<templated description, see any recently published fleetdm/fleet advisory for the sections>"}
EOF
gh api -X PATCH /repos/fleetdm/fleet/security-advisories/<ghsa> --input /tmp/adv.json

# Reject a submission that isn't a vulnerability or duplicates an accepted one (comment in the UI first)
gh api -X PATCH /repos/fleetdm/fleet/security-advisories/<ghsa> -f state=closed
gh api /repos/fleetdm/fleet/security-advisories/<ghsa> -q '{ghsa_id,state,severity}'   # PATCH can silently no-op; always re-read
```

Rules:
- Credits need a GitHub login. A reporter without one is thanked by name in the Credits text.
- Patched version stays empty until the fix ships. Fill it in at release.
- Comments on advisories have no API. Open the advisory page signed in and use the comment box.
- Publishing and "Request CVE" are the CTO's. Leave the advisory as a draft.

## Bugbop

Reads are free. Writes notify the hunter; one at a time, and verify each with a read.

```bash
BUGBOP=$(find ~/.claude/plugins -path '*bugbop/scripts/bugbop.py' | head -1)   # from the bugbop:bugbop plugin
python3 "$BUGBOP" GET /reports/<id>                       # includes internal_notes, suspected_duplicate_of_id
python3 "$BUGBOP" GET /reports/<id>/comments --summary
python3 "$BUGBOP" POST /reports/<id>/comments --data '{"content":"..."}'
python3 "$BUGBOP" PATCH /reports/<id> --data '{"status":"open","severity":"medium","internal_notes":"Tracked via https://github.com/fleetdm/security/issues/<n>"}'
python3 "$BUGBOP" POST /reports/<id>/close_as_duplicate   # same program only; uses suspected_duplicate_of_id
```

The script needs `BUGBOP_TOKEN` in the environment; never paste the token into chat.

A report whose bug was fixed and shipped before it arrived, with the canonical tracker outside Bugbop (email disclosure, advisory submission), can't be closed as `duplicate`. Set `not_applicable`, severity to Fleet's rating of the underlying bug, and an internal note naming the tracker and advisory. Use the "Fixed before the report" reply.

Bugbop's "Suspected duplicate" banner matches titles, not root causes. Two reports with similar titles can be different bugs. Read both before clicking.

The web UI works when the API is unavailable. The comment composer is a rich-text editor, so paste or type into it rather than setting a field programmatically.
