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

Make sure the repo is set to default (Needed only once) 
```
  gh repo set-default
```


## Before you begin

Make sure all tickets are tagged with the correct milestone.

Filter by both the milestone you expect and also double check `no milestone` to make sure you haven't missed anything, and that all tickets are in the correct GitHub Projects column.

Make sure you are on the `main` branch, and that you have all updates pulled locally.

> Add the `-d` flag to any command to dry run and test the next step.


## Minor Release

**1. Create release candidate**

```sh
./tools/release/publish_release.sh -m
```

This will create the minor release candidate branch, changelog PRs (`main` and RC branch), and the release QA issue.


**2. Complete quality assurance**

A Quality Assurance Engineer from each product group needs to [confirm that their product group is ready](https://fleetdm.com/handbook/engineering#indicate-your-product-group-is-release-ready) before proceeding to the next step.


**3. Merge changelog and version bump**

Finalize and merge the two PRs. Check out the RC branch locally and pull the latest updates so that the changelog commit is the most recent commit on the branch.

**4. Tag minor release**

```sh
./tools/release/publish_release.sh -mg
```

Wait for build to run, which typically takes about fifteen minutes. 


**5. Publish minor release**

```sh
./tools/release/publish_release.sh -muq
```

Wait for publish process to complete.

**6. Update the fleetdm/terraform and fleetdm/fleet-gitops repos**

Update all Fleet version references in our [fleetdm/terraform](https://github.com/fleetdm/fleet-terraform) repo and submit a PR. Then update `DEFAULT_FLEETCTL_VERSION` in `.github/gitops-action/action.yml` in [fleetdm/fleet-gitops](https://github.com/fleetdm/fleet-gitops) and submit a PR.


**7. Merge milestone pull requests**

Merge any pull requests associated with this release milestone, which include reference documentation, feature guides, and a release announcement article. 

Wait for the release article to appear on the [Fleet articles page](https://fleetdm.com/articles).


**8. Post to LinkedIn company page**

When the release article is published, create a LinkedIn post on Fleet's company page. Copy the previous version announcement post, update the headline features and associated emojis, and update the URL to point to the new release article. Publish the LinkedIn post, then copy a link to the post. 

Open the Fleet releaser script and search for a variable called `linkedin_post_url`. Change the associated value to the new LinkedIn post URL. 


**9. Announce release**

```sh
./tools/release/publish_release.sh -mnu -v {current_version}
```

Change `{current_version}` to the current version that was just released. For example: `./tools/release/publish_release.sh -mnu -v 4.50.0`. 

Open the Fleet channels in the osquery Slack and MacAdmins Slack and update the topic to point to the new release. 


**10. Conclude the milestone**

Complete the [conclude the milestone ritual](https://fleetdm.com/handbook/engineering#conclude-current-milestone).


## Patch release

**1. Confirm commits, milestones, and cherry-picks**

Ensure all tickets that you want in the patch release are milestoned to that release, and all PRs into `main` for associated work, including follow-ups, are linked to an issue with the milestone. For confidential issues, create a placeholder issue (e.g. https://github.com/fleetdm/fleet/issues/39483) to link against, as the patch release tooling does not interact with the confidential repo.

If a higher-version minor release RC is currently active, make sure all chosen commits are also in the RC branch, either because they were made before the RC cut or because they were cherry-picked in after the cut. For example, any commit landing in 4.80.1 should also land in 4.81.0, so when a new minor release goes GA users upgrading from a patch release don't experience regressions.

**2. Create release candidate**

```sh
./tools/release/publish_release.sh
```

This will create the patch release candidate branch, changelog PRs (`main` and RC branch), and the release QA issue.


**3. Complete quality assurance**

A Quality Assurance Engineer from each product group needs to [confirm that their product group is ready](https://fleetdm.com/handbook/engineering#indicate-your-product-group-is-release-ready) before proceeding to the next step.


**4. Merge changelog and version bump**

Finalize and merge the two PRs. Check out the RC branch locally and pull the latest updates so that the changelog commit is the most recent commit on the branch.


**5. Tag patch release**

```sh
./tools/release/publish_release.sh -g
```

Wait for build to run, which typically takes about fifteen minutes. 


**6. Publish patch release**

```sh
./tools/release/publish_release.sh -u
```

> During the publish process, the release script will attempt to publish `fleetctl` to NPM. If this times out or otherise fails, you need to publish to NPM manually. From the `/tools/fleetctl-npm/` directory, run `npm publish`.

**7. Update the fleetdm/terraform and fleetdm/fleet-gitops repos**

Update all Fleet version references in our [fleetdm/terraform](https://github.com/fleetdm/fleet-terraform) repo and submit a PR. Then, if this release is _not_ a backport, update `DEFAULT_FLEETCTL_VERSION` in `.github/gitops-action/action.yml` in [fleetdm/fleet-gitops](https://github.com/fleetdm/fleet-gitops) and submit a PR.

**8. Announce the release**

The release script will announce the patch in the #general channel. Open the Fleet channels in the osquery Slack and MacAdmins Slack and update the topic to point to the new release. 


**9. Conclude the milestone**

Complete the [conclude the milestone ritual](https://fleetdm.com/handbook/engineering#conclude-current-milestone).

## Using the Backport check script

```sh
./tools/release/backport-check.sh fleet-v4.81.0 rc-patch-fleet-v4.81.1 rc-minor-fleet-v4.82.0
```

Example output
```
Indexing rc-minor-fleet-v4.82.0 since merge-base with fleet-v4.81.0 (6e9d46202e9b54e1b83542179f40ed880586d3f4)...

=== INCLUDED (present on rc-minor-fleet-v4.82.0) ===
STATUS     PATCH_SHA     MINOR_SHA     MATCH     SUBJECT
--------   ------------  ------------  --------  ----------------------------------------
INCLUDED   8798f6cee5ea  b1d0a5c2da8c  subject   End user UI: Update logo loading spinner styling (#39234)
INCLUDED   67c2d503f23d  fa4b7426f1db  subject   End user /enroll page for macOS: Download button should have semi-bold font weight (#39301)
INCLUDED   1f09d64f4df7  fac6ca5f2afd  subject   Add ellipsis to cut-off placeholder text in search fields (#39112)
INCLUDED   a3c6f7e3a1c6  ba1862595e4e  subject   Add route for Microsoft Entra home page (for tenant ID) (#39216)
...
INCLUDED   4ec78002e0d6  3b825449757f  subject   Set secure cookie in SSO callback (#40765) (#40806)

=== MISSING (not found on rc-minor-fleet-v4.82.0) ===
STATUS     PATCH_SHA     SUBJECT
--------   ------------  ----------------------------------------
MISSING    cb08454ddfef  improve windows resending (#40365)
MISSING    3ef6f40a10fc  Batch select query in CleanupExcessQueryResultRows (#40491)
MISSING    9e64a65f29c9  Improved validation for packages (#40407)
MISSING    5df0a554fe1d  Add migration to update host_certificates_template UUID column size (… (#40709)
MISSING    82890f883369  Exorcise edited_enroll_secrets from v4.81.1 (#40714)
MISSING    1adbfb89429b  Cherry-pick: Remove "do not enqueue setup experience items >24 hours after enrollment" logic for macOS hosts (#40739) (#40748)
MISSING    ac068800375a  Adding changes for Fleet v4.81.1 (#40704)
MISSING    729c324074c3  Avoid panics on VPP install command errors when command not initiated by Fleet VPP install -> 4.81.0 (#40395)
```

For each item in missing you can verify manually by opening the PR's at the end of each line to trace back to the original issue and validate if a related PR made it into the minor branch you are targeting.

To include any code missed checkout the minor branch, branch off into a new branch to pull them in and then `git cherry-pick <patch_sha>` to add the missing code. Resolve any conflicts and open a PR based off the minor branch.
