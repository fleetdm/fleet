# Commands and mechanics

Everything here is `gh` or the Bugbop plugin script. Writes that notify a reporter (Bugbop comments and status changes, advisory comments) are listed last on purpose: draft them, read them once more, then send.

## Dedup sweep

`gh search issues` only matches titles reliably. Dump every issue and grep the bodies.

```bash
# Private trackers (needs maintainer access to both repos)
gh issue list --repo fleetdm/security --state all --limit 500 \
  --json number,title,state,createdAt,closedAt,labels,milestone,body,comments > /tmp/sec.json
gh issue list --repo fleetdm/confidential --state all --limit 1000 --label ":security" \
  --json number,title,state,createdAt,closedAt,labels,body > /tmp/conf.json 2>/dev/null || \
gh issue list --repo fleetdm/confidential --state all --limit 1000 \
  --json number,title,state,createdAt,closedAt,labels,body > /tmp/conf.json

# Grep bodies and comments for exact names from the report: endpoint path, query parameter, Go function
python3 - "<term1>" "<term2>" <<'EOF'
import json,re,sys
terms=[re.escape(t) for t in sys.argv[1:]]
for f in ("/tmp/sec.json","/tmp/conf.json"):
    repo="security" if "sec" in f else "confidential"
    for i in json.load(open(f)):
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
python3 scripts/bugbop.py GET /reports --all --summary | grep -iE "<term1>|<term2>"

# Advisory running list (shipped fixes waiting on an advisory)
gh search issues --repo fleetdm/confidential "advisory running list" --state open --json number,title
```

The dedup rule: same root cause and same sink means duplicate, even with a different identifier or endpoint. Same symptom through a different code path is a new issue that cross-references the old one.

## Verify against main

```bash
git fetch -q origin main
git grep -n "<function or route>" origin/main -- server/ ee/
git log --format='%h %ad %s' --date=short -S "<guard or check text>" origin/main -- <file>   # when was it added or removed
git tag -l 'fleet-v4.*.0' | sort -V | while read t; do git merge-base --is-ancestor <sha> $t && { echo "first in $t"; break; }; done
```

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

Product group board numbers and EMs are on the [product groups page](https://fleetdm.com/handbook/company/product-groups); look them up rather than trusting a copy.

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
```

Rules:
- Credits need a GitHub login. A reporter without one is thanked by name in the Credits text.
- Patched version stays empty until the fix ships. Fill it in at release.
- Comments on advisories have no API. Open the advisory page signed in and use the comment box.
- Publishing and "Request CVE" are the CTO's. Leave the advisory as a draft.

## Bugbop

Reads are free. Writes notify the hunter; one at a time, and verify each with a read.

```bash
python3 scripts/bugbop.py GET /reports/<id>                       # includes internal_notes, suspected_duplicate_of_id
python3 scripts/bugbop.py GET /reports/<id>/comments --summary
python3 scripts/bugbop.py POST /reports/<id>/comments --data '{"content":"..."}'
python3 scripts/bugbop.py PATCH /reports/<id> --data '{"status":"open","severity":"medium","internal_notes":"Tracked via https://github.com/fleetdm/security/issues/<n>"}'
python3 scripts/bugbop.py POST /reports/<id>/close_as_duplicate   # same program only; uses suspected_duplicate_of_id
```

`scripts/bugbop.py` lives in the `bugbop:bugbop` plugin skill directory. The script needs `BUGBOP_TOKEN` in the environment; never paste the token into chat.

Bugbop's "Suspected duplicate" banner matches titles, not root causes. Two reports with similar titles can be different bugs. Read both before clicking.

The web UI works when the API is unavailable. The comment composer is a rich-text editor, so paste or type into it rather than setting a field programmatically.
