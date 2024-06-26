
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

## Minor Release (typically end of sprint)

example
```
# Build release candidate and changelogs and QA ticket
./tools/release/publish_release.sh -m

# Do QA until ready to release

# - QA is passed on all teams and ready for release
# - Merge changelog and versions update PR into main
# - git pull main locally with the changelog as the latest commit

# Tag minor
./tools/release/publish_release.sh -mg

# - Wait for build to run

# Publish minor
./tools/release/publish_release.sh -muq

# - Wait for publish process to complete.
# - Merge release article and wait for website to build.
# - When the release article is published, create a LinkedIn post on Fleet's company page. 
# - Copy te LinkedIn post URL as the value for the linkedin_post_url variable in the general_announce_info() function.
# - Go update osquery-slack version

# Announce release
# Change {current_version} to the current version that was just released
# For example, ./tools/release/publish_release.sh -mnu -v 4.50.0
./tools/release/publish_release.sh -mnu -v {current_version}
```

...
:cloud: :rocket: The latest version of Fleet is 4.50.0.
More info: https://github.com/fleetdm/fleet/releases/tag/fleet-v4.50.0
Release article: https://fleetdm.com/releases/fleet-4.50.0
LinkedIn post: https://www.linkedin.com/feed/update/urn:li:activity:7199509896705232898/
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

