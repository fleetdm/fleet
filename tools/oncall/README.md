# Oncall

You can use the `oncall.sh` script to find out if there are any open issues or PRs from the community:
```sh
gh auth login
./tools/oncall/oncall.sh issues
./tools/oncall/oncall.sh prs
./tools/oncall/oncall.sh prs -v
```

`prs` columns: number | opened | issue | author | title

With `-v`: number | opened | issue | tested | author | assignee | link | title | labels

With `-s`: one Slack mrkdwn line per PR, with linked PR and issue numbers and short dates (used by the `oncall-community-prs.yml` workflow).
