#!/usr/bin/env bash
# Fleet project board helper over the REST API (gh project's GraphQL calls hit secondary rate limits
# when several agents share a token). Status names match without their emoji, case-insensitive.
#   board.sh bugs <project> [status]          TSV of bug items: number, status, assignees, labels, title
#   board.sh statuses <project>               Status column names
#   board.sh move <project> <status> <issue>...
set -euo pipefail
cmd=$1 proj=$2
base=/orgs/fleetdm/projectsV2/$proj
norm='ascii_downcase | sub("^[^a-z]+"; "")'

field=$(gh api "$base/fields" --paginate --jq '.[] | select(.name == "Status")')
fid=$(jq -r .id <<<"$field")
items() {
  gh api "$base/items?per_page=100&fields=$fid" --paginate --jq '.[] | {
    id, n: .content.number, title: .content.title,
    labels: [.content.labels[]?.name], assignees: [.content.assignees[]?.login],
    status: ([.fields[]? | select(.name == "Status") | .value.name.raw // .value.name] | first // "")}'
}

case $cmd in
bugs)
  items | jq -r --arg s "${3:-}" "
    select(.labels | index(\"bug\"))
    | select(\$s == \"\" or (.status | $norm) == (\$s | $norm))
    | [.n, .status, (.assignees | join(\",\")), (.labels | join(\",\")), .title] | @tsv"
  ;;
statuses)
  jq -r '.options[] | .name.raw // .name' <<<"$field"
  ;;
move)
  want=$3; shift 3
  opt=$(jq -r --arg w "$want" "[.options[] | select((.name.raw // .name | $norm) == (\$w | $norm))] | if length == 1 then .[0].id else error(\"no single status named \" + \$w) end" <<<"$field")
  snap=$(items)
  for n in "$@"; do
    iid=$(jq -r --argjson n "$n" 'select(.n == $n) | .id' <<<"$snap")
    if [ -z "$iid" ]; then echo "#$n not on board $proj" >&2; continue; fi
    gh api -X PATCH "$base/items/$iid" --input - >/dev/null <<<"{\"fields\":[{\"id\":$fid,\"value\":\"$opt\"}]}" && echo "#$n -> $want"
  done
  ;;
*) echo "unknown command: $cmd" >&2; exit 2 ;;
esac
