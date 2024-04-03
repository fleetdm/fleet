
# Releasing Fleet

## Setup

This script release requires various secrets to utilize chat GPT for formatting
as well as posting to Slack channels automatically

```
  OPEN_API_KEY           Open API key used for fallback if not provided via -o or --open-api-key option
  SLACK_GENERAL_TOKEN    Slack token to publish via curl to #general
  SLACK_HELP_INFRA_TOKEN Slack token to publish via curl to #help-infrastructure
  SLACK_HELP_ENG_TOKEN   Slack token to publish via curl to #help-engineering
```

This requires:
 `jq` `gh` `git` `curl` `awk` `sed` `make` `ack` `grep`

The script will check that each of these are installed and available before running

## Before running the script

Make sure all tickets are tagged with the correct milestone.

I recommend filtering by both the milestone you expect and also double check `no milestone` to make sure you haven't missed anything

For example no tickets still in Ready / In Progress should be in the milestone we are about to release.

## Main Release (end of sprint)

example
```
# Build release candidate and changelogs and QA ticket
./tools/release/publish_release.sh -a
# Do QA until ready to release

# QA is passed on all teams and ready for release

# Tag main
./tools/release/publish_release.sh -ag
# Publish main
./tools/release/publish_release.sh -au
# Go update osquery-slack version
```

...
TODO example output
...


## Patch Release (end of week / critical)

example
```
# Build release candidate and changelogs and QA ticket
./tools/release/publish_release.sh
# Do QA until ready to release

# QA is passed on all teams and ready for release

# Tag patch
./tools/release/publish_release.sh -g
# Publish patch
./tools/release/publish_release.sh -u
# Go update osquery-slack version
```

...
TODO example output
...

